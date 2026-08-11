// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package main

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestScaffoldGoModule checks that `gad gomod init` writes a parseable module.go
// and a samples.gad, refuses to overwrite, and validates the module name.
func TestScaffoldGoModule(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "greet")
	require.NoError(t, scaffoldGoModule("greet", dir))

	mod, err := os.ReadFile(filepath.Join(dir, "module.go"))
	require.NoError(t, err)
	require.Contains(t, string(mod), "package greet")
	require.Contains(t, string(mod), `const ModuleName = "greet"`)
	require.Contains(t, string(mod), "gad:samples [module,auto] greet/samples.gad")
	// The generated Go source must parse.
	_, err = parser.ParseFile(token.NewFileSet(), "module.go", mod, parser.AllErrors)
	require.NoError(t, err)

	sam, err := os.ReadFile(filepath.Join(dir, "samples.gad"))
	require.NoError(t, err)
	require.Contains(t, string(sam), "//snippet hello")
	require.Contains(t, string(sam), `//= "hello Gad"`)

	// A second init over the same directory must not clobber existing files.
	require.Error(t, scaffoldGoModule("greet", dir))
}

// TestGomodInitNameValidation checks the module-name rule.
func TestGomodInitNameValidation(t *testing.T) {
	for _, ok := range []string{"greet", "my_mod", "http2"} {
		require.True(t, reModuleName.MatchString(ok), ok)
	}
	for _, bad := range []string{"Greet", "my-mod", "2mod", "", "a.b"} {
		require.False(t, reModuleName.MatchString(bad), bad)
	}
}
