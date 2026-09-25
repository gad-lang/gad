package gad_test

import (
	"context"
	"testing"

	gad "github.com/gad-lang/gad"
	"github.com/stretchr/testify/require"
)

// TestInvokerImportInForkedVM guards the forked-VM module specs: a function
// that imports a module, run through an Invoker on a VM forked from an Eval (as
// `gad test` runs each test function), used to panic with "index out of range"
// because the fork got the bytecode's modules but not the VM's module specs.
func TestInvokerImportInForkedVM(t *testing.T) {
	builtins := gad.NewBuiltins().Build()
	mm := gad.NewModuleMap()
	mm.AddSourceModule("m", []byte(`export x = 7`))
	opts := gad.CompileOptions{CompilerOptions: gad.CompilerOptions{ModuleMap: mm}}
	eval := gad.NewEval(builtins, gad.NewSymbolTable(builtins.Builtins().NameSet), opts, &gad.RunOpts{})
	eval.VM.Builtins = builtins
	_, bc, err := eval.RunScript(context.Background(), []byte(`f := func() => import("m").x; return f`))
	require.NoError(t, err)
	require.Equal(t, 2, bc.NumModules)
	fn := eval.Locals[0]
	inv := gad.NewInvoker(eval.VM, fn)
	inv.Acquire()
	defer inv.Release()
	got, err := inv.Invoke(gad.Args{}, nil)
	require.NoError(t, err)
	require.Equal(t, gad.Int(7), got)
}
