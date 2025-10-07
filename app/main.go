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

		var response DNSResponse

		header := &response.Header

		header.AddID(reqHeader.ID)
		header.AddQR(QueryTypeReply)
		header.AddOPCODE(reqHeader.Opcode())
		header.AddAA(boolToByte(false))
		header.AddTC(boolToByte(false))
		header.AddRD(reqHeader.RecursionDesired())
		header.AddZ(boolToByte(false))
		header.AddRCODE(respCode)
		header.AddQDCOUNT(reqHeader.QDCount)
		header.AddANCOUNT(1)
		header.AddNSCOUNT(0)
		header.AddARCOUNT(0)

		question := &response.Question
		
		question.AddName(parsedQuestion.DomainName)
		question.AddType(parsedQuestion.Type)
		question.AddClass(parsedQuestion.Class)

		ip, err := ipToBytes("127.0.0.1")
		if err != nil {
			fmt.Println("Failed to convert IP to bytes:", err)
			continue
		}
        
		answer:=&response.Answer

		answer.AddQuestion(question)
		answer.AddTTL(60)
		answer.AddDataLength(4)
		answer.AddData(ip)

		_, err = udpConn.WriteToUDP(response.Bytes(), source)
		if err != nil {
			fmt.Println("Failed to send response:", err)
		}
	}
}
