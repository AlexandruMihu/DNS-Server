package main

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
		pkt := make([]byte, n)
		copy(pkt, buf[:n])

		go handleRequest(pkt, src, udpConn, resolverAddr)
	}
}

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

	if !success || len(forwardResponses) == 0 {
		if err := sendSERVFAIL(udpConn, src, reqHeader, questions); err != nil {
			fmt.Println("failed to send SERVFAIL:", err)
		}
		return
	}

	out := buildMergedResponse(reqHeader, questions, forwardResponses, firstRespHeader)
	if _, err := udpConn.WriteToUDP(out, src); err != nil {
		fmt.Println("Failed to send response to client:", err)
	}
}

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

