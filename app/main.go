package main

import (
	"fmt"
	"net"
)
// Akashisang
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
	
		receivedData := string(buf[:size])
		fmt.Printf("Received %d bytes from %s: %s\n", size, source, receivedData)
	
		var response DNSResponse

		header := &response.Header

		header.AddID(1234)
		header.AddQR(1)
		header.AddOPCODE(0)
		header.AddAA(0)
		header.AddTC(0)
		header.AddRA(0)
		header.AddZ(0)
		header.AddRCODE(0)
		header.AddQDCOUNT(1)
		header.AddANCOUNT(0)
		header.AddNSCOUNT(0)
		header.AddARCOUNT(0)

		question :=response.Question

		question.AddName("codecrafters.io")
		question.AddType(QuestionTypeA)
		question.AddClass(QuestionClassIN)

		_, err = udpConn.WriteToUDP(response.Bytes(), source)
		if err != nil {
			fmt.Println("Failed to send response:", err)
		}
	}
}
