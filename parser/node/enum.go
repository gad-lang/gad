package node

import (
	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/token"
)

// EnumFieldExpr is one field of an enum body: an optional `bit` prefix, an
// optional `+`/`-` sign, the field Name (`_` is a placeholder that advances the
// running value but is not added to the enum) and an optional explicit `= Value`
// (which may reference earlier fields, e.g. `All = Read | Write`).
type EnumFieldExpr struct {
	Bit     bool        // written with a leading `bit`
	Sign    token.Token // token.Add, token.Sub, or token.Illegal for none
	SignPos source.Pos
	Name    *IdentExpr
	Assign  source.Pos
	Value   Expr              // explicit value; nil when defaulted
	Doc     *ast.CommentGroup // doc comment preceding the field; or nil
}

func (e *EnumFieldExpr) ExprNode() {}

func (e *EnumFieldExpr) Pos() source.Pos {
	if e.Bit && e.SignPos.IsValid() {
		return e.SignPos
	}
	return e.Name.Pos()
}

func (e *EnumFieldExpr) End() source.Pos {
	if e.Value != nil {
		return e.Value.End()
	}
	return e.Name.End()
}

func (e *EnumFieldExpr) String() string { return Code(e) }

func (e *EnumFieldExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteLeadDoc(e.Doc)
	if e.Bit {
		ctx.WriteString("bit ")
	}
	if e.Sign == token.Add || e.Sign == token.Sub {
		ctx.WriteString(e.Sign.String())
	}
	e.Name.WriteCode(ctx)
	if e.Value != nil {
		ctx.WriteString(" = ")
		e.Value.WriteCode(ctx)
	}
}

// EnumExpr is an enum literal: `enum [Name] { field, … }`. It compiles to an
// Enum constant whose field values are computed at compile time (incrementing
// integers by default, bit flags under `bit`, or explicit expressions).
// NameExpr is nil for an anonymous, expression-form enum.
type EnumExpr struct {
	EnumToken TokenLit
	NameExpr  Expr
	Fields    []*EnumFieldExpr
	LBrace    source.Pos
	RBrace    source.Pos
	Doc       *ast.CommentGroup // doc comment preceding the enum; or nil
}

func (e *EnumExpr) ExprNode() {}

func (e *EnumExpr) Pos() source.Pos {
	if e.EnumToken.Pos != source.NoPos {
		return e.EnumToken.Pos
	}
	return e.LBrace
}

func (e *EnumExpr) End() source.Pos { return e.RBrace + 1 }

func (e *EnumExpr) String() string { return Code(e) }

func (e *EnumExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteLeadDoc(e.Doc)
	ctx.WriteString("enum")
	if e.NameExpr != nil {
		ctx.WriteString(" ")
		e.NameExpr.WriteCode(ctx)
	}
	ctx.WriteString(" {")
	writeEnumFields(ctx, e.Fields)
	ctx.WriteString("}")
}

// writeEnumFields emits the enum fields one per indented line when formatting
// with a prefix and `, `-separated inline otherwise.
func writeEnumFields(ctx *CodeWriteContext, fields []*EnumFieldExpr) {
	ctx.WriteItemsSep(ctx.HasPrefix(), len(fields), ", ", "", func(i int) {
		fields[i].WriteCode(ctx)
	}, func(newLine bool) {
		if newLine {
			ctx.WriteSecondLine()
		}
	})
	if len(fields) > 0 && ctx.HasPrefix() {
		ctx.WritePrefix()
	}
}

// EnumStmt is the statement form `enum Name { … }`. It compiles to
// `const Name = <enum expression>`.
type EnumStmt struct {
	EnumExpr
}

func (*EnumStmt) StmtNode() {}
