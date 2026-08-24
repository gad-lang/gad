package langsym_test

import (
	"strings"
	"testing"

	"github.com/gad-lang/gad/langsym"
	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/source"
	"github.com/stretchr/testify/require"
)

func parse(t *testing.T, src string) (*parser.File, *source.File) {
	t.Helper()
	fs := source.NewFileSet()
	sf := fs.AddFileData("t.gad", -1, []byte(src))
	po := &parser.ParserOptions{Mode: parser.ParseComments}
	f, err := parser.NewParserWithOptions(sf, po, nil).ParseFile()
	require.NoError(t, err, src)
	return f, sf
}

// nth returns the offset of the nth (0-based) occurrence of sub in s.
func nth(s, sub string, n int) int {
	off := -1
	for i := 0; i <= n; i++ {
		next := strings.Index(s[off+1:], sub)
		if next < 0 {
			return -1
		}
		off += next + 1
	}
	return off
}

func TestDefinitionBasic(t *testing.T) {
	src := "x := 1\ny := x + 1\nprintln(y)\n"
	f, sf := parse(t, src)

	// the `x` used in `y := x + 1` resolves to the `x` declared at offset 0.
	def, ok := langsym.Definition(f, sf, nth(src, "x", 1))
	require.True(t, ok)
	require.Equal(t, 0, def)

	// the `y` in `println(y)` resolves to the `y` declared on line 2.
	def, ok = langsym.Definition(f, sf, nth(src, "y", 1))
	require.True(t, ok)
	require.Equal(t, nth(src, "y", 0), def)
}

func TestDefinitionShadowing(t *testing.T) {
	src := "" +
		"a := 1\n" +
		"f := func() {\n" +
		"\tb := a\n" + // this a is the OUTER a
		"\ta := 2\n" + // shadows a
		"\tc := a\n" + // this a is the INNER a
		"}\n"
	f, sf := parse(t, src)

	outerA := nth(src, "a", 0) // `a := 1`
	innerA := nth(src, "a := 2", 0)

	// b := a  -> outer a (inner a not declared yet at this point)
	def, ok := langsym.Definition(f, sf, nth(src, "b := a", 0)+len("b := "))
	require.True(t, ok)
	require.Equal(t, outerA, def, "b's a should be the outer a")

	// c := a  -> inner a
	def, ok = langsym.Definition(f, sf, nth(src, "c := a", 0)+len("c := "))
	require.True(t, ok)
	require.Equal(t, innerA, def, "c's a should be the inner a")
}

func TestCompletionsInScopeWithDoc(t *testing.T) {
	src := "" +
		"/// The greeting shown to the user.\n" +
		"msg := \"hi\"\n" +
		"count := 3\n" +
		"later := 9\n" +
		"println(m)\n"
	f, sf := parse(t, src)

	// at the `m` in println(m): msg and count are in scope; `later` too (declared
	// earlier). The completion list carries msg's lead doc comment.
	syms := langsym.Completions(f, sf, nth(src, "println(m", 0)+len("println("))
	byName := map[string]langsym.Symbol{}
	for _, s := range syms {
		byName[s.Label] = s
	}
	require.Contains(t, byName, "msg")
	require.Contains(t, byName, "count")
	require.Equal(t, "The greeting shown to the user.", byName["msg"].Doc)
	require.Equal(t, "variable", byName["msg"].Kind)
}

func TestCompletionsScopeVisibility(t *testing.T) {
	src := "" +
		"a := 1\n" +
		"f := func() {\n" +
		"\tb := 2\n" +
		"\tprintln(b)\n" + // here: a, f, b visible; c not (declared after)
		"\tc := 3\n" +
		"}\n"
	f, sf := parse(t, src)
	syms := langsym.Completions(f, sf, nth(src, "println(b", 0)+len("println("))
	names := map[string]bool{}
	for _, s := range syms {
		names[s.Label] = true
	}
	require.True(t, names["a"], "outer a visible")
	require.True(t, names["b"], "local b visible")
	require.True(t, names["f"], "sibling f visible")
	require.False(t, names["c"], "c declared after the caret must not be visible")
}

func TestDefinitionParam(t *testing.T) {
	src := "f := func(p) {\n\treturn p + 1\n}\n"
	f, sf := parse(t, src)
	// `p` in the body resolves to the parameter `p`.
	def, ok := langsym.Definition(f, sf, nth(src, "p", 1))
	require.True(t, ok)
	require.Equal(t, nth(src, "p", 0), def)
}

// TestDefinitionInInterpolation resolves an identifier used inside a `#"…{x}…"`
// string interpolation (the island is parsed on demand; the literal text around
// it is not an identifier).
func TestDefinitionInInterpolation(t *testing.T) {
	src := "name := \"x\"\nprintln(#\"{name}abc\")\n"
	f, sf := parse(t, src)

	// `name` inside the interpolation resolves to the `name` declared at offset 0.
	def, ok := langsym.Definition(f, sf, nth(src, "name", 1))
	require.True(t, ok)
	require.Equal(t, 0, def)

	// the literal `abc` after the island is not an identifier declaration.
	_, ok = langsym.Definition(f, sf, nth(src, "abc", 0))
	require.False(t, ok)
}
