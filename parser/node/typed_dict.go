package node

import (
	"strings"

	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/source"
)

// TypedDictTypeExpr is a typed dict type (issue #19): `type dict[K, V]` (a
// value, anonymous), `type NAME dict[K, V]` written where a value goes (a value,
// named — NAME is not declared) and `type dict` (`dict[str, any]`). A new object
// type, a child of the generic `DictType[K, V]`: its instances are bound to it,
// every write checked.
type TypedDictTypeExpr struct {
	TypePos source.Pos
	// Name is the type's name; nil: anonymous.
	Name *IdentExpr
	// Args are K and V (or V alone); none: [str, any].
	Args   []Expr
	EndPos source.Pos
	Meta   *KeyValueArrayLit
}

func (e *TypedDictTypeExpr) ExprNode() {}

func (e *TypedDictTypeExpr) Pos() source.Pos { return e.TypePos }

func (e *TypedDictTypeExpr) End() source.Pos { return e.EndPos }

func (e *TypedDictTypeExpr) String() string {
	var b strings.Builder
	b.WriteString("type ")
	if e.Name != nil {
		b.WriteString(e.Name.Name + " ")
	}
	b.WriteString("dict")
	if len(e.Args) > 0 {
		b.WriteString("[")
		for i, a := range e.Args {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(a.String())
		}
		b.WriteString("]")
	}
	return b.String()
}

func (e *TypedDictTypeExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteString(e.String())
}

// TypedDictTypeStmt is the declaration `type NAME dict[K, V]` (issue #19): NAME
// declared as the typed dict type named NAME.
type TypedDictTypeStmt struct {
	TypedDictTypeExpr
	Doc *ast.CommentGroup
}

func (s *TypedDictTypeStmt) StmtNode() {}

func (s *TypedDictTypeStmt) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteLeadDoc(s.Doc)
	if s.Meta != nil && len(s.Meta.Elements) > 0 {
		ctx.WriteString("[")
		for i, el := range s.Meta.Elements {
			if i > 0 {
				ctx.WriteString(", ")
			}
			el.WriteCode(ctx)
		}
		ctx.WriteString("]")
		ctx.WriteSecondLine()
		ctx.WritePrefix()
	}
	s.TypedDictTypeExpr.WriteCode(ctx)
}
