package gad

import "testing"

// compileSpec compiles src against a fresh spec and returns the spec so its
// Flags (set by the compiler) can be inspected.
func compileSpec(t *testing.T, src string) *ModuleSpec {
	t.Helper()
	builtins := NewBuiltins()
	st := NewSymbolTable(builtins.NameSet)
	spec := &ModuleSpec{ModuleInfo: ModuleInfo{Name: MainName}, Flags: ModuleMain}
	if _, err := CompileModule(st, spec, []byte(src), CompileOptions{}); err != nil {
		t.Fatalf("compile %q: %v", src, err)
	}
	return spec
}

// TestModuleFlags checks the bitmask helpers on ModuleSpec.
func TestModuleFlags(t *testing.T) {
	s := &ModuleSpec{}
	if s.IsMain() || s.IsRawArgv() {
		t.Fatal("zero spec should have no flags")
	}
	s.Flags |= ModuleMain
	if !s.IsMain() || s.IsRawArgv() {
		t.Fatalf("after ModuleMain: IsMain=%v IsRawArgv=%v", s.IsMain(), s.IsRawArgv())
	}
	s.Flags |= ModuleRawArgv
	if !s.IsMain() || !s.IsRawArgv() {
		t.Fatal("both flags should be set")
	}
	if !s.Flags.Has(ModuleMain|ModuleRawArgv) || (ModuleMain).Has(ModuleRawArgv) {
		t.Fatal("Has must require all bits")
	}
}

// TestModuleRawArgvDetection checks that a lone variadic `param (*argv)` sets
// ModuleRawArgv, while other param shapes do not.
func TestModuleRawArgvDetection(t *testing.T) {
	rawArgv := []string{
		"param (*argv)\nreturn argv",
		"param (*rest)\nreturn rest", // name is irrelevant
	}
	for _, src := range rawArgv {
		if s := compileSpec(t, src); !s.IsRawArgv() {
			t.Fatalf("expected ModuleRawArgv for %q", src)
		}
	}

	notRawArgv := []string{
		"return 1",                        // no params
		"param (a)\nreturn a",             // single non-variadic
		"param (a, *rest)\nreturn rest",   // a leading positional before the variadic
		"param (;name)\nreturn name",      // a named param
		"param (*argv; x=1)\nreturn argv", // variadic + a named param
	}
	for _, src := range notRawArgv {
		if s := compileSpec(t, src); s.IsRawArgv() {
			t.Fatalf("did not expect ModuleRawArgv for %q", src)
		}
	}
}

// TestIsMainConsistentWithOptimizer checks that `@main` (OpIsMain) folds to the
// module's real ModuleMain flag even with the optimizer on (regression: the
// optimizer built its module spec without copying the flags, so @main folded to
// false for a main module).
func TestIsMainConsistentWithOptimizer(t *testing.T) {
	run := func(flags ModuleFlags) Object {
		builtins := NewBuiltins()
		st := NewSymbolTable(builtins.NameSet)
		spec := &ModuleSpec{ModuleInfo: ModuleInfo{Name: MainName}, Flags: flags}
		opts := CompileOptions{CompilerOptions: CompilerOptions{
			OptimizeConst: true, OptimizeExpr: true, OptimizerMaxCycle: 100,
		}}
		cr, err := CompileModule(st, spec, []byte("return @main"), opts)
		if err != nil {
			t.Fatal(err)
		}
		ret, err := NewVM(builtins.Build(), cr.Bytecode).Run()
		if err != nil {
			t.Fatal(err)
		}
		return ret
	}
	if run(ModuleMain) != True {
		t.Fatal("@main should be true for a main module (optimizer on)")
	}
	if run(0) != False {
		t.Fatal("@main should be false for a non-main module")
	}
}

// TestModuleRawArgvIndependentOfMain checks a non-main module can carry
// ModuleRawArgv (the flag is derived from the params, not from ModuleMain).
func TestModuleRawArgvIndependentOfMain(t *testing.T) {
	builtins := NewBuiltins()
	st := NewSymbolTable(builtins.NameSet)
	spec := &ModuleSpec{ModuleInfo: ModuleInfo{Name: "mod"}} // not main
	if _, err := CompileModule(st, spec, []byte("param (*argv)\nreturn argv"), CompileOptions{}); err != nil {
		t.Fatal(err)
	}
	if spec.IsMain() {
		t.Fatal("spec must not be main")
	}
	if !spec.IsRawArgv() {
		t.Fatal("non-main module with param (*argv) should still be ModuleRawArgv")
	}
}
