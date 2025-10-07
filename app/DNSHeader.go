package main

import "encoding/binary"

type DNSHeader struct {
	ID      uint16
	Flags   uint16
	QDCount uint16
	ANCount uint16
	NSCount uint16
	ARCount uint16
}

func (r *DNSHeader) AddID(id uint16) *DNSHeader {
	r.ID = id
	return r
}

func (r *DNSHeader) AddQR(b byte) *DNSHeader {
	r.Flags |= uint16(b) << 15
	return r
}

func (r *DNSHeader) AddOPCODE(b byte) *DNSHeader {
	r.Flags |= uint16(b) << 14
	return r
}

func (r *DNSHeader) AddAA(b byte) *DNSHeader {
	r.Flags |= uint16(b) << 10
	return r
}

func (r *DNSHeader) AddTC(b byte) *DNSHeader {
	r.Flags |= uint16(b) << 9
	return r
}

func (r *DNSHeader) AddRD(b byte) *DNSHeader {
	r.Flags |= uint16(b) << 8
	return r
}

func (r *DNSHeader) AddRA(b byte) *DNSHeader {
	r.Flags |= uint16(b) << 7
	return r
}

func (r *DNSHeader) AddZ(b byte) *DNSHeader {
	r.Flags |= uint16(b) << 4
	return r
}

func (r *DNSHeader) AddRCODE(b byte) *DNSHeader {
	r.Flags |= uint16(b)
	return r
}

func (r *DNSHeader) AddQDCOUNT(b uint16) *DNSHeader {
	r.QDCount = b
	return r
}

func (r *DNSHeader) AddANCOUNT(b uint16) *DNSHeader {
	r.ANCount = b
	return r
}

func (r *DNSHeader) AddNSCOUNT(b uint16) *DNSHeader {
	r.NSCount = b
	return r
}

func (r *DNSHeader) AddARCOUNT(b uint16) *DNSHeader {
	r.ARCount = b
	return r
}

func ParseHeader(buf []byte) *DNSHeader {
	if len(buf) < 12 {
		return nil 
	}

	return &DNSHeader{
		ID:      binary.BigEndian.Uint16(buf[0:2]),
		Flags:   binary.BigEndian.Uint16(buf[2:4]),
		QDCount: binary.BigEndian.Uint16(buf[4:6]),
		ANCount: binary.BigEndian.Uint16(buf[6:8]),
		NSCount: binary.BigEndian.Uint16(buf[8:10]),
		ARCount: binary.BigEndian.Uint16(buf[10:12]),
	}
}