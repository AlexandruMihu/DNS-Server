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
	ResponseCodeNoError        = 0
	ResponseCodeFormatError    = 1
	ResponseCodeServerFailure  = 2
	ResponseCodeNameError      = 3
	ResponseCodeNotImplemented = 4
	ResponseCodeRefused        = 5
)

// DNS Query / Response type
const (
	QueryTypeQuery = 0
	QueryTypeReply = 1
)

