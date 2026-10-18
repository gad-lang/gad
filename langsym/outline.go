package langsym

import (
	"strings"

	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
)

// OutlineSym is one node of a file's structure outline: a named declaration with
// its kind, 0-based byte Offset (for navigation), 1-based Line/Column, and nested
// members (a class's fields/props/methods, an enum's values, …).
type OutlineSym struct {
	Name     string       `json:"name"`
	Kind     string       `json:"kind"` // const|var|func|class|mixin|type|interface|enum|method|property|field|new|met|value|overload
	Detail   string       `json:"detail,omitempty"`
	Offset   int          `json:"offset"`
	Line     int          `json:"line"`
	Column   int          `json:"column"`
	Children []OutlineSym `json:"children,omitempty"`
}

// Outline returns the top-level structure of a parsed file as a tree of symbols,
// for an editor's structure/outline view. It recognises const/var declarations,
// functions, classes, mixins, interfaces, enums and `met` declarations, nesting a
// type's own members (fields, properties, methods, constructors, enum values)
// underneath it.
func Outline(f *parser.File, sf *source.File) []OutlineSym {
	var out []OutlineSym
	for _, stmt := range f.Stmts {
		if s, ok := outlineStmt(sf, stmt); ok {
			out = append(out, s)
		}
	}
	return out
}

// outlineStmt maps a top-level statement to a symbol (ok=false to skip it).
func outlineStmt(sf *source.File, stmt node.Stmt) (OutlineSym, bool) {
	switch s := stmt.(type) {
	case *node.DeclStmt:
		if gd, _ := s.Decl.(*node.GenDecl); gd != nil {
			return outlineGenDecl(sf, gd)
		}
	case *node.FuncStmt:
		if s.Func != nil {
			return outlineFunc(sf, s.Func, "func"), true
		}
	case *node.TypeDeclStmt:
		return outlineClass(sf, &s.TypeLitExpr), true
	case *node.InterfaceStmt:
		return outlineInterface(sf, &s.InterfaceExpr), true
	case *node.EnumStmt:
		return outlineEnum(sf, &s.EnumExpr), true
	case *node.TypedArrayTypeStmt:
		// `type NAME []…` — a typed array type; its optional member body lists
		// like a class's.
		sym := leaf(sf, s.NameExpr.Name, "type", s.Pos())
		if s.Body != nil {
			sym.Children = outlineClass(sf, s.Body).Children
		}
		return sym, true
	case *node.ExprStmt:
		return outlineExpr(sf, s.Expr)
	case *node.FuncWithMethodsStmt:
		return outlineFuncWithMethods(sf, &s.FuncWithMethodsExpr), true
	case *node.ExportStmt:
		return outlineExport(sf, s)
	}
	return OutlineSym{}, false
}

// outlineExport maps an `export` to what it exports: a declaration
// (`export class default { … }`, `export type T []Item`, `export enum E …`,
// `export func f() …` — in its prelude), a type literal or a function as its
// value, or else the name exported, a var (a const for `export const`).
func outlineExport(sf *source.File, s *node.ExportStmt) (OutlineSym, bool) {
	if s.Prelude != nil {
		if sym, ok := outlineStmt(sf, s.Prelude); ok {
			return sym, true
		}
	}
	if s.ValueExpr != nil {
		if sym, ok := outlineExpr(sf, s.ValueExpr); ok {
			return sym, true
		}
	}
	id, _ := s.KeyExpr.(*node.IdentExpr)
	if id == nil || id.Empty {
		return OutlineSym{}, false
	}
	kind := "var"
	if s.Const {
		kind = "const"
	}
	return leaf(sf, id.Name, kind, id.Pos()), true
}

// outlineExpr maps an expression statement (a `met`, or an anonymous type literal
// used as a statement) to a symbol.
func outlineExpr(sf *source.File, e node.Expr) (OutlineSym, bool) {
	switch x := e.(type) {
	case *node.MethodExpr:
		if fn, _ := x.Expr.(*node.FuncExpr); fn != nil {
			return outlineFunc(sf, fn, "met"), true
		}
	case *node.FuncExpr:
		if named := funcName(x); named != "" {
			return outlineFunc(sf, x, "func"), true
		}
	case *node.FuncWithMethodsExpr:
		if x.NameExpr != nil {
			return outlineFuncWithMethods(sf, x), true
		}
	case *node.TypeLitExpr:
		if x.NameExpr != nil {
			return outlineClass(sf, x), true
		}
	case *node.InterfaceExpr:
		if x.NameExpr != nil {
			return outlineInterface(sf, x), true
		}
	case *node.EnumExpr:
		if x.NameExpr != nil {
			return outlineEnum(sf, x), true
		}
	}
	return OutlineSym{}, false
}

// outlineGenDecl maps a `const`/`var` group to a symbol. A single-spec value that
// is a type literal (`const C = class { … }`) is surfaced as that type; otherwise
// each declared identifier is a const/var leaf, and a multi-name group nests them
// under a single node.
func outlineGenDecl(sf *source.File, gd *node.GenDecl) (OutlineSym, bool) {
	kind := "var"
	if gd.Tok.String() == "const" {
		kind = "const"
	}
	var leaves []OutlineSym
	for _, spec := range gd.Specs {
		vs, _ := spec.(*node.ValueSpec)
		if vs == nil {
			continue
		}
		// `const Name = class/mixin/interface/enum { … }` → surface the type.
		if len(vs.Idents) == 1 && len(vs.Values) == 1 {
			if sym, ok := outlineNamedTypeValue(sf, vs.Idents[0], vs.Values[0]); ok {
				leaves = append(leaves, sym)
				continue
			}
		}
		for _, id := range vs.Idents {
			if id != nil && !id.Empty {
				leaves = append(leaves, leaf(sf, id.Name, kind, id.Pos()))
			}
		}
	}
	switch len(leaves) {
	case 0:
		return OutlineSym{}, false
	case 1:
		return leaves[0], true
	default:
		grp := leaf(sf, kind, kind, gd.Pos())
		grp.Children = leaves
		return grp, true
	}
}

// outlineNamedTypeValue surfaces `Name = <type literal>` as the type symbol named
// by Name (so an anonymous `class {…}` bound to a const shows its members).
func outlineNamedTypeValue(sf *source.File, id *node.IdentExpr, val node.Expr) (OutlineSym, bool) {
	if id == nil || id.Empty {
		return OutlineSym{}, false
	}
	switch v := val.(type) {
	case *node.TypeLitExpr:
		s := outlineClass(sf, v)
		s.Name = id.Name
		s.Offset, s.Line, s.Column = at(sf, id.Pos())
		return s, true
	case *node.InterfaceExpr:
		s := outlineInterface(sf, v)
		s.Name = id.Name
		s.Offset, s.Line, s.Column = at(sf, id.Pos())
		return s, true
	case *node.EnumExpr:
		s := outlineEnum(sf, v)
		s.Name = id.Name
		s.Offset, s.Line, s.Column = at(sf, id.Pos())
		return s, true
	case *node.FuncExpr:
		s := outlineFunc(sf, v, "func")
		s.Name = id.Name
		s.Offset, s.Line, s.Column = at(sf, id.Pos())
		return s, true
	}
	return OutlineSym{}, false
}

func outlineClass(sf *source.File, c *node.TypeLitExpr) OutlineSym {
	kind := "class"
	switch {
	case c.Mixin:
		kind = "mixin"
	case c.Static:
		kind = "type" // a marker `type Name { … }`
	}
	sym := leaf(sf, className(c), kind, c.Pos())
	for _, f := range c.Fields {
		if f.Name != nil && f.Name.Ident != nil {
			sym.Children = append(sym.Children, typedMember(sf, f.Name, "field", f.Pos()))
		}
	}
	for _, p := range c.Props {
		if id, _ := p.NameExpr.(*node.IdentExpr); id != nil {
			prop := leaf(sf, id.Name, "property", p.Pos())
			prop.Detail = propType(p.Methods)
			sym.Children = append(sym.Children, prop)
		}
	}
	if len(c.New) > 0 {
		sym.Children = append(sym.Children, overloaded(sf, "new", "new", c.New[0].Pos(), c.New))
	}
	// A marker type's `call(…)` factory overloads.
	if len(c.Call) > 0 {
		sym.Children = append(sym.Children, overloaded(sf, "call", "new", c.Call[0].Pos(), c.Call))
	}
	for _, m := range c.Methods {
		if id, _ := m.NameExpr.(*node.IdentExpr); id != nil {
			sym.Children = append(sym.Children, overloaded(sf, id.Name, "method", m.Pos(), m.Methods))
		}
	}
	return sym
}

func outlineInterface(sf *source.File, i *node.InterfaceExpr) OutlineSym {
	sym := leaf(sf, interfaceName(i), "interface", i.Pos())
	for _, m := range i.Members {
		if m.Name != nil && m.Name.Ident != nil {
			sym.Children = append(sym.Children, typedMember(sf, m.Name, "field", m.Pos()))
		}
	}
	for _, m := range i.Methods {
		if m.NameExpr == nil {
			continue
		}
		meth := leaf(sf, m.NameExpr.Name, "method", m.Pos())
		switch len(m.Headers) {
		case 0:
		case 1:
			meth.Detail = signature(&m.Headers[0].FuncHeader.Params, m.Headers[0].FuncHeader.Return)
		default:
			for _, h := range m.Headers {
				meth.Children = append(meth.Children,
					leaf(sf, signature(&h.FuncHeader.Params, h.FuncHeader.Return), "overload", h.Pos()))
			}
		}
		sym.Children = append(sym.Children, meth)
	}
	return sym
}

// typedMember is a field (of a class, of an interface) with its type as its
// detail — `str`, `?Link`, `int|str` —; a field typed by a class or an
// interface written in place is `class` or `interface`, its members under it.
func typedMember(sf *source.File, ti *node.TypedIdentExpr, kind string, p source.Pos) OutlineSym {
	sym := leaf(sf, ti.Ident.Name, kind, p)
	if len(ti.Type) == 1 {
		switch t := ti.Type[0].Expr.(type) {
		case *node.TypeLitExpr:
			inner := outlineClass(sf, t)
			sym.Detail, sym.Children = nullable(ti, inner.Kind), inner.Children
			return sym
		case *node.InterfaceExpr:
			inner := outlineInterface(sf, t)
			sym.Detail, sym.Children = nullable(ti, "interface"), inner.Children
			return sym
		}
	}
	names := make([]string, 0, len(ti.Type))
	for _, t := range ti.Type {
		names = append(names, typeText(t))
	}
	if len(names) > 0 || ti.Nullable {
		sym.Detail = nullable(ti, strings.Join(names, "|"))
	}
	return sym
}

// nullable is typ marked `?` when the field accepts nil.
func nullable(ti *node.TypedIdentExpr, typ string) string {
	if ti.Nullable {
		return "?" + typ
	}
	return typ
}

// typeText is a type as written; a class or an interface written in place is
// its kind.
func typeText(t *node.TypeExpr) string {
	switch t.Expr.(type) {
	case *node.TypeLitExpr:
		return "class"
	case *node.InterfaceExpr:
		return "interface"
	}
	return t.Expr.String()
}

// propType is the type a property gives: the return of its getter (the
// accessor without parameters), "" when it says none.
func propType(methods []*node.FuncMethod) string {
	for _, m := range methods {
		if len(m.Return) > 0 && m.Params.Args.NumFields() == 0 {
			return strings.TrimSpace(strings.Trim(strings.TrimSpace(node.FormatFuncReturn(m.Return)), "<>"))
		}
	}
	return ""
}

// overloaded is a function of several signatures — a method, `new`, `call`,
// a func with methods —: one signature its detail; several, each a child
// (an "overload") where it is written.
func overloaded(sf *source.File, name, kind string, p source.Pos, methods []*node.FuncMethod) OutlineSym {
	sym := leaf(sf, name, kind, p)
	switch len(methods) {
	case 0:
	case 1:
		sym.Detail = signature(&methods[0].Params, methods[0].Return)
	default:
		for _, m := range methods {
			sym.Children = append(sym.Children, leaf(sf, signature(&m.Params, m.Return), "overload", m.Pos()))
		}
	}
	return sym
}

// signature is the header of a function as written: its parameters and its
// return types (`(a int; b=1) <str>`).
func signature(params *node.FuncParams, ret []*node.TypedIdentExpr) string {
	return params.String() + node.FormatFuncReturn(ret)
}

func outlineFuncWithMethods(sf *source.File, f *node.FuncWithMethodsExpr) OutlineSym {
	name := "func"
	if f.NameExpr != nil {
		if id, _ := f.NameExpr.(*node.IdentExpr); id != nil {
			name = id.Name
		} else {
			name = node.Code(f.NameExpr)
		}
	}
	return overloaded(sf, name, "func", f.Pos(), f.Methods)
}

func outlineEnum(sf *source.File, e *node.EnumExpr) OutlineSym {
	sym := leaf(sf, enumName(e), "enum", e.Pos())
	for _, f := range e.Fields {
		if f.Name != nil {
			sym.Children = append(sym.Children, leaf(sf, f.Name.Name, "value", f.Name.Pos()))
		}
	}
	return sym
}

func outlineFunc(sf *source.File, fn *node.FuncExpr, kind string) OutlineSym {
	name := funcName(fn)
	if name == "" {
		name = "func"
	}
	sym := leaf(sf, name, kind, fn.Pos())
	if fn.Type != nil {
		sym.Detail = signature(&fn.Type.Params, fn.Type.Return)
	}
	return sym
}

// --- helpers ---

func leaf(sf *source.File, name, kind string, p source.Pos) OutlineSym {
	off, line, col := at(sf, p)
	return OutlineSym{Name: name, Kind: kind, Offset: off, Line: line, Column: col}
}

// at returns the 0-based offset and 1-based line/column of a source position.
func at(sf *source.File, p source.Pos) (offset, line, column int) {
	fp := sf.SafePosition(p)
	return fp.Offset, fp.Line, fp.Column
}

func className(c *node.TypeLitExpr) string {
	if id, _ := c.NameExpr.(*node.IdentExpr); id != nil {
		return id.Name
	}
	switch {
	case c.Mixin:
		return "mixin"
	case c.Static:
		return "type"
	}
	return "class"
}

func interfaceName(i *node.InterfaceExpr) string {
	if id, _ := i.NameExpr.(*node.IdentExpr); id != nil {
		return id.Name
	}
	return "interface"
}

func enumName(e *node.EnumExpr) string {
	if id, _ := e.NameExpr.(*node.IdentExpr); id != nil {
		return id.Name
	}
	return "enum"
}

// funcName returns a function's name (a plain name, or the `recv.method` /
// `recv["m"]` selector of a `met`), or "" when anonymous.
func funcName(fn *node.FuncExpr) string {
	if fn.Type == nil || fn.Type.NameExpr == nil {
		return ""
	}
	switch n := fn.Type.NameExpr.(type) {
	case *node.IdentExpr:
		return n.Name
	default:
		return node.Code(n)
	}
}
