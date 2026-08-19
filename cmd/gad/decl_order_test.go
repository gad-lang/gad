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

// TestDeclShortVar checks a lone `var name = value` collapses to `name := value`
// (any standalone var), while valueless/const/global are left alone.
func TestDeclShortVar(t *testing.T) {
	require.Equal(t, "x := 1\n", fmtGad(t, "var x = 1\n"))
	require.Equal(t, "b := 3\n", fmtGad(t, "var (b = 3)\n"))
	require.Contains(t, fmtGad(t, "var x\n"), "var x")
	require.Contains(t, fmtGad(t, "const x = 1\n"), "const x = 1")

	// Semantics unchanged, and idempotent.
	src := "var x = 41\nx + 1\n"
	out := fmtGad(t, src)
	require.Equal(t, runGad(t, src), runGad(t, out))
	require.Equal(t, out, fmtGad(t, out))
}

// TestDeclOrderIdempotent checks the reordering is a fixed point.
func TestDeclOrderIdempotent(t *testing.T) {
	src := "var (z = 3, m = 1, aa = 2, k)\n"
	out := fmtGad(t, src)
	require.Equal(t, out, fmtGad(t, out))
	require.Contains(t, out, "var (k, aa = 2, m = 1, z = 3)")
}

// TestDeclMerge checks adjacent same-kind declarations merge into one paren
// group, that a blank line or a floating statement breaks the run, and that
// merging preserves semantics (a later item may reference an earlier one).
func TestDeclMerge(t *testing.T) {
	require.Contains(t, fmtGad(t, "a := 2\nb := 3\n"), "var (a = 2, b = 3)")
	require.Contains(t, fmtGad(t, "var x = 1\ny := 2\n"), "var (x = 1, y = 2)")

	require.NotContains(t, fmtGad(t, "a := 2\n\nb := 3\n"), "var (")           // blank line
	require.NotContains(t, fmtGad(t, "a := 2\nprintln(1)\nb := 3\n"), "var (") // interrupted

	// Semantics: `b` references the earlier `a`; the merged/reordered form agrees.
	src := "a := 1\nb := a\n[a, b]\n"
	require.Equal(t, "[1, 1]", runGad(t, src))
	require.Equal(t, runGad(t, src), runGad(t, fmtGad(t, src)))
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
