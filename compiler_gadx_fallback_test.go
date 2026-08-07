// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCallManySpreadsWithGadxFallback is a regression test: a call with a
// non-trailing positional spread once recursed forever (stack overflow) when the
// Gadx compile fallback was installed — the form CompileFile installs by default
// for every .gad file run by the CLI. It must compile and run correctly,
// including the arrow-body variadic/named form.
func TestCallManySpreadsWithGadxFallback(t *testing.T) {
	run := func(src string) Object {
		t.Helper()
		builtins := NewBuiltins()
		st := NewSymbolTable(builtins.NameSet)
		opts := CompileOptions{}
		opts.FallbackFunc = gadxCompileFallback
		cr, err := Compile(st, []byte(src), opts)
		require.NoError(t, err)
		ret, err := NewVM(builtins.Build(), cr.Bytecode).Run(nil)
		require.NoError(t, err)
		return ret
	}

	require.Equal(t, Array{Int(1), Int(2), Int(9)},
		run(`f := func(*a) { return a }; return f(*[1, 2], 9)`))
	require.Equal(t, Array{Int(1), Int(2), Int(3), Int(4), Int(5), Int(6)},
		run(`f := func(*a) { return a }; return f(1, *[2, 3], 4, *[5, 6])`))
	require.Equal(t, Array{Int(1), Int(2), Int(3)},
		run(`collect := func(*args; **kw) => [args, dict(kw)]; return collect(*[1, 2], 3)[0]`))
}
