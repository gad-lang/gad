package encoder

import (
	"io"
)

type Reader interface {
	io.ReadSeeker
	io.ByteReader
	io.ReaderAt
}

type Writer interface {
	io.Writer
	io.ByteWriter
	io.StringWriter
	BytesWritten() int
}

type EmbeddedWriter interface {
	io.Writer
	BytesWritten() int
}

type stdReader struct {
	io.ReadSeeker
	readByteFunc func() (byte, error)
}

func (s *stdReader) ReadByte() (byte, error) {
	return s.readByteFunc()
}

func (s *stdReader) ReadAt(p []byte, off int64) (n int, err error) {
	var curPos int64
	if curPos, err = s.Seek(0, io.SeekCurrent); err != nil {
		return
	}
	if curPos, err = s.Seek(off, io.SeekStart); err != nil {
		return
	}
	if n, err = s.Read(p); err != nil {
		return
	}
	_, err = s.Seek(curPos, io.SeekStart)
	return
}

func NewReader(r io.ReadSeeker) Reader {
	if r, ok := r.(Reader); ok {
		return r
	}
	sr := &stdReader{ReadSeeker: r}
	if br, ok := r.(io.ByteReader); ok {
		sr.readByteFunc = func() (byte, error) {
			return br.ReadByte()
		}
	} else {
		var buf [1]byte
		sr.readByteFunc = func() (b byte, err error) {
			_, err = r.Read(buf[:])
			return buf[0], err
		}
	}
	return sr
}

type stdWriter struct {
	io.Writer
	bytesWritten int
}

func (w *stdWriter) BytesWritten() int {
	return w.bytesWritten
}

func (w *stdWriter) Write(p []byte) (n int, err error) {
	n, err = w.Writer.Write(p)
	w.bytesWritten += n
	return
}

func (w *stdWriter) WriteByte(c byte) error {
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
	return &stdWriter{Writer: w}
}
