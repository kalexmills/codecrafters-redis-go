package main

import (
	"codecrafters-redis-go/app/resp"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
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
		var input resp.Array
		if err := dec.Decode(&input); err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			fmt.Println("error decoding command: ", err.Error())
			continue
		}
		cmd, err := parseCommand(input)
		if err != nil {
			fmt.Println("error parsing command: ", err.Error())
			continue
		}
		switch cmd.Root() {
		case CmdPing:
			handlePing(enc, cmd)
		case CmdEcho:
			handleEcho(enc, cmd)
		case CmdUnknown:
			fmt.Println("unrecognized command")
		}
	}
}

func handlePing(enc *resp.Encoder, cmd Command) {
	if len(cmd.Args) == 1 {
		if err := enc.Encode("PONG"); err != nil {
			fmt.Println("error responding to PING:", err)
		}
		return
	}
	if len(cmd.Args) != 2 {
		if err := enc.Encode(fmt.Errorf("ERR wrong number arguments for command")); err != nil {
			fmt.Println("error sending error:", err)
		}
		return
	}
	if err := enc.Encode(cmd.Args[1]); err != nil {
		fmt.Println("error responding to PING:", err)
	}
}

func handleEcho(enc *resp.Encoder, cmd Command) {
	if len(cmd.Args) != 2 {
		if err := enc.Encode(fmt.Errorf("ERR wrong number arguments for command")); err != nil {
			fmt.Println("error sending error:", err)
		}
		return
	}
	if err := enc.Encode(cmd.Args[1]); err != nil {
		fmt.Println("error responding to ECHO:", err)
	}
}

type Command struct {
	Args [][]byte
}

// Root returns the root command key.
func (c Command) Root() Cmd {
	switch strings.ToUpper(string(c.Args[0])) {
	case string(CmdPing):
		return CmdPing
	case string(CmdEcho):
		return CmdEcho
	default:
		return CmdUnknown
	}
}

type Cmd string

const (
	CmdPing    Cmd = "PING"
	CmdEcho    Cmd = "ECHO"
	CmdUnknown Cmd = "UNKNOWN"
)

// parseCommand takes a resp.Array consisting of bulk strings and parses it into a Command.
func parseCommand(raw resp.Array) (Command, error) {
	var result Command
	if len(raw) == 0 {
		return result, fmt.Errorf("empty array")
	}
	for i := 0; i < len(raw); i++ {
		bytes, ok := raw[i].([]byte)
		if !ok {
			return result, fmt.Errorf("index %d not a bulk string", i)
		}
		result.Args = append(result.Args, bytes)
	}
	return result, nil
}
