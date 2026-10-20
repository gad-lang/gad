package node

import "github.com/gad-lang/gad/parser/source"

// IndexEndExpr is `end` inside the brackets of an index or a slice, as Julia's:
// the last index of what is indexed — `len(x) - 1` (gad counts from 0) —, so
// `a[end]` is the last item and `a[end-1]` the one before it. The innermost
// brackets own it: in `a[b[end]]` it is b's.
type IndexEndExpr struct {
	TokenPos source.Pos
}

func (e *IndexEndExpr) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *IndexEndExpr) Pos() source.Pos {
	return e.TokenPos
}

// End returns the position of first character immediately after the node.
func (e *IndexEndExpr) End() source.Pos {
	return e.TokenPos + 3 // len("end") == 3
}

func (e *IndexEndExpr) String() string {
	return "end"
}

func (e *IndexEndExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString("end")
}
