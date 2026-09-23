package node

import (
	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/source"
)

// TypedArrayTypeStmt is a named typed-array type declaration:
//
//	type numerics []<int|uint|float|decimal>   // element types in an envelope
//	type ints []int                            // one bare element type
//	type users []{ name; id }                  // an inline interface element
//	type users [] interface { name; id }       // the same, long form
//	type grid [][]int                          // deeper nesting
//
// It compiles to `const NAME = TypedArrayType("NAME", depth, elem; meta=…)`: a
// TypedArrayType whose instances (`numerics(1, 2.5)`) are typed arrays.
type TypedArrayTypeStmt struct {
	TypePos  source.Pos
	NameExpr *IdentExpr
	// Depth is the number of `[]` written (>= 1).
	Depth int
	// ElemTypes are the element types (`<int|uint>` or a bare `int`); nil when the
	// element is an inline interface (ElemIface).
	ElemTypes []*TypeExpr
	RAngle    source.Pos // the envelope's closing `>`; NoPos for a bare type
	// ElemIface is the inline interface element (`[]{ … }`); nil otherwise.
	ElemIface *InterfaceExpr
	Doc       *ast.CommentGroup
	// Meta is the optional `[k=v, …]` metadata block preceding the declaration.
	Meta *KeyValueArrayLit
	// Body is the optional class-like member block after the element
	// (`type T []int { label str; new(…) {…}; props {…}; methods {…} }`): fields,
	// constructor overloads, properties and methods (no parents, `use`, `this` or
	// `call`). nil when there is none.
	Body *TypeLitExpr
}

func (s *TypedArrayTypeStmt) StmtNode() {}

func (s *TypedArrayTypeStmt) Pos() source.Pos { return s.TypePos }

func (s *TypedArrayTypeStmt) End() source.Pos {
	switch {
	case s.Body != nil:
		return s.Body.End()
	case s.ElemIface != nil:
		return s.ElemIface.End()
	case s.RAngle.IsValid():
		return s.RAngle + 1
	case len(s.ElemTypes) > 0:
		return s.ElemTypes[len(s.ElemTypes)-1].End()
	}
	return s.NameExpr.End()
}

func (s *TypedArrayTypeStmt) String() string { return Code(s) }

// WriteCode renders `type NAME []…ELEM`, the element as `int`, `<int | uint>` or
// `{ … }` (an inline interface; the long `interface { … }` form prints short).
func (s *TypedArrayTypeStmt) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteLeadDoc(s.Doc)
	writeMeta(ctx, s.Meta)
	ctx.WriteString("type ")
	s.NameExpr.WriteCode(ctx)
	ctx.WriteString(" ")
	for i := 0; i < s.Depth; i++ {
		ctx.WriteString("[]")
	}
	if s.ElemIface != nil {
		ctx.WriteString("{")
		writeInterfaceBody(ctx, s.ElemIface)
		ctx.WriteString("}")
	} else {
		ctx.WriteString((&InterfaceExpr{ElemTypes: s.ElemTypes}).elemTypesCode())
	}
	if s.Body != nil {
		ctx.WriteString(" {")
		writeTypeLitBody(ctx, s.Body)
	}
}
