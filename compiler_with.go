package gad

import (
	"fmt"

	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/token"
)

// withTempHandle is the synthetic variable a `with` binds its resource to when
// the resource is not a bare identifier and no `as`/target name is given.
const withTempHandle = "__with__"

// withScaffold parses the block scaffold shared by every `with` form: register
// the exit hook as a `deferb` (so it runs at block exit, with the block's $err),
// then enter the resource. handle is the resource's identifier.
func withScaffold(handle string) ([]node.Stmt, error) {
	src := fmt.Sprintf("deferb { gad.exit(%[1]s, $err) }\ngad.enter(%[1]s)", handle)
	return parseGadSnippet(src)
}

// withBinding builds `handle <tok> resource` (a `:=` define or `=` assign) used
// to bind a `with` resource to its handle.
func withBinding(handle *node.IdentExpr, tok token.Token, resource node.Expr) node.Stmt {
	return node.SAssign([]node.Expr{handle}, []node.Expr{resource}, tok, handle.NamePos)
}

// compileWithStmt desugars a `with` statement into a block that pairs
// gad.enter/gad.exit around the body (the exit via `deferb`, so it runs on
// every exit including an error). See doc/operators.md.
//
//	with R { body }           ->  { deferb { gad.exit(R, $err) }; gad.enter(R); body }
//	with R as f { body }      ->  { f := R; deferb { gad.exit(f, $err) }; gad.enter(f); body }
//	with x = R { body }       ->  x = R; { deferb { gad.exit(x, $err) }; gad.enter(x); body }
//	with x := R { body }      ->  x := R; { deferb { gad.exit(x, $err) }; gad.enter(x); body }
func (c *Compiler) compileWithStmt(nd *node.WithStmt) error {
	var (
		handle string
		pre    []node.Stmt // emitted before the block (assign/define forms)
		inBind node.Stmt   // binding prepended inside the block (as/temp forms)
	)

	switch nd.Bind {
	case node.WithBindAssign:
		handle = nd.Ident.Name
		pre = []node.Stmt{withBinding(nd.Ident, token.Assign, nd.Resource)}
	case node.WithBindDefine:
		handle = nd.Ident.Name
		pre = []node.Stmt{withBinding(nd.Ident, token.Define, nd.Resource)}
	case node.WithBindAs:
		handle = nd.Ident.Name
		inBind = withBinding(nd.Ident, token.Define, nd.Resource)
	default: // WithBindNone
		if id, ok := nd.Resource.(*node.IdentExpr); ok {
			handle = id.Name
		} else {
			handle = withTempHandle
			inBind = withBinding(node.EIdent(handle, nd.WithPos), token.Define, nd.Resource)
		}
	}

	blockStmts, err := withScaffold(handle)
	if err != nil {
		return c.Errorf(nd, "with: %v", err)
	}
	if inBind != nil {
		blockStmts = append([]node.Stmt{inBind}, blockStmts...)
	}
	blockStmts = append(blockStmts, nd.Body.Stmts...)

	block := &node.BlockStmt{Stmts: blockStmts, LBrace: nd.Body.LBrace, RBrace: nd.Body.RBrace}

	for _, s := range append(pre, block) {
		if err = c.Compile(s); err != nil {
			return err
		}
	}
	return nil
}

// compileWithExpr desugars the `with` expression forms into an immediately-invoked
// closure that enters R (with the exit hook deferred) and yields a value:
//
//	with R [as f]: V     ->  (func() { [f := R]; deferb { gad.exit(f, $err) }; gad.enter(f); return V })()
//	with R [as f] { B }  ->  (func() { [f := R]; deferb { gad.exit(f, $err) }; gad.enter(f); B; return f })()
//
// The block form yields the resource itself, so a caller can build into it and
// receive it (e.g. `return with Tag(parent, …) as tag { … }`).
func (c *Compiler) compileWithExpr(nd *node.WithExpr) error {
	var (
		handle string
		inBind node.Stmt
	)
	switch {
	case nd.Ident != nil:
		handle = nd.Ident.Name
		inBind = withBinding(nd.Ident, token.Define, nd.Resource)
	default:
		if id, ok := nd.Resource.(*node.IdentExpr); ok {
			handle = id.Name
		} else {
			handle = withTempHandle
			inBind = withBinding(node.EIdent(handle, nd.WithPos), token.Define, nd.Resource)
		}
	}

	bodyStmts, err := withScaffold(handle)
	if err != nil {
		return c.Errorf(nd, "with: %v", err)
	}
	if inBind != nil {
		bodyStmts = append([]node.Stmt{inBind}, bodyStmts...)
	}
	if nd.Body != nil {
		// Block form: run the body, then yield the resource handle.
		bodyStmts = append(bodyStmts, nd.Body.Stmts...)
		bodyStmts = append(bodyStmts, node.SReturn(nd.WithPos, node.EIdent(handle, nd.WithPos)))
	} else {
		bodyStmts = append(bodyStmts, node.SReturn(nd.ColonPos, nd.Value))
	}

	// Build the IIFE by parsing an empty closure and splicing the body in.
	tmpl, err := parseGadSnippet("$__with_iife__ := func() {}")
	if err != nil {
		return c.Errorf(nd, "with: %v", err)
	}
	fn, ok := withFuncOf(tmpl)
	if !ok {
		return c.Errorf(nd, "with: closure slot not found")
	}
	fn.Body = &node.BlockStmt{Stmts: bodyStmts}

	return c.Compile(&node.CallExpr{Func: fn})
}

// withFuncOf extracts the `func() {}` literal parsed from the IIFE template.
func withFuncOf(stmts []node.Stmt) (*node.FuncExpr, bool) {
	if len(stmts) != 1 {
		return nil, false
	}
	as, ok := stmts[0].(*node.AssignStmt)
	if !ok || len(as.RHS) != 1 {
		return nil, false
	}
	fn, ok := as.RHS[0].(*node.FuncExpr)
	return fn, ok
}
