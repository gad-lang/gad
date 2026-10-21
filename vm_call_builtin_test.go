package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// vm.CallBuiltin dispatches as a script's call does: an overload a script
// registered on the builtin is the one called.
func TestVMCallBuiltin(t *testing.T) {
	res, err := Compile(NewSymbolTable(NewBuiltins().NameSet), []byte(`
class Box { n = 0 }
met len(b Box) => b.n
return Box(; n=7)`), CompileOptions{})
	if err != nil {
		t.Fatal(err)
	}
	vm := NewVM(NewBuiltins().Build(), res.Bytecode)
	box, err := vm.Run()
	if err != nil {
		t.Fatal(err)
	}
	if v, err := vm.CallBuiltin(BuiltinLen, nil, Array{Int(1), Int(2)}); err != nil || v != Int(2) {
		t.Fatalf("len([1, 2]) = %v, %v", v, err)
	}
	if v, err := vm.CallBuiltin(BuiltinLen, nil, box); err != nil || v != Int(7) {
		t.Fatalf("len(box) = %v, %v; want the overload's 7", v, err)
	}
}
