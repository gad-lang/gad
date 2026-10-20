package gad

import (
	"bytes"
	"io"
	"strconv"
	"strings"

	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/node"
)

// The inline checks (issue #22): `= EXPR` checks the value of the statement
// before it, `<<< text` / `<<<⏎ lines ⏎>>>` the output since the previous
// check. They are compiled only with CompilerOptions.Checks — what a test, an
// example or a doc fence runs with —; without it they compile to nothing.

// ErrCheck is the error of a check that fails.
var ErrCheck = &Error{Name: "CheckError"}

// checkValueFunc is `= EXPR`: got and want compared as str(…).
var checkValueFunc = &Function{
	FuncName: "check",
	Value: func(c Call) (Object, error) {
		got, err := checkStr(c.VM, c.Args.Get(0))
		if err != nil {
			return nil, err
		}
		want, err := checkStr(c.VM, c.Args.Get(1))
		if err != nil {
			return nil, err
		}
		if got != want {
			return nil, ErrCheck.NewErrorf("got %s, want %s", got, want)
		}
		return Nil, nil
	},
}

// checkStr is str(o), as the `str` builtin makes it.
func checkStr(vm *VM, o Object) (string, error) {
	s, err := vm.CallBuiltin(BuiltinStr, nil, o)
	if err != nil {
		return "", err
	}
	return s.ToString(), nil
}

// checkTee is the writer under a VM's stdout while checks run: what is
// written goes through, and is recorded for the output checks — all of it,
// last where the previous check ended, marks where the blocks a `<<<` follows
// started (a block's output is all it printed, whatever checks are in it).
type checkTee struct {
	w     io.Writer
	all   bytes.Buffer
	last  int
	marks []int
}

func (t *checkTee) Write(p []byte) (int, error) {
	t.all.Write(p)
	return t.w.Write(p)
}

// since is the output from offset from on; the next check starts after it.
func (t *checkTee) since(from int) string {
	b := t.all.Bytes()
	from = min(from, len(b))
	t.last = len(b)
	return string(b[from:])
}

// checkStartFunc records the output of the program from where it is called
// (the start of a file with output checks): the base writer of the VM's
// stdout wrapped in a checkTee, once.
var checkStartFunc = &Function{
	FuncName: "checkStart",
	Value: func(c Call) (Object, error) {
		if outputTee(c.VM) == nil {
			sw := c.VM.StdOut
			if len(sw.writers) == 0 {
				sw.writers = append(sw.writers, io.Discard)
			}
			sw.writers[0] = &checkTee{w: sw.writers[0]}
		}
		return Nil, nil
	},
}

// checkMarkFunc marks where a block a `<<<` follows starts.
var checkMarkFunc = &Function{
	FuncName: "checkMark",
	Value: func(c Call) (Object, error) {
		if t := outputTee(c.VM); t != nil {
			t.marks = append(t.marks, t.all.Len())
		}
		return Nil, nil
	},
}

func outputTee(vm *VM) *checkTee {
	if vm == nil || vm.StdOut == nil || len(vm.StdOut.writers) == 0 {
		return nil
	}
	t, _ := vm.StdOut.writers[0].(*checkTee)
	return t
}

// checkOutputFunc is `<<< text`: the output since the previous check must be
// text — both with each line's trailing whitespace and the blank lines at
// their edges dropped.
var checkOutputFunc = &Function{
	FuncName: "checkOutput",
	Value: func(c Call) (Object, error) {
		t := outputTee(c.VM)
		if t == nil {
			return nil, ErrCheck.NewError("output check: the output is not recorded")
		}
		from := t.last
		if block := c.Args.Length() > 1 && !c.Args.Get(1).IsFalsy(); block && len(t.marks) > 0 {
			// `{ … }` then `<<<`: all the block printed
			from = t.marks[len(t.marks)-1]
			t.marks = t.marks[:len(t.marks)-1]
		}
		got := normalizeCheckOutput(t.since(from))
		want := normalizeCheckOutput(c.Args.Get(0).ToString())
		if got != want {
			return nil, ErrCheck.NewErrorf("output: got %s, want %s", strconv.Quote(got), strconv.Quote(want))
		}
		return Nil, nil
	},
}

func normalizeCheckOutput(s string) string {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t")
	}
	for len(lines) > 0 && lines[0] == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

// hasOutputCheck reports whether stmts hold an output check.
func hasOutputCheck(stmts []node.Stmt) (found bool) {
	for _, s := range stmts {
		node.Walk(s, func(n ast.Node) bool {
			if _, ok := n.(*node.OutputCheckStmt); ok {
				found = true
			}
			return !found
		})
		if found {
			return
		}
	}
	return
}

// emitCheckCall emits fn(args…), its value dropped, at nd.
func (c *Compiler) emitCheckCall(nd ast.Node, fn *Function, args ...func() error) error {
	c.emit(nd, OpConstant, c.addConstant(fn))
	for _, a := range args {
		if err := a(); err != nil {
			return err
		}
	}
	c.emit(nd, OpCall, len(args), 0)
	c.emit(nd, OpPop)
	return nil
}

// compileValueCheck compiles `= EXPR` checking got — the statement before it,
// compiled by got.
func (c *Compiler) compileValueCheck(vc *node.ValueCheckStmt, got func() error) error {
	return c.emitCheckCall(vc, checkValueFunc, got, func() error { return c.Compile(vc.Expr) })
}

// checkedValue is the expression whose value is that of s, for a `=` after it:
// an assignment or a definition of one name (its name, read again); nil for
// any other statement.
func checkedValue(s node.Stmt) node.Expr {
	switch t := s.(type) {
	case *node.AssignStmt:
		if len(t.LHS) == 1 {
			return t.LHS[0]
		}
	case *node.DeclStmt:
		if g, ok := t.Decl.(*node.GenDecl); ok && len(g.Specs) == 1 {
			if vs, ok := g.Specs[0].(*node.ValueSpec); ok && len(vs.Idents) == 1 {
				return vs.Idents[0]
			}
		}
	}
	return nil
}

// compileOutputCheck compiles `<<< text`: of stdout, or of what was rendered
// into its Target.
func (c *Compiler) compileOutputCheck(oc *node.OutputCheckStmt) error {
	return c.compileOutputCheckOf(oc, false)
}

// compileOutputCheckOf compiles `<<< text`; of a block: all the block printed
// (from its mark, compileBlockOutputCheck).
func (c *Compiler) compileOutputCheckOf(oc *node.OutputCheckStmt, block bool) error {
	if !c.opts.Checks {
		return nil
	}
	text := func() error {
		c.emit(oc, OpConstant, c.addConstant(Str(oc.Text)))
		return nil
	}
	if oc.Target != nil {
		return c.emitCheckCall(oc, checkRenderedFunc, func() error { return c.Compile(oc.Target) }, text)
	}
	if block {
		return c.emitCheckCall(oc, checkOutputFunc, text, func() error {
			c.emit(oc, OpTrue)
			return nil
		})
	}
	return c.emitCheckCall(oc, checkOutputFunc, text)
}

// compileBlockOutputCheck compiles `{ … }` followed by `<<<`: the output check
// is of all the block printed.
func (c *Compiler) compileBlockOutputCheck(b node.Stmt, oc *node.OutputCheckStmt) error {
	if err := c.emitCheckCall(b, checkMarkFunc); err != nil {
		return err
	}
	if err := c.Compile(b); err != nil {
		return err
	}
	return c.compileOutputCheckOf(oc, true)
}

// RenderedChecker is a render tree an output check reads (gadx's elements):
// CheckRendered is the rendering of what was added to it since its previous
// call.
type RenderedChecker interface {
	CheckRendered(vm *VM) (string, error)
}

// checkRenderedFunc is `<<< text` of a render tree: what was rendered into it
// since the previous check must be text — the lines of text joined, each
// without its indentation (a rendering has no line breaks of its own).
var checkRenderedFunc = &Function{
	FuncName: "checkRendered",
	Value: func(c Call) (Object, error) {
		r, ok := c.Args.Get(0).(RenderedChecker)
		if !ok {
			return nil, ErrCheck.NewErrorf("output check: %s is not a render tree", c.Args.Get(0).Type().Name())
		}
		got, err := r.CheckRendered(c.VM)
		if err != nil {
			return nil, err
		}
		var b strings.Builder
		for _, l := range strings.Split(c.Args.Get(1).ToString(), "\n") {
			b.WriteString(strings.TrimSpace(l))
		}
		got, want := strings.TrimSpace(got), b.String()
		if got != want {
			return nil, ErrCheck.NewErrorf("rendered: got %s, want %s", strconv.Quote(got), strconv.Quote(want))
		}
		return Nil, nil
	},
}

// compileExample compiles `example [NAME] { … }`: in a scope that stays open
// to the end of the statements around it (compileStmtList closes it) — what
// comes after it, a later example included, sees its definitions, and a later
// example may declare a name again (its own scope, nested in this one). A
// `global` in it is of the module, as at the top level.
func (c *Compiler) compileExample(ex *node.ExampleStmt) error {
	c.symbolTable = c.symbolTable.Fork(true)
	c.symbolTable.example = true
	c.exampleScopes++
	return c.compileStmts(ex.Body.Stmts...)
}
