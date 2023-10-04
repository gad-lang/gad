package gad

import (
	"bytes"
	"fmt"
	"io"
)

type ToWriter interface {
	Object
	WriteTo(w Writer) (n Int, err error)
}

type writer struct {
	io.Writer
	typ ObjectType
}

func NewWriter(w io.Writer) Writer {
	return &writer{Writer: w}
}

func NewTypedWriter(w io.Writer, typ ObjectType) Writer {
	return &writer{Writer: w, typ: typ}
}

func (w *writer) Type() ObjectType {
	if w.typ == nil {
		return TWriter
	}
	return w.typ
}

func (w *writer) String() string {
	return fmt.Sprintf("writer of %v", w.Writer)
}

func (w *writer) IsFalsy() bool {
	return false
}

func (w *writer) GoWriter() io.Writer {
	return w.Writer
}

func (w *writer) Equal(right Object) bool {
	switch t := right.(type) {
	case *writer:
		return w.Writer == t.Writer
	case Writer:
		return w.Writer == t.GoWriter()
	default:
		return false
	}
}

type reader struct {
	io.Reader
}

func NewReader(r io.Reader) Reader {
	return &reader{Reader: r}
}

func (r *reader) Type() ObjectType {
	return TReader
}

func (r *reader) String() string {
	return fmt.Sprintf("reader of %v", r.Reader)
}

func (r *reader) IsFalsy() bool {
	return false
}

func (r *reader) GoReader() io.Reader {
	return r.Reader
}

func (r *reader) Equal(right Object) bool {
	switch t := right.(type) {
	case *reader:
		return r.Reader == t.Reader
	case Reader:
		return r.Reader == t.GoReader()
	default:
		return false
	}
}

type Buffer struct {
	bytes.Buffer
}

var (
	_ Writer           = new(Buffer)
	_ Reader           = new(Buffer)
	_ LengthGetter     = new(Buffer)
	_ IndexGetter      = new(Buffer)
	_ IndexSetter      = new(Buffer)
	_ NameCallerObject = new(Buffer)
	_ BytesConverter   = new(Buffer)
)

// Iterate implements Object interface.
func (o *Buffer) Iterate() Iterator {
	return &BytesIterator{V: o.Bytes()}
}

// IndexSet implements Object interface.
func (o *Buffer) IndexSet(_ *VM, index, value Object) error {
	var idx int
	switch v := index.(type) {
	case Int:
		idx = int(v)
	case Uint:
		idx = int(v)
	default:
		return NewIndexTypeError("int|uint", index.Type().Name())
	}

	if idx >= 0 && idx < o.Len() {
		switch v := value.(type) {
		case Int:
			o.Bytes()[idx] = byte(v)
		case Uint:
			o.Bytes()[idx] = byte(v)
		default:
			return NewIndexValueTypeError("int|uint", value.Type().Name())
		}
		return nil
	}
	return ErrIndexOutOfBounds
}

// IndexGet represents string values and implements Object interface.
func (o *Buffer) IndexGet(_ *VM, index Object) (Object, error) {
	var idx int
	switch v := index.(type) {
	case Int:
		idx = int(v)
	case Uint:
		idx = int(v)
	default:
		return nil, NewIndexTypeError("int|uint|char", index.Type().Name())
	}

	if idx >= 0 && idx < o.Len() {
		return Int(o.Bytes()[idx]), nil
	}
	return nil, ErrIndexOutOfBounds
}

func (o *Buffer) GoReader() io.Reader {
	return &o.Buffer
}

func (o *Buffer) Type() ObjectType {
	return typeOf(o)
}

func (o *Buffer) IsFalsy() bool {
	return o.Len() == 0
}

func (o *Buffer) Equal(right Object) bool {
	switch t := right.(type) {
	case *Buffer:
		return o == t
	default:
		return false
	}
}

func (o *Buffer) GoWriter() io.Writer {
	return &o.Buffer
}

func (o *Buffer) CallName(name string, c Call) (Object, error) {
	switch name {
	case "reset":
		o.Reset()
	default:
		return nil, ErrInvalidIndex.NewError(name)
	}
	return Nil, nil
}

func (o *Buffer) ToBytes() (Bytes, error) {
	return o.Bytes(), nil
}

var DiscardWriter = NewTypedWriter(io.Discard, TDiscardWriter)
