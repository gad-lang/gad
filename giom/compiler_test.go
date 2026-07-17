package giom

import (
	"bytes"
	"testing"

	"github.com/gad-lang/gad"
)

// TestCompilerCompile verifies that NewCompiler(...).Compile() produces runnable
// bytecode for multiple inputs. Each input is compiled with its own symbol table
// (as the render engine does per template): the tree model binds a root `tag` at
// the module top level, so a symbol table is not shared across independent
// compiles.
func TestCompilerCompile(t *testing.T) {
	builtins := AppendBuiltins(gad.NewBuiltins())
	opts := gad.CompileOptions{CompilerOptions: gad.CompilerOptions{
		FallbackFunc: CompileFallback,
		ModuleMap:    testModuleMap(),
	}}

	run := func(src string) string {
		t.Helper()
		st := gad.NewSymbolTable(builtins.NameSet)
		_, bc, err := NewCompiler(st, opts).Compile([]byte(src))
		if err != nil {
			t.Fatalf("compile: %v\nsrc: %s", err, src)
		}
		var buf bytes.Buffer
		vm := gad.NewVM(builtins.Build(), bc)
		ret, err := vm.RunOpts(&gad.RunOpts{StdOut: &buf, Globals: gad.Dict{}})
		if err != nil {
			t.Fatalf("run: %v", err)
		}
		if el, ok := ret.(Element); ok {
			if _, err := el.WriteTo(vm, &buf); err != nil {
				t.Fatalf("write: %v", err)
			}
		}
		return buf.String()
	}

	if got := run("@main\n    p one\n"); got != "<p>one</p>" {
		t.Fatalf("first compile: got %q, want %q", got, "<p>one</p>")
	}
	// Reuse the same Compiler for a second input.
	if got := run("@main\n    span two\n"); got != "<span>two</span>" {
		t.Fatalf("second compile: got %q, want %q", got, "<span>two</span>")
	}
}

// TestCompileDelegatesToCompiler verifies the package-level Compile is
// equivalent to NewCompiler(...).Compile().
func TestCompileDelegatesToCompiler(t *testing.T) {
	src := "@main\n    b hi\n"
	builtins := AppendBuiltins(gad.NewBuiltins())
	newOpts := func() gad.CompileOptions {
		return gad.CompileOptions{CompilerOptions: gad.CompilerOptions{
			FallbackFunc: CompileFallback,
			ModuleMap:    testModuleMap(),
		}}
	}
	newSt := func() *gad.SymbolTable {
		return gad.NewSymbolTable(builtins.NameSet)
	}

	_, bcFunc, err := Compile(newSt(), []byte(src), newOpts())
	if err != nil {
		t.Fatalf("package Compile: %v", err)
	}
	_, bcMethod, err := NewCompiler(newSt(), newOpts()).Compile([]byte(src))
	if err != nil {
		t.Fatalf("Compiler.Compile: %v", err)
	}

	out := func(bc *gad.Bytecode) string {
		var buf bytes.Buffer
		vm := gad.NewVM(builtins.Build(), bc)
		ret, err := vm.RunOpts(&gad.RunOpts{StdOut: &buf, Globals: gad.Dict{}})
		if err != nil {
			t.Fatalf("run: %v", err)
		}
		if el, ok := ret.(Element); ok {
			if _, err := el.WriteTo(vm, &buf); err != nil {
				t.Fatalf("write: %v", err)
			}
		}
		return buf.String()
	}
	if a, b := out(bcFunc), out(bcMethod); a != b {
		t.Fatalf("outputs differ: package=%q method=%q", a, b)
	}
}
