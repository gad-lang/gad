package gad

import (
	"fmt"

	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
)

// deferWrapperTemplate is spliced around a function body that uses `defer`.
// The original body is moved into $__body so its return value can be captured
// into $ret; registered handlers (closures capturing $ret/$err) then run LIFO,
// honouring the defer_ok/defer_err variants, before the function finally
// re-raises a surviving error or returns the (possibly modified) $ret.
const deferWrapperTemplate = `
var ($ret, $err)
$__defers := []
$__body := func() {}
try {
	$ret = $__body()
} catch $e {
	$err = $e
}
for $__i := len($__defers) - 1; $__i >= 0; $__i-- {
	$__d := $__defers[$__i]
	if $__d[1] == 1 && $err != nil { continue }
	if $__d[1] == 2 && $err == nil { continue }
	try { $__d[0]() } catch $de { $err = $de }
}
if $err != nil { throw $err }
return $ret
`

// parseGadSnippet parses a small gad source snippet into its top-level
// statements. It is used to build the defer desugaring from templates instead
// of constructing the AST by hand.
func parseGadSnippet(src string) ([]node.Stmt, error) {
	fileSet := source.NewFileSet()
	srcFile := fileSet.AppendFileData("<defer>", []byte(src))
	f, err := parser.NewParser(srcFile, nil).ParseFile()
	if err != nil {
		return nil, err
	}
	return f.Stmts, nil
}

// stmtsHaveDefer reports whether any statement (not descending into nested
// function literals) is a DeferStmt.
func stmtsHaveDefer(stmts []node.Stmt) bool {
	for _, s := range stmts {
		if stmtHasDefer(s) {
			return true
		}
	}
	return false
}

func stmtHasDefer(s node.Stmt) bool {
	switch s := s.(type) {
	case nil:
		return false
	case *node.DeferStmt:
		return s != nil
	case *node.BlockStmt:
		return s != nil && stmtsHaveDefer(s.Stmts)
	case *node.IfStmt:
		return s != nil && (stmtHasDefer(s.Init) || stmtHasDefer(s.Body) || stmtHasDefer(s.Else))
	case *node.ForStmt:
		return s != nil && (stmtHasDefer(s.Init) || stmtHasDefer(s.Body) || stmtHasDefer(s.Post))
	case *node.ForInStmt:
		return s != nil && (stmtHasDefer(s.Body) || stmtHasDefer(s.Else))
	case *node.TryStmt:
		if s == nil {
			return false
		}
		if stmtHasDefer(s.Body) {
			return true
		}
		if s.Catch != nil && stmtHasDefer(s.Catch.Body) {
			return true
		}
		if s.Finally != nil && stmtHasDefer(s.Finally.Body) {
			return true
		}
	}
	return false
}

// wrapDeferBody returns a new function body that desugars `defer` by wrapping
// the original body in the defer runner (see deferWrapperTemplate).
func (c *Compiler) wrapDeferBody(body *node.BlockStmt) (*node.BlockStmt, error) {
	stmts, err := parseGadSnippet(deferWrapperTemplate)
	if err != nil {
		return nil, fmt.Errorf("defer wrapper: %w", err)
	}

	var placed bool
	for _, s := range stmts {
		as, ok := s.(*node.AssignStmt)
		if !ok || len(as.LHS) != 1 || len(as.RHS) != 1 {
			continue
		}
		id, ok := as.LHS[0].(*node.IdentExpr)
		if !ok || id.Name != "$__body" {
			continue
		}
		fe, ok := as.RHS[0].(*node.FuncExpr)
		if !ok {
			continue
		}
		fe.Body = body
		placed = true
		break
	}
	if !placed {
		return nil, fmt.Errorf("defer wrapper: $__body slot not found")
	}
	return &node.BlockStmt{Stmts: stmts}, nil
}

// compileDeferStmt compiles a `defer`/`defer_ok`/`defer_err` statement to a
// registration on the enclosing function's $__defers list:
//
//	$__defers = append($__defers, [func() { <handler> }, <variant>])
//
// The handler closure captures the enclosing $ret/$err (and any other locals)
// so it can read and modify them when it runs at function exit.
func (c *Compiler) compileDeferStmt(nd *node.DeferStmt) error {
	var bodyStmts node.Stmts
	if nd.Body != nil {
		bodyStmts = nd.Body.Stmts
	} else {
		call := nd.Call
		if _, ok := call.(*node.CallExpr); !ok {
			// `defer handler` -> call the handler
			call = &node.CallExpr{Func: call}
		}
		bodyStmts = node.Stmts{&node.ExprStmt{Expr: call}}
	}

	src := fmt.Sprintf(`$__defers = append($__defers, [func() {}, %d])`, int(nd.Variant))
	stmts, err := parseGadSnippet(src)
	if err != nil {
		return c.errorf(nd, "defer: %v", err)
	}

	assign, ok := stmts[0].(*node.AssignStmt)
	if !ok {
		return c.errorf(nd, "defer: malformed registration")
	}
	fe, ok := deferHandlerFuncOf(assign)
	if !ok {
		return c.errorf(nd, "defer: handler slot not found")
	}
	fe.Body = &node.BlockStmt{Stmts: bodyStmts}

	if _, ok := c.symbolTable.Resolve("$__defers"); !ok {
		return c.errorf(nd, "defer is only allowed inside a function body")
	}
	return c.Compile(assign)
}

// deferHandlerFuncOf extracts the `func() {}` literal from the parsed
// registration `$__defers = append($__defers, [func() {}, N])`.
func deferHandlerFuncOf(assign *node.AssignStmt) (*node.FuncExpr, bool) {
	if len(assign.RHS) != 1 {
		return nil, false
	}
	call, ok := assign.RHS[0].(*node.CallExpr)
	if !ok || len(call.Args.Values) != 2 {
		return nil, false
	}
	arr, ok := call.Args.Values[1].(*node.ArrayExpr)
	if !ok || len(arr.Elements) != 2 {
		return nil, false
	}
	fe, ok := arr.Elements[0].(*node.FuncExpr)
	return fe, ok
}
