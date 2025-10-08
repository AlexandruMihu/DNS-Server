package main

import "strings"
import "encoding/binary"
import "errors"

type QuestionType uint16

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

func ParseName(buf []byte, offset int) (string, int, error) {
	if offset >= len(buf) {
		return "", offset, errors.New("offset beyond buffer")
	}

	var labels []string
	jumped := false
	origOffset := offset
	steps := 0
	for {
		steps++
		if steps > len(buf)+5 {
			return "", offset, errors.New("too many steps while parsing name (possible loop)")
		}
		if offset >= len(buf) {
			return "", offset, errors.New("unexpected end of buffer while reading name")
		}
		b := buf[offset]

		if b&0xC0 == 0xC0 {
			if offset+1 >= len(buf) {
				return "", offset, errors.New("pointer truncated")
			}
			pointer := int(binary.BigEndian.Uint16(buf[offset:offset+2]) & 0x3FFF)
			if pointer >= len(buf) {
				return "", offset, errors.New("pointer out of range")
			}
			if !jumped {
				origOffset = offset + 2
			}
			offset = pointer
			jumped = true
			continue
		}

		if b == 0 {
			offset++
			break
		}

		length := int(b)
		offset++
		if offset+length > len(buf) {
			return "", offset, errors.New("label length extends past buffer")
		}
		labels = append(labels, string(buf[offset:offset+length]))
		offset += length
	}

	if jumped {
		return strings.Join(labels, "."), origOffset, nil
	}
	return strings.Join(labels, "."), offset, nil
}

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

func ParseQuestion(buf []byte, offset int) (*DNSQuestion, int, error) {
	name, off, err := ParseName(buf, offset)
	if err != nil {
		return nil, offset, err
	}
	if off+4 > len(buf) {
		return nil, off, errors.New("buffer too small for type/class")
	}
	typ := binary.BigEndian.Uint16(buf[off : off+2])
	class := binary.BigEndian.Uint16(buf[off+2 : off+4])
	off += 4

	q := &DNSQuestion{
		DomainName: name,
		Type:       QuestionType(typ),
		Class:      QuestionClass(class),
	}
	return q, off, nil
}

func forwardQuestion(q *DNSQuestion, origHeader *DNSHeader, resolverAddr string) (forwardResp, error) {
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

	answerBytes := make([]byte, len(resp)-off)
	copy(answerBytes, resp[off:])

	return forwardResp{
		header:      respHeader,
		answerBytes: answerBytes,
		anCount:     respHeader.ANCount,
	}, nil
}