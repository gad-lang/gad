package gad

import (
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/token"
)

// compileContractStmt compiles a `contract NAME { *P…; target T; meta { … } }`
// declaration to `const NAME = <the contract>` (compileContractExpr):
//
//	const NAME = Contract("NAME", [P…], interface { target T },
//	    (target) => interface { … }; meta=[…])
//
// The target is checked as a field typed T (`interface { target T }`); the
// meta is a function of the target, so `target` is a type and a value in it —
// `options []target = target.@members` —, its interface built for each target.
func (c *Compiler) compileContractStmt(nd *node.ContractStmt) error {
	pos := nd.Pos()
	return c.Compile(&node.DeclStmt{Decl: &node.GenDecl{
		TokPos: pos,
		Tok:    token.Const,
		Specs: []node.Spec{&node.ValueSpec{
			Idents: []*node.IdentExpr{nd.NameExpr},
			Values: []node.Expr{&nd.ContractExpr},
		}},
	}})
}

// compileContractExpr compiles a contract as a value — `contract { … }`, or a
// named one — to the `Contract(…)` call that builds it (compileContractStmt).
func (c *Compiler) compileContractExpr(nd *node.ContractExpr) error {
	pos := nd.Pos()
	name := "contract"
	if nd.NameExpr != nil {
		name = nd.NameExpr.Name
	}
	var targetIface node.Expr = &node.NilLit{TokenPos: pos}
	if len(nd.Target) > 0 {
		targetIface = &node.InterfaceExpr{
			LBrace: pos,
			RBrace: pos,
			Members: []*node.InterfaceMemberExpr{{
				Kind: node.IfaceField,
				Name: &node.TypedIdentExpr{Ident: node.EIdent("target", nd.TargetPos), Type: nd.Target},
			}},
		}
	}
	var factory node.Expr = &node.NilLit{TokenPos: pos}
	if nd.MetaIface != nil {
		factory = &node.ClosureExpr{
			Params: node.FuncParams{Args: node.ArgsList{Values: []*node.TypedIdentExpr{
				node.ETypedIdent(node.EIdent("target", nd.MetaIface.LBrace)),
			}}},
			Body: nd.MetaIface,
		}
	}
	parents := &node.ArrayExpr{LBrack: pos, RBrack: pos, Elements: nd.Parents}

	var named node.CallExprNamedArgs
	if m := metaArgExpr(nd.Meta); m != nil {
		named.AppendS("meta", m)
	}
	// the keys of its meta required — no `?`, no default —: the overloads
	// of contracts are told apart by them first
	required := 0
	if nd.MetaIface != nil {
		for _, m := range nd.MetaIface.Members {
			if m.Kind == node.IfaceField && !m.Name.Nullable && m.Default == nil {
				required++
			}
		}
	}
	if required > 0 {
		named.AppendS("required", &node.IntLit{Value: int64(required), ValuePos: pos})
	}
	call := &node.CallExpr{
		Func: node.EIdent(BuiltinContract.String(), pos),
		CallArgs: node.CallArgs{
			Args:      node.CallExprPositionalArgs{Values: []node.Expr{node.Str(name, pos), parents, targetIface, factory}},
			NamedArgs: named,
		},
	}
	return c.Compile(call)
}
