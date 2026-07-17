package parser

import (
	"bytes"

	gnode "github.com/gad-lang/gad/parser/node"
)

type firstSpacerTrimWriter struct {
	w     bytes.Buffer
	wrote bool
}

func (w *firstSpacerTrimWriter) Write(p []byte) (n int, err error) {
	if w.wrote {
		return w.w.Write(p)
	}

	p2 := bytes.TrimLeftFunc(p, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})

	if len(p2) == 0 {
		n = len(p)
		return
	}

	w.wrote = true
	return w.w.Write(p2)
}

func (w *firstSpacerTrimWriter) String() string { return w.w.String() }
func (w *firstSpacerTrimWriter) Bytes() []byte  { return w.w.Bytes() }

// Format writes giom statements as formatted GAD output.
func Format(stmts gnode.Stmts) (_ []byte, err error) {
	var buf firstSpacerTrimWriter
	gnode.CodeW(&buf, stmts, gnode.CodeWithPrefix("\t"), gnode.CodeFormat())
	return buf.Bytes(), nil
}
