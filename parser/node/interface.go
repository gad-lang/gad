package node

import (
	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/source"
)

// InterfaceMemberKind selects the kind of a simple (name + optional types)
// interface body member: a field, or a getter/setter/property accessor.
type InterfaceMemberKind uint8

const (
	// IfaceField is a typed field: `name` or `name Type`.
	IfaceField InterfaceMemberKind = iota
	// IfaceGet is a getter: `get name` or `get name Type`.
	IfaceGet
	// IfaceSet is a setter: `set name` or `set name Type`.
	IfaceSet
	// IfaceProp is a property (getter + setter shortcut): `prop name [Type]`.
	IfaceProp
)

func (k InterfaceMemberKind) String() string {
	switch k {
	case IfaceGet:
		return "get"
	case IfaceSet:
		return "set"
	case IfaceProp:
		return "prop"
	default:
		return ""
	}
}

// InterfaceMemberExpr is a field/getter/setter/property in an interface body: an
// optional `get`/`set`/`prop` keyword followed by a typed ident (`name` or
// `name Type1|Type2`).
type InterfaceMemberExpr struct {
	Kind  InterfaceMemberKind
	KwPos source.Pos // position of get/set/prop keyword; NoPos for a field
	Name  *TypedIdentExpr
	Doc   *ast.CommentGroup
}

func (e *InterfaceMemberExpr) ExprNode() {}

func (e *InterfaceMemberExpr) Pos() source.Pos {
	if e.KwPos.IsValid() {
		return e.KwPos
	}
	return e.Name.Pos()
}

func (e *InterfaceMemberExpr) End() source.Pos { return e.Name.End() }

func (e *InterfaceMemberExpr) String() string { return Code(e) }

func (e *InterfaceMemberExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteLeadDoc(e.Doc)
	if kw := e.Kind.String(); kw != "" {
		ctx.WriteString(kw)
		ctx.WriteString(" ")
	}
	e.Name.WriteCode(ctx)
}

// InterfaceExpr is an interface literal describing a structural contract:
//
//	interface [Name] {
//	  extends { Parent, … }
//	  field, field Type          // typed fields
//	  get g, set s, prop p        // accessors
//	  method(params) <return>     // required methods (func-header shape)
//	  parse { (params) <return> } // meti-style headers grouped as `parse`
//	}
type InterfaceExpr struct {
	InterfaceToken TokenLit
	NameExpr       Expr   // *IdentExpr or nil (anonymous)
	Parents        []Expr // extends { … } — no alias
	ExtendsDoc     *ast.CommentGroup
	Members        []*InterfaceMemberExpr // fields, getters, setters, props (source order)
	Methods        []*FuncHeaderExpr      // named method headers
	Parse          []*FuncHeaderExpr      // the `parse { … }` anonymous headers
	ParseDoc       *ast.CommentGroup
	LBrace         source.Pos
	RBrace         source.Pos
	Doc            *ast.CommentGroup // doc comment preceding the interface; or nil
}

func (e *InterfaceExpr) ExprNode() {}

func (e *InterfaceExpr) Pos() source.Pos {
	if e.InterfaceToken.Pos != source.NoPos {
		return e.InterfaceToken.Pos
	}
	return e.LBrace
}

func (e *InterfaceExpr) End() source.Pos { return e.RBrace + 1 }

func (e *InterfaceExpr) String() string { return Code(e) }

// NameIdent returns the interface name identifier, or nil when anonymous.
func (e *InterfaceExpr) NameIdent() *IdentExpr {
	id, _ := e.NameExpr.(*IdentExpr)
	return id
}

func (e *InterfaceExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteLeadDoc(e.Doc)
	ctx.WriteString("interface")
	if e.NameExpr != nil {
		ctx.WriteString(" ")
		e.NameExpr.WriteCode(ctx)
	}
	ctx.WriteString(" {")
	ctx.Depth++
	if len(e.Parents) > 0 {
		ctx.WriteLeadDoc(e.ExtendsDoc)
		ctx.WriteString("extends {")
		for i, p := range e.Parents {
			if i > 0 {
				ctx.WriteString(", ")
			}
			ctx.WriteString(p.String())
		}
		ctx.WriteString("}")
		ctx.WriteSemi()
	}
	for _, m := range e.Members {
		m.WriteCode(ctx)
		ctx.WriteSemi()
	}
	for _, m := range e.Methods {
		ctx.WriteLeadDoc(m.Doc)
		// Interface methods render without the `<…>` header brackets.
		ctx.WriteString(m.FuncHeader.String())
		ctx.WriteSemi()
	}
	if len(e.Parse) > 0 {
		ctx.WriteLeadDoc(e.ParseDoc)
		ctx.WriteString("parse {")
		ctx.Depth++
		for _, h := range e.Parse {
			ctx.WriteString(h.FuncHeader.String())
			ctx.WriteSemi()
		}
		ctx.Depth--
		ctx.WriteString("}")
		ctx.WriteSemi()
	}
	ctx.Depth--
	ctx.WriteString("}")
}

// InterfaceStmt is the statement form `interface Name { … }`, which binds a
// const to the interface value.
type InterfaceStmt struct {
	InterfaceExpr
}

func (s *InterfaceStmt) StmtNode() {}
