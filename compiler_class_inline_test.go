package gad

import (
	"strings"
	"testing"

	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
)

// compileFile compiles src and returns what it parsed to, as compiled.
func compileFile(t *testing.T, src string) *CompileResult {
	t.Helper()
	res, err := Compile(NewSymbolTable(NewBuiltins().NameSet), []byte(src), CompileOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// classNames are the implied names of the anonymous classes of the file, by
// the field that holds them.
func classNames(f ast.Node) map[string]string {
	out := map[string]string{}
	node.Walk(f, func(n ast.Node) bool {
		if cf, ok := n.(*node.ClassFieldExpr); ok && cf.Name != nil && cf.Name.Ident != nil {
			for _, typ := range cf.Name.Type {
				if lit, ok := typ.Expr.(*node.TypeLitExpr); ok {
					out[cf.Name.Ident.Name] = lit.ImpliedName
				}
			}
		}
		return true
	})
	return out
}

// The compiler names the anonymous classes the fields are typed by: their path
// from the outermost class, its name first — `#N` for an anonymous one.
func TestCompileFieldClassNames(t *testing.T) {
	res := compileFile(t, "class P { a class { b class { c int } }; x class { y int } }")
	got := classNames(res.File)
	for field, want := range map[string]string{"a": "P.a", "b": "P.a.b", "x": "P.x"} {
		if got[field] != want {
			t.Errorf("%s: %q, want %q", field, got[field], want)
		}
	}

	res = compileFile(t, "q := class { a class { b int } }")
	if name := classNames(res.File)["a"]; !strings.HasPrefix(name, "#") || !strings.HasSuffix(name, ".a") {
		t.Errorf("below an anonymous class: %q", name)
	}

	// a mixin or a marker type is not a field's class, and is not named
	res = compileFile(t, "class P { a int }")
	if len(classNames(res.File)) != 0 {
		t.Errorf("no anonymous class: %v", classNames(res.File))
	}
}

// parsedClass is the class literal src, a class statement, parses to.
func parsedClass(t *testing.T, src string) *node.TypeLitExpr {
	t.Helper()
	f, err := parser.NewParserWithOptions(source.NewFileSet().AppendFileData("t.gad", []byte(src)), nil, nil).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	return &f.Stmts[0].(*node.TypeDeclStmt).TypeLitExpr
}

// `** EXPR` lowers to the define call's `spread` argument, the expressions in
// order, after the declared members.
func TestCompileClassSpreadLowering(t *testing.T) {
	lit := parsedClass(t, "class O { a = 1; ** one; ** two }")
	call, err := (&Compiler{}).classCallExpr(lit)
	if err != nil {
		t.Fatal(err)
	}
	code := node.Code(call)
	if !strings.Contains(code, "spread=[one, two]") {
		t.Errorf("the spread argument: %s", code)
	}
	if strings.Index(code, "fields=") > strings.Index(code, "spread=") {
		t.Errorf("the declared members come first: %s", code)
	}

	// only spreads: the class still gets its define handler
	lit = parsedClass(t, "class O { ** one; ** two }")
	call, err = (&Compiler{}).classCallExpr(lit)
	if err != nil {
		t.Fatal(err)
	}
	if code := node.Code(call); !strings.Contains(code, "spread=[one, two]") {
		t.Errorf("a class of spreads only: %s", code)
	}
}

// A class with `** EXPR` and anonymous field classes compiles to bytecode that
// runs, the spread evaluated where the class is declared.
func TestCompileClassSpreadEvaluatedAtDeclaration(t *testing.T) {
	res := compileFile(t, `n := 0
f := func() { n++; return {fields: {v: n}} }
class O { ** f() }
return [O().v, O().v, n]`)
	ret, err := NewVM(NewBuiltins().Build(), res.Bytecode).Run()
	if err != nil {
		t.Fatal(err)
	}
	if got := ret.(Array); len(got) != 3 || got[0] != Int(1) || got[1] != Int(1) || got[2] != Int(1) {
		t.Errorf("the spread runs once, when the class is declared: %v", got)
	}
}
