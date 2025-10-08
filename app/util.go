package main

import "net"

type DNSAnswer struct {
	Question   DNSQuestion
	TimeToLive uint32
	DataLength uint16
	Data       []byte
}

type forwardResp struct {
	header      *DNSHeader
	answerBytes []byte
	anCount     uint16
}

type DNSHeader struct {
	ID      uint16
	Flags   uint16
	QDCount uint16
	ANCount uint16
	NSCount uint16
	ARCount uint16
}

type QuestionType uint16

type QuestionClass uint16

type DNSQuestion  struct {
	DomainName string
	Type       QuestionType
	Class      QuestionClass
}

func ipToBytes(ip string) ([]byte, error) {
	return net.ParseIP(ip).To4(), nil
}

func boolToByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}

const (
	QuestionClassIN QuestionClass = 1
)

const (
	OpcodeQuery  = 0
	OpcodeIQuery = 1 
	OpcodeStatus = 2 
)

const (
	ResponseCodeNoError        byte = 0
	ResponseCodeFormatError    byte = 1
	ResponseCodeServerFailure  byte = 2
	ResponseCodeNameError      byte = 3
	ResponseCodeNotImplemented byte = 4
	ResponseCodeRefused        byte = 5
)

const (
	QueryTypeQuery = 0
	QueryTypeReply = 1
)

const (
	QuestionTypeA     QuestionType = 1
	QuestionTypeNS    QuestionType = 2
	QuestionTypeMD    QuestionType = 3 
	QuestionTypeMF    QuestionType = 4 
	QuestionTypeCNAME QuestionType = 5
	QuestionTypeSOA   QuestionType = 6
	QuestionTypeWKS   QuestionType = 11
	QuestionTypePTR   QuestionType = 12
	QuestionTypeMX    QuestionType = 15
	QuestionTypeTXT   QuestionType = 16
)
