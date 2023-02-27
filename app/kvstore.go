package main

import (
	"codecrafters-redis-go/app/resp"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var kvstore = make(map[string]stored)

type stored struct {
	Value  []byte
	Expiry *time.Time
}

func (s *stored) Unexpired() bool {
	return s.Expiry == nil || time.Now().Before(*s.Expiry)
}

func handleSet(enc *resp.Encoder, cmd Command) {
	if len(cmd.Args) < 3 {
		encError(enc, "wrong number of arguments for command")
	}
	key := string(cmd.Args[1])
	value := cmd.Args[2]
	opts, err := setOptions(cmd)
	if err != nil {
		encError(enc, err.Error())
	}

	existing, ok := kvstore[key]
	kvstore[key] = stored{
		Value:  value,
		Expiry: opts.Expiry(),
	}

	if ok {
		if err := enc.Encode(existing.Value); err != nil {
			fmt.Println("error responding to SET:", err)
		}
	} else {
		if err := enc.Encode("OK"); err != nil {
			fmt.Println("error responding to SET:", err)
		}
	}
}

type SetOpts struct {
	PX int // PX is the expiration time in milliseconds
}

func (o *SetOpts) Expiry() *time.Time {
	if o.PX > 0 {
		out := time.Now().Add(time.Duration(o.PX) * time.Millisecond)
		return &out
	}
	return nil
}

func setOptions(cmd Command) (SetOpts, error) {
	var result SetOpts
	if len(cmd.Args) < 4 {
		return result, nil
	}
	for i := 3; i < len(cmd.Args)-1; i++ {
		switch strings.ToUpper(string(cmd.Args[i])) {
		case "PX":
			val, err := strconv.Atoi(string(cmd.Args[i+1]))
			if err != nil {
				return result, fmt.Errorf("expected integer argument to PX")
			}
			result.PX = val
		}
	}
	return result, nil
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
		var out []byte
		if existing.Unexpired() {
			out = existing.Value
		} else {
			out = nil
		}
		if err := enc.Encode(out); err != nil {
			fmt.Println("error responding to GET:", err)
		}
	}
}
