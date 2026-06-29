package node

import (
	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/source"
)

// ClassParentExpr is one parent in a class `extends` list: a parent type
// expression with an optional alias (`extends Base as B`).
type ClassParentExpr struct {
	Type  Expr
	Alias *IdentExpr // optional; nil when written without `as alias`
}

func (e *ClassParentExpr) Pos() source.Pos { return e.Type.Pos() }

func (e *ClassParentExpr) End() source.Pos {
	if e.Alias != nil {
		return e.Alias.End()
	}
	return e.Type.End()
}

func (e *ClassParentExpr) String() string {
	if e.Alias != nil {
		return e.Type.String() + " as " + e.Alias.String()
	}
	return e.Type.String()
}

func (e *ClassParentExpr) WriteCode(ctx *CodeWriteContext) {
	e.Type.WriteCode(ctx)
	if e.Alias != nil {
		ctx.WriteString(" as ")
		e.Alias.WriteCode(ctx)
	}
}

// ClassFieldExpr is a declared field in a class body: `name`, `name = value`,
// `name Type = value`, or a computed default `name = (= expr)` (Value is then a
// *ComputedExpr, evaluated per instance).
type ClassFieldExpr struct {
	Name   *TypedIdentExpr
	Assign source.Pos
	Value  Expr              // default value; nil when none
	Doc    *ast.CommentGroup // doc comment preceding the field; or nil
}

func (e *ClassFieldExpr) ExprNode() {}

func (e *ClassFieldExpr) Pos() source.Pos { return e.Name.Pos() }

func (e *ClassFieldExpr) End() source.Pos {
	if e.Value != nil {
		return e.Value.End()
	}
	return e.Name.End()
}

func (e *ClassFieldExpr) String() string { return Code(e) }

func (e *ClassFieldExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteLeadDoc(e.Doc)
	e.Name.WriteCode(ctx)
	if e.Value != nil {
		ctx.WriteString(" = ")
		e.Value.WriteCode(ctx)
	}
}

// ClassMemberExpr is a named method (or, inside `props {}`, a property) in a
// class body: a name plus one (single form) or several (brace-block form)
// FuncMethod overloads, sharing the func-with-methods/prop body syntax.
type ClassMemberExpr struct {
	NameExpr Expr
	Methods  []*FuncMethod
	Block    bool // written in the brace-block form `name { (…) … }`
	LBrace   source.Pos
	RBrace   source.Pos
	Doc      *ast.CommentGroup // doc comment preceding the member; or nil
}

func (e *ClassMemberExpr) ExprNode() {}

func (e *ClassMemberExpr) Pos() source.Pos {
	if e.NameExpr != nil {
		return e.NameExpr.Pos()
	}
	if len(e.Methods) > 0 {
		return e.Methods[0].Pos()
	}
	return e.LBrace
}

func (e *ClassMemberExpr) End() source.Pos {
	if e.RBrace.IsValid() {
		return e.RBrace + 1
	}
	if l := len(e.Methods); l > 0 {
		return e.Methods[l-1].End()
	}
	return source.NoPos
}

func (e *ClassMemberExpr) String() string { return Code(e) }

func (e *ClassMemberExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteLeadDoc(e.Doc)
	if e.NameExpr != nil {
		e.NameExpr.WriteCode(ctx)
	}
	if e.Block {
		ctx.WriteString(" {")
		writeClassMethodBlock(ctx, e.Methods)
		ctx.WriteString("}")
		return
	}
	if len(e.Methods) == 1 {
		e.Methods[0].WriteCode(ctx)
	}
}

// ClassExpr is a class literal:
//
//	class [Name] [extends P, …] { fields, props {…}, new …, methods {…} }
//
// It lowers (in the compiler) to a
//
//	Class(name; define=(Type, define) => define(; extends=…, fields=…,
//	    properties=…, methods=…, new=…))
//
// call. The `define` callback binds `Type` to the in-construction class so each
// method, property accessor and constructor can take a typed `this Type` first
// parameter (injected by the compiler). NameExpr is nil for an anonymous,
// expression-form class.
type ClassExpr struct {
	ClassToken TokenLit
	NameExpr   Expr
	Parents    []*ClassParentExpr
	Fields     []*ClassFieldExpr
	Props      []*ClassMemberExpr
	PropsDoc   *ast.CommentGroup
	New        []*FuncMethod
	NewDoc     *ast.CommentGroup
	Methods    []*ClassMemberExpr
	MethodsDoc *ast.CommentGroup
	LBrace     source.Pos
	RBrace     source.Pos
	Doc        *ast.CommentGroup // doc comment preceding the class; or nil
}

func (e *ClassExpr) ExprNode() {}

func (e *ClassExpr) Pos() source.Pos {
	if e.ClassToken.Pos != source.NoPos {
		return e.ClassToken.Pos
	}
	return e.LBrace
}

func (e *ClassExpr) End() source.Pos { return e.RBrace + 1 }

func (e *ClassExpr) String() string { return Code(e) }

func (e *ClassExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteLeadDoc(e.Doc)
	ctx.WriteString("class")
	if e.NameExpr != nil {
		ctx.WriteString(" ")
		e.NameExpr.WriteCode(ctx)
	}
	if len(e.Parents) > 0 {
		ctx.WriteString(" extends ")
		for i, p := range e.Parents {
			if i > 0 {
				ctx.WriteString(", ")
			}
			p.WriteCode(ctx)
		}
	}
	ctx.WriteString(" {")

	ctx.Depth++
	for _, f := range e.Fields {
		ctx.WriteSecondLine()
		ctx.WritePrefix()
		f.WriteCode(ctx)
	}
	if len(e.Props) > 0 {
		ctx.WriteSecondLine()
		ctx.WritePrefix()
		ctx.WriteLeadDoc(e.PropsDoc)
		ctx.WriteString("props {")
		writeClassMembers(ctx, e.Props)
		ctx.WriteString("}")
	}
	if len(e.New) > 0 {
		ctx.WriteSecondLine()
		ctx.WritePrefix()
		ctx.WriteLeadDoc(e.NewDoc)
		ctx.WriteString("new {")
		writeClassMethodBlock(ctx, e.New)
		ctx.WriteString("}")
	}
	if len(e.Methods) > 0 {
		ctx.WriteSecondLine()
		ctx.WritePrefix()
		ctx.WriteLeadDoc(e.MethodsDoc)
		ctx.WriteString("methods {")
		writeClassMembers(ctx, e.Methods)
		ctx.WriteString("}")
	}
	ctx.Depth--

	ctx.WriteSecondLine()
	ctx.WritePrefix()
	ctx.WriteString("}")
}

// writeClassMembers emits the entries of a `props {}` / `methods {}` block, one
// per indented line.
func writeClassMembers(ctx *CodeWriteContext, members []*ClassMemberExpr) {
	ctx.Depth++
	for _, m := range members {
		ctx.WriteSecondLine()
		ctx.WritePrefix()
		m.WriteCode(ctx)
	}
	ctx.Depth--
	ctx.WriteSecondLine()
	ctx.WritePrefix()
}

// writeClassMethodBlock emits the overloads of a brace-block member (`name {
// (…) … }`) or the `new {}` block, one per indented line.
func writeClassMethodBlock(ctx *CodeWriteContext, methods []*FuncMethod) {
	ctx.Depth++
	for _, m := range methods {
		ctx.WriteSecondLine()
		ctx.WritePrefix()
		m.WriteCode(ctx)
	}
	ctx.Depth--
	ctx.WriteSecondLine()
	ctx.WritePrefix()
}

// ClassStmt is the statement form `class Name [extends …] { … }`. It compiles
// to `const Name = <class expression>`.
type ClassStmt struct {
	ClassExpr
}

func (*ClassStmt) StmtNode() {}
