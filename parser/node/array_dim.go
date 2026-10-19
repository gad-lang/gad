package node

import (
	"strings"

	"github.com/gad-lang/gad/parser/source"
)

// ArrayDim is the size of a level of an array type, written between its
// brackets: `[N]` (exactly N: Exact), `[..B]` (at most B), `[A..]` (at least
// A), `[A..B]` (between A and B) — Min and Max with Range. A level written `[]`
// has no ArrayDim (nil). The sizes are int expressions.
type ArrayDim struct {
	LBrack source.Pos
	RBrack source.Pos
	Exact  Expr
	Range  bool
	Min    Expr
	Max    Expr
}

// String is the level as written: `[2]`, `[..3]`, `[2..]`, `[2..5]`; `[]` for
// nil.
func (d *ArrayDim) String() string {
	if d == nil {
		return "[]"
	}
	var b strings.Builder
	b.WriteByte('[')
	if d.Range {
		if d.Min != nil {
			b.WriteString(d.Min.String())
		}
		b.WriteString("..")
		if d.Max != nil {
			b.WriteString(d.Max.String())
		}
	} else if d.Exact != nil {
		b.WriteString(d.Exact.String())
	}
	b.WriteByte(']')
	return b.String()
}

// ArrayDimsString is the levels of an array type: dims (nil entries `[]`)
// padded with `[]` to depth.
func ArrayDimsString(depth int, dims []*ArrayDim) string {
	var b strings.Builder
	for i := 0; i < depth; i++ {
		if i < len(dims) {
			b.WriteString(dims[i].String())
		} else {
			b.WriteString("[]")
		}
	}
	return b.String()
}

// HasArrayDims reports whether a level of dims has a size.
func HasArrayDims(dims []*ArrayDim) bool {
	for _, d := range dims {
		if d != nil {
			return true
		}
	}
	return false
}
