package node

import "github.com/gad-lang/gad/parser/source"

// Return represents an return expression.
type Return struct {
	ReturnPos source.Pos
	Result    Expr
}

// Pos returns the position of first character belonging to the node.
func (s *Return) Pos() source.Pos {
	return s.ReturnPos
}

// End returns the position of first character immediately after the node.
func (s *Return) End() source.Pos {
	return s.Result.End()
}

func (s *Return) String() string {
	var expr string
	if s.Result != nil {
		expr = " " + s.Result.String()
	}
	return "return" + expr
}

func (s *Return) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString("return")
	if s.Result != nil {
		ctx.WriteSingleByte(' ')
		s.Result.WriteCode(ctx)
	}
}
