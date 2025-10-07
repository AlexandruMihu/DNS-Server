package main

import "encoding/binary"

type DNSAnswer struct {
	Question   DNSQuestion
	TimeToLive uint32
	DataLength uint16
	Data       []byte
}


func (a *DNSAnswer) Bytes() []byte {
	buf := make([]byte, 0, len(a.Question.DomainName)+10+len(a.Data))
	buf = append(buf, a.Question.Bytes()...)
	l := len(buf)
	buf = append(buf, byte(0), byte(0), byte(0), byte(0), byte(0), byte(0)) // zeroed bytes
	binary.BigEndian.PutUint32(buf[l:], a.TimeToLive)
	binary.BigEndian.PutUint16(buf[l+4:], a.DataLength)
	buf = append(buf, a.Data...)
	return buf
}

func (a *DNSAnswer ) AddQuestion(question DNSQuestion)  { a.Question = question }
func (a *DNSAnswer ) AddTTL(ttl uint32) { a.TimeToLive = ttl }
func (a *DNSAnswer ) AddDataLength(dataLength uint16) { a.DataLength = dataLength }
func (a *DNSAnswer ) AddData(data []byte) { a.Data = data }
