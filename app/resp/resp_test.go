package resp_test

import (
	"bytes"
	"codecrafters-redis-go/app/resp"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

func TestEncoder_Encode(t *testing.T) {
	tests := []struct {
		name     string
		v        any
		expected []byte
	}{
		{
			name:     "nil",
			v:        nil,
			expected: []byte("$-1\r\n"),
		},
		{
			name:     "string",
			v:        "hello",
			expected: []byte("+hello\r\n"),
		},
		{
			name:     "string empty",
			v:        "",
			expected: []byte("+\r\n"),
		},
		{
			name:     "error",
			v:        fmt.Errorf("error"),
			expected: []byte("-error\r\n"),
		},
		{
			name:     "error empty",
			v:        fmt.Errorf(""),
			expected: []byte("-\r\n"),
		},
		{
			name:     "[]byte empty",
			v:        []byte{},
			expected: []byte("$0\r\n\r\n"),
		},
		{
			name:     "[]byte",
			v:        []byte{'a', 'b', 'c', 'd', 'e'},
			expected: []byte("$5\r\nabcde\r\n"),
		},
		{
			name:     "int64",
			v:        int64(123),
			expected: []byte(":123\r\n"),
		},
		{
			name:     "int32",
			v:        int32(-5757),
			expected: []byte(":-5757\r\n"),
		},
		{
			name:     "int",
			v:        234,
			expected: []byte(":234\r\n"),
		},
		{
			name:     "Int",
			v:        resp.Int(-56777),
			expected: []byte(":-56777\r\n"),
		},
		{
			name:     "integer zero",
			v:        resp.Int(0),
			expected: []byte(":0\r\n"),
		},
		{
			name:     "array empty",
			v:        resp.Array{},
			expected: []byte("*0\r\n"),
		},
		{
			name:     "array ints",
			v:        resp.Array{1, 4, 7, 2, 9, 10},
			expected: []byte("*6\r\n:1\r\n:4\r\n:7\r\n:2\r\n:9\r\n:10\r\n"),
		},
		{
			name:     "array strs",
			v:        resp.Array{"hello", "world"},
			expected: []byte("*2\r\n+hello\r\n+world\r\n"),
		},
		{
			name:     "array errors",
			v:        resp.Array{fmt.Errorf("e1"), fmt.Errorf("e2")},
			expected: []byte("*2\r\n-e1\r\n-e2\r\n"),
		},
		{
			name:     "array bytes",
			v:        resp.Array{[]byte{'a', 'b', 'c', 'd'}, []byte{}},
			expected: []byte("*2\r\n$4\r\nabcd\r\n$0\r\n\r\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b bytes.Buffer
			err := resp.NewEncoder(&b).Encode(tt.v)
			assert.NoError(t, err)

			assert.EqualValues(t, tt.expected, b.Bytes())
		})
	}
}

func TestDecoder_Decode_strings(t *testing.T) {
	tests := []struct {
		name     string
		bytes    string
		expected string
	}{
		{
			name:     "hello world",
			bytes:    "+hello world\r\n",
			expected: "hello world",
		},
		{
			name:     "empty str",
			bytes:    "+\r\n",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := bytes.NewBufferString(tt.bytes)
			var out string
			err := resp.NewDecoder(b).Decode(&out)
			assert.NoError(t, err)

			assert.EqualValues(t, tt.expected, out)
		})
	}
}

func TestDecoder_Decode_errors(t *testing.T) {
	tests := []struct {
		name     string
		bytes    string
		expected error
	}{
		{
			name:     "error",
			bytes:    "-error\r\n",
			expected: errors.New("error"),
		},
		{
			name:     "empty",
			bytes:    "-\r\n",
			expected: errors.New(""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := bytes.NewBufferString(tt.bytes)
			var out error
			err := resp.NewDecoder(b).Decode(&out)
			assert.NoError(t, err)

			assert.EqualValues(t, tt.expected, out)
		})
	}
}

func TestDecoder_Decode_ints(t *testing.T) {
	tests := []struct {
		name     string
		bytes    string
		expected resp.Int
		wantErr  error
	}{
		{
			name:     "negative",
			bytes:    ":-246\r\n",
			expected: -246,
		},
		{
			name:     "zero",
			bytes:    ":0\r\n",
			expected: 0,
		},
		{
			name:     "max int 64",
			bytes:    ":9223372036854775807\r\n",
			expected: math.MaxInt64,
		},
		{
			name:     "min int 64",
			bytes:    ":-9223372036854775808\r\n",
			expected: math.MinInt64,
		},
		{
			name:    "overflow",
			bytes:   ":9223372036854775808\r\n",
			wantErr: resp.ErrParse,
		},
		{
			name:    "underflow",
			bytes:   ":-9223372036854775809\r\n",
			wantErr: resp.ErrParse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := bytes.NewBufferString(tt.bytes)
			var out resp.Int
			err := resp.NewDecoder(b).Decode(&out)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)

			assert.EqualValues(t, tt.expected, out)
		})
	}
}

func TestDecoder_Decode_bulkString(t *testing.T) {
	tests := []struct {
		name     string
		bytes    string
		expected []byte
		wantErr  error
	}{
		{
			name:     "nil",
			bytes:    "$-1\r\n",
			expected: nil,
		},
		{
			name:     "hello world",
			bytes:    "$11\r\nhello world\r\n",
			expected: []byte("hello world"),
		},
		{
			name:     "binary unsafe",
			bytes:    "$6\r\n\r\n\r\n\r\n\r\n",
			expected: []byte("\r\n\r\n\r\n"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := bytes.NewBufferString(tt.bytes)
			var out []byte
			err := resp.NewDecoder(b).Decode(&out)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)

			assert.EqualValues(t, tt.expected, out)
		})
	}
}

func TestDecoder_Decode_arrays(t *testing.T) {
	tests := []struct {
		name     string
		bytes    string
		expected resp.Array
		wantErr  error
	}{
		{
			name:     "array of nils",
			bytes:    "*2\r\n$-1\r\n$-1\r\n",
			expected: resp.Array{nil, nil},
		},
		{
			name:     "empty array",
			bytes:    "*0\r\n\r\n",
			expected: resp.Array{},
		},
		{
			name:     "nested empty arrays",
			bytes:    "*2\r\n*0\r\n*0\r\n\r\n",
			expected: resp.Array{resp.Array{}, resp.Array{}},
		},
		{
			name:     "incoming command",
			bytes:    "*2\r\n$4\r\nLLEN\r\n$6\r\nmylist\r\n",
			expected: resp.Array{[]byte("LLEN"), []byte("mylist")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := bytes.NewBufferString(tt.bytes)
			var out resp.Array
			err := resp.NewDecoder(b).Decode(&out)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)

			assert.EqualValues(t, tt.expected, out)
		})
	}
}
