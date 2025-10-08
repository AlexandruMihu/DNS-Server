package main

import (
	"flag"
	"fmt"
	"net"
)

var _ = net.ListenUDP


func main() {
	resolver := flag.String("resolver", "", "upstream resolver address (ip:port)")
	flag.Parse()
	if *resolver == "" {
		fmt.Println("Usage: ./your_server --resolver <ip:port>")
		return
	}

	if err := runServer("127.0.0.1:2053", *resolver); err != nil {
		fmt.Println("server error:", err)
	}
}


