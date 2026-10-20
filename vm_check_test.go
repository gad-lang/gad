package gad_test

import (
	"bytes"
	"strings"
	"testing"

	. "github.com/gad-lang/gad"
	fmtmod "github.com/gad-lang/gad/stdlib/fmt"
)

func checksOn() *VMTestOpts {
	return newOpts().CompileOptions(func(o *CompileOptions) { o.Checks = true })
}

// The inline checks (issue #22): `= EXPR`, `<<< text`, `<<<⏎ … ⏎>>>`.
func TestVMInlineChecks(t *testing.T) {
	// a value: of an expression, an assignment, a definition, compared as str
	testExpectRun(t, "x := 10\n= 10\nx + 1\n= 11\nconst y = [1, 2]\n= [1, 2]\nreturn x", checksOn(), Int(10))
	testExpectRun(t, "1u\n= 1\nreturn 0", checksOn(), Int(0)) // str: 1u and 1 print alike
	expectErrHas(t, "x := 10\n= 11", checksOn(), "CheckError: got 10, want 11")
	expectErrHas(t, "if true {}\n= 1", checksOn().CompilerError(), "`=` after a statement with no value")
	expectErrHas(t, "= 1", checksOn().CompilerError(), "`=` after a statement with no value")
	// the expression of an ExprStmt is evaluated once
	testExpectRun(t, "n := 0\nf := func() { n++; return n }\nf()\n= 1\nreturn n", checksOn(), Int(1))

	// output since the previous check
	testExpectRun(t, "println(\"a\\nb\")\n<<<\na\nb\n>>>\nprintln(\"c\");<<< c\nprint(\"d\\te\")\n<<< d\\te\nreturn 1", checksOn(), Int(1))
	expectErrHas(t, "println(\"a\")\n<<< b", checksOn(), `CheckError: output: got "a", want "b"`)
	// in a function: checked too
	expectErrHas(t, "func f() {\n\t2 * 2\n\t= 5\n}\nf()", checksOn(), "got 4, want 5")
	// in a function, indented
	testExpectRun(t, "func f() {\n\tprintln(1)\n\t<<<\n\t1\n\t>>>\n\treturn 2\n}\nreturn f()", checksOn(), Int(2))

	// the return first, then what it printed — builtin and fmt's print
	testExpectRun(t, "n := print(\"ab\")\n= 2\n<<< ab\nreturn n", checksOn(), Int(2))
	testExpectRun(t, "fmt := import(\"fmt\")\nfmt.print(\"a\", 1)\n= 2\n<<< a1\nfmt.println(\"hi\", 1)\n= 5\n<<<\nhi 1\n>>>\nreturn 0", checksOn().Module("fmt", fmtmod.ModuleInit), Int(0))
	expectErrHas(t, "fmt := import(\"fmt\")\nfmt.printf(\"%d!\", 5)\n= 2\n<<< 6!", checksOn().Module("fmt", fmtmod.ModuleInit), `CheckError: output: got "5!", want "6!"`)

	// `<<<` after a block: all it printed, whatever checks are in it (a value
	// check does not take the output: the return, then the output)
	testExpectRun(t, "{\n\tprintln(\"a\")\n\t<<< a\n\tx := 1\n\t= 1\n\tprintln(\"b\")\n\t<<< b\n\tprintln(\"c\")\n}\n<<<\na\nb\nc\n>>>\nprintln(\"d\")\n<<< d\nreturn 0", checksOn(), Int(0))
	expectErrHas(t, "{\n\tprintln(\"a\")\n}\n<<< b", checksOn(), `CheckError: output: got "a", want "b"`)

	// off: nothing compiled, nothing checked
	testExpectRun(t, "x := 10\n= 11\nprintln(\"a\")\n<<< b\nreturn x", nil, Int(10))
	// the operators unchanged
	testExpectRun(t, "x := 1; y := x\nreturn [x, y]", checksOn(), Array{Int(1), Int(1)})
}

// With checks off the bytecode is the program's without them.
func TestVMInlineChecksOffSameBytecode(t *testing.T) {
	bi := NewBuiltins()
	compile := func(src string) *Bytecode {
		res, err := Compile(NewSymbolTable(bi.NameSet), []byte(src), CompileOptions{})
		if err != nil {
			t.Fatal(err)
		}
		return res.Bytecode
	}
	a := compile("x := 10\nprintln(x)\nreturn x")
	b := compile("x := 10\n= 10\nprintln(x)\n<<< 10\nreturn x")
	var ab, bb bytes.Buffer
	a.Fprint(nil, &ab)
	b.Fprint(nil, &bb)
	// the same instructions and constants (the source map: other positions)
	strip := func(s string) string { return s[:strings.Index(s, "SourceMap:")] }
	if strip(ab.String()) != strip(bb.String()) {
		t.Fatalf("bytecode differs:\n%s\n---\n%s", ab.String(), bb.String())
	}
}

// gadt: the checks inside a code block — `=` and `<<<` where a statement
// starts there, not right after `{%` (`{%= … %}` emits).
func TestVMInlineChecksGadt(t *testing.T) {
	testExpectRun(t, "a{%\nx := 10\n= 10\n%}b{%\n<<<\nab\n>>>\nreturn x %}", checksOn().Mixed(), Int(10))
	expectErrHas(t, "a{%\n<<<\nb\n>>>\n%}", checksOn().Mixed(), `CheckError: output: got "a", want "b"`)
}

// fmt.print/printf/println write to the VM's stdout (a redirection — obstart, a
// test's output check — sees them), not to the process's.
func TestVMFmtPrintToVMStdout(t *testing.T) {
	testExpectRun(t, `fmt := import("fmt"); obstart(); fmt.print("a", 1); fmt.printf("%d!", 5); fmt.println("x"); return str(obend())`,
		newOpts().Module("fmt", fmtmod.ModuleInit), Str("a15!x\n"))
}

// `example [NAME] { … }`: its code of the scope around it — a later example and
// the code after it see its definitions —, its own scope (a name declared
// again), `<<<` after it all it printed, `=` after it an error.
func TestVMExample(t *testing.T) {
	testExpectRun(t, "base := 10\nexample a {\n\ty := base + 1\n\t= 11\n\tprintln(\"a\")\n\t<<< a\n\tprintln(\"b\")\n}\n<<<\na\nb\n>>>\n"+
		"example b {\n\ty + 1\n\t= 12\n\ty := 0\n\t= 0\n}\nreturn y", checksOn(), Int(0))
	expectErrHas(t, "example {\n\t1\n}\n= 1", checksOn().CompilerError(), "`=` after a statement with no value")
	// a global in an example is of the module
	testExpectRun(t, "example {\n\tx := 1\n}\nexample {\n\tglobal (g = 5)\n\tg\n\t= 5\n}\nreturn g", checksOn(), Int(5))
	// off: an example runs as code, its checks nothing
	testExpectRun(t, "example a {\n\tx := 1\n\t= 2\n}\nreturn x", nil, Int(1))
}
