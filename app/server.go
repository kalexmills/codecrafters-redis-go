package main

import (
	"codecrafters-redis-go/app/resp"
	"fmt"
	"net"
	"os"
)

func main() {

	fmt.Println("starting server on 6379")
	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}
	c, err := l.Accept()
	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}
	if err := resp.NewEncoder(c).Encode("OK"); err != nil {
		fmt.Println("Error responding to PING on startup: ", err.Error())
		os.Exit(1)
	}
}
