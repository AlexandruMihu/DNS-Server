package main

import "encoding/binary"

type forwardResp struct {
	header      *DNSHeader
	answerBytes []byte
	anCount     uint16
}

func buildMergedResponse(reqHeader *DNSHeader, questions []*DNSQuestion, frs []forwardResp, firstResp *DNSHeader) []byte {
	merged := DNSHeader{}
	merged.ID = reqHeader.ID 
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

	totalAnswers := uint16(0)
	answersOut := make([]byte, 0)
	for _, fr := range frs {
		totalAnswers += fr.anCount
		answersOut = append(answersOut, fr.answerBytes...)
	}
	merged.AddANCOUNT(totalAnswers)
	merged.AddNSCOUNT(0)
	merged.AddARCOUNT(0)

	out := make([]byte, 0, 512)
	out = append(out, headerToBytes(&merged)...)
	for _, q := range questions {
		out = append(out, q.Bytes()...)
	}
	out = append(out, answersOut...)
	return out
}
