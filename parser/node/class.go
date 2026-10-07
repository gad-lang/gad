package node

import (
	"sort"
	"strings"

	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/source"
)

// ClassParentExpr is one parent class, written as a `*Parent` spread body item:
// a parent type expression (an IdentExpr or SelectorExpr) with an optional alias
// written after a colon (`*Base: B`).
type ClassParentExpr struct {
	Type  Expr
	Alias *IdentExpr // optional; nil when written without `: alias`
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
		return e.Type.String() + ": " + e.Alias.String()
	}
	return e.Type.String()
}

func (e *ClassParentExpr) WriteCode(ctx *CodeWriteContext) {
	e.Type.WriteCode(ctx)
	if e.Alias != nil {
		ctx.WriteString(": ")
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
	// Meta is the field's `[k=v, …]` metadata (or nil); read as `Class.field.@meta`.
	Meta *KeyValueArrayLit
	// Group marks a field written as a group — `{ … }`, `? { … }` —: named
	// `$N` (N its count in the body), typed by the anonymous class of the
	// block; it is written back as the block.
	Group bool
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
	writeMeta(ctx, e.Meta)
	if e.Group && len(e.Name.Type) == 1 {
		if cls, ok := e.Name.Type[0].Expr.(*TypeLitExpr); ok {
			if e.Name.Nullable {
				ctx.WriteString("? ")
			}
			ctx.WriteString("{")
			writeTypeLitBody(ctx, cls)
			return
		}
	}
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
	// Meta is the member's `[k=v, …]` metadata (or nil).
	Meta *KeyValueArrayLit
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
	writeMeta(ctx, e.Meta)
	if e.NameExpr != nil {
		e.NameExpr.WriteCode(ctx)
	}
	if e.Block {
		ctx.WriteString(" {")
		writeClassMethods(ctx, e.Methods)
		ctx.WriteString("}")
		return
	}
	if len(e.Methods) == 1 {
		e.Methods[0].WriteCode(ctx)
	}
}

// TypeLitExpr is a type literal — a `class`, `mixin` or marker `type` body,
// distinguished by the Mixin / Static flags:
//
//	class [Name] { *P, …, fields, props {…}, new …, methods {…} }
//	mixin [Name] { … this {…} … }        // Mixin
//	type  [Name] { … call(…) … }         // Static (marker: no instances)
//
// Parent classes are `*Parent` spread body items (optionally aliased as
// `*Parent: Alias`), alongside fields and the `props`/`new`/`methods` groups.
// A class lowers (in the compiler) to a
//
//	Class(name; define=(Type, define) => define(; extends=…, fields=…,
//	    props=…, methods=…, new=…))
//
// call. The `define` callback binds `Type` to the in-construction class so each
// method, property accessor and constructor can take a typed `this Type` first
// parameter (injected by the compiler). NameExpr is nil for an anonymous,
// expression-form class.
type TypeLitExpr struct {
	ClassToken TokenLit
	// Mixin marks a `mixin … { … }` literal: it parses like a class (parents,
	// fields, props, methods) plus an optional `this` interface block, has no `new`
	// clause, and lowers to `gad.Mixin(...)` instead of `Class(...)`.
	Mixin bool
	// Static marks a `type … { … }` marker-type literal: it parses like a class
	// (static fields, props, methods) but has no instances — its members are on the
	// type itself (`this` is the type), it may declare a `call(…)` factory instead
	// of `new`, and it lowers to a StaticType.
	Static   bool
	NameExpr Expr
	// TypeParams are a generic class's type parameters — `class Box[T, K
	// number] { … }` —, each a name and its optional constraint. A generic
	// class is a template: each `Box[int]` the module uses is a class of its
	// own, the parameters replaced by the arguments (see the compiler's
	// generics).
	TypeParams []*TypedIdentExpr
	// TypeArgs are, in a generic's instance (or its template), each type
	// parameter and the types it is there — `Box[int]`: T int —, what
	// `Class.@tparams` reports. The compiler sets it; it is not written back.
	TypeArgs []*TypedIdentExpr
	// ImpliedName names an anonymous class a field is typed by (`a class { … }`)
	// by its path from the outermost class holding it, that class's name first
	// — `PageOptions.a.b`, `#1.a.b` below an anonymous one —, so an instance
	// says where in the record it is. The compiler sets it. It is not written back (the source has no
	// name there); NameExpr, when set, wins.
	ImpliedName string
	Parents     []*ClassParentExpr
	ExtendsDoc  *ast.CommentGroup
	// Use are the mixins a class pulls in (`use A, B`); class-only.
	Use    []Expr
	UseDoc *ast.CommentGroup
	Fields []*ClassFieldExpr
	// Spreads are the `**Expr` body items of a class: members given at run
	// time, EXPR evaluating to `{fields: …, methods: …, props: …}` (a dict or a
	// key-value array), added after the declared ones.
	Spreads []Expr
	// This is a mixin's optional `this { … }` interface block: it declares the
	// interface the `this` parameter of the mixin's props/methods must satisfy.
	// Parsed as an anonymous interface body; mixin-only.
	This     *InterfaceExpr
	ThisDoc  *ast.CommentGroup
	Props    []*ClassMemberExpr
	PropsDoc *ast.CommentGroup
	New      []*FuncMethod
	NewDoc   *ast.CommentGroup
	// Call are a marker type's `call(…)` factory overloads (Static only); it is
	// the analogue of `New` for a `type … { … }`, but the result is arbitrary (a
	// factory), not an instance. Empty for classes and mixins.
	Call       []*FuncMethod
	CallDoc    *ast.CommentGroup
	Methods    []*ClassMemberExpr
	MethodsDoc *ast.CommentGroup
	LBrace     source.Pos
	RBrace     source.Pos
	Doc        *ast.CommentGroup // doc comment preceding the class; or nil
	// Meta is the optional `[k=v, …]` metadata block preceding the class.
	Meta *KeyValueArrayLit
	// Group marks the body of a group (`{ … }` in a body): its fields are in
	// the order written — its columns —, never sorted by the formatter.
	Group bool
}

// keyword returns "mixin", "type" (marker) or "class" for formatting/diagnostics.
func (e *TypeLitExpr) keyword() string {
	switch {
	case e.Mixin:
		return "mixin"
	case e.Static:
		return "type"
	default:
		return "class"
	}
}

func (e *TypeLitExpr) ExprNode() {}

func (e *TypeLitExpr) Pos() source.Pos {
	if e.ClassToken.Pos != source.NoPos {
		return e.ClassToken.Pos
	}
	return e.LBrace
}

func (e *TypeLitExpr) End() source.Pos { return e.RBrace + 1 }

func (e *TypeLitExpr) String() string { return Code(e) }

func (e *TypeLitExpr) WriteCode(ctx *CodeWriteContext) {
	ctx.WriteLeadDoc(e.Doc)
	writeMeta(ctx, e.Meta)
	ctx.WriteString(e.keyword())
	if e.NameExpr != nil {
		ctx.WriteString(" ")
		e.NameExpr.WriteCode(ctx)
		ctx.WriteString(FormatTypeParams(e.TypeParams))
	}
	ctx.WriteString(" {")
	writeTypeLitBody(ctx, e)
}

// writeTypeLitBody emits a class-like body's items and the closing `}` (the
// caller wrote the opening `{`). Shared by class/mixin/type literals and the
// member block of a typed array type (`type T []int { … }`).
func writeTypeLitBody(ctx *CodeWriteContext, e *TypeLitExpr) {
	// Body items in canonical order: the parent spreads (`*Parent`), the `use`
	// clause (class), fields, the `this` interface block (mixin), then the `props`,
	// `new` and `methods` groups.
	var items []func()
	for i, parent := range e.Parents {
		i, parent := i, parent
		items = append(items, func() {
			if i == 0 {
				ctx.WriteLeadDoc(e.ExtendsDoc)
			}
			ctx.WriteString("*")
			parent.WriteCode(ctx)
		})
	}
	if len(e.Use) > 0 {
		items = append(items, func() {
			ctx.WriteLeadDoc(e.UseDoc)
			ctx.WriteString("use ")
			// The mixin list wraps greedily: when the next name would overflow the
			// line it breaks after a comma onto a new line indented one level under
			// `use`. Short lists stay inline (`use A, B`).
			ctx.Depth++
			ctx.WriteGreedy(len(e.Use), ", ", ",", func(i int) { e.Use[i].WriteCode(ctx) })
			ctx.Depth--
		})
	}
	// each field's comments — above it, at the end of its line — claimed in
	// the order written, so they follow it wherever it is put
	fieldComments := map[*ClassFieldExpr]*itemComments{}
	for i, ic := range ctx.claimBodyComments(len(e.Fields), e.LBrace, e.RBrace, func(i int) Node { return e.Fields[i] }) {
		fieldComments[e.Fields[i]] = ic
	}
	for _, f := range formatClassFields(e) {
		f := f
		items = append(items, func() {
			fieldComments[f].writeLead(ctx)
			f.WriteCode(ctx)
			fieldComments[f].writeTrail(ctx)
		})
	}
	for _, sp := range e.Spreads {
		sp := sp
		items = append(items, func() {
			ctx.WriteString("**")
			sp.WriteCode(ctx)
		})
	}
	if e.This != nil {
		items = append(items, func() {
			ctx.WriteLeadDoc(e.ThisDoc)
			ctx.WriteString("this {")
			writeInterfaceBody(ctx, e.This)
			ctx.WriteString("}")
		})
	}
	if len(e.Props) > 0 {
		props := sortedClassMembers(e.Props)
		items = append(items, func() {
			ctx.WriteLeadDoc(e.PropsDoc)
			ctx.WriteString("props {")
			writeClassMembers(ctx, props)
			ctx.WriteString("}")
		})
	}
	if len(e.New) > 0 {
		items = append(items, func() {
			ctx.WriteLeadDoc(e.NewDoc)
			ctx.WriteString("new {")
			writeClassMethods(ctx, e.New)
			ctx.WriteString("}")
		})
	}
	if len(e.Methods) > 0 {
		methods := sortedClassMembers(e.Methods)
		items = append(items, func() {
			ctx.WriteLeadDoc(e.MethodsDoc)
			ctx.WriteString("methods {")
			writeClassMembers(ctx, methods)
			ctx.WriteString("}")
		})
	}

	writeBraceItems(ctx, len(items), func(i int) { items[i]() })
	ctx.WriteString("}")
}

// writeBraceItems emits count items of a brace block, one per indented line when
// formatting with a prefix and `; `-separated inline otherwise, leaving the
// cursor positioned for the closing brace. Mirrors FuncWithMethodsExpr.
func writeBraceItems(ctx *CodeWriteContext, count int, do func(i int)) {
	ctx.WriteItemsSep(ctx.HasPrefix(), count, "; ", "", do, func(newLine bool) {
		if newLine {
			ctx.WriteSecondLine()
		}
	})
	if count > 0 && ctx.HasPrefix() {
		ctx.WritePrefix()
	}
}

// writeClassMembers emits the entries of a `props {}` / `methods {}` block.
func writeClassMembers(ctx *CodeWriteContext, members []*ClassMemberExpr) {
	writeBraceItems(ctx, len(members), func(i int) { members[i].WriteCode(ctx) })
}

// writeClassMethods emits the overloads of a brace-block member (`name { (…) …
// }`) or the `new {}` block.
func writeClassMethods(ctx *CodeWriteContext, methods []*FuncMethod) {
	writeBraceItems(ctx, len(methods), func(i int) { methods[i].WriteCode(ctx) })
}

// classFieldName returns a field's name for ordering (empty when unnamed).
func classFieldName(f *ClassFieldExpr) string {
	if f.Name != nil && f.Name.Ident != nil {
		return f.Name.Ident.Name
	}
	return ""
}

// classFieldGroup ranks a field for the canonical body order: untyped no-default
// (0), typed no-default (1), untyped with-default (2), typed with-default (3).
func classFieldGroup(f *ClassFieldExpr) int {
	typed := f.Name != nil && len(f.Name.Type) > 0
	hasDefault := f.Value != nil
	switch {
	case !typed && !hasDefault:
		return 0
	case typed && !hasDefault:
		return 1
	case !typed && hasDefault:
		return 2
	default:
		return 3
	}
}

// formatClassFields are the fields of e in the order they are written: the
// canonical one (sortedClassFields), unless their order means something —
// the class is `[ordered]`, or it has groups, whose names (`$N`) are their
// order — and then as declared.
func formatClassFields(e *TypeLitExpr) []*ClassFieldExpr {
	if e.Group || hasMetaKey(e.Meta, "ordered") {
		return e.Fields
	}
	for _, f := range e.Fields {
		if f.Group {
			return e.Fields
		}
	}
	return sortedClassFields(e.Fields)
}

// hasMetaKey says the metadata m has the key name.
func hasMetaKey(m *KeyValueArrayLit, name string) bool {
	if m == nil {
		return false
	}
	for _, el := range m.Elements {
		if s := el.String(); s == name || strings.HasPrefix(s, name+"=") {
			return true
		}
	}
	return false
}

// sortedClassFields returns the fields in the canonical formatting order: the
// four field groups (untyped no-default, typed no-default, untyped with-default,
// typed with-default), each sorted by name. The input slice is not mutated.
func sortedClassFields(fields []*ClassFieldExpr) []*ClassFieldExpr {
	out := make([]*ClassFieldExpr, len(fields))
	copy(out, fields)
	sort.SliceStable(out, func(i, j int) bool {
		if gi, gj := classFieldGroup(out[i]), classFieldGroup(out[j]); gi != gj {
			return gi < gj
		}
		return classFieldName(out[i]) < classFieldName(out[j])
	})
	return out
}

// sortedClassMembers returns props/methods sorted by name (declaration order is
// preserved among same-named entries). The input slice is not mutated.
func sortedClassMembers(members []*ClassMemberExpr) []*ClassMemberExpr {
	out := make([]*ClassMemberExpr, len(members))
	copy(out, members)
	sort.SliceStable(out, func(i, j int) bool {
		return classMemberName(out[i]) < classMemberName(out[j])
	})
	return out
}

// classMemberName returns a prop/method member's name for ordering.
func classMemberName(m *ClassMemberExpr) string {
	if id, _ := m.NameExpr.(*IdentExpr); id != nil {
		return id.Name
	}
	return ""
}

// TypeDeclStmt is the named statement form of a type literal — `class Name { … }`,
// `mixin Name { … }` or `type Name { … }` (the marker form). It wraps the
// TypeLitExpr (whose Mixin/Static flags say which) and compiles to
// `const Name = <the type literal>`.
type TypeDeclStmt struct {
	TypeLitExpr
}

func (*TypeDeclStmt) StmtNode() {}
