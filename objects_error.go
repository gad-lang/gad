package gad

import (
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"

	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/repr"
)

// Error represents Error Object and implements error and Object interfaces.
type Error struct {
	Name    string
	Message string
	Cause   error
}

func WrapError(cause error) *Error {
	switch err := cause.(type) {
	case *Error:
		return err
	default:
		t := reflect.TypeOf(err)
		name := strings.TrimPrefix(t.String(), "*")
		return &Error{Name: name, Cause: cause}
	}
}

var (
	_ Object = (*Error)(nil)
	_ Copier = (*Error)(nil)
)

func (o *Error) Unwrap() error {
	return o.Cause
}

func (o *Error) Type() ObjectType {
	return TError
}

func (o *Error) ToString() string {
	return o.Error()
}

// Copy implements Copier interface.
func (o *Error) Copy() Object {
	return &Error{
		Name:    o.Name,
		Message: o.Message,
		Cause:   o.Cause,
	}
}

// Error implements error interface.
func (o *Error) Error() string {
	var (
		name = o.Name
		msg  = o.Message
	)
	if name == "" {
		name = "error"
	}
	if o.Cause != nil {
		var cause string
		switch ct := o.Cause.(type) {
		case *Error:
			if len(ct.Message) > 0 || ct.Cause != nil {
				cause = ct.Error()
			}
		default:
			return fmt.Sprintf("%s: %s", name, ct.Error())
		}

		if len(cause) > 0 {
			if msg == "" {
				msg = cause
			} else {
				msg += "; caused by: " + repr.Quote(cause)
			}
		}
	}
	return fmt.Sprintf("%s: %s", name, msg)
}

// Equal implements Object interface.
func (o *Error) Equal(right Object) bool {
	if v, ok := right.(*Error); ok {
		return v == o
	}
	return false
}

// IsFalsy implements Object interface.
func (o *Error) IsFalsy() bool { return true }

// IndexGet implements Object interface.
func (o *Error) IndexGet(_ *VM, index Object) (Object, error) {
	s := index.ToString()
	switch s {
	case "name":
		return Str(o.Name), nil
	case "message":
	try:
		if len(o.Message) == 0 && o.Cause != nil {
			if ce, ok := o.Cause.(*Error); ok {
				o = ce
				goto try
			}
			return Str(o.Cause.Error()), nil
		}
		return Str(o.Message), nil
	case "unwrap":
		if ge, _ := o.Cause.(*Error); ge != nil {
			return ge, nil
		}
		return Nil, nil
	case "cause":
		e := o
		for e.Cause != nil {
			if ge, _ := e.Cause.(*Error); ge != nil {
				if ge.Message == "" && ge.Name == e.Name {
					break
				}
				e = ge
			} else {
				break
			}
		}
		return e, nil
	case "New":
		return &Function{
			FuncName: "New",
			Value: func(c Call) (Object, error) {
				l := c.Args.Length()
				switch l {
				case 1:
					return o.NewError(c.Args.Get(0).ToString()), nil
				case 0:
					return o.NewError(o.Message), nil
				default:
					msgs := make([]string, l)
					for i := range msgs {
						msgs[i] = c.Args.Get(i).ToString()
					}
					return o.NewError(msgs...), nil
				}
			},
		}, nil
	default:
		return nil, ErrInvalidIndex.NewError(s)
	}
}

// NewError creates a new Error and sets original Error as its cause which can be unwrapped.
func (o *Error) NewError(messages ...string) *Error {
	cp := o.Copy().(*Error)
	cp.Message = strings.Join(messages, " ")
	cp.Cause = o
	return cp
}

// NewErrorf creates a new Error and sets original Error as its cause which can be unwrapped, using formatable message.
func (o *Error) NewErrorf(format string, arg ...any) *Error {
	cp := o.Copy().(*Error)
	cp.Message = fmt.Sprintf(format, arg...)
	cp.Cause = o
	return cp
}

func (o *Error) Wrapf(cause error, format string, arg ...any) *Error {
	cp := o.Copy().(*Error)
	cp.Message = fmt.Sprintf(format, arg...)
	cp.Cause = cause
	return cp
}

func (o *Error) Wrap(cause error, msg string) *Error {
	cp := o.Copy().(*Error)
	cp.Message = msg
	cp.Cause = cause
	return cp
}

// RuntimeError represents a runtime error that wraps Error and includes trace information.
type RuntimeError struct {
	Err     *Error
	fileSet *source.FileSet
	Trace   []source.Pos
}

var (
	_ Object = (*RuntimeError)(nil)
	_ Copier = (*RuntimeError)(nil)
)

func (o *RuntimeError) FileSet() *source.FileSet {
	return o.fileSet
}

func (o *RuntimeError) Unwrap() error {
	if o.Err != nil {
		return o.Err
	}
	return nil
}

func (o *RuntimeError) addTrace(pos source.Pos) {
	if len(o.Trace) > 0 {
		if o.Trace[len(o.Trace)-1] == pos {
			return
		}
	}
	o.Trace = append(o.Trace, pos)
}

func (*RuntimeError) Type() ObjectType {
	return TError
}

func (o *RuntimeError) ToString() string {
	return o.Error()
}

// Copy implements Copier interface.
func (o *RuntimeError) Copy() Object {
	var err *Error
	if o.Err != nil {
		err = o.Err.Copy().(*Error)
	}

	return &RuntimeError{
		Err:     err,
		fileSet: o.fileSet,
		Trace:   append([]source.Pos{}, o.Trace...),
	}
}

// Error implements error interface.
func (o *RuntimeError) Error() string {
	if o.Err == nil {
		return ReprQuote("nil")
	}
	return o.Err.Error()
}

// Equal implements Object interface.
func (o *RuntimeError) Equal(right Object) bool {
	if o.Err != nil {
		return o.Err.Equal(right)
	}
	return false
}

// IsFalsy implements Object interface.
func (o *RuntimeError) IsFalsy() bool { return true }

// IndexGet implements Object interface.
func (o *RuntimeError) IndexGet(vm *VM, index Object) (Object, error) {
	if o.Err != nil {
		s := index.ToString()
		if s == "New" {
			return &Function{
				FuncName: "New",
				Value: func(c Call) (Object, error) {
					l := c.Args.Length()
					switch l {
					case 1:
						return o.NewError(c.Args.Get(0).ToString()), nil
					case 0:
						return o.NewError(o.Err.Message), nil
					default:
						msgs := make([]string, l)
						for i := range msgs {
							msgs[i] = c.Args.Get(i).ToString()
						}
						return o.NewError(msgs...), nil
					}
				},
			}, nil
		}
		return o.Err.IndexGet(vm, index)
	}

	return Nil, nil
}

// NewError creates a new Error and sets original Error as its cause which can be unwrapped.
func (o *RuntimeError) NewError(messages ...string) *RuntimeError {
	cp := o.Copy().(*RuntimeError)
	cp.Err.Message = strings.Join(messages, " ")
	cp.Err.Cause = o
	return cp
}

// StackTrace returns stack trace if set otherwise returns nil.
func (o *RuntimeError) StackTrace() source.FilePosStackTrace {
	if o.fileSet == nil {
		if o.Trace != nil {
			sz := len(o.Trace)
			trace := make(source.FilePosStackTrace, sz)
			j := 0
			for i := sz - 1; i >= 0; i-- {
				trace[j] = source.FilePos{
					Offset: int(o.Trace[i]),
				}
				j++
			}
			return trace
		}
		return nil
	}

	sz := len(o.Trace)
	trace := make(source.FilePosStackTrace, sz)
	j := 0
	for i := sz - 1; i >= 0; i-- {
		trace[j] = o.fileSet.Position(o.Trace[i])
		j++
	}
	return trace
}

// Format implements fmt.Formater interface.
func (o *RuntimeError) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v', 's':
		switch {
		case s.Flag('+'):
			io.WriteString(s, o.ToString())
			o.StackTrace().Format(s, verb)

			e := o.Unwrap()
			for e != nil {
				if e, ok := e.(*RuntimeError); ok && o != e {
					e.Format(s, verb)
				}
				if err, ok := e.(interface{ Unwrap() error }); ok {
					e = err.Unwrap()
				} else {
					break
				}
			}
		default:
			_, _ = io.WriteString(s, o.ToString())
		}
	case 'q':
		_, _ = io.WriteString(s, strconv.Quote(o.ToString()))
	}
}
