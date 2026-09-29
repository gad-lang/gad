package gad

import (
	"reflect"
	"strings"

	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/token"
)

// Generic classes and interfaces.
//
// A class or an interface may declare type parameters, each a name and,
// optionally, the type it stands for — its alias:
//
//	class Pair[K str, V int] { key K; value V }
//	interface Getter[T] { get() <T> }
//
// The declaration is a template. Each `Pair[A, B]` of the block is a class of
// its own, named so (`Pair[str, float]`), declared beside the template with
// the parameters replaced by the arguments: K by A, V by B. An argument left
// out is its parameter's alias (`Pair[str]` is `Pair[str, int]`), and a
// parameter with none is `any`. The name alone, `Pair`, is the template with
// every parameter its alias: `Pair` is `Pair[str, int]`, written as declared.
//
// The instances are made when the block is compiled (monomorphization): they
// are consts of the block like the template (compiler_hoist.go), in any order,
// and `Pair[str, int]` twice is the same class. A parameter is replaced where
// it names a type — a field's, a param's, a return's, a parent's, an
// argument's of another generic (`next? Node[T]`) — and where a value names it
// (`T(v)`); not as the name of a field or a param, nor after a `.`.
//
// An instance is of the block the generic is declared in: a module that
// imports one gets the template (`Pair`), and makes no instance of it.

// maxGenericInstances caps the instances of a block: a generic whose
// instances need ever more of them (`class L[T] { next? L[L[T]] }`) stops
// there.
const maxGenericInstances = 1000

type genericDecl struct {
	name   string
	params []*node.TypedIdentExpr
	stmt   node.Stmt // the *node.TypeDeclStmt or *node.InterfaceStmt
	insts  []node.Stmt
}

type genericsExpander struct {
	c        *Compiler
	generics map[string]*genericDecl
	seen     map[string]bool
	queue    []genericWork
	count    int
	err      error
}

type genericWork struct {
	g    *genericDecl
	key  string
	args [][]*node.TypeExpr
}

// expandGenerics returns stmts with its generic declarations made templates:
// each replaced by the declaration with its parameters' aliases, followed by
// the instances the block uses, and each use of one (`Box[int]`) by the name of
// its instance. stmts is returned as is when it declares no generic.
func (c *Compiler) expandGenerics(stmts []node.Stmt) ([]node.Stmt, error) {
	var ge *genericsExpander
	for _, s := range stmts {
		d := s
		if e, ok := s.(*node.ExportStmt); ok && e.Prelude != nil {
			d = e.Prelude
		}
		var (
			name   node.Expr
			params []*node.TypedIdentExpr
		)
		switch t := d.(type) {
		case *node.TypeDeclStmt:
			name, params = t.NameExpr, t.TypeParams
		case *node.InterfaceStmt:
			name, params = t.NameExpr, t.TypeParams
		}
		id, _ := name.(*node.IdentExpr)
		if id == nil || len(params) == 0 {
			continue
		}
		if ge == nil {
			ge = &genericsExpander{c: c, generics: map[string]*genericDecl{}, seen: map[string]bool{}}
		}
		if _, dup := ge.generics[id.Name]; dup {
			return nil, c.Errorf(id, "%q redeclared in this block", id.Name)
		}
		for i, p := range params {
			for _, q := range params[:i] {
				if q.Ident.Name == p.Ident.Name {
					return nil, c.Errorf(p.Ident, "type parameter %q declared twice", p.Ident.Name)
				}
			}
		}
		ge.generics[id.Name] = &genericDecl{name: id.Name, params: params, stmt: d}
	}
	if ge == nil {
		return stmts, nil
	}

	out := make([]node.Stmt, 0, len(stmts))
	owner := make([]*genericDecl, 0, len(stmts)) // the template out[i] is, or nil
	for _, s := range stmts {
		d := s
		exp, _ := s.(*node.ExportStmt)
		if exp != nil && exp.Prelude != nil {
			d = exp.Prelude
		}
		g := ge.genericOf(d)
		if g != nil {
			// the template: its parameters their aliases
			bare := ge.instance(g, g.name, nil)
			if exp != nil {
				cp := *exp
				cp.Prelude = bare
				s = &cp
			} else {
				s = bare
			}
		} else {
			ge.rewrite(reflect.ValueOf(&s).Elem(), nil)
		}
		out = append(out, s)
		owner = append(owner, g)
	}
	for len(ge.queue) > 0 && ge.err == nil {
		w := ge.queue[0]
		ge.queue = ge.queue[1:]
		w.g.insts = append(w.g.insts, ge.instance(w.g, w.key, w.args))
	}
	if ge.err != nil {
		return nil, ge.err
	}

	// each template's instances right after it
	res := make([]node.Stmt, 0, len(out)+ge.count)
	for i, s := range out {
		res = append(res, s)
		if g := owner[i]; g != nil {
			res = append(res, g.insts...)
		}
	}
	return res, nil
}

// genericOf is the generic d declares, or nil.
func (ge *genericsExpander) genericOf(d node.Stmt) *genericDecl {
	var name node.Expr
	switch t := d.(type) {
	case *node.TypeDeclStmt:
		name = t.NameExpr
	case *node.InterfaceStmt:
		name = t.NameExpr
	default:
		return nil
	}
	id, _ := name.(*node.IdentExpr)
	if id == nil {
		return nil
	}
	if g := ge.generics[id.Name]; g != nil && g.stmt == d {
		return g
	}
	return nil
}

// instance is the declaration of g named name, its parameters replaced by args
// — by their aliases where args has none.
func (ge *genericsExpander) instance(g *genericDecl, name string, args [][]*node.TypeExpr) node.Stmt {
	env := make(map[string][]*node.TypeExpr, len(g.params))
	for i, p := range g.params {
		switch {
		case i < len(args):
			env[p.Ident.Name] = args[i]
		case len(p.Type) > 0:
			env[p.Ident.Name] = p.Type
		default:
			env[p.Ident.Name] = []*node.TypeExpr{{Expr: node.EIdent(TAny.Name(), p.Ident.Pos())}}
		}
	}
	// the aliases are written in the scope of the declaration: a parameter
	// named in one is not replaced
	cp := cloneNode(reflect.ValueOf(g.stmt)).Interface().(node.Stmt)
	var nameExpr *node.Expr
	switch t := cp.(type) {
	case *node.TypeDeclStmt:
		t.TypeParams = nil
		nameExpr = &t.NameExpr
	case *node.InterfaceStmt:
		t.TypeParams = nil
		nameExpr = &t.NameExpr
	}
	pos := (*nameExpr).Pos()
	*nameExpr = nil // not a use: not rewritten
	ge.rewrite(reflect.ValueOf(&cp).Elem(), env)
	*nameExpr = node.EIdent(name, pos)

	// what `@tparams` reports: each parameter and its types here
	targs := make([]*node.TypedIdentExpr, len(g.params))
	for i, p := range g.params {
		targs[i] = &node.TypedIdentExpr{Ident: p.Ident, Type: ge.expandTypes([]*node.TypeExpr{{Expr: p.Ident}}, env)}
	}
	switch t := cp.(type) {
	case *node.TypeDeclStmt:
		t.TypeArgs = targs
	case *node.InterfaceStmt:
		t.TypeArgs = targs
	}
	return cp
}

var (
	exprType     = reflect.TypeOf((*node.Expr)(nil)).Elem()
	typeExprsTyp = reflect.TypeOf([]*node.TypeExpr(nil))
)

// rewrite replaces, in the settable v, the parameters of env by their types,
// and each use of a generic by its instance's name.
func (ge *genericsExpander) rewrite(v reflect.Value, env map[string][]*node.TypeExpr) {
	if ge.err != nil {
		return
	}
	switch v.Kind() {
	case reflect.Interface:
		if v.IsNil() {
			return
		}
		e := v.Elem()
		if v.Type() == exprType {
			if r := ge.replace(e.Interface(), env); r != nil {
				v.Set(reflect.ValueOf(r))
				return
			}
		}
		if e.Kind() == reflect.Ptr {
			ge.rewrite(e, env)
		}
	case reflect.Ptr:
		if v.IsNil() || v.Elem().Kind() != reflect.Struct {
			return
		}
		switch n := v.Interface().(type) {
		case *node.TypedIdentExpr:
			// the name is not a use
			ge.rewrite(reflect.ValueOf(&n.Type).Elem(), env)
			return
		case *node.SelectorExpr:
			ge.rewrite(reflect.ValueOf(&n.X).Elem(), env)
			return
		}
		ge.rewrite(v.Elem(), env)
	case reflect.Struct:
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			if t.Field(i).IsExported() {
				ge.rewrite(v.Field(i), env)
			}
		}
	case reflect.Slice:
		if v.Type() == typeExprsTyp && env != nil {
			v.Set(reflect.ValueOf(ge.expandTypes(v.Interface().([]*node.TypeExpr), env)))
		}
		for i := 0; i < v.Len(); i++ {
			ge.rewrite(v.Index(i), env)
		}
	}
}

// expandTypes is ts, each type naming a parameter of env replaced by its
// types: `x T` with T `int|str` is `x int|str`.
func (ge *genericsExpander) expandTypes(ts []*node.TypeExpr, env map[string][]*node.TypeExpr) []*node.TypeExpr {
	var out []*node.TypeExpr
	for i, t := range ts {
		id, _ := t.Expr.(*node.IdentExpr)
		if id == nil || env[id.Name] == nil {
			if out != nil {
				out = append(out, t)
			}
			continue
		}
		if out == nil {
			out = append(make([]*node.TypeExpr, 0, len(ts)), ts[:i]...)
		}
		for _, r := range env[id.Name] {
			out = append(out, cloneNode(reflect.ValueOf(r)).Interface().(*node.TypeExpr))
		}
	}
	if out == nil {
		return ts
	}
	return out
}

// replace is what e is replaced by: a parameter of env, its types; a use of a
// generic, its instance's name. nil when e stays.
func (ge *genericsExpander) replace(e any, env map[string][]*node.TypeExpr) node.Expr {
	switch n := e.(type) {
	case *node.IdentExpr:
		if ts := env[n.Name]; ts != nil {
			return unionExpr(ts)
		}
	case *node.IndexExpr:
		if id, _ := n.X.(*node.IdentExpr); id != nil && ge.generics[id.Name] != nil {
			ge.rewrite(reflect.ValueOf(&n.Index).Elem(), env)
			return ge.use(id, []node.Expr{n.Index})
		}
	case *node.TypeArgsExpr:
		if id, _ := n.X.(*node.IdentExpr); id != nil && ge.generics[id.Name] != nil {
			ge.rewrite(reflect.ValueOf(&n.Args).Elem(), env)
			return ge.use(id, n.Args)
		}
	}
	return nil
}

// use is the name of the instance `id[args]`, queued to be declared.
func (ge *genericsExpander) use(id *node.IdentExpr, args []node.Expr) node.Expr {
	g := ge.generics[id.Name]
	if len(args) > len(g.params) {
		ge.err = ge.c.Errorf(id, "%s has %d type parameters, given %d type arguments", g.name, len(g.params), len(args))
		return nil
	}
	types := make([][]*node.TypeExpr, len(args))
	parts := make([]string, len(g.params))
	for i, p := range g.params {
		var ts []*node.TypeExpr
		if i < len(args) {
			ts = unionTypes(args[i])
			types[i] = ts
		} else {
			ts = p.Type
		}
		if len(ts) == 0 {
			parts[i] = TAny.Name()
			continue
		}
		s := make([]string, len(ts))
		for j, t := range ts {
			s[j] = t.String()
		}
		parts[i] = strings.Join(s, "|")
	}
	key := g.name + "[" + strings.Join(parts, ", ") + "]"
	if !ge.seen[key] {
		if ge.count++; ge.count > maxGenericInstances {
			ge.err = ge.c.Errorf(id, "%s: more than %d instances of the generics of this block", key, maxGenericInstances)
			return nil
		}
		ge.seen[key] = true
		ge.queue = append(ge.queue, genericWork{g: g, key: key, args: types})
	}
	return node.EIdent(key, id.Pos())
}

// unionTypes are the types of a type argument: `int|str` is two.
func unionTypes(e node.Expr) []*node.TypeExpr {
	switch t := e.(type) {
	case *node.BinaryExpr:
		if t.Token == token.Or {
			return append(unionTypes(t.LHS), unionTypes(t.RHS)...)
		}
	case *node.TypeExpr:
		return []*node.TypeExpr{t}
	case *node.ParenExpr:
		return unionTypes(t.Expr)
	}
	return []*node.TypeExpr{{Expr: e}}
}

// unionExpr is ts written as an expression: `int|str`.
func unionExpr(ts []*node.TypeExpr) node.Expr {
	var out node.Expr
	for _, t := range ts {
		e := cloneNode(reflect.ValueOf(t.Expr)).Interface().(node.Expr)
		if out == nil {
			out = e
		} else {
			out = &node.BinaryExpr{LHS: out, RHS: e, Token: token.Or, TokenPos: e.Pos()}
		}
	}
	return out
}

// typeArgsExpr is the `[T=int, K=int|str]` key-value array of args, each
// parameter and its types — several, an array of them —, what `@tparams`
// reports; nil when there are none.
func typeArgsExpr(args []*node.TypedIdentExpr) node.Expr {
	if len(args) == 0 {
		return nil
	}
	kva := &node.KeyValueArrayLit{}
	for _, a := range args {
		var v node.Expr
		if len(a.Type) == 1 {
			v = a.Type[0].Expr
		} else {
			arr := &node.ArrayExpr{}
			for _, t := range a.Type {
				arr.Elements = append(arr.Elements, t.Expr)
			}
			v = arr
		}
		kva.Elements = append(kva.Elements, &node.KeyValueLit{Key: node.EIdent(a.Ident.Name, a.Ident.Pos()), Value: v})
	}
	return kva
}

// cloneNode is a deep copy of the syntax tree v.
func cloneNode(v reflect.Value) reflect.Value {
	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			return v
		}
		cp := reflect.New(v.Elem().Type())
		cp.Elem().Set(v.Elem())
		if v.Elem().Kind() == reflect.Struct {
			cloneFields(cp.Elem())
		}
		return cp
	case reflect.Interface:
		if v.IsNil() {
			return v
		}
		cp := reflect.New(v.Type()).Elem()
		cp.Set(cloneNode(v.Elem()))
		return cp
	case reflect.Slice:
		if v.IsNil() {
			return v
		}
		cp := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
		for i := 0; i < v.Len(); i++ {
			cp.Index(i).Set(cloneNode(v.Index(i)))
		}
		return cp
	case reflect.Struct:
		cp := reflect.New(v.Type()).Elem()
		cp.Set(v)
		cloneFields(cp)
		return cp
	}
	return v
}

func cloneFields(s reflect.Value) {
	t := s.Type()
	for i := 0; i < s.NumField(); i++ {
		if t.Field(i).IsExported() {
			f := s.Field(i)
			f.Set(cloneNode(f))
		}
	}
}
