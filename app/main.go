package main

import (
	"fmt"
	"net"
)
// Ensures gofmt doesn't remove the "net" import in stage 1 (feel free to remove this!)
var _ = net.ListenUDP

func main() {

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
	
	buf := make([]byte, 512)
	
	for {
		size, source, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error receiving data:", err)
			break
		}

		receivedData := buf[:size] // <-- keep as []byte, not string
		fmt.Printf("Received %d bytes from %s\n", size, source)

		reqHeader := ParseHeader(receivedData)
		if reqHeader == nil {
			fmt.Println("Failed to parse header (packet too small)")
			continue
		}

		respCode := ResponseCodeNoError
		if reqHeader.Opcode() != OpcodeQuery {
			respCode = ResponseCodeNotImplemented
		}

		qCount := int(reqHeader.QDCount)
		offset := 12
		questions := make([]*DNSQuestion, 0, qCount)
		for i := 0; i < qCount; i++ {
			q, nextOff, perr := ParseQuestion(receivedData, offset)
			if perr != nil {
				fmt.Println("Failed to parse question:", perr)
				break
			}
			questions = append(questions, q)
			offset = nextOff
		}
		// if we didn't parse as many as advertised, treat as error
		if len(questions) != qCount {
			fmt.Println("Did not parse expected number of questions")
			continue
		}

		var respHeader DNSHeader
		h := &respHeader
		h.AddID(reqHeader.ID)
		h.AddQR(QueryTypeReply)
		h.AddOPCODE(reqHeader.Opcode())
		h.AddAA(boolToByte(false))
		h.AddTC(boolToByte(false))
		h.AddRD(reqHeader.RecursionDesired())
		h.AddRA(0)
		h.AddZ(boolToByte(false))
		h.AddRCODE(respCode)

		// Mirror the number of questions and provide one answer per question
		h.AddQDCOUNT(reqHeader.QDCount)
		h.AddANCOUNT(uint16(len(questions)))
		h.AddNSCOUNT(0)
		h.AddARCOUNT(0)

		// Header bytes
		headerBytes := make([]byte, 12)
		binary.BigEndian.PutUint16(headerBytes[0:2], h.ID)
		binary.BigEndian.PutUint16(headerBytes[2:4], h.Flags)
		binary.BigEndian.PutUint16(headerBytes[4:6], h.QDCount)
		binary.BigEndian.PutUint16(headerBytes[6:8], h.ANCount)
		binary.BigEndian.PutUint16(headerBytes[8:10], h.NSCount)
		binary.BigEndian.PutUint16(headerBytes[10:12], h.ARCount)

		out := make([]byte, 0, 512)
		out = append(out, headerBytes...)

		// Append each question (uncompressed)
		for _, q := range questions {
			out = append(out, q.Bytes()...)
		}

		// Prepare answers: one A-record per question (A, IN, TTL=60, RDLEN=4, RDATA=127.0.0.1)
		ip, _ := ipToBytes("127.0.0.1")
		for _, q := range questions {
			var ans DNSAnswer
			ans.AddQuestion(q)
			ans.AddTTL(60)
			ans.AddDataLength(4)
			ans.AddData(ip)
			out = append(out, ans.Bytes()...)
		}

		// Send the response
		_, err = udpConn.WriteToUDP(out, source)
		if err != nil {
			fmt.Println("Failed to send response:", err)
		}
	}
}
