package gad

import (
	"testing"
)

// TestVMCallNoFork verifies that invoking a compiled function through vm.Call
// while the VM is running its loop runs it on the SAME VM (a sub-run) rather
// than forking a child: the pool acquires no child VM. It also checks the call
// returns the correct result and that an external (not-running) vm.Call still
// works via the fork path.
func TestVMCallNoFork(t *testing.T) {
	bi := NewBuiltins()
	// A builtin that invokes its first argument via vm.Call with two ints.
	bi.Set("callit", &Function{FuncName: "callit", Value: func(c Call) (Object, error) {
		return c.VM.Call(c.Args.Get(0), Args{Array{Int(10), Int(5)}}, nil)
	}})

	st := NewSymbolTable(bi.NameSet)
	res, err := Compile(st, []byte(`add := func(a, b) { return a + b }; return callit(add)`), CompileOptions{})
	if err != nil {
		t.Fatal(err)
	}

	vm := NewVM(bi.Build(), res.Bytecode)
	ret, err := vm.Run()
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if ret.ToString() != "15" {
		t.Fatalf("result = %s, want 15", ret.ToString())
	}
	// The in-loop vm.Call ran on the same VM: no child VM was acquired.
	if n := len(vm.pool.vms); n != 0 {
		t.Fatalf("pool acquired %d child VM(s); want 0 (same-VM sub-run)", n)
	}
}

// TestVMCallExternal verifies vm.Call works from an external (not-running) VM,
// falling back to the fork path for a compiled function.
func TestVMCallExternal(t *testing.T) {
	st := NewSymbolTable(NewBuiltins().NameSet)
	res, err := Compile(st, []byte(`return func(a, b) { return a * b }`), CompileOptions{})
	if err != nil {
		t.Fatal(err)
	}
	vm := NewVM(NewBuiltins().Build(), res.Bytecode)
	fn, err := vm.Run()
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	// vm is no longer running here; Call forks a child for the compiled function.
	got, err := vm.Call(fn, Args{Array{Int(6), Int(7)}}, nil)
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if got.ToString() != "42" {
		t.Fatalf("result = %s, want 42", got.ToString())
	}
}
