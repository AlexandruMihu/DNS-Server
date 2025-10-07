package main

import "net"

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
	OpcodeQuery  = 0 // Standard query
	OpcodeIQuery = 1 // Inverse query
	OpcodeStatus = 2 // Server status request
)

// DNS Response codes
const (
	ResponseCodeNoError        byte = 0
	ResponseCodeFormatError    byte = 1
	ResponseCodeServerFailure  byte = 2
	ResponseCodeNameError      byte = 3
	ResponseCodeNotImplemented byte = 4
	ResponseCodeRefused        byte = 5
)

// DNS Query / Response type
const (
	QueryTypeQuery = 0
	QueryTypeReply = 1
)

