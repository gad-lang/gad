package langsym

import (
	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
)

// OutlineSym is one node of a file's structure outline: a named declaration with
// its kind, 0-based byte Offset (for navigation), 1-based Line/Column, and nested
// members (a class's fields/props/methods, an enum's values, …).
type OutlineSym struct {
	Name     string       `json:"name"`
	Kind     string       `json:"kind"` // const|var|func|class|mixin|interface|enum|method|property|field|new|met|value
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
	case *node.ExprStmt:
		return outlineExpr(sf, s.Expr)
	}
	return OutlineSym{}, false
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
	if c.Mixin {
		kind = "mixin"
	}
	sym := leaf(sf, className(c), kind, c.Pos())
	for _, f := range c.Fields {
		if f.Name != nil && f.Name.Ident != nil {
			sym.Children = append(sym.Children, leaf(sf, f.Name.Ident.Name, "field", f.Pos()))
		}
	}
	for _, p := range c.Props {
		if id, _ := p.NameExpr.(*node.IdentExpr); id != nil {
			sym.Children = append(sym.Children, leaf(sf, id.Name, "property", p.Pos()))
		}
	}
	for _, m := range c.New {
		sym.Children = append(sym.Children, leaf(sf, "new", "new", m.Pos()))
	}
	for _, m := range c.Methods {
		if id, _ := m.NameExpr.(*node.IdentExpr); id != nil {
			sym.Children = append(sym.Children, leaf(sf, id.Name, "method", m.Pos()))
		}
	}
	return sym
}

func outlineInterface(sf *source.File, i *node.InterfaceExpr) OutlineSym {
	sym := leaf(sf, interfaceName(i), "interface", i.Pos())
	for _, m := range i.Members {
		if m.Name != nil && m.Name.Ident != nil {
			sym.Children = append(sym.Children, leaf(sf, m.Name.Ident.Name, "field", m.Pos()))
		}
	}
	for _, m := range i.Methods {
		if m.NameExpr != nil {
			sym.Children = append(sym.Children, leaf(sf, m.NameExpr.Name, "method", m.Pos()))
		}
	}
	return sym
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
	return leaf(sf, name, kind, fn.Pos())
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
	if c.Mixin {
		return "mixin"
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
