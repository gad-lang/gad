package gad

import (
	"reflect"
	"strings"

	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/token"
)

// A const is visible in the whole block it is declared in, not only after its
// declaration: a statement — or another const — may refer to one declared
// further down. The named declarations are consts too (`func f`, `class C`,
// `mixin M`, `type T { … }`, `type T []E`, `type T <A|B>`, `interface I`,
// `meti M`, `enum E`), so they may be written in any order and refer to one
// another.
//
// How: each const is assigned where it is written — unless something before
// needs its value, in which case it is assigned first, right before that
// (its dependencies, in order). One used before, but not needed yet (by a
// function body, say), is declared right before that use, nil, and assigned
// where it is written; a class, a mixin and a marker type are declared made,
// empty, and defined where they are written: a reference to one needs only the
// type, so classes may refer to each other in a cycle (a field of A typed B,
// and of B typed A). Extending one, or using it as a mixin, needs its
// definition: that is a dependency. Code in order compiles as before.
//
// A cycle of values (`const a = b; const b = a`) is an error. A const whose
// value uses a variable of the block declared after the place it is needed is
// an unresolved reference, as it would be written there. A reference inside a
// function body is not a dependency: the body runs later, and sees the const
// through its (shared) slot.
//
// A const declared with iota, without a value, or by destructuring keeps the
// old rule: visible from its declaration on.

type hoistKind int

const (
	// hoistValue is a const assigned once, where it is written (or first
	// needed): a `const x = …`, a func, an interface, an enum, a typed array
	// type, a type union, a meti.
	hoistValue hoistKind = iota
	// hoistType is a class, a mixin or a marker type: made, empty, where it
	// is first used (hoistSlot), defined where it is written (or first needed).
	hoistType
)

type hoistItem struct {
	name  *node.IdentExpr
	kind  hoistKind
	plain bool      // a `const x = …`: its value may use a class (instantiate it)
	decl  node.Node // what its dependencies are read from
	unit  node.Stmt // what is compiled to assign it
	typ   *node.TypeLitExpr
	state int     // 0 pending, 1 being assigned, 2 assigned
	slot  *Symbol // declared before its declaration (hoistSlot)
}

type hoistScope struct {
	items   map[string]*hoistItem
	order   []*hoistItem
	byStmt  map[node.Stmt][]*hoistItem
	pending int
	path    []*hoistItem
}

// hoisted compiles the statement list stmts by f, with its consts visible in
// the whole list (see above).
func (c *Compiler) hoisted(stmts []node.Stmt, f func() error) error {
	hs, err := c.hoistCollect(stmts)
	if err != nil {
		return err
	}
	prev := c.hoist
	c.hoist = hs
	defer func() { c.hoist = prev }()
	return f()
}

// hoistCollect gathers the consts of the list: nil when there is none.
func (c *Compiler) hoistCollect(stmts []node.Stmt) (*hoistScope, error) {
	var hs *hoistScope
	add := func(owner node.Stmt, it *hoistItem) error {
		if hs == nil {
			hs = &hoistScope{items: map[string]*hoistItem{}, byStmt: map[node.Stmt][]*hoistItem{}}
		}
		if it.name.Name == "_" {
			return nil
		}
		if _, dup := hs.items[it.name.Name]; dup {
			return c.Errorf(it.name, "%q redeclared in this block", it.name.Name)
		}
		hs.items[it.name.Name] = it
		hs.order = append(hs.order, it)
		hs.byStmt[owner] = append(hs.byStmt[owner], it)
		hs.pending++
		return nil
	}

	for _, s := range stmts {
		d := s
		if e, ok := s.(*node.ExportStmt); ok && e.Prelude != nil {
			d = e.Prelude
		}
		var it *hoistItem
		switch t := d.(type) {
		case *node.DeclStmt:
			g, _ := t.Decl.(*node.GenDecl)
			if g == nil || g.Tok != token.Const || !hoistableConst(g) {
				continue
			}
			for _, sp := range g.Specs {
				spec := sp.(*node.ValueSpec)
				for i, id := range spec.Idents {
					one := &node.ValueSpec{Idents: []*node.IdentExpr{id}, Values: []node.Expr{spec.Values[i]}}
					if err := add(s, &hoistItem{
						name:  id,
						kind:  hoistValue,
						plain: true,
						decl:  spec.Values[i],
						unit:  &node.DeclStmt{Decl: &node.GenDecl{TokPos: g.TokPos, Tok: token.Const, Specs: []node.Spec{one}}},
					}); err != nil {
						return nil, err
					}
				}
			}
			continue
		case *node.TypeDeclStmt:
			if id, _ := t.NameExpr.(*node.IdentExpr); id != nil {
				it = &hoistItem{name: id, kind: hoistType, decl: &t.TypeLitExpr, unit: t, typ: &t.TypeLitExpr}
			}
		case *node.InterfaceStmt:
			if id, _ := t.NameExpr.(*node.IdentExpr); id != nil {
				it = &hoistItem{name: id, decl: &t.InterfaceExpr, unit: t}
			}
		case *node.MethodInterfaceStmt:
			if id, _ := t.NameExpr.(*node.IdentExpr); id != nil {
				it = &hoistItem{name: id, decl: &t.MethodInterfaceExpr, unit: t}
			}
		case *node.EnumStmt:
			if id, _ := t.NameExpr.(*node.IdentExpr); id != nil {
				it = &hoistItem{name: id, decl: &t.EnumExpr, unit: t}
			}
		case *node.FuncStmt:
			if id, _ := t.Func.Type.NameExpr.(*node.IdentExpr); id != nil {
				it = &hoistItem{name: id, decl: t.Func, unit: t}
			}
		case *node.FuncWithMethodsStmt:
			if id, _ := t.NameExpr.(*node.IdentExpr); id != nil {
				it = &hoistItem{name: id, decl: &t.FuncWithMethodsExpr, unit: t}
			}
		case *node.TypedArrayTypeStmt:
			if t.NameExpr != nil {
				it = &hoistItem{name: t.NameExpr, decl: t, unit: t}
			}
		}
		if it != nil {
			if err := add(s, it); err != nil {
				return nil, err
			}
		}
	}
	return hs, nil
}

// hoistableConst reports whether each const of g has its own value: one using
// iota, one repeating the value before and a destructuring keep the old rule.
func hoistableConst(g *node.GenDecl) bool {
	for _, sp := range g.Specs {
		spec, _ := sp.(*node.ValueSpec)
		if spec == nil || spec.Pattern != nil || len(spec.Values) != len(spec.Idents) {
			return false
		}
		for _, id := range spec.Idents {
			if id.Name == "iota" {
				return false // an error, where it is written
			}
		}
		for _, v := range spec.Values {
			if v == nil {
				return false
			}
			iota := false
			hoistRefs(v, func(id *node.IdentExpr) { iota = iota || id.Name == "iota" })
			if iota {
				return false
			}
		}
	}
	return true
}

// hoistSlot declares the const it, not assigned yet, where something before
// its declaration uses it: the name gets its slot, nil — a type, the empty one
// its declaration defines (`Class("Name")`, `Mixin("Name")`,
// `StaticType("Name")`). A const nothing uses before is declared where it is
// written, as any other.
func (c *Compiler) hoistSlot(it *hoistItem) error {
	if it.state == 2 || it.slot != nil {
		return nil
	}
	sym, exists := c.symbolTable.DefineLocal(it.name.Name)
	if exists {
		return c.Errorf(it.name, "%q redeclared in this block", it.name.Name)
	}
	if it.kind == hoistType {
		fn := BuiltinNewClass
		switch {
		case it.typ.Mixin:
			fn = BuiltinNewMixin
		case it.typ.Static:
			fn = BuiltinNewStaticType
		}
		pos := it.name.Pos()
		if err := c.Compile(&node.CallExpr{
			Func:     node.EIdent(fn.String(), pos),
			CallArgs: node.CallArgs{Args: node.CallExprPositionalArgs{Values: []node.Expr{node.Str(it.name.Name, pos)}}},
		}); err != nil {
			return err
		}
	} else {
		c.emit(it.name, OpNil)
	}
	c.emit(it.name, OpDefineLocal, sym.Index)
	sym.Assigned = true
	sym.Constant = true
	sym.hoistPending = true
	it.slot = sym
	return nil
}

// hoistSlots declares the consts n uses — in function bodies too — that are
// not assigned yet (hoistSlot); self is the one n declares.
func (c *Compiler) hoistSlots(n ast.Node, self map[*hoistItem]bool) error {
	hs := c.hoist
	var uses []*hoistItem
	hoistWalk(reflect.ValueOf(n), true, func(id *node.IdentExpr) {
		if it := hs.items[id.Name]; it != nil && !self[it] && it.state != 2 && it.slot == nil {
			uses = append(uses, it)
		}
	})
	for _, it := range uses {
		if err := c.hoistSlot(it); err != nil {
			return err
		}
	}
	return nil
}

// hoistStmt compiles the statement s of the list with its consts: assigns
// the consts it declares, and first the ones it needs. It reports whether s
// was compiled (a statement that only declares consts).
func (c *Compiler) hoistStmt(s node.Stmt) (done bool, err error) {
	hs := c.hoist
	if hs == nil {
		return false, nil
	}
	if items := hs.byStmt[s]; items != nil {
		for _, it := range items {
			if err = c.hoistAssign(it); err != nil {
				return
			}
		}
		if e, ok := s.(*node.ExportStmt); ok {
			// the export itself, its declaration assigned
			cp := *e
			cp.Prelude = nil
			return true, c.Compile(&cp)
		}
		return true, nil
	}
	if hs.pending == 0 {
		return false, nil
	}
	if err = c.hoistSlots(s, nil); err != nil {
		return
	}
	err = c.hoistNeeds(s, true, nil)
	return
}

// hoistNeeds assigns the consts n needs, before it: those it refers to out of
// a function body. A class is needed as a value only when strict (it may be
// instantiated); otherwise the one hoistSlot made serves.
func (c *Compiler) hoistNeeds(n ast.Node, strict bool, self *hoistItem) error {
	hs := c.hoist
	var need []*hoistItem
	hoistRefs(n, func(id *node.IdentExpr) {
		if it := hs.items[id.Name]; it != nil && it != self && (strict || it.kind != hoistType) {
			need = append(need, it)
		}
	})
	for _, it := range need {
		if err := c.hoistAssign(it); err != nil {
			return err
		}
	}
	if strict {
		// n runs now: what it calls may run too, and use the consts of its
		// body
		seen := map[*hoistItem]bool{}
		for _, it := range need {
			if err := c.hoistBodies(it, seen); err != nil {
				return err
			}
		}
	}
	return nil
}

// hoistBodies assigns the consts the function bodies of it use, and theirs:
// it may be called now. One being assigned (a recursion) is left: its slot is
// set before a call returns.
func (c *Compiler) hoistBodies(it *hoistItem, seen map[*hoistItem]bool) error {
	if seen[it] {
		return nil
	}
	seen[it] = true
	hs := c.hoist
	var uses []*hoistItem
	hoistWalk(reflect.ValueOf(it.decl), true, func(id *node.IdentExpr) {
		if u := hs.items[id.Name]; u != nil && u != it {
			uses = append(uses, u)
		}
	})
	for _, u := range uses {
		if u.state == 0 {
			if err := c.hoistAssign(u); err != nil {
				return err
			}
		}
		if err := c.hoistBodies(u, seen); err != nil {
			return err
		}
	}
	return nil
}

// hoistAssign assigns the const it, after the ones it needs.
func (c *Compiler) hoistAssign(it *hoistItem) error {
	hs := c.hoist
	switch it.state {
	case 2:
		return nil
	case 1:
		var names []string
		start := 0
		for i, p := range hs.path {
			if p == it {
				start = i
			}
		}
		for _, p := range hs.path[start:] {
			names = append(names, p.name.Name)
		}
		names = append(names, it.name.Name)
		return c.Errorf(it.name, "initialization cycle: %s", strings.Join(names, " refers to "))
	}
	it.state = 1
	hs.path = append(hs.path, it)
	defer func() { hs.path = hs.path[:len(hs.path)-1] }()

	if err := c.hoistNeeds(it.decl, it.plain, it); err != nil {
		return err
	}
	if it.kind == hoistType {
		// what it extends and the mixins it uses are defined before it
		for _, p := range it.typ.Parents {
			if err := c.hoistNeeds(p, true, it); err != nil {
				return err
			}
		}
		for _, u := range it.typ.Use {
			if err := c.hoistNeeds(u, true, it); err != nil {
				return err
			}
		}
	}
	// an interface may name itself (`interface Node { next? Node }`): its
	// types are read when a value is checked, the slot assigned by then
	_, iface := it.decl.(*node.InterfaceExpr)
	if err := c.hoistSlots(it.decl, map[*hoistItem]bool{it: !iface}); err != nil {
		return err
	}
	if err := c.Compile(it.unit); err != nil {
		return err
	}
	it.state = 2
	hs.pending--
	return nil
}

// hoistedType is the symbol of the type named name hoistSlot made, not yet
// defined: its declaration defines that one (the builtin receives it in place
// of the name).
func (c *Compiler) hoistedType(name string) *Symbol {
	if c.hoist == nil {
		return nil
	}
	if it := c.hoist.items[name]; it == nil || it.kind != hoistType {
		return nil
	}
	if s := c.symbolTable.store[name]; s != nil && s.hoistPending {
		return s
	}
	return nil
}

// hoistRefs calls f with each identifier n refers to — not a name it declares
// (a field, a member, a parameter, a key, a selector), and not one inside a
// function body.
func hoistRefs(n ast.Node, f func(*node.IdentExpr)) {
	hoistWalk(reflect.ValueOf(n), false, f)
}

var (
	hoistIdentT  = reflect.TypeOf((*node.IdentExpr)(nil))
	hoistBodyOf  = map[reflect.Type]bool{reflect.TypeOf(node.FuncExpr{}): true, reflect.TypeOf(node.FuncMethod{}): true, reflect.TypeOf(node.ClosureExpr{}): true}
	hoistNameFld = map[string]bool{"NameExpr": true, "Ident": true, "Name": true, "Sel": true, "Key": true, "Rest": true}
)

// hoistWalk is hoistRefs; bodies: with the function bodies.
func hoistWalk(v reflect.Value, bodies bool, f func(*node.IdentExpr)) {
	switch v.Kind() {
	case reflect.Interface:
		if !v.IsNil() {
			hoistWalk(v.Elem(), bodies, f)
		}
	case reflect.Ptr:
		if v.IsNil() {
			return
		}
		if v.Type() == hoistIdentT {
			f(v.Interface().(*node.IdentExpr))
			return
		}
		hoistWalk(v.Elem(), bodies, f)
	case reflect.Struct:
		t := v.Type()
		body := !bodies && hoistBodyOf[t]
		for i := 0; i < v.NumField(); i++ {
			fld := t.Field(i)
			if fld.PkgPath != "" {
				continue
			}
			if body && (fld.Name == "Body" || fld.Name == "BodyExpr") {
				continue
			}
			fv := v.Field(i)
			if hoistNameFld[fld.Name] && isHoistIdent(fv) {
				continue
			}
			hoistWalk(fv, bodies, f)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			hoistWalk(v.Index(i), bodies, f)
		}
	}
}

// isHoistIdent reports whether v holds an identifier directly.
func isHoistIdent(v reflect.Value) bool {
	for v.Kind() == reflect.Interface {
		if v.IsNil() {
			return false
		}
		v = v.Elem()
	}
	return v.Type() == hoistIdentT
}

// hoistExported assigns the consts the module's exports give, before the
// exports: a const exported before its declaration.
func (c *Compiler) hoistExported(dict *node.DictExpr) error {
	if c.hoist == nil || c.hoist.pending == 0 {
		return nil
	}
	if err := c.hoistSlots(dict, nil); err != nil {
		return err
	}
	return c.hoistNeeds(dict, true, nil)
}
