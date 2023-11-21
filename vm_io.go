package gad

import "io"

type StackWriter struct {
	last    int
	writers []io.Writer
}

func NewStackWriter(writers ...io.Writer) *StackWriter {
	return &StackWriter{writers: writers}
}

func (w *StackWriter) Type() ObjectType {
	return TWriter
}

func (w *StackWriter) ToString() string {
	return "stackWriter"
}

func (w *StackWriter) IsFalsy() bool {
	return w.last < 0
}

func (w *StackWriter) Equal(right Object) bool {
	if o, _ := right.(*StackWriter); o != nil {
		return o == w
	}
	return false
}

func (w *StackWriter) GoWriter() io.Writer {
	return w.writers[w.last]
}

func (w *StackWriter) Write(p []byte) (n int, err error) {
	return w.writers[w.last].Write(p)
}

func (w *StackWriter) Push(sw io.Writer) {
	w.writers = append(w.writers, sw)
	w.last++
}

func (w *StackWriter) Pop() Writer {
	last := w.writers[w.last]
	w.writers = w.writers[:w.last]
	w.last--

	switch t := last.(type) {
	case Writer:
		return t
	default:
		return NewWriter(t)
	}
}

func (w *StackWriter) Old() Writer {
	if w.last == 0 {
		return nil
	}
	switch t := w.writers[w.last-1].(type) {
	case Writer:
		return t
	default:
		return NewWriter(t)
	}
}

func (w *StackWriter) Current() Writer {
	switch t := w.writers[w.last].(type) {
	case Writer:
		return t
	default:
		return NewWriter(t)
	}
}

func (w *StackWriter) Flush() (n Int, err error) {
	if w.last == 0 {
		return
	}
	old, cur := w.writers[w.last-1], w.writers[w.last]

	switch t := cur.(type) {
	case io.WriterTo:
		var n_ int64
		n_, err = t.WriteTo(old)
		n = Int(n_)
	case io.Reader:
		var n_ int64
		n_, err = io.Copy(old, t)
		n = Int(n_)
	default:
		err = ErrType.NewError("current writer in't io.Reader|io.WriterTo")
	}
	return
}

type StackReader struct {
	last    int
	readers []io.Reader
}

func NewStackReader(readers ...io.Reader) *StackReader {
	return &StackReader{readers: readers, last: len(readers) - 1}
}

func (s *StackReader) Type() ObjectType {
	return TReader
}

func (s *StackReader) ToString() string {
	return "stackReader"
}

func (s *StackReader) IsFalsy() bool {
	return s.last < 0
}

func (s *StackReader) Equal(right Object) bool {
	if o, _ := right.(*StackReader); o != nil {
		return o == s
	}
	return false
}

func (s *StackReader) GoReader() io.Reader {
	return s.readers[s.last]
}

func (s *StackReader) Read(p []byte) (n int, err error) {
	return s.readers[s.last].Read(p)
}

func (s *StackReader) Push(r io.Reader) {
	s.readers = append(s.readers, r)
	s.last++
}

func (s *StackReader) Pop() {
	s.readers = s.readers[:s.last]
	s.last--
}

func (vm *VM) Write(b []byte) (int, error) {
	return vm.StdOut.Write(b)
}

func (vm *VM) Read(b []byte) (int, error) {
	return vm.StdIn.Read(b)
}
