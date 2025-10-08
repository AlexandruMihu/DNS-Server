package main

import (
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"
)

var _ = net.ListenUDP

func main() {

	resolver := flag.String("resolver", "", "upstream resolver address (ip:port)")
	flag.Parse()
	if *resolver == "" {
		fmt.Println("Usage: ./your_server --resolver <ip:port>")
		return
	}

	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:2053")
	if err != nil {
		fmt.Println("Failed to resolve UDP address:", err)
		return
	}
	
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		fmt.Println("Failed to bind to address:", err)
		return
	}
	defer udpConn.Close()
	
	rand.Seed(time.Now().UnixNano())

	buf := make([]byte, 4096)
	
	fmt.Println("DNS forwarder listening on 127.0.0.1:2053, forwarding to", *resolver)

	for {
		size, source, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error receiving data:", err)
			continue
		}
		receivedData := make([]byte, size)
		copy(receivedData, buf[:size])
		go handlePacket(receivedData, source, udpConn, *resolver)
	}
}

func handlePacket(pkt []byte, source *net.UDPAddr, listenConn *net.UDPConn, resolverAddr string) {
	reqHeader := ParseHeader(pkt)
	if reqHeader == nil {
		fmt.Println("Failed to parse header (packet too small)")
		return
	}

	qCount := int(reqHeader.QDCount)
	offset := 12
	questions := make([]*DNSQuestion, 0, qCount)
	for i := 0; i < qCount; i++ {
		q, nextOff, perr := ParseQuestion(pkt, offset)
		if perr != nil {
			fmt.Println("Failed to parse question:", perr)
			return
		}
		questions = append(questions, q)
		offset = nextOff
	}
	if len(questions) != qCount {
		fmt.Println("Did not parse expected number of questions")
		return
	}

	// We will forward each question separately to resolver (resolver expects 1 question),
	// then collect answer sections and merge them back into a single response to the client.

	type forwardResp struct {
		header     *DNSHeader
		answerBytes []byte
		anCount    uint16
	}

	forwardResponses := make([]forwardResp, 0, len(questions))
	var firstRespHeader *DNSHeader

	// perform per-question forwards
	for _, q := range questions {
		fwdID := uint16(rand.Intn(0x10000))
		fwdHeader := DNSHeader{}
		// Build header: ID=fwdID, QR=0 (query), OPCODE same as original, RD same as original
		fwdHeader.AddID(fwdID)
		fwdHeader.Flags = 0
		fwdHeader.AddQR(QueryTypeQuery) // QR = 0
		fwdHeader.AddOPCODE(reqHeader.Opcode())
		fwdHeader.AddRD(reqHeader.RecursionDesired())
		fwdHeader.AddQDCOUNT(1)
		fwdHeader.AddANCOUNT(0)
		fwdHeader.AddNSCOUNT(0)
		fwdHeader.AddARCOUNT(0)

		// assemble packet
		headerBytes := make([]byte, 12)
		binary.BigEndian.PutUint16(headerBytes[0:2], fwdHeader.ID)
		binary.BigEndian.PutUint16(headerBytes[2:4], fwdHeader.Flags)
		binary.BigEndian.PutUint16(headerBytes[4:6], fwdHeader.QDCount)
		binary.BigEndian.PutUint16(headerBytes[6:8], fwdHeader.ANCount)
		binary.BigEndian.PutUint16(headerBytes[8:10], fwdHeader.NSCount)
		binary.BigEndian.PutUint16(headerBytes[10:12], fwdHeader.ARCount)

		out := make([]byte, 0, 512)
		out = append(out, headerBytes...)
		out = append(out, q.Bytes()...)

		// send to resolver
		raddr, err := net.ResolveUDPAddr("udp", resolverAddr)
		if err != nil {
			fmt.Println("Failed to resolve resolver address:", err)
			return
		}
		conn, err := net.DialUDP("udp", nil, raddr)
		if err != nil {
			fmt.Println("Failed to dial resolver:", err)
			return
		}

		conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
		_, err = conn.Write(out)
		if err != nil {
			fmt.Println("Failed to send to resolver:", err)
			conn.Close()
			return
		}

		conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		respBuf := make([]byte, 4096)
		n, err := conn.Read(respBuf)
		conn.Close()
		if err != nil {
			fmt.Println("Timeout/failed waiting for resolver response:", err)
			return
		}
		resp := respBuf[:n]

		respHeader := ParseHeader(resp)
		if respHeader == nil {
			fmt.Println("Resolver sent truncated/invalid header")
			return
		}
		if respHeader.QDCount != 1 {
			// unexpected from resolver
			fmt.Println("Resolver response QDCount != 1, skipping")
			return
		}

		// parse its question to find where answers start
		off := 12
		_, off2, perr := ParseQuestion(resp, off)
		if perr != nil {
			fmt.Println("Failed to parse question in resolver response:", perr)
			return
		}
		answersStart := off2
		if answersStart > len(resp) {
			fmt.Println("Resolver response malformed (answersStart beyond length)")
			return
		}
		answerBytes := make([]byte, len(resp)-answersStart)
		copy(answerBytes, resp[answersStart:])

		forwardResponses = append(forwardResponses, forwardResp{
			header:     respHeader,
			answerBytes: answerBytes,
			anCount:    respHeader.ANCount,
		})
		if firstRespHeader == nil {
			firstRespHeader = respHeader
		}
	}

	// build merged response to original client
	mergedHeader := DNSHeader{}
	mergedHeader.ID = reqHeader.ID // preserve original ID (very important)
	// Construct flags:
	// - QR = reply
	// - OPCODE = same as original
	// - RD = original RD
	// - RA and RCODE we copy from the resolver response (first resp)
	mergedHeader.Flags = 0
	mergedHeader.AddQR(QueryTypeReply)         // QR = 1
	mergedHeader.AddOPCODE(reqHeader.Opcode()) // OPCODE
	mergedHeader.AddAA(0)
	mergedHeader.AddTC(0)
	mergedHeader.AddRD(reqHeader.RecursionDesired())
	if firstRespHeader != nil {
		// RA from resolver
		ra := byte((firstRespHeader.Flags >> 7) & 1)
		mergedHeader.AddRA(ra)
		// RCODE from resolver (low 4 bits)
		rcode := byte(firstRespHeader.Flags & 0x0F)
		mergedHeader.AddRCODE(rcode)
	} else {
		mergedHeader.AddRA(0)
		mergedHeader.AddRCODE(ResponseCodeServerFailure)
	}

	mergedHeader.AddZ(0)
	mergedHeader.AddQDCOUNT(uint16(len(questions)))

	// append all answers bytes and sum ANCOUNT
	totalAnswers := uint16(0)
	answersOut := make([]byte, 0)
	for _, fr := range forwardResponses {
		totalAnswers += fr.anCount
		answersOut = append(answersOut, fr.answerBytes...)
	}
	mergedHeader.AddANCOUNT(totalAnswers)
	mergedHeader.AddNSCOUNT(0)
	mergedHeader.AddARCOUNT(0)

	// create header bytes
	headerBytes := make([]byte, 12)
	binary.BigEndian.PutUint16(headerBytes[0:2], mergedHeader.ID)
	binary.BigEndian.PutUint16(headerBytes[2:4], mergedHeader.Flags)
	binary.BigEndian.PutUint16(headerBytes[4:6], mergedHeader.QDCount)
	binary.BigEndian.PutUint16(headerBytes[6:8], mergedHeader.ANCount)
	binary.BigEndian.PutUint16(headerBytes[8:10], mergedHeader.NSCount)
	binary.BigEndian.PutUint16(headerBytes[10:12], mergedHeader.ARCount)

	out := make([]byte, 0, 512)
	out = append(out, headerBytes...)
	// append original questions (so pointers in answers referencing offset 12 remain valid)
	for _, q := range questions {
		out = append(out, q.Bytes()...)
	}
	// append answers collected
	out = append(out, answersOut...)

	_, err := listenConn.WriteToUDP(out, source)
	if err != nil {
		fmt.Println("Failed to send response to client:", err)
	}
}