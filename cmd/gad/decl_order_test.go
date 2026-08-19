//go:build !js
// +build !js

package main

import (
	"context"
	"strings"
	"testing"

	gad "github.com/gad-lang/gad"
	"github.com/stretchr/testify/require"
)

// fmtGad formats src with the default (column-aware) formatter.
func fmtGad(t *testing.T, src string) string {
	t.Helper()
	o := &fmtOptions{codeFlags: fmtFormatFlag()}
	out, err := o.formatSource("x.gad", []byte(src), false)
	require.NoError(t, err, src)
	return out
}

// runGad compiles and runs src, returning the last value's string form.
func runGad(t *testing.T, src string) string {
	t.Helper()
	ev := gad.NewEval(nil, nil, gad.CompileOptions{})
	ret, _, err := ev.RunScript(context.Background(), []byte(src))
	require.NoError(t, err, src)
	require.NotNil(t, ret, src)
	return ret.ToString()
}

// TestDeclOrderGrouping checks var items are grouped (valueless, plain,
// expression, computed, closure, func) and sorted by name within a group.
func TestDeclOrderGrouping(t *testing.T) {
	src := "var (f = func() {\nreturn 1\n}, e = a2 + 2, p = 5, x, cv = (= a2 + 1), c = () => 9, q = 1)\n"
	out := fmtGad(t, src)

	order := []string{"x", "p = 5", "q = 1", "e = ", "cv = ", "c = ", "f = "}
	prev := -1
	for _, tok := range order {
		i := strings.Index(out, tok)
		require.GreaterOrEqualf(t, i, 0, "%q not found in:\n%s", tok, out)
		require.Greaterf(t, i, prev, "%q out of order in:\n%s", tok, out)
		prev = i
	}
}

// TestDeclOrderIdempotent checks the reordering is a fixed point.
func TestDeclOrderIdempotent(t *testing.T) {
	src := "var (z = 3, m = 1, aa = 2, k)\n"
	out := fmtGad(t, src)
	require.Equal(t, out, fmtGad(t, out))
	require.Contains(t, out, "var (k, aa = 2, m = 1, z = 3)")
}

// TestDeclOrderPreservesResolution runs the original and the reordered program
// and asserts identical results: `b`/`c` must see the OUTER `a`, `d` the group
// `a`, regardless of how the group is laid out.
func TestDeclOrderPreservesResolution(t *testing.T) {
	src := "a := 1\n" +
		"(func() {\n" +
		"\tvar (x, b = a, c = () => a, a = 2, d = a + 2)\n" +
		"\treturn [b, c(), d]\n" +
		"})()\n"
	out := fmtGad(t, src)

	got := runGad(t, src)
	require.Equal(t, "[1, 1, 4]", got, "baseline semantics")
	require.Equal(t, got, runGad(t, out), "formatted output changed semantics:\n%s", out)
}
