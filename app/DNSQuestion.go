package main
//Jlrine2
import "strings"

const (
	QuestionTypeA     QuestionType = 1
	QuestionTypeNS    QuestionType = 2
	QuestionTypeMD    QuestionType = 3 // Obsolete Use MX RFC 1035
	QuestionTypeMF    QuestionType = 4 // Obsolete Use MX RFC 1035
	QuestionTypeCNAME QuestionType = 5
	QuestionTypeSOA   QuestionType = 6
	QuestionTypeWKS   QuestionType = 11 // Experimental RFC 1035
	QuestionTypePTR   QuestionType = 12
	QuestionTypeMX    QuestionType = 15
	QuestionTypeTXT   QuestionType = 16
)

type QuestionClass uint16

const (
	QuestionClassIN QuestionClass = 1
)

type Question struct {
	DomainName string
	Type       QuestionType
	Class      QuestionClass
}

func encodeDomainName(domainName string) []byte {
	parts := strings.Split(domainName, ".")
	l := len([]byte(domainName)) + 2
	encoded := make([]byte, 0, l)
	for _, part := range parts {
		encodedPart := append([]byte{byte(len(part))}, []byte(part)...)
		encoded = append(encoded, encodedPart...)
	}
	encoded = append(encoded, byte(0))
	return encoded
}

func (q *Question) Bytes() []byte {
	name := encodeDomainName(q.DomainName)
	buf := make([]byte, 0, len(name)+4)
	buf = append(buf, name...)
	tmp := make([]byte, 2)
	binary.BigEndian.PutUint16(tmp, uint16(q.Type))
	buf = append(buf, tmp...)
	binary.BigEndian.PutUint16(tmp, uint16(q.Class))
	buf = append(buf, tmp...)
	return buf
}

func (q *Question) AddName(name string)  { q.DomainName = name }
func (q *Question) AddType(t QuestionType) { q.Type = t }
func (q *Question) AddClass(c QuestionClass) { q.Class = c }