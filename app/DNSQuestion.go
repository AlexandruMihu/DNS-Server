package main
//Jlrine2
import "strings"
import "encoding/binary"

type QuestionType uint16

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

type DNSQuestion  struct {
	DomainName string
	Type       QuestionType
	Class      QuestionClass
}

func encodeDomainName(domainName string) []byte {
	
	domainName = strings.TrimSuffix(domainName, ".")

	parts := strings.Split(domainName, ".")
	encoded := make([]byte, 0, len(domainName)+2)

	for _, part := range parts {
		if part == "" {
			continue
		}
		encoded = append(encoded, byte(len(part)))
		encoded = append(encoded, []byte(part)...)
	}
	encoded = append(encoded, 0) 
	return encoded
}


func (q *DNSQuestion) Bytes() []byte {
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

func (q *DNSQuestion ) AddName(name string)  { q.DomainName = name }
func (q *DNSQuestion ) AddType(t QuestionType) { q.Type = t }
func (q *DNSQuestion ) AddClass(c QuestionClass) { q.Class = c }

func ParseQuestion(buf []byte, offset int) (*DNSQuestion, int, error) {
	if offset >= len(buf) {
		return nil, offset, errors.New("buffer too small for question")
	}

	labels := make([]string, 0)
	for {
		if offset >= len(buf) {
			return nil, offset, errors.New("unexpected end of buffer while reading name")
		}
		length := int(buf[offset])
		offset++
		if length == 0 {
			break // end of name
		}
		if offset+length > len(buf) {
			return nil, offset, errors.New("label length goes past buffer")
		}
		labels = append(labels, string(buf[offset:offset+length]))
		offset += length
	}
	domain := strings.Join(labels, ".")
	
	if offset+4 > len(buf) {
		return nil, offset, errors.New("buffer too small for type/class")
	}
	typ := binary.BigEndian.Uint16(buf[offset : offset+2])
	class := binary.BigEndian.Uint16(buf[offset+2 : offset+4])
	offset += 4

	q := &DNSQuestion{
		DomainName: domain,
		Type:       QuestionType(typ),
		Class:      QuestionClass(class),
	}
	return q, offset, nil
}