package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"time"
)

var _ = net.ListenUDP

// forwardResp holds the upstream response header and the raw answer bytes.
type forwardResp struct {
	header      *DNSHeader
	answerBytes []byte
	anCount     uint16
}

func main() {
	resolver := flag.String("resolver", "", "upstream resolver address (ip:port)")
	flag.Parse()
	if *resolver == "" {
		fmt.Println("Usage: ./your_server --resolver <ip:port>")
		return
	}

	if err := runServer("127.0.0.1:2053", *resolver); err != nil {
		fmt.Println("server error:", err)
	}
}

// runServer binds to listenAddr and forwards queries to resolverAddr.
func runServer(listenAddr, resolverAddr string) error {
	udpAddr, err := net.ResolveUDPAddr("udp", listenAddr)
	if err != nil {
		return fmt.Errorf("resolve listen addr: %w", err)
	}
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("bind listen addr: %w", err)
	}
	defer udpConn.Close()

	rand.Seed(time.Now().UnixNano())
	buf := make([]byte, 4096)

	fmt.Println("DNS forwarder listening on", listenAddr, "forwarding to", resolverAddr)

	for {
		n, src, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("read error:", err)
			continue
		}
		// copy packet for goroutine safety
		pkt := make([]byte, n)
		copy(pkt, buf[:n])

		go handleRequest(pkt, src, udpConn, resolverAddr)
	}
}

// handleRequest orchestrates parsing, forwarding and responding.
func handleRequest(pkt []byte, src *net.UDPAddr, udpConn *net.UDPConn, resolverAddr string) {
	reqHeader := ParseHeader(pkt)
	if reqHeader == nil {
		fmt.Println("Failed to parse header (packet too small)")
		return
	}

	questions, err := parseQuestionsFromPacket(pkt, int(reqHeader.QDCount))
	if err != nil {
		fmt.Println("Failed to parse questions:", err)
		return
	}

	// Forward each question, collect forwardResponses
	forwardResponses := make([]forwardResp, 0, len(questions))
	var firstRespHeader *DNSHeader
	success := true

	for _, q := range questions {
		fr, err := forwardQuestion(q, reqHeader, resolverAddr)
		if err != nil {
			fmt.Println("forward error:", err)
			success = false
			break
		}
		forwardResponses = append(forwardResponses, fr)
		if firstRespHeader == nil {
			firstRespHeader = fr.header
		}
	}

	// If forwarding failed or we collected no answers — return SERVFAIL with original questions
	if !success || len(forwardResponses) == 0 {
		if err := sendSERVFAIL(udpConn, src, reqHeader, questions); err != nil {
			fmt.Println("failed to send SERVFAIL:", err)
		}
		return
	}

	// Build a merged reply and send it
	out := buildMergedResponse(reqHeader, questions, forwardResponses, firstRespHeader)
	if _, err := udpConn.WriteToUDP(out, src); err != nil {
		fmt.Println("Failed to send response to client:", err)
	}
}

// parseQuestionsFromPacket parses qCount questions starting at offset 12.
func parseQuestionsFromPacket(pkt []byte, qCount int) ([]*DNSQuestion, error) {
	offset := 12
	questions := make([]*DNSQuestion, 0, qCount)
	for i := 0; i < qCount; i++ {
		q, nextOff, err := ParseQuestion(pkt, offset)
		if err != nil {
			return nil, err
		}
		questions = append(questions, q)
		offset = nextOff
	}
	return questions, nil
}

// forwardQuestion sends a single-question DNS request to resolver and returns the upstream answer section bytes.
func forwardQuestion(q *DNSQuestion, origHeader *DNSHeader, resolverAddr string) (forwardResp, error) {
	// Build forward header (one-question query)
	fwdHeader := DNSHeader{}
	fwdHeader.AddID(uint16(rand.Intn(0x10000)))
	fwdHeader.Flags = 0
	fwdHeader.AddQR(QueryTypeQuery)
	fwdHeader.AddOPCODE(origHeader.Opcode())
	fwdHeader.AddRD(origHeader.RecursionDesired())
	fwdHeader.AddQDCOUNT(1)
	fwdHeader.AddANCOUNT(0)
	fwdHeader.AddNSCOUNT(0)
	fwdHeader.AddARCOUNT(0)

	headerBytes := headerToBytes(&fwdHeader)
	out := append(headerBytes, q.Bytes()...)

	raddr, err := net.ResolveUDPAddr("udp", resolverAddr)
	if err != nil {
		return forwardResp{}, fmt.Errorf("resolve resolver addr: %w", err)
	}

	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return forwardResp{}, fmt.Errorf("dial resolver: %w", err)
	}
	defer conn.Close()

	conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	if _, err := conn.Write(out); err != nil {
		return forwardResp{}, fmt.Errorf("write to resolver: %w", err)
	}

	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	respBuf := make([]byte, 4096)
	n, err := conn.Read(respBuf)
	if err != nil {
		return forwardResp{}, fmt.Errorf("read from resolver: %w", err)
	}
	resp := respBuf[:n]

	respHeader := ParseHeader(resp)
	if respHeader == nil {
		return forwardResp{}, fmt.Errorf("invalid resolver response header")
	}

	// Advance past ALL questions in the resolver response (respHeader.QDCount)
	off := 12
	for i := 0; i < int(respHeader.QDCount); i++ {
		_, newOff, err := ParseQuestion(resp, off)
		if err != nil {
			return forwardResp{}, fmt.Errorf("parsing resolver question: %w", err)
		}
		off = newOff
	}

	if off > len(resp) {
		return forwardResp{}, fmt.Errorf("resolver response malformed (answers start beyond length)")
	}

	// Everything after 'off' is the answer/authority/additional sections as raw bytes.
	answerBytes := make([]byte, len(resp)-off)
	copy(answerBytes, resp[off:])

	return forwardResp{
		header:      respHeader,
		answerBytes: answerBytes,
		anCount:     respHeader.ANCount,
	}, nil
}

// buildMergedResponse constructs the final DNS response sent to the original client.
// It preserves the original request ID and question section, and appends all answers collected.
func buildMergedResponse(reqHeader *DNSHeader, questions []*DNSQuestion, frs []forwardResp, firstResp *DNSHeader) []byte {
	merged := DNSHeader{}
	merged.ID = reqHeader.ID // preserve original ID
	merged.Flags = 0
	merged.AddQR(QueryTypeReply)
	merged.AddOPCODE(reqHeader.Opcode())
	merged.AddAA(0)
	merged.AddTC(0)
	merged.AddRD(reqHeader.RecursionDesired())

	if firstResp != nil {
		ra := byte((firstResp.Flags >> 7) & 1)
		merged.AddRA(ra)
		rcode := byte(firstResp.Flags & 0x0F)
		merged.AddRCODE(rcode)
	} else {
		merged.AddRA(0)
		merged.AddRCODE(ResponseCodeServerFailure)
	}

	merged.AddZ(0)
	merged.AddQDCOUNT(uint16(len(questions)))

	// collect answers and sum ANCOUNT
	totalAnswers := uint16(0)
	answersOut := make([]byte, 0)
	for _, fr := range frs {
		totalAnswers += fr.anCount
		answersOut = append(answersOut, fr.answerBytes...)
	}
	merged.AddANCOUNT(totalAnswers)
	merged.AddNSCOUNT(0)
	merged.AddARCOUNT(0)

	// assemble final bytes
	out := make([]byte, 0, 512)
	out = append(out, headerToBytes(&merged)...)
	for _, q := range questions {
		out = append(out, q.Bytes()...)
	}
	out = append(out, answersOut...)
	return out
}

// sendSERVFAIL constructs a minimal SERVFAIL reply with the original questions and sends it.
func sendSERVFAIL(udpConn *net.UDPConn, dst *net.UDPAddr, reqHeader *DNSHeader, questions []*DNSQuestion) error {
	h := DNSHeader{}
	h.ID = reqHeader.ID
	h.Flags = 0
	h.AddQR(QueryTypeReply)
	h.AddOPCODE(reqHeader.Opcode())
	h.AddRD(reqHeader.RecursionDesired())
	h.AddRA(0)
	h.AddRCODE(ResponseCodeServerFailure)
	h.AddQDCOUNT(uint16(len(questions)))
	h.AddANCOUNT(0)
	h.AddNSCOUNT(0)
	h.AddARCOUNT(0)

	out := make([]byte, 0, 512)
	out = append(out, headerToBytes(&h)...)
	for _, q := range questions {
		out = append(out, q.Bytes()...)
	}
	_, err := udpConn.WriteToUDP(out, dst)
	return err
}

// headerToBytes serializes the DNSHeader into 12 bytes.
func headerToBytes(h *DNSHeader) []byte {
	b := make([]byte, 12)
	binary.BigEndian.PutUint16(b[0:2], h.ID)
	binary.BigEndian.PutUint16(b[2:4], h.Flags)
	binary.BigEndian.PutUint16(b[4:6], h.QDCount)
	binary.BigEndian.PutUint16(b[6:8], h.ANCount)
	binary.BigEndian.PutUint16(b[8:10], h.NSCount)
	binary.BigEndian.PutUint16(b[10:12], h.ARCount)
	return b
}
