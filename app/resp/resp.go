package resp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
)

// Int is an integer number with up to 64 bits of precision, as a signed integer.
type Int int64

// Array is an array of anything, used by Redis to send and receive sequences of values.
type Array []any

const (
	prefixString     byte = '+'
	prefixError           = '-'
	prefixInt             = ':'
	prefixArray           = '*'
	prefixBulkString      = '$'
)

type Encoder struct {
	w io.Writer
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

func (e *Encoder) Encode(v any) error {
	var err error
	switch v := v.(type) {
	case int:
		err = e.encodeInt(Int(v))
	case int64:
		err = e.encodeInt(Int(v))
	case int32:
		err = e.encodeInt(Int(v))
	case Int:
		err = e.encodeInt(v)
	case string:
		err = e.encodeStr(v)
	case []byte:
		err = e.encodeBulkStr(v)
	case error:
		err = e.encodeErr(v)
	case Array:
		err = e.encodeArray(v)
	}
	return err
}

func (e *Encoder) encodeInt(v Int) error {
	return encodePrefix(e.w, prefixInt, func() error {
		_, err := e.w.Write([]byte(strconv.Itoa(int(v))))
		return err
	})
}

func (e *Encoder) encodeErr(err error) error {
	return encodePrefix(e.w, prefixError, func() error {
		_, err := e.w.Write([]byte(err.Error()))
		return err
	})
}

func (e *Encoder) encodeStr(str string) error {
	return encodePrefix(e.w, prefixString, func() error {
		_, err := e.w.Write([]byte(str))
		return err
	})
}

func (e *Encoder) encodeBulkStr(bytes []byte) error {
	return encodePrefix(e.w, prefixBulkString, func() error {
		_, err := e.w.Write([]byte(fmt.Sprintf("%d\r\n", len(bytes))))
		if err != nil {
			return err
		}
		_, err = e.w.Write(bytes)
		return err
	})
}

func (e *Encoder) encodeArray(arr Array) error {
	_, err := e.w.Write([]byte(fmt.Sprintf("%c%d\r\n", prefixArray, len(arr))))
	if err != nil {
		return err
	}
	for idx, x := range arr {
		if err := e.Encode(x); err != nil {
			return fmt.Errorf("idx[%d]: %w", idx, err)
		}
	}
	return nil
}

// encodePrefix writes b to w, calls f with t, then writes "\r\n" to w. Any error which occurs is returned.
func encodePrefix(w io.Writer, b byte, f func() error) error {
	if _, err := w.Write([]byte{b}); err != nil {
		return err
	}
	if err := f(); err != nil {
		return err
	}
	if _, err := w.Write([]byte("\r\n")); err != nil {
		return err
	}
	return nil
}

type Decoder struct {
	r *bufio.Reader
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: bufio.NewReader(r)}
}

var ErrProtocol = errors.New("protocol error")
var ErrParse = errors.New("parse error")
var ErrUnexpectedType = errors.New("unexpected type")

func (d *Decoder) Decode(v any) error {
	bytes, err := d.r.Peek(1)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("%w: unexpected EOF", ErrProtocol)
		}
		return fmt.Errorf("error decoding next entity: %w", err)
	}
	prefix := bytes[0]
	switch prefix {
	case prefixString:
		return d.decodeString(v)
	case prefixError:
		return d.decodeError(v)
	case prefixInt:
		return d.decodeInt(v)
	case prefixBulkString:
		return d.decodeBulkString(v)
	case prefixArray:
		return d.decodeArray(v)
	default:
		return fmt.Errorf("%w: bad prefix %d", ErrUnexpectedType, prefix)
	}
}

func (d *Decoder) decodeString(v any) error {
	bytes, err := d.readNext()
	if err != nil {
		return err
	}
	target, ok := v.(*string)
	if !ok {
		return fmt.Errorf("%w: expected v to have type *string", ErrUnexpectedType)
	}
	*target = string(bytes)
	return nil
}

func (d *Decoder) decodeError(v any) error {
	bytes, err := d.readNext()
	if err != nil {
		return err
	}
	target, ok := v.(*error)
	if !ok {
		return fmt.Errorf("%w: expected v to have type *error", ErrUnexpectedType)
	}
	*target = errors.New(string(bytes))
	return nil
}

func (d *Decoder) decodeInt(v any) error {
	bytes, err := d.readNext()
	if err != nil {
		return err
	}
	target, ok := v.(*Int)
	if !ok {
		return fmt.Errorf("%w: expected v to have type *resp.Int", ErrUnexpectedType)
	}
	val, err := readInt(bytes)
	if err != nil {
		return err
	}
	*target = Int(val)
	return nil
}

func (d *Decoder) decodeBulkString(v any) error {
	bytes, err := d.readNext()
	if err != nil {
		return err
	}
	target, ok := v.(*[]byte)
	if !ok {
		return fmt.Errorf("%w: expected v to have type *[]byte", ErrUnexpectedType)
	}
	length, err := readLength(bytes, d.r)
	if err != nil {
		return err
	}
	fmt.Println("length", length)
	if length > 512*1024*1024 { // maximum 512 MiB, according to protocol
		return fmt.Errorf("%w: length of bulk string exceeds 512 MiB", ErrProtocol)
	}
	// read the entire payload
	*target = make([]byte, length)
	n, err := io.ReadFull(d.r, *target)
	if err != nil {
		return err
	}
	if n != length {
		return fmt.Errorf("%w: expected %d bytes, read %d before EOF", ErrProtocol, length, n)
	}
	return nil
}

func (d *Decoder) decodeArray(v any) error {
	bytes, err := d.readNext()
	if err != nil {
		return err
	}
	target, ok := v.(*Array)
	if !ok {
		return fmt.Errorf("%w: expected v to have type *resp.Array", ErrUnexpectedType)
	}
	length, err := readLength(bytes, d.r)
	if err != nil {
		return err
	}

	result := make(Array, length)
	for i := 0; i < length; i++ {
		bytes, err := d.r.Peek(1)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return fmt.Errorf("%w: unexpected EOF", ErrProtocol)
			}
			return fmt.Errorf("error decoding entity at idx %d: %w", i, err)
		}
		switch bytes[0] {
		case prefixInt:
			var target int64
			if err := d.decodeInt(&target); err != nil {
				return fmt.Errorf("error decoding int at idx %d: %w", i, err)
			}
			result[i] = target
		case prefixString:
			var target string
			if err := d.decodeString(&target); err != nil {
				return fmt.Errorf("error decoding string at idx %d: %w", i, err)
			}
			result[i] = target
		case prefixError:
			var target error
			if err := d.decodeError(&target); err != nil {
				return fmt.Errorf("error decoding error at idx %d: %w", i, err)
			}
			result[i] = target
		case prefixBulkString:
			var target []byte
			if err := d.decodeBulkString(&target); err != nil {
				return fmt.Errorf("error decoding bulk string at idx %d: %w", i, err)
			}
			result[i] = target
		case prefixArray:
			var target Array
			if err := d.decodeArray(&target); err != nil {
				return fmt.Errorf("error decoding array at idx %d: %w", i, err)
			}
			result[i] = target
		default:
			return fmt.Errorf("unexpected prefix: %d", bytes[0])
		}
	}
	*target = result
	return nil
}

// readNext advances the reader to the next new line and returns bytes between the current prefix and
func (d *Decoder) readNext() ([]byte, error) {
	bytes, err := d.r.ReadBytes('\n') // TODO: why not to \n?!
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("error decoding next entity: %w", err)
	}
	return bytes[1 : len(bytes)-2], err
}

// readInt reads an integer from the provided sequence of bytes.
func readInt(bytes []byte) (result int, err error) {
	result, err = strconv.Atoi(string(bytes))
	if err != nil {
		return 0, fmt.Errorf("%w: not a 64-bit signed integer", ErrParse)
	}
	return result, err
}

func readLength(bytes []byte, r *bufio.Reader) (int, error) {
	length, err := readInt(bytes)
	if err != nil {
		return 0, err
	}
	return length, err
}
