// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad_test

import (
	"testing"

	gad "github.com/gad-lang/gad"
	"github.com/stretchr/testify/require"
)

// compileInclude compiles mainSrc with each entry of mods registered as a plain
// Gad source module (so `include ("name")` resolves it).
func compileInclude(mainSrc string, mods map[string]string) (*gad.CompileResult, *gad.Builtins, error) {
	builtins := gad.NewBuiltins()
	mm := gad.NewModuleMap()
	for name, src := range mods {
		mm.AddSourceModule(name, []byte(src))
	}
	st := gad.NewSymbolTable(builtins.NameSet)
	opts := gad.CompileOptions{}
	opts.ModuleMap = mm
	opts.ModuleFile = "main.gad"
	cr, err := gad.Compile(st, []byte(mainSrc), opts)
	return cr, builtins, err
}

func runInclude(t *testing.T, mainSrc string, mods map[string]string) gad.Object {
	t.Helper()
	cr, builtins, err := compileInclude(mainSrc, mods)
	require.NoError(t, err)
	ret, err := gad.NewVM(builtins.Build(), cr.Bytecode).SetRecover(true).RunOpts(&gad.RunOpts{})
	require.NoError(t, err)
	return ret
}

// TestIncludeInlineScope verifies `include` compiles the named file inline into
// the current scope: a variable it defines is visible after the include (unlike
// `import`, which isolates the module).
func TestIncludeInlineScope(t *testing.T) {
	ret := runInclude(t, `include ("name.gad"); return x + 1`,
		map[string]string{"name.gad": "x := 10"})
	require.Equal(t, "11", ret.ToString())
}

// TestIncludeInlineOrder proves the included statements are emitted in place: a
// variable declared before the include is visible to the included code, and the
// included code's effects are visible after it.
func TestIncludeInlineOrder(t *testing.T) {
	ret := runInclude(t, `a := 1; include ("add.gad"); return a`,
		map[string]string{"add.gad": "a = a + 41"})
	require.Equal(t, "42", ret.ToString())
}

// TestIncludeMultiple verifies the multi-file forms compile every file inline,
// in order.
func TestIncludeMultiple(t *testing.T) {
	mods := map[string]string{"a.gad": "a := 2", "b.gad": "b := 3"}

	ret := runInclude(t, `include ("a.gad", "b.gad"); return a * b`, mods)
	require.Equal(t, "6", ret.ToString())

	// Same files, each still compiled once per include site.
	ret = runInclude(t, `include ("a.gad", "b.gad"); return a + b`, mods)
	require.Equal(t, "5", ret.ToString())
}

// TestIncludeSourceName verifies `@file` reports the included source while its
// code runs, and restores to the module's file afterwards.
func TestIncludeSourceName(t *testing.T) {
	ret := runInclude(t, `include ("name.gad"); return [inner, @file]`,
		map[string]string{"name.gad": "inner := @file"})
	arr, ok := ret.(gad.Array)
	require.True(t, ok, "want array, got %T", ret)
	require.Equal(t, "name.gad", arr[0].ToString())
	// Outside the include, @file falls back to the module URL (empty for the
	// unnamed main module here).
	require.Equal(t, "", arr[1].ToString())
}

// TestIncludeFilesStack verifies `@files` exposes the source stack (the module
// at the base, then each active include, innermost last).
func TestIncludeFilesStack(t *testing.T) {
	ret := runInclude(t, `include ("outer.gad"); return stack`,
		map[string]string{
			"outer.gad": `include ("inner.gad")`,
			"inner.gad": `stack := @files`,
		})
	arr, ok := ret.(gad.Array)
	require.True(t, ok, "want array, got %T", ret)
	require.Len(t, arr, 3)
	require.Equal(t, "outer.gad", arr[1].ToString())
	require.Equal(t, "inner.gad", arr[2].ToString())
}

// TestIncludeMod verifies `@mod` yields the module object (the including
// module), even inside an included file.
func TestIncludeMod(t *testing.T) {
	ret := runInclude(t, `include ("name.gad"); return same`,
		map[string]string{"name.gad": "same := @mod == @mod"})
	require.Equal(t, "true", ret.ToString())
}

// TestIncludeNested verifies transitive includes compose (A includes B includes
// C), all sharing one scope.
func TestIncludeNested(t *testing.T) {
	ret := runInclude(t, `include ("a.gad"); return a + b + c`,
		map[string]string{
			"a.gad": `a := 1; include ("b.gad")`,
			"b.gad": `b := 2; include ("c.gad")`,
			"c.gad": `c := 3`,
		})
	require.Equal(t, "6", ret.ToString())
}

// TestIncludeCycle verifies a cyclic include is a compile error, not an infinite
// loop.
func TestIncludeCycle(t *testing.T) {
	_, _, err := compileInclude(`include ("self.gad")`,
		map[string]string{"self.gad": `include ("self.gad")`})
	require.Error(t, err)
	require.Contains(t, err.Error(), "include cycle")
}

// TestIncludeNotFound verifies an unknown include path is a compile error.
func TestIncludeNotFound(t *testing.T) {
	_, _, err := compileInclude(`include ("missing.gad")`, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}
