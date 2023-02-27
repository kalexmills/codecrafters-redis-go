package main

import (
	"codecrafters-redis-go/app/resp"
	"fmt"
)

var kvstore = make(map[string][]byte)

func handleSet(enc *resp.Encoder, cmd Command) {
	if len(cmd.Args) < 3 {
		encError(enc, "wrong number of arguments for command")
	}
	key := string(cmd.Args[1])
	value := cmd.Args[2]

	existing, ok := kvstore[key]
	kvstore[key] = value
	if ok {
		if err := enc.Encode(existing); err != nil {
			fmt.Println("error responding to SET:", err)
		}
	} else {
		if err := enc.Encode("OK"); err != nil {
			fmt.Println("error responding to SET:", err)
		}
	}
}

func handleGet(enc *resp.Encoder, cmd Command) {
	if len(cmd.Args) != 2 {
		encError(enc, "wrong nmuber of arguments for command")
	}
	key := string(cmd.Args[1])

	existing, ok := kvstore[key]
	if !ok {
		if err := enc.Encode(nil); err != nil {
			fmt.Println("error responding to GET:", err)
		}
	} else {
		if err := enc.Encode(existing); err != nil {
			fmt.Println("error responding to GET:", err)
		}
	}
}
