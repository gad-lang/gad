package gad

import (
	"fmt"

	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/token"
)

// TestRegistryPrefix marks the top-level const bindings that `test`/`bench`
// statements lower to. The `gad test` runner discovers them by this prefix. Each
// binds a `[kind, name, func(t){…}, doc]` array.
const TestRegistryPrefix = "__gadTest_"

// compileTestStmt lowers a `test NAME { … }` / `bench NAME { … }` statement to a
// top-level const binding
//
//	const __gadTest_<pos> = ["test"|"bench", "NAME", func(t) { … }, "doc"]
//
// so the body runs with an injected `t` parameter and the `gad test` runner can
// discover it (by the testRegistryPrefix) and invoke it with a fresh test
// context. When run outside `gad test` the binding is simply an unused value.
func (c *Compiler) compileTestStmt(nd *node.TestStmt) error {
	pos := nd.Pos()

	// func(t) { <body> }
	fnType := node.NewFuncType(pos, pos, pos, node.ArgsList{
		Values: []*node.TypedIdentExpr{node.ETypedIdent(node.EIdent("t", pos))},
	})
	fn := node.EFunc(fnType, nd.Body)

	var doc string
	if nd.Doc != nil {
		doc = nd.Doc.Text()
	}

	entry := &node.ArrayExpr{Elements: []node.Expr{
		node.Str(nd.Kind.String(), pos),
		node.Str(nd.Name, pos),
		fn,
		node.Str(doc, pos),
	}}

	name := node.EIdent(fmt.Sprintf("%s%d", TestRegistryPrefix, int(nd.KwPos)), pos)
	return c.Compile(&node.DeclStmt{
		Decl: &node.GenDecl{
			Tok: token.Const,
			Specs: []node.Spec{
				&node.ValueSpec{
					Idents: []*node.IdentExpr{name},
					Values: []node.Expr{entry},
				},
			},
		},
	})
}
