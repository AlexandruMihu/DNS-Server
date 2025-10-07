package main

import "net"

func ipToBytes(ip string) ([]byte, error) {
	return net.ParseIP(ip).To4(), nil
}