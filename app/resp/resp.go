package resp

import (
	"io"
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
	case string:
		err = prefix(e.w, '+', e.encodeStr, v)
	}
	return err
}

func (e *Encoder) encodeStr(str string) error {
	_, err := e.w.Write([]byte(str))
	return err
}

// prefix writes b to w, calls f with t, then writes `\r\n` to w. Any error which occurs is returned.
func prefix[T any](w io.Writer, b byte, f func(T) error, t T) error {
	if _, err := w.Write([]byte{b}); err != nil {
		return err
	}
	if err := f(t); err != nil {
		return err
	}
	if _, err := w.Write([]byte(`\r\n`)); err != nil {
		return err
	}
	return nil
}
