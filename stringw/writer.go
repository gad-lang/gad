package stringw

import (
	"fmt"
	"io"
	"strings"
)

type StringWriter interface {
	io.Writer
	io.StringWriter
}

type StringerTo interface {
	StringTo(w StringWriter)
}

func ToStringW(w StringWriter, v any) {
	switch t := v.(type) {
	case io.WriterTo:
		t.WriteTo(w)
	case StringerTo:
		t.StringTo(w)
	default:
		fmt.Fprint(w, v)
	}
}

func ToString(v any) string {
	var s strings.Builder
	ToStringW(&s, v)
	return s.String()
}

func ToStringSlice[T any](w StringWriter, sep string, s []T) {
	l := len(s)
	if l == 0 {
		return
	}
	for i := 0; i < l-1; i++ {
		ToStringW(w, s[i])
		w.WriteString(sep)
	}
	ToStringW(w, s[l-1])
}

func Each[T any](s []T, do func(e any, last bool)) {
	l := len(s)
	if l == 0 {
		return
	}
	for i := 0; i < l-1; i++ {
		do(s[i], false)
	}
	do(s[l-1], true)
}
