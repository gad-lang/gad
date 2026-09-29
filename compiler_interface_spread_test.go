package gad

import (
	"strings"
	"testing"
)

// An interface with `**Expr` compiles to InterfaceSpread(<the interface
// declared without them>, EXPR…), run where it is declared; one written as a
// type takes none; a header in a spread may name the interface `@self`.
func TestCompileInterfaceSpread(t *testing.T) {
	res := compileFile(t, "extra := {fields: {b: int}}\ninterface I { a int; **extra }\nreturn I")
	ret, err := NewVM(NewBuiltins().Build(), res.Bytecode).Run()
	if err != nil {
		t.Fatal(err)
	}
	iface := ret.(*Interface)
	if len(iface.Fields) != 2 || iface.Fields[0].Name != "a" || iface.Fields[1].Name != "b" {
		t.Errorf("fields: %v", iface.String())
	}

	// written as the type of an interface's field: a constant, no spread
	_, err = Compile(NewSymbolTable(NewBuiltins().NameSet),
		[]byte("extra := {}\ninterface O { x interface { a int; **extra } }"), CompileOptions{})
	if err == nil || !strings.Contains(err.Error(), "**Expr needs an interface declared") {
		t.Errorf("a spread in an interface written as a type: %v", err)
	}

	// `@self` in a spread's header is the interface
	res = compileFile(t, `f := func(x int, o) => 1
interface I { **{funcs: {f: (; fn=f, headers=[<(x int, @self)>])}} }
return I`)
	ret, err = NewVM(NewBuiltins().Build(), res.Bytecode).Run()
	if err != nil {
		t.Fatal(err)
	}
	cf := ret.(*Interface).ContextFuncs
	if len(cf) != 1 || len(cf[0].Headers) != 1 || !cf[0].Headers[0].Params[1].(*TypedIdent).Self {
		t.Errorf("the @self param: %+v", cf)
	}

	// `@self` outside an interface's spread stays unknown
	_, err = Compile(NewSymbolTable(NewBuiltins().NameSet), []byte("h := <(x int, @self)>"), CompileOptions{})
	if err == nil {
		t.Error("@self outside an interface: want an error")
	}
}
