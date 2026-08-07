// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad_test

import (
	"testing"

	gad "github.com/gad-lang/gad"
	"github.com/gad-lang/gad/gadx"
	"github.com/stretchr/testify/require"
)

// TestSourceKindStringExt covers the canonical dialect name and extension.
func TestSourceKindStringExt(t *testing.T) {
	cases := []struct {
		kind     gad.SourceKind
		str, ext string
	}{
		{gad.SourceKindGad, "GAD", ".gad"},
		{gad.SourceKindGadt, "GADT", ".gadt"},
		{gad.SourceKindGadx, "GADX", ".gadx"},
	}
	for _, c := range cases {
		require.Equal(t, c.str, c.kind.String())
		require.Equal(t, c.ext, c.kind.Ext())
	}
	require.Equal(t, gad.SourceKindGad, gad.SourceKindForExt("m.gad"))
	require.Equal(t, gad.SourceKindGadt, gad.SourceKindForExt("m.gadt"))
	require.Equal(t, gad.SourceKindGadx, gad.SourceKindForExt("m.gadx"))
	require.Equal(t, gad.SourceKindGad, gad.SourceKindForExt("m.txt"))
}

// compileImport compiles a main module that imports "mod", with mod registered
// as a SourceCode of the given kind. gadxBuiltins registers the gadx namespace.
func compileImport(mainSrc, modSrc string, kind gad.SourceKind, gadxBuiltins bool) (*gad.CompileResult, *gad.Builtins, error) {
	builtins := gad.NewBuiltins()
	if gadxBuiltins {
		builtins = gadx.AppendBuiltins(builtins)
	}
	mm := gad.NewModuleMap()
	mm.AddSourceModuleKind("mod", []byte(modSrc), kind)
	st := gad.NewSymbolTable(builtins.NameSet)
	opts := gad.CompileOptions{}
	opts.ModuleMap = mm
	cr, err := gad.Compile(st, []byte(mainSrc), opts)
	return cr, builtins, err
}

func runImport(t *testing.T, mainSrc, modSrc string, kind gad.SourceKind, gadxBuiltins bool) gad.Object {
	t.Helper()
	cr, builtins, err := compileImport(mainSrc, modSrc, kind, gadxBuiltins)
	require.NoError(t, err)
	ret, err := gad.NewVM(builtins.Build(), cr.Bytecode).SetRecover(true).RunOpts(&gad.RunOpts{})
	require.NoError(t, err)
	return ret
}

// TestSourceCodeImportDialects proves the imported module's SourceCode.Kind
// selects the front-end: a .gadt module is parsed as a mixed template and a
// .gadx module through the Gadx front-end — dialects that do NOT parse as plain
// Gad, so a wrong kind fails.
func TestSourceCodeImportDialects(t *testing.T) {
	// A .gadt template whose code island exports a value: importing runs it.
	answer := runImport(t, `m := import("mod"); return m.answer`,
		"{% export answer = 6 * 7 %}", gad.SourceKindGadt, false)
	require.Equal(t, "42", answer.ToString())

	// The same template bytes fail when (wrongly) compiled as plain Gad.
	_, _, err := compileImport(`import("mod"); return 1`,
		"{% export answer = 6 * 7 %}", gad.SourceKindGad, false)
	require.Error(t, err, "template source must not parse as plain Gad")

	// A .gadx module uses indentation/tag syntax: it compiles through the Gadx
	// front-end (with the gadx builtins) but not as plain Gad.
	_, _, err = compileImport(`import("mod"); return 1`, "span Hi", gad.SourceKindGadx, true)
	require.NoError(t, err, "gadx source should compile under SourceKindGadx")

	_, _, err = compileImport(`import("mod"); return 1`, "span Hi", gad.SourceKindGad, false)
	require.Error(t, err, "gadx source must not parse as plain Gad")
}

// TestSourceCodePlainByteBackCompat verifies AddSourceModule (a bare []byte
// source, compiled as SourceKindGad) still works.
func TestSourceCodePlainByteBackCompat(t *testing.T) {
	builtins := gad.NewBuiltins()
	mm := gad.NewModuleMap()
	mm.AddSourceModule("mod", []byte(`export v = 6 * 7`))
	st := gad.NewSymbolTable(builtins.NameSet)
	opts := gad.CompileOptions{}
	opts.ModuleMap = mm
	cr, err := gad.Compile(st, []byte(`m := import("mod"); return m.v`), opts)
	require.NoError(t, err)
	ret, err := gad.NewVM(builtins.Build(), cr.Bytecode).SetRecover(true).RunOpts(&gad.RunOpts{})
	require.NoError(t, err)
	require.Equal(t, "42", ret.ToString())
}
