package main

import "encoding/binary"

type DNSResponse struct {
	Header DNSHeader
	Question DNSQuestion
	Answer DNSAnswer
	Body   []byte
}

func (r *DNSResponse) Bytes() []byte {
	header := make([]byte, 12)
	binary.BigEndian.PutUint16(header[0:2], r.Header.ID)
	binary.BigEndian.PutUint16(header[2:4], r.Header.Flags)
	binary.BigEndian.PutUint16(header[4:6], r.Header.QDCount)
	binary.BigEndian.PutUint16(header[6:8], r.Header.ANCount)
	binary.BigEndian.PutUint16(header[8:10], r.Header.NSCount)
	binary.BigEndian.PutUint16(header[10:12], r.Header.ARCount)

	questionBytes := r.Question.Bytes()

    answerBytes := r.Answer.Bytes()

	buf := append(header, questionBytes,answerBytes,r.Body)

	return buf
}
