package gad

import (
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/token"
)

// compileClassStmt compiles the statement form `class Name [extends …] { … }`
// to `const Name = <class expression>`.
func (c *Compiler) compileClassStmt(nd *node.ClassStmt) error {
	name, _ := nd.NameExpr.(*node.IdentExpr)
	if name == nil {
		return c.Errorf(nd, "class statement requires a name identifier")
	}
	call, err := c.classCallExpr(&nd.ClassExpr)
	if err != nil {
		return err
	}
	return c.Compile(&node.DeclStmt{
		Decl: &node.GenDecl{
			Tok: token.Const,
			Specs: []node.Spec{
				&node.ValueSpec{
					Idents: []*node.IdentExpr{name},
					Values: []node.Expr{call},
				},
			},
		},
	})
}

// compileClassExpr compiles a class expression by lowering it to the equivalent
// Class(...) constructor call.
func (c *Compiler) compileClassExpr(nd *node.ClassExpr) error {
	call, err := c.classCallExpr(nd)
	if err != nil {
		return err
	}
	return c.Compile(call)
}

// classCallExpr lowers a class literal to
//
//	Class("Name"; define=(Type, define) => define(; extends=…, fields=…,
//	    properties=…, methods=…, new=…))
//
// The `define` callback binds `Type` to the in-construction class, so methods
// take a typed `this Type` first parameter (enabling type/arity overload
// dispatch). Property accessors and constructors take an untyped `this`: their
// param types are resolved when the accessor/constructor frame is current (not
// the define callback), where the local `Type` symbol would resolve to `this`
// rather than the class. See classNewExpr.
func (c *Compiler) classCallExpr(nd *node.ClassExpr) (*node.CallExpr, error) {
	if nd.Mixin {
		return c.mixinCallExpr(nd)
	}

	pos := nd.Pos()

	var name string
	if id, _ := nd.NameExpr.(*node.IdentExpr); id != nil {
		name = id.Name
	}

	clsIdent := node.EIdent("cls", pos)
	defineIdent := node.EIdent("define", pos)

	var inner node.CallExprNamedArgs
	if len(nd.Parents) > 0 {
		inner.AppendS("extends", classExtendsExpr(nd))
	}
	if len(nd.Use) > 0 {
		inner.AppendS("mixins", &node.ArrayExpr{Elements: nd.Use})
	}
	if len(nd.Fields) > 0 {
		fieldsExpr, initFieldsExpr := classFieldsExpr(nd)
		inner.AppendS("fields", fieldsExpr)
		if initFieldsExpr != nil {
			inner.AppendS("initFields", initFieldsExpr)
		}
	}
	if len(nd.Props) > 0 {
		props, err := c.classPropertiesExpr(nd, nil)
		if err != nil {
			return nil, err
		}
		inner.AppendS("properties", props)
	}
	if len(nd.Methods) > 0 {
		methods, err := c.classMethodsExpr(nd, clsIdent)
		if err != nil {
			return nil, err
		}
		inner.AppendS("methods", methods)
	}
	if len(nd.New) > 0 {
		inner.AppendS("new", classNewExpr(nd))
	}

	// (cls, define) => define(; …)
	callback := &node.ClosureExpr{
		Params: node.FuncParams{
			Args: node.ArgsList{
				Values: []*node.TypedIdentExpr{
					{Ident: clsIdent},
					{Ident: defineIdent},
				},
			},
		},
		Lambda: node.Token{Token: token.Lambda},
		Body: &node.CallExpr{
			Func:     defineIdent,
			CallArgs: node.CallArgs{NamedArgs: inner},
		},
	}

	// Class(name, (Type, define) => define(; …)) — the define handler is the
	// second positional argument. An empty class body needs no handler.
	args := []node.Expr{node.Str(name, pos)}
	if len(inner.Values) > 0 {
		args = append(args, callback)
	}

	return &node.CallExpr{
		Func: node.EIdent(BuiltinNewClass.String(), pos),
		CallArgs: node.CallArgs{
			Args: node.CallExprPositionalArgs{Values: args},
		},
	}, nil
}

// mixinCallExpr lowers a `mixin [Name] { … }` literal to
//
//	Mixin("Name", (mx, define) => { [interface Name$this { … }] define(; …) })
//
// A mixin shares the class body grammar (parent mixins via `extends=`, fields,
// properties, methods) but is not instantiable. When a `this { … }` block is
// present it is lowered to a local `interface Name$this { … }` const inside the
// callback and referenced as the injected `this Name$this` param type of every
// property and method (so their `this` must satisfy it); without a `this` block
// those params are untyped. Parent mixins are passed as `extends=[…]` of *Mixin
// values, and field defaults follow the same rules as a class (see
// classFieldsExpr).
func (c *Compiler) mixinCallExpr(nd *node.ClassExpr) (*node.CallExpr, error) {
	pos := nd.Pos()

	var name string
	if id, _ := nd.NameExpr.(*node.IdentExpr); id != nil {
		name = id.Name
	}

	defineIdent := node.EIdent("define", pos)
	mxIdent := node.EIdent("mx", pos)

	// The `this` of every property and method is SEMANTICALLY the mixin's
	// `@interface` (its `this` block + parents + own members). It is left untyped in
	// the lowering (no per-call receiver validation); a using class is validated
	// once against `mixin.@classInterface` in Class.useMixins instead, and editor
	// autocomplete resolves `this` to the mixin's members from the AST/@interface.
	ifaceName := name + "$this"
	var thisType node.Expr // untyped `this`
	_ = mxIdent

	var inner node.CallExprNamedArgs
	if len(nd.Parents) > 0 {
		inner.AppendS("extends", classExtendsExpr(nd))
	}
	// The `this { … }` interface is passed to define so the mixin can expose it as
	// `@this` (and build `@interface` extending it); it is referenced by the const
	// name the block is lowered to, in scope inside the callback body below.
	if nd.This != nil {
		inner.AppendS("this", node.EIdent(ifaceName, pos))
	}
	if len(nd.Fields) > 0 {
		fieldsExpr, initFieldsExpr := classFieldsExpr(nd)
		inner.AppendS("fields", fieldsExpr)
		if initFieldsExpr != nil {
			inner.AppendS("initFields", initFieldsExpr)
		}
	}
	if len(nd.Props) > 0 {
		props, err := c.classPropertiesExpr(nd, thisType)
		if err != nil {
			return nil, err
		}
		inner.AppendS("properties", props)
	}
	if len(nd.Methods) > 0 {
		methods, err := c.classMethodsExpr(nd, thisType)
		if err != nil {
			return nil, err
		}
		inner.AppendS("methods", methods)
	}

	defineCall := &node.CallExpr{
		Func:     defineIdent,
		CallArgs: node.CallArgs{NamedArgs: inner},
	}

	// The callback body: `define(; …)` on its own, or — with a `this` block —
	// `{ interface Name$this { … }; define(; …) }` so the interface const is in
	// scope for the method/property `this` param types.
	var body node.Expr = defineCall
	if nd.This != nil {
		iface := *nd.This
		iface.NameExpr = node.EIdent(ifaceName, pos)
		body = &node.BlockExpr{BlockStmt: &node.BlockStmt{
			Stmts: node.Stmts{
				&node.InterfaceStmt{InterfaceExpr: iface},
				&node.ExprStmt{Expr: defineCall},
			},
		}}
	}

	callback := &node.ClosureExpr{
		Params: node.FuncParams{
			Args: node.ArgsList{
				Values: []*node.TypedIdentExpr{
					{Ident: mxIdent},
					{Ident: defineIdent},
				},
			},
		},
		Lambda: node.Token{Token: token.Lambda},
		Body:   body,
	}

	args := []node.Expr{node.Str(name, pos)}
	if len(inner.Values) > 0 || nd.This != nil {
		args = append(args, callback)
	}

	return &node.CallExpr{
		Func: node.EIdent(BuiltinNewMixin.String(), pos),
		CallArgs: node.CallArgs{
			Args: node.CallExprPositionalArgs{Values: args},
		},
	}, nil
}

// classExtendsExpr builds the `extends=[…]` array: each parent is its type
// expression, or a `[type, "alias"]` pair when an alias was given.
func classExtendsExpr(nd *node.ClassExpr) node.Expr {
	elems := make([]node.Expr, len(nd.Parents))
	for i, p := range nd.Parents {
		if p.Alias != nil {
			elems[i] = &node.ArrayExpr{Elements: []node.Expr{p.Type, node.Str(p.Alias.Name, p.Alias.Pos())}}
		} else {
			elems[i] = p.Type
		}
	}
	return &node.ArrayExpr{Elements: elems}
}

// classFieldsExpr builds the `fields=(; …)` key-value array and, when any field
// has a per-instance (non-literal, non-ComputedExpr) default, the matching
// `initFields=() => (; name=expr, …)` closure that computes them all with one
// call.
//
// A field's default is kept inline as its value when it is a literal (evaluated
// once, shared) or a `(= expr)` ComputedExpr (evaluated per field, per instance).
// Any other default expression (`g + 5`, `f()`, `[1, 2]`) would otherwise be
// evaluated a single time at class-definition and shared across instances, so it
// is moved into initFields and re-evaluated per construction. Such a field is
// emitted with no inline value (a flag), so it has no shared static default.
func classFieldsExpr(nd *node.ClassExpr) (fields node.Expr, initFields node.Expr) {
	elems := make(node.Exprs, len(nd.Fields))
	var inits node.Exprs
	for i, f := range nd.Fields {
		var key node.Expr = f.Name
		if len(f.Name.Type) == 0 {
			key = f.Name.Ident
		}
		value := f.Value
		if value != nil && !isStaticFieldDefault(value) {
			inits = append(inits, &node.KeyValueLit{Key: f.Name.Ident, Value: value})
			value = nil // emit the field as a flag: no shared static default
		}
		elems[i] = &node.KeyValueLit{Key: key, Value: value}
	}
	fields = &node.KeyValueArrayLit{Elements: elems}
	if len(inits) > 0 {
		// () => (; name1 = expr1, name2 = expr2, …)
		initFields = node.EClosure(
			&node.FuncParams{},
			nd.Pos(),
			token.Lambda,
			&node.KeyValueArrayLit{Elements: inits},
		)
	}
	return
}

// isStaticFieldDefault reports whether a field default may stay inline in the
// `fields=(; …)` array: a scalar literal (shared, evaluated once) or a `(= expr)`
// ComputedExpr (evaluated per field per instance). Every other expression is a
// per-instance default that must move into initFields (see classFieldsExpr).
func isStaticFieldDefault(e node.Expr) bool {
	switch e.(type) {
	case *node.IntLit, *node.UintLit, *node.FloatLit, *node.DecimalLit,
		*node.BoolLit, *node.FlagLit, *node.CharLit, *node.StrLit, *node.NilLit,
		*node.ComputedExpr:
		return true
	}
	return false
}

// classMethodsExpr builds the `methods=[…]` array of named functions; each
// overload is a separate named function entry sharing the method name.
func (c *Compiler) classMethodsExpr(nd *node.ClassExpr, typeIdent node.Expr) (node.Expr, error) {
	var elems []node.Expr
	for _, m := range nd.Methods {
		name, _ := m.NameExpr.(*node.IdentExpr)
		if name == nil {
			return nil, c.Errorf(m, "class method requires a name identifier")
		}
		for _, fm := range m.Methods {
			elems = append(elems, classMemberFunc(name, fm, typeIdent))
		}
	}
	return &node.ArrayExpr{Elements: elems}, nil
}

// classPropertiesExpr builds the `properties={…}` dict mapping each property
// name to a func-with-methods value holding its accessor overloads (a zero-arg
// getter and one-arg setters).
func (c *Compiler) classPropertiesExpr(nd *node.ClassExpr, typeIdent node.Expr) (node.Expr, error) {
	elems := make([]*node.DictElementLit, 0, len(nd.Props))
	for _, p := range nd.Props {
		name, _ := p.NameExpr.(*node.IdentExpr)
		if name == nil {
			return nil, c.Errorf(p, "class property requires a name identifier")
		}
		fwm := &node.FuncWithMethodsExpr{Methods: classInjectThis(p.Methods, typeIdent)}
		elems = append(elems, node.EDictElementStr(name.Name, name.Pos(), name.Pos(), fwm))
	}
	return &node.DictExpr{Elements: elems}, nil
}

// classNewExpr builds the `new=` value: an (anonymous) func-with-methods holding
// the constructor overloads, each with a `new` first parameter prepended. `new`
// is the ClassInitiator (see ClassInitiator.Call); a constructor body builds the
// instance with a `new(; fields)` super-call.
func classNewExpr(nd *node.ClassExpr) node.Expr {
	return &node.FuncWithMethodsExpr{Methods: classInjectParam(nd.New, "new")}
}

// classInjectParam returns copies of the methods with an untyped parameter named
// `name` prepended.
func classInjectParam(methods []*node.FuncMethod, name string) []*node.FuncMethod {
	out := make([]*node.FuncMethod, len(methods))
	for i, m := range methods {
		cp := *m
		cp.Params.Args.PrependValue(&node.TypedIdentExpr{Ident: node.EIdent(name, source.NoPos)})
		out[i] = &cp
	}
	return out
}

// classMemberFunc builds a named function literal for one method overload, with
// a typed `this cls` parameter prepended.
func classMemberFunc(name *node.IdentExpr, m *node.FuncMethod, typeIdent node.Expr) *node.FuncExpr {
	params := m.Params
	params.Args.PrependValue(thisParam(typeIdent))
	return &node.FuncExpr{
		Type: &node.FuncType{
			FuncHeader: node.FuncHeader{NameExpr: name, Params: params, Return: m.Return},
		},
		Body:      m.Body,
		BodyExpr:  m.BodyExpr,
		LambdaPos: m.LambdaPos,
	}
}

// classInjectThis returns copies of the methods with a `this` first parameter
// prepended (typed `this Type` when typeIdent is non-nil, untyped otherwise),
// for the func-with-methods used by properties and `new`.
func classInjectThis(methods []*node.FuncMethod, typeIdent node.Expr) []*node.FuncMethod {
	out := make([]*node.FuncMethod, len(methods))
	for i, m := range methods {
		cp := *m
		cp.Params.Args.PrependValue(thisParam(typeIdent))
		out[i] = &cp
	}
	return out
}

// thisParam builds the injected `this` parameter. When typeIdent is non-nil it
// is typed `this Type` (Type being the define callback's class-type parameter);
// a nil typeIdent yields an untyped `this`.
func thisParam(typeIdent node.Expr) *node.TypedIdentExpr {
	if typeIdent == nil {
		return &node.TypedIdentExpr{Ident: node.EIdent("this", source.NoPos)}
	}
	return &node.TypedIdentExpr{
		Ident: node.EIdent("this", typeIdent.Pos()),
		Type:  []*node.TypeExpr{{Expr: typeIdent}},
	}
}
