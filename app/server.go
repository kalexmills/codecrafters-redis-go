package main

import (
	"codecrafters-redis-go/app/resp"
	"fmt"
	"net"
	"os"
)

func main() {
	fmt.Println("starting server on 6379")
	l, err := net.Listen("tcp", ":6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}
	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go handleConnection(c)
	}
}

func handleConnection(c net.Conn) {
	enc := resp.NewEncoder(c)
	dec := resp.NewDecoder(c)
	for {
		cmd := "PING"
		if err := dec.Decode(&cmd); err != nil {
			fmt.Println("error decoding command: ", err.Error())
			//os.Exit(1)
		}
		switch cmd {
		case "PING":
			if err := enc.Encode("PONG"); err != nil {
				fmt.Println("error responding to PING", err.Error())
				os.Exit(1)
			}
		}
	}
}
