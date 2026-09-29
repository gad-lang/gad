package node

import (
	"strings"

	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/token"
)

// TypeArgsExpr is a generic class or interface given its type arguments —
// `Box[int]`, `Pair[str, int]` —, where a type goes and where a value does.
// (One argument written as a value, `Box[int]`, parses as an IndexExpr; the
// compiler reads either against the generics in scope.)
type TypeArgsExpr struct {
	X      Expr
	LBrack source.Pos
	Args   []Expr
	RBrack source.Pos
}

func (e *TypeArgsExpr) ExprNode() {}

// Pos returns the position of first character belonging to the node.
func (e *TypeArgsExpr) Pos() source.Pos { return e.X.Pos() }

// End returns the position of first character immediately after the node.
func (e *TypeArgsExpr) End() source.Pos { return e.RBrack + 1 }

func (e *TypeArgsExpr) String() string {
	args := make([]string, len(e.Args))
	for i, a := range e.Args {
		args[i] = typeArgString(a)
	}
	return e.X.String() + "[" + strings.Join(args, ", ") + "]"
}

func (e *TypeArgsExpr) WriteCode(ctx *CodeWriteContext) {
	e.X.WriteCode(ctx)
	ctx.WriteString("[")
	for i, a := range e.Args {
		if i > 0 {
			ctx.WriteString(", ")
		}
		ctx.WriteString(typeArgString(a))
	}
	ctx.WriteString("]")
}

// typeArgString is the type argument e as a type is written: a union
// `int|str`, not the expression `(int | str)`.
func typeArgString(e Expr) string {
	if b, ok := e.(*BinaryExpr); ok && b.Token == token.Or {
		return typeArgString(b.LHS) + "|" + typeArgString(b.RHS)
	}
	return e.String()
}
