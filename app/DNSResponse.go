package main

import "encoding/binary"

type DNSResponse struct {
	Header DNSHeader
	Body   []byte
}

func (r *DNSResponse) Bytes() []byte {
	buf := make([]byte, 12+len(r.Body))
	binary.BigEndian.PutUint16(buf[0:2], r.Header.ID)
	binary.BigEndian.PutUint16(buf[2:4], r.Header.Flags)
	binary.BigEndian.PutUint16(buf[4:6], r.Header.QDCount)
	binary.BigEndian.PutUint16(buf[6:8], r.Header.ANCount)
	binary.BigEndian.PutUint16(buf[8:10], r.Header.NSCount)
	binary.BigEndian.PutUint16(buf[10:12], r.Header.ARCount)
	copy(buf[12:], r.Body)
	return buf
}