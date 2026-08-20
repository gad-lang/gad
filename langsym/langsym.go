// Package langsym is a small language service over the Gad AST: scope-aware
// symbol resolution for go-to-definition and completion. It powers the `gad def`
// / `gad complete` commands the editor plugins call, so the logic lives once in
// gad rather than being duplicated per editor.
package langsym

import (
	"reflect"

	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/token"
)

// Decl is a declared name, where it was declared, and the declaring node (for
// documentation lookup).
type Decl struct {
	Name string
	Pos  source.Pos
	Node node.Node
}

// scope is one lexical scope: its declarations (in source order) and its span,
// with child scopes for nested functions/blocks.
type scope struct {
	parent   *scope
	children []*scope
	decls    []Decl
	start    source.Pos
	end      source.Pos
}

func (s *scope) add(name string, pos source.Pos, n node.Node) {
	if name != "" && name != "_" {
		s.decls = append(s.decls, Decl{Name: name, Pos: pos, Node: n})
	}
}

// resolver builds and queries the scope tree for a file.
type resolver struct {
	file *source.File
	root *scope
}

func newResolver(f *parser.File, sf *source.File) *resolver {
	r := &resolver{file: sf}
	r.root = &scope{start: f.Pos(), end: f.End() + 1}
	for _, s := range f.Stmts {
		r.walk(s, r.root)
	}
	return r
}

func (r *resolver) pos(offset int) source.Pos { return source.Pos(r.file.Base + offset) }

func (r *resolver) child(parent *scope, n ast.Node) *scope {
	s := &scope{parent: parent, start: n.Pos(), end: n.End() + 1}
	parent.children = append(parent.children, s)
	return s
}

// scopeAt returns the deepest scope whose span contains p.
func (r *resolver) scopeAt(p source.Pos) *scope {
	cur := r.root
	for {
		var next *scope
		for _, c := range cur.children {
			if c.start <= p && p < c.end {
				next = c
				break
			}
		}
		if next == nil {
			return cur
		}
		cur = next
	}
}

// walk records declarations and scopes for n under sc.
func (r *resolver) walk(n ast.Node, sc *scope) {
	switch x := n.(type) {
	case *node.AssignStmt:
		if x.Token == token.Define {
			for _, lhs := range x.LHS {
				if id, ok := lhs.(*node.IdentExpr); ok && !id.Empty {
					sc.add(id.Name, id.Pos(), id)
				}
			}
		}
		for _, rhs := range x.RHS {
			r.walk(rhs, sc)
		}
		return

	case *node.GenDecl:
		for _, spec := range x.Specs {
			switch sp := spec.(type) {
			case *node.ValueSpec:
				for _, id := range sp.Idents {
					sc.add(id.Name, id.Pos(), id)
				}
				if sp.Pattern != nil {
					r.addTypedIdents(sp.Pattern, sc)
				}
				for _, v := range sp.Values {
					if v != nil {
						r.walk(v, sc)
					}
				}
			case *node.ParamSpec:
				r.addTypedIdent(sp.Ident, sc)
			case *node.NamedParamSpec:
				r.addTypedIdent(sp.Ident, sc)
				if sp.Value != nil {
					r.walk(sp.Value, sc)
				}
			}
		}
		return

	case *node.FuncExpr:
		fsc := r.child(sc, x)
		r.addTypedIdents(x.Type, fsc)
		if x.Body != nil {
			for _, s := range x.Body.Stmts {
				r.walk(s, fsc)
			}
		}
		if x.BodyExpr != nil {
			r.walk(x.BodyExpr, fsc)
		}
		return

	case *node.ClosureExpr:
		csc := r.child(sc, x)
		r.addTypedIdents(&x.Params, csc)
		if x.Body != nil {
			r.walk(x.Body, csc)
		}
		return

	case *node.BlockStmt:
		bsc := r.child(sc, x)
		for _, s := range x.Stmts {
			r.walk(s, bsc)
		}
		return

	case *node.ForStmt:
		lsc := r.child(sc, x)
		if x.Init != nil {
			r.walk(x.Init, lsc)
		}
		if x.Cond != nil {
			r.walk(x.Cond, lsc)
		}
		if x.Post != nil {
			r.walk(x.Post, lsc)
		}
		if x.Body != nil {
			for _, s := range x.Body.Stmts {
				r.walk(s, lsc)
			}
		}
		return

	case *node.ForInStmt:
		lsc := r.child(sc, x)
		if x.Key != nil && !x.Key.Empty {
			lsc.add(x.Key.Name, x.Key.Pos(), x.Key)
		}
		if x.Value != nil && !x.Value.Empty {
			lsc.add(x.Value.Name, x.Value.Pos(), x.Value)
		}
		if x.Iterable != nil {
			r.walk(x.Iterable, sc) // the iterable is evaluated in the outer scope
		}
		if x.Body != nil {
			for _, s := range x.Body.Stmts {
				r.walk(s, lsc)
			}
		}
		return
	}

	// Any other node: descend into its children in the same scope, so nested
	// functions/closures/declarations inside arbitrary expressions are found.
	r.walkChildren(reflect.ValueOf(n), sc)
}

// addTypedIdent adds a param/return identifier (TypedIdentExpr.Ident).
func (r *resolver) addTypedIdent(ti *node.TypedIdentExpr, sc *scope) {
	if ti != nil && ti.Ident != nil && !ti.Ident.Empty {
		sc.add(ti.Ident.Name, ti.Ident.Pos(), ti.Ident)
	}
}

// addTypedIdents collects every TypedIdentExpr's identifier under n (a parameter
// list or destructuring pattern), without descending into type annotations.
func (r *resolver) addTypedIdents(n ast.Node, sc *scope) {
	if n == nil {
		return
	}
	node.Walk(n, func(x ast.Node) bool {
		if ti, ok := x.(*node.TypedIdentExpr); ok {
			r.addTypedIdent(ti, sc)
			return false // skip its Type annotation
		}
		if id, ok := x.(*node.IdentExpr); ok && !id.Empty {
			sc.add(id.Name, id.Pos(), id) // bare idents in a destructuring pattern
		}
		return true
	})
}

// walkChildren descends generically into a value's ast.Node children.
func (r *resolver) walkChildren(v reflect.Value, sc *scope) {
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		if !v.IsNil() {
			r.walkChildren(v.Elem(), sc)
		}
	case reflect.Struct:
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			if t.Field(i).PkgPath == "" {
				r.walkField(v.Field(i), sc)
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			r.walkField(v.Index(i), sc)
		}
	}
}

func (r *resolver) walkField(v reflect.Value, sc *scope) {
	if v.Kind() == reflect.Interface || v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return
		}
	}
	if v.CanInterface() {
		if n, ok := v.Interface().(node.Node); ok {
			r.walk(n, sc)
			return
		}
	}
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Array, reflect.Struct:
		r.walkChildren(v, sc)
	}
}
