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

// IncludeResolver reads the source of an `include`d file for the language
// service, so an included file's top-level declarations show up in the
// includer's completion and go-to-definition (since `include` compiles inline).
// fromFile is the name of the file that contains the include (for relative path
// resolution); path is the include argument. It returns the included source, its
// canonical name, and ok=false when it cannot be resolved (the include is then
// ignored). It is nil by default (no cross-file resolution); the editor host
// (e.g. `gad complete`/`gad def`) sets it to read from disk.
var IncludeResolver func(fromFile, path string) (src []byte, name string, ok bool)

// Decl is a declared name, where it was declared, the declaring node, and its
// lead doc comment (when the declaration carries one).
type Decl struct {
	Name string
	Pos  source.Pos
	Node node.Node
	Doc  *ast.CommentGroup
	// Kind is the completion kind of a named type declaration — "class",
	// "mixin", "type" (a marker or typed array type), "interface" or "enum" —
	// and "" for a plain variable/constant.
	Kind string
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
	s.addDoc(name, pos, n, nil)
}

func (s *scope) addDoc(name string, pos source.Pos, n node.Node, doc *ast.CommentGroup) {
	s.addKind(name, pos, n, doc, "")
}

func (s *scope) addKind(name string, pos source.Pos, n node.Node, doc *ast.CommentGroup, kind string) {
	if name != "" && name != "_" {
		s.decls = append(s.decls, Decl{Name: name, Pos: pos, Node: n, Doc: doc, Kind: kind})
	}
}

// resolver builds and queries the scope tree for a file.
type resolver struct {
	file     *source.File
	root     *scope
	comments []*ast.CommentGroup
	// visited guards against include cycles while collecting cross-file decls.
	visited map[string]bool
}

func newResolver(f *parser.File, sf *source.File) *resolver {
	r := &resolver{file: sf, comments: f.Comments, visited: map[string]bool{}}
	r.root = &scope{start: f.Pos(), end: f.End() + 1}
	for _, s := range f.Stmts {
		r.walk(s, r.root)
	}
	return r
}

// typeDeclName returns the name, doc and completion kind of a named type
// declaration statement: `class`/`mixin`/`type NAME { … }`, `interface NAME …`,
// `enum NAME { … }` and a typed array type `type NAME []…`. id is nil for any
// other node (or an anonymous declaration).
func typeDeclName(n ast.Node) (id *node.IdentExpr, doc *ast.CommentGroup, kind string) {
	switch x := n.(type) {
	case *node.TypeDeclStmt:
		id, _ = x.NameExpr.(*node.IdentExpr)
		kind = "class"
		switch {
		case x.Mixin:
			kind = "mixin"
		case x.Static:
			kind = "type"
		}
		return id, x.Doc, kind
	case *node.InterfaceStmt:
		id, _ = x.NameExpr.(*node.IdentExpr)
		return id, x.Doc, "interface"
	case *node.EnumStmt:
		id, _ = x.NameExpr.(*node.IdentExpr)
		return id, x.Doc, "enum"
	case *node.TypedArrayTypeStmt:
		return x.NameExpr, x.Doc, "type"
	}
	return nil, nil, ""
}

// leadDoc returns the free-floating comment group immediately preceding the
// declaration at pos (its last line is the line just above pos), used for `:=`
// declarations whose AST node carries no Doc field of its own.
func (r *resolver) leadDoc(pos source.Pos) *ast.CommentGroup {
	line := r.file.SafePosition(pos).Line
	if line <= 1 {
		return nil
	}
	for _, g := range r.comments {
		if r.file.SafePosition(g.End()).Line == line-1 {
			return g
		}
	}
	return nil
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
	// A named type declaration binds its name (a const) in the enclosing scope;
	// its body is then walked as usual (method bodies open their own scopes).
	if id, doc, kind := typeDeclName(n); id != nil {
		sc.addKind(id.Name, id.Pos(), id, doc, kind)
	}
	switch x := n.(type) {
	case *node.AssignStmt:
		if x.Token == token.Define {
			// A doc on a `name := func … {}` lives on the function literal.
			var doc *ast.CommentGroup
			if len(x.RHS) == 1 {
				if fe, ok := x.RHS[0].(*node.FuncExpr); ok {
					doc = fe.Doc
				}
			}
			for _, lhs := range x.LHS {
				if id, ok := lhs.(*node.IdentExpr); ok && !id.Empty {
					sc.addDoc(id.Name, id.Pos(), id, doc)
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
				doc := sp.Doc
				if doc == nil {
					doc = x.Doc // a single-spec group's doc sits on the GenDecl
				}
				for _, id := range sp.Idents {
					sc.addDoc(id.Name, id.Pos(), id, doc)
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

	case *node.IncludeStmt:
		// `include ("a.gad", …)` compiles each file inline, so its top-level
		// declarations become visible here (from the include line onward). Pull
		// them in via the host-provided IncludeResolver.
		if IncludeResolver != nil {
			for _, pth := range x.Paths {
				r.addIncludeDecls(pth.Value(), x.Pos(), sc)
			}
		}
		return
	}

	// Any other node: descend into its children in the same scope, so nested
	// functions/closures/declarations inside arbitrary expressions are found.
	r.walkChildren(reflect.ValueOf(n), sc)
}

// addIncludeDecls resolves an included file and adds its top-level declarations
// to sc, re-homed at the include site `at` so they are in scope from the include
// line onward. Nested includes are followed (flattened here); cycles are guarded
// by r.visited.
func (r *resolver) addIncludeDecls(path string, at source.Pos, sc *scope) {
	if path == "" || r.visited[path] {
		return
	}
	src, name, ok := IncludeResolver(r.file.Name, path)
	if !ok {
		return
	}

	sf := source.NewFileSet().AddFileData(name, -1, src)
	po := &parser.ParserOptions{Mode: parser.ParseComments}
	file, err := parser.NewParserWithOptions(sf, po, nil).ParseFile()
	if err != nil {
		return
	}

	// Collect the included file's top-level decls with a sub-resolver that shares
	// the visited set (so a transitive cycle back to an active file is skipped).
	sub := &resolver{file: sf, comments: file.Comments, visited: r.visited}
	sub.root = &scope{start: file.Pos(), end: file.End() + 1}
	r.visited[path] = true
	for _, s := range file.Stmts {
		sub.walk(s, sub.root)
	}
	delete(r.visited, path)

	for i := range sub.root.decls {
		d := &sub.root.decls[i]
		doc := d.Doc
		if doc == nil {
			// A `name := …` decl carries no Doc node; its lead comment is associated
			// by line against the INCLUDED file's comments — resolve it now (the
			// includer's leadDoc could not, it has the wrong file/lines).
			doc = sub.leadDoc(d.Pos)
		}
		sc.addDoc(d.Name, at, d.Node, doc)
	}
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
