package encoder

import (
	"bufio"
	"io"
)

type Reader interface {
	io.Reader
	io.ByteReader
	io.RuneReader
}

type Writer interface {
	io.Writer
	io.ByteWriter
	io.StringWriter
}

func NewReader(r io.Reader) Reader {
	if r, ok := r.(Reader); ok {
		return r
	}
	return bufio.NewReader(r)
}

type stdWriter struct {
	io.Writer
}

func (w stdWriter) WriteByte(c byte) error {
	_, err := w.Write([]byte{c})
	return err
}

func (w stdWriter) WriteString(s string) (n int, err error) {
	return w.Write([]byte(s))
}

func NewWriter(w io.Writer) Writer {
	if w, ok := w.(Writer); ok {
		return w
	}
	return &stdWriter{w}
}
