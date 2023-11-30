package gad

import (
	"fmt"
	"io"
)

type ObjectToWriter interface {
	WriteTo(vm *VM, w io.Writer, obj Object) (handled bool, n int64, err error)
}

type ObjectToWriterFunc func(vm *VM, w io.Writer, obj Object) (handled bool, n int64, err error)

func (f ObjectToWriterFunc) WriteTo(vm *VM, w io.Writer, obj Object) (handled bool, n int64, err error) {
	return f(vm, w, obj)
}

var DefaultObjectToWrite ObjectToWriterFunc = func(vm *VM, w io.Writer, obj Object) (handled bool, n int64, err error) {
	if ToWritable(obj) {
		n, err = obj.(ToWriter).WriteTo(vm, w)
	} else {
		var n32 int
		n32, err = fmt.Fprint(w, obj)
		n += int64(n32)
	}
	handled = true
	return
}

type ObjectToWriters []ObjectToWriter

func (o ObjectToWriters) WriteTo(vm *VM, w io.Writer, obj Object) (handled bool, n int64, err error) {
	for _, handler := range o {
		if handled, n, err = handler.WriteTo(vm, w, obj); handled {
			return
		}
	}
	return
}

func (o ObjectToWriters) Prepend(handlers ...ObjectToWriter) ObjectToWriters {
	return append(handlers, o...)
}

func (o ObjectToWriters) Append(handlers ...ObjectToWriter) ObjectToWriters {
	return append(o, handlers...)
}
