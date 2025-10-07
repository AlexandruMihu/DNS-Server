package main

import (
	"encoding/binary"
	"strconv"
	"strings"
)

func ipToBytes(ip string) ([]byte, error) {
	parts := strings.Split(ip, ".")
	buf := make([]byte, 4)
	joined := strings.Join(parts, "")
	asInt, err := strconv.Atoi(joined)
	if err != nil {
		return nil, err
	}
	binary.BigEndian.PutUint32(buf, uint32(asInt))
	return buf, nil
}