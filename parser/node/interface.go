package node

import (
	"sort"
	"strings"

	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/source"
)

// sortedInterfaceMembers returns the interface's fields and property-like members
// (get/set/prop) in canonical order: fields first, grouped untyped-before-typed
// and sorted by name within each group, then the property-like members sorted by
// name (declaration order preserved among same-named getter/setter pairs). The
// input slice is not mutated.
func sortedInterfaceMembers(members []*InterfaceMemberExpr) []*InterfaceMemberExpr {
	out := make([]*InterfaceMemberExpr, len(members))
	copy(out, members)
	rank := func(m *InterfaceMemberExpr) int {
		if m.Kind != IfaceField {
			return 2 // property-like members sort after all fields
		}
		if m.Name != nil && len(m.Name.Type) > 0 {
			return 1 // typed field
		}
		return 0 // untyped field
	}
	name := func(m *InterfaceMemberExpr) string {
		if m.Name != nil && m.Name.Ident != nil {
			return m.Name.Ident.Name
		}
		return ""
	}
	sort.SliceStable(out, func(i, j int) bool {
		if ri, rj := rank(out[i]), rank(out[j]); ri != rj {
			return ri < rj
		}
		return name(out[i]) < name(out[j])
	})
	return out
}

// sortedInterfaceMethods returns the required methods sorted by name. The input
// slice is not mutated.
func sortedInterfaceMethods(methods []*InterfaceMethodExpr) []*InterfaceMethodExpr {
	out := make([]*InterfaceMethodExpr, len(methods))
	copy(out, methods)
	sort.SliceStable(out, func(i, j int) bool {
		var ni, nj string
		if out[i].NameExpr != nil {
			ni = out[i].NameExpr.Name
		}
		if out[j].NameExpr != nil {
			nj = out[j].NameExpr.Name
		}
		return ni < nj
	})
	return out
}

// sortedInterfaceContextFuncs returns the `funcs { … }` members sorted by their
// function expression's source form. The input slice is not mutated.
func sortedInterfaceContextFuncs(funcs []*InterfaceContextFuncExpr) []*InterfaceContextFuncExpr {
	out := make([]*InterfaceContextFuncExpr, len(funcs))
	copy(out, funcs)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].FnExpr.String() < out[j].FnExpr.String()
	})
	return out
}

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
	// Meta is the optional `[k=v, …]` metadata block declared between the doc
	// comment and the member; it compiles to a KeyValueArray reachable as
	// `Iface.member.@meta`.
	Meta *KeyValueArrayLit
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
	writeMeta(ctx, e.Meta)
	if kw := e.Kind.String(); kw != "" {
		ctx.WriteString(kw)
		ctx.WriteString(" ")
	}
	// A field whose type is an anonymous nested interface always renders in the
	// short form `name: { … }` (a slice interface as `name: []{ … }`, deeper as
	// `[][]…`), never `name interface { … }` — both parse to the same field.
	if e.Kind == IfaceField && e.Name != nil && len(e.Name.Type) == 1 {
		if iface, ok := e.Name.Type[0].Expr.(*InterfaceExpr); ok && iface.NameExpr == nil {
			ctx.WriteString(e.Name.nameCode())
			ctx.WriteString(": ")
			for i := 0; i < iface.ArrayDepth; i++ {
				ctx.WriteString("[]")
			}
			ctx.WriteString("{")
			writeInterfaceBody(ctx, iface)
			ctx.WriteString("}")
			return
		}
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
	// ArrayDepth is the number of `[]` written after the interface name
	// (`interface P [] { … }`, `interface P [][][] { … }`; anonymous
	// `interface [] { … }`): the interface then matches an array nested to this
	// depth whose leaf elements each satisfy the body (or ElemTypes). 0 for a
	// plain interface.
	ArrayDepth int
	// ElemTypes are the leaf element types of a slice-of-types interface,
	// `interface P []<int|uint>` (or `interface P []int`): the named analogue of
	// an anonymous `[]<int|uint>` slice type. Such an interface has no body; nil
	// otherwise. RAngle is the envelope's closing `>` (NoPos for a bare type).
	ElemTypes    []*TypeExpr
	RAngle       source.Pos
	NameExpr     Expr   // *IdentExpr or nil (anonymous)
	Parents      []Expr // *Parent spreads — no alias
	ExtendsDoc   *ast.CommentGroup
	Members      []*InterfaceMemberExpr      // fields, getters, setters, props (source order)
	Methods      []*InterfaceMethodExpr      // required methods (one or more signatures each)
	ContextFuncs []*InterfaceContextFuncExpr // context-function checks (`funcs { … }`)
	// Rest is the `**name` rest-capture field: when the interface is used to cast
	// a dict (`d :: I`), keys not named by the interface are collected into a dict
	// bound to this name in the result. Nil when the interface has no `**` member.
	Rest    *IdentExpr
	RestDoc *ast.CommentGroup
	LBrace  source.Pos
	RBrace  source.Pos
	Doc     *ast.CommentGroup // doc comment preceding the interface; or nil
	// Meta is the optional `[k=v, …]` metadata block preceding the interface.
	Meta *KeyValueArrayLit
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
	// Meta is the optional `[k=v, …]` metadata block for this method.
	Meta *KeyValueArrayLit
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
	writeMeta(ctx, e.Meta)
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

func (e *InterfaceExpr) End() source.Pos {
	if len(e.ElemTypes) > 0 {
		if e.RAngle.IsValid() {
			return e.RAngle + 1
		}
		return e.ElemTypes[len(e.ElemTypes)-1].End()
	}
	return e.RBrace + 1
}

// elemTypesCode renders the ElemTypes of a slice-of-types interface: one type
// bare (`int`), several in the `<…>` envelope (`<int | uint>`), like a
// SliceTypeExpr element.
func (e *InterfaceExpr) elemTypesCode() string {
	if len(e.ElemTypes) == 1 {
		return e.ElemTypes[0].String()
	}
	names := make([]string, len(e.ElemTypes))
	for i, t := range e.ElemTypes {
		names[i] = t.String()
	}
	return "<" + strings.Join(names, " | ") + ">"
}

func (e *InterfaceExpr) String() string { return Code(e) }

// NameIdent returns the interface name identifier, or nil when anonymous.
func (e *InterfaceExpr) NameIdent() *IdentExpr {
	id, _ := e.NameExpr.(*IdentExpr)
	return id
}

// MetaCode renders a `[k=v, …]` metadata block as source (`[a=1, b]`), or ""
// when meta is nil/empty — for documentation and tooling.
func MetaCode(meta *KeyValueArrayLit) string {
	if meta == nil || len(meta.Elements) == 0 {
		return ""
	}
	parts := make([]string, len(meta.Elements))
	for i, el := range meta.Elements {
		parts[i] = el.String()
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// writeMeta emits a `[k=v, …]` metadata block on its own line before a
// doc-commentable element (declared between the doc comment and the element).
func writeMeta(ctx *CodeWriteContext, meta *KeyValueArrayLit) {
	if meta == nil || len(meta.Elements) == 0 {
		return
	}
	ctx.WriteString("[")
	for i, el := range meta.Elements {
		if i > 0 {
			ctx.WriteString(", ")
		}
		ctx.WriteString(el.String())
	}
	// Formatting: the block sits on its own line, between the doc comment and
	// the element; compact output keeps it inline.
	if ctx.HasPrefix() {
		ctx.WriteString("]\n", ctx.CurrentPrefix())
		return
	}
	ctx.WriteString("] ")
}

func (e *InterfaceExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteLeadDoc(e.Doc)
	writeMeta(ctx, e.Meta)
	ctx.WriteString("interface")
	if e.NameExpr != nil {
		ctx.WriteString(" ")
		e.NameExpr.WriteCode(ctx)
	}
	// A slice interface: `interface P []{ … }` / `interface P []<int | uint>`
	// (anonymous `interface []{ … }`); the `[]`s follow the name.
	if e.ArrayDepth > 0 {
		ctx.WriteString(" ")
		for i := 0; i < e.ArrayDepth; i++ {
			ctx.WriteString("[]")
		}
		if len(e.ElemTypes) > 0 {
			ctx.WriteString(e.elemTypesCode())
			return
		}
		ctx.WriteString("{")
	} else {
		ctx.WriteString(" {")
	}
	writeInterfaceBody(ctx, e)
	ctx.WriteString("}")
}

// writeInterfaceBody emits an interface's members (parents, fields/getters/…,
// methods, `**rest`, `funcs {}`) between the enclosing braces — shared by the
// `interface { … }` form and a mixin's `this { … }` block.
func writeInterfaceBody(ctx *CodeWriteContext, e *InterfaceExpr) {
	if ctx.HasPrefix() {
		writeInterfaceBodyLines(ctx, e)
		return
	}
	ctx.Depth++
	for i, p := range e.Parents {
		if i == 0 {
			ctx.WriteLeadDoc(e.ExtendsDoc)
		}
		ctx.WriteString("*")
		ctx.WriteString(p.String())
		ctx.WriteSemi()
	}
	for _, m := range sortedInterfaceMembers(e.Members) {
		m.WriteCode(ctx)
		ctx.WriteSemi()
	}
	for _, m := range sortedInterfaceMethods(e.Methods) {
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
		for _, m := range sortedInterfaceContextFuncs(e.ContextFuncs) {
			m.WriteCode(ctx)
			ctx.WriteSemi()
		}
		ctx.Depth--
		ctx.WriteString("}")
		ctx.WriteSemi()
	}
	ctx.Depth--
}

// writeInterfaceBodyLines is the multi-line (formatting with a prefix) layout of
// writeInterfaceBody: one member per indented line and the closing brace back at
// the interface's own indentation, like a class body (see writeBraceItems).
func writeInterfaceBodyLines(ctx *CodeWriteContext, e *InterfaceExpr) {
	var items []func()
	for i, p := range e.Parents {
		i, p := i, p
		items = append(items, func() {
			if i == 0 {
				ctx.WriteLeadDoc(e.ExtendsDoc)
			}
			ctx.WriteString("*")
			ctx.WriteString(p.String())
		})
	}
	for _, m := range sortedInterfaceMembers(e.Members) {
		m := m
		items = append(items, func() { m.WriteCode(ctx) })
	}
	for _, m := range sortedInterfaceMethods(e.Methods) {
		m := m
		items = append(items, func() { m.WriteCode(ctx) })
	}
	if e.Rest != nil {
		items = append(items, func() {
			ctx.WriteLeadDoc(e.RestDoc)
			ctx.WriteString("**")
			e.Rest.WriteCode(ctx)
		})
	}
	if len(e.ContextFuncs) > 0 {
		funcs := sortedInterfaceContextFuncs(e.ContextFuncs)
		items = append(items, func() {
			ctx.WriteString("funcs {")
			writeBraceItems(ctx, len(funcs), func(i int) { funcs[i].WriteCode(ctx) })
			ctx.WriteString("}")
		})
	}
	writeBraceItems(ctx, len(items), func(i int) { items[i]() })
}

// InterfaceStmt is the statement form `interface Name { … }`, which binds a
// const to the interface value.
type InterfaceStmt struct {
	InterfaceExpr
}

func (s *InterfaceStmt) StmtNode() {}
