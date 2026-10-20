package gad

import (
	"strconv"

	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/token"
)

// typedArrayDefineExpr lowers the optional member body of a typed array type
// (`type T []int { fields; props {…}; new(…) {…}; methods {…} }`) to the define
// handler `(T, define) => define(; fields=…, initFields=…, props=…, methods=…,
// new=…)`, reusing the class lowering: methods take a typed `this T` (so overloads
// dispatch on the typed array), properties and constructors an untyped one. It
// returns nil for an empty body.
func (c *Compiler) typedArrayDefineExpr(body *node.TypeLitExpr) (node.Expr, error) {
	pos := body.Pos()
	typeIdent := node.EIdent("T", pos)
	defineIdent := node.EIdent("define", pos)

	var inner node.CallExprNamedArgs
	if len(body.Fields) > 0 {
		fieldsExpr, initFieldsExpr := classFieldsExpr(body)
		inner.AppendS("fields", fieldsExpr)
		if initFieldsExpr != nil {
			inner.AppendS("initFields", initFieldsExpr)
		}
	}
	if len(body.Props) > 0 {
		props, err := c.classPropertiesExpr(body, nil)
		if err != nil {
			return nil, err
		}
		inner.AppendS("props", props)
	}
	if len(body.Methods) > 0 {
		methods, err := c.classMethodsExpr(body, typeIdent)
		if err != nil {
			return nil, err
		}
		inner.AppendS("methods", methods)
	}
	if len(body.New) > 0 {
		inner.AppendS("new", classNewExpr(body))
	}
	if len(inner.Values) == 0 {
		return nil, nil
	}
	return &node.ClosureExpr{
		Params: node.FuncParams{Args: node.ArgsList{Values: []*node.TypedIdentExpr{
			{Ident: typeIdent},
			{Ident: defineIdent},
		}}},
		Lambda: node.Token{Token: token.Lambda},
		Body: &node.CallExpr{
			Func:     defineIdent,
			CallArgs: node.CallArgs{NamedArgs: inner},
		},
	}, nil
}

// compileTypedArrayTypeStmt compiles a `type NAME []…ELEM` declaration to
//
//	const NAME = TypedArrayType("NAME", depth, elem; meta=[…])
//
// where elem is the element type value: the single type written (`[]int`), a
// type union of several (`[]<int|uint>`), or the inline interface
// (`[]{ name; id }`). The metadata block, when present, is passed as `meta=` and
// evaluated at run time like any key-value array.
func (c *Compiler) compileTypedArrayTypeStmt(nd *node.TypedArrayTypeStmt) error {
	pos := nd.Pos()
	var elem node.Expr
	switch {
	case nd.ElemIface != nil:
		elem = nd.ElemIface
	case len(nd.ElemTypes) == 1:
		elem = nd.ElemTypes[0].Expr
	case len(nd.ElemTypes) > 1:
		elem = &node.TypeUnionExpr{TypePos: pos, Types: nd.ElemTypes}
	default:
		return c.Errorf(nd, "typed array type %s has no element type", nd.NameExpr.Name)
	}

	var named node.CallExprNamedArgs
	if m := metaArgExpr(nd.Meta); m != nil {
		named.AppendS("meta", m)
	}
	if node.HasArrayDims(nd.Dims) {
		// the sizes of the levels: an array type of them alone (its sizes
		// naming variables are bound where the type is declared)
		named.AppendS("sizes", &node.ArrayTypeExpr{LBrack: pos, Depth: nd.Depth, Dims: nd.Dims,
			Types: []*node.TypeExpr{node.EType(node.EIdent("any", pos))}})
	}

	args := []node.Expr{
		node.Str(nd.NameExpr.Name, pos),
		&node.IntLit{Value: int64(nd.Depth), Literal: strconv.Itoa(nd.Depth), ValuePos: pos},
		elem,
	}
	if nd.Body != nil {
		define, err := c.typedArrayDefineExpr(nd.Body)
		if err != nil {
			return err
		}
		if define != nil {
			args = append(args, define)
		}
	}

	call := &node.CallExpr{
		Func: node.EIdent(BuiltinNewTypedArrayType.String(), pos),
		CallArgs: node.CallArgs{
			Args:      node.CallExprPositionalArgs{Values: args},
			NamedArgs: named,
		},
	}
	return c.Compile(&node.DeclStmt{Decl: &node.GenDecl{
		TokPos: pos,
		Tok:    token.Const,
		Specs: []node.Spec{&node.ValueSpec{
			Idents: []*node.IdentExpr{nd.NameExpr},
			Values: []node.Expr{call},
		}},
	}})
}

// typeArgsIndex is `X[A, B]` as `X[[A, B]]` (each argument a union when written
// `a|b`): a builtin generic type indexed by its type arguments.
func typeArgsIndex(nt *node.TypeArgsExpr) node.Expr {
	args := make([]node.Expr, len(nt.Args))
	for i, a := range nt.Args {
		args[i] = typeIndexUnions(&node.IndexExpr{X: nt.X, Index: a}).(*node.IndexExpr).Index
	}
	return &node.IndexExpr{X: nt.X, LBrack: nt.LBrack, RBrack: nt.RBrack,
		Index: &node.ArrayExpr{Elements: args, LBrack: nt.LBrack, RBrack: nt.RBrack}}
}

// typedDictTypeCall is a typed dict type (issue #19) as the call
// `TypedDictType(NAME, DictType[K, V]; meta=[…])` (NAME nil when anonymous; the
// generic `dict` when no type arguments are written).
func typedDictTypeCall(nd *node.TypedDictTypeExpr) *node.CallExpr {
	pos := nd.Pos()
	base, genericName, ctor := "dict", "DictType", BuiltinNewTypedDictType.String()
	if nd.Kind == "keyValueArray" {
		base, genericName, ctor = "keyValueArray", "KeyValueArrayType", BuiltinNewTypedKVAType.String()
	}
	var generic node.Expr = node.EIdent(base, pos)
	if len(nd.Args) > 0 {
		generic = typeArgsIndex(&node.TypeArgsExpr{X: node.EIdent(genericName, pos), LBrack: pos, Args: nd.Args})
	}
	var name node.Expr = &node.NilLit{TokenPos: pos}
	if nd.Name != nil {
		name = node.Str(nd.Name.Name, pos)
	}
	var named node.CallExprNamedArgs
	if m := metaArgExpr(nd.Meta); m != nil {
		named.AppendS("meta", m)
	}
	return &node.CallExpr{
		Func: node.EIdent(ctor, pos),
		CallArgs: node.CallArgs{
			Args:      node.CallExprPositionalArgs{Values: []node.Expr{name, generic}},
			NamedArgs: named,
		},
	}
}

// compileTypedDictTypeStmt compiles `type NAME dict[K, V]` to
// `const NAME = TypedDictType("NAME", DictType[K, V])`.
func (c *Compiler) compileTypedDictTypeStmt(nd *node.TypedDictTypeStmt) error {
	return c.Compile(&node.DeclStmt{Decl: &node.GenDecl{
		TokPos: nd.Pos(),
		Tok:    token.Const,
		Specs: []node.Spec{&node.ValueSpec{
			Idents: []*node.IdentExpr{nd.Name},
			Values: []node.Expr{typedDictTypeCall(&nd.TypedDictTypeExpr)},
		}},
	}})
}
