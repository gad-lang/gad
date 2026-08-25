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
//	  *Parent, …                  // parent interfaces (spread)
//	  field, field Type          // typed fields
//	  get g, set s, prop p        // accessors
//	  method(params) <return>     // required methods (func-header shape)
//	  parse { (params) <return> } // meti-style headers grouped as `parse`
//	}
type InterfaceExpr struct {
	InterfaceToken TokenLit
	// ArrayDepth is the number of `[]` written right after the `interface` keyword
	// (`interface[] P`, `interface[][][] P`): the interface then matches an array
	// nested to this depth whose leaf elements each satisfy the body. 0 for a plain
	// interface.
	ArrayDepth int
	NameExpr   Expr // *IdentExpr or nil (anonymous)
	Parents        []Expr // *Parent spreads — no alias
	ExtendsDoc     *ast.CommentGroup
	Members        []*InterfaceMemberExpr      // fields, getters, setters, props (source order)
	Methods        []*InterfaceMethodExpr      // required methods (one or more signatures each)
	ContextFuncs   []*InterfaceContextFuncExpr // context-function checks (`funcs { … }`)
	// Rest is the `**name` rest-capture field: when the interface is used to cast
	// a dict (`d :: I`), keys not named by the interface are collected into a dict
	// bound to this name in the result. Nil when the interface has no `**` member.
	Rest    *IdentExpr
	RestDoc *ast.CommentGroup
	LBrace  source.Pos
	RBrace  source.Pos
	Doc     *ast.CommentGroup // doc comment preceding the interface; or nil
}

// InterfaceContextFuncExpr is one entry of an interface's `funcs { … }` section:
// `FnExpr <(params)>` or `FnExpr { (params); … }`. It requires that the function
// value `FnExpr` (an ident, selector or any expression, captured by value where
// the interface is declared) has, for each header, a signature matching it —
// with the special `@self` positional param standing for the interface's own
// type. Every header must contain at least one `@self`.
type InterfaceContextFuncExpr struct {
	FnExpr  Expr              // the context function (ident or selector, …)
	Headers []*FuncHeaderExpr // required signature(s), anonymous
	Block   bool              // written in the brace-block form `{ … }`
	LBrace  source.Pos
	RBrace  source.Pos
	Doc     *ast.CommentGroup
}

func (e *InterfaceContextFuncExpr) ExprNode() {}

func (e *InterfaceContextFuncExpr) Pos() source.Pos { return e.FnExpr.Pos() }

func (e *InterfaceContextFuncExpr) End() source.Pos {
	if e.RBrace.IsValid() {
		return e.RBrace + 1
	}
	if n := len(e.Headers); n > 0 {
		return e.Headers[n-1].End()
	}
	return e.FnExpr.End()
}

func (e *InterfaceContextFuncExpr) String() string { return Code(e) }

// WriteCode writes one context-function entry `FnExpr <header>` (or the block
// form `FnExpr { (…); … }`). It is emitted inside the interface's `funcs { … }`
// section, so it carries no `:` prefix.
func (e *InterfaceContextFuncExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteLeadDoc(e.Doc)
	e.FnExpr.WriteCode(ctx)
	if e.Block {
		ctx.WriteString(" {")
		ctx.Depth++
		for _, h := range e.Headers {
			ctx.WriteString(h.FuncHeader.String())
			ctx.WriteSemi()
		}
		ctx.Depth--
		ctx.WriteString("}")
		return
	}
	if len(e.Headers) == 1 {
		ctx.WriteString(" ")
		ctx.WriteString(e.Headers[0].String())
	}
}

// InterfaceMethodExpr is a required method of an interface: a name and one or
// more signatures (func-header shape, without the `<…>` brackets). Written
// either single `name(params) <return>` or block `name { (params) <return>, … }`
// (the block form is how the `parse` example groups several signatures).
type InterfaceMethodExpr struct {
	NameExpr *IdentExpr
	Headers  []*FuncHeaderExpr // the signature(s), anonymous (the name is on NameExpr)
	Block    bool              // written in the brace-block form
	LBrace   source.Pos
	RBrace   source.Pos
	Doc      *ast.CommentGroup
}

func (e *InterfaceMethodExpr) ExprNode() {}

func (e *InterfaceMethodExpr) Pos() source.Pos { return e.NameExpr.Pos() }

func (e *InterfaceMethodExpr) End() source.Pos {
	if e.RBrace.IsValid() {
		return e.RBrace + 1
	}
	if n := len(e.Headers); n > 0 {
		return e.Headers[n-1].End()
	}
	return e.NameExpr.End()
}

func (e *InterfaceMethodExpr) String() string { return Code(e) }

func (e *InterfaceMethodExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteLeadDoc(e.Doc)
	e.NameExpr.WriteCode(ctx)
	if e.Block {
		ctx.WriteString(" {")
		ctx.Depth++
		for _, h := range e.Headers {
			ctx.WriteString(h.FuncHeader.String())
			ctx.WriteSemi()
		}
		ctx.Depth--
		ctx.WriteString("}")
		return
	}
	if len(e.Headers) == 1 {
		ctx.WriteString(e.Headers[0].FuncHeader.String())
	}
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
	for i := 0; i < e.ArrayDepth; i++ {
		ctx.WriteString("[]")
	}
	if e.NameExpr != nil {
		ctx.WriteString(" ")
		e.NameExpr.WriteCode(ctx)
	}
	ctx.WriteString(" {")
	ctx.Depth++
	for i, p := range e.Parents {
		if i == 0 {
			ctx.WriteLeadDoc(e.ExtendsDoc)
		}
		ctx.WriteString("*")
		ctx.WriteString(p.String())
		ctx.WriteSemi()
	}
	for _, m := range e.Members {
		m.WriteCode(ctx)
		ctx.WriteSemi()
	}
	for _, m := range e.Methods {
		m.WriteCode(ctx)
		ctx.WriteSemi()
	}
	if e.Rest != nil {
		ctx.WriteLeadDoc(e.RestDoc)
		ctx.WriteString("**")
		e.Rest.WriteCode(ctx)
		ctx.WriteSemi()
	}
	if len(e.ContextFuncs) > 0 {
		ctx.WriteString("funcs {")
		ctx.Depth++
		for _, m := range e.ContextFuncs {
			m.WriteCode(ctx)
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
