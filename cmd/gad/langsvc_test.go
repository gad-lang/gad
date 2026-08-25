package main

import (
	"strings"
	"testing"

	"github.com/gad-lang/gad/langsym"
	"github.com/stretchr/testify/require"
)

// nthIndex returns the byte offset of the nth (0-based) occurrence of sub in s.
func nthIndex(s, sub string, n int) int {
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

// TestLangsymParseGadx checks that the language service parses a `.gadx` file
// through the Gadx front-end (lowered to Gad with positions preserved), so
// go-to-definition resolves an interpolation `{= name }` back to the component
// parameter `name`.
func TestLangsymParseGadx(t *testing.T) {
	src := "@comp greeting(; name = \"world\")\n    span Hello {= name }\n"

	file, sf, err := langsymParse("greeting.gadx", []byte(src))
	require.NoError(t, err)

	// caret on the `name` used inside `{= name }` (the last occurrence)
	use := nthIndex(src, "name", 1)
	def, ok := langsym.Definition(file, sf, use)
	require.True(t, ok, "definition should resolve for a .gadx interpolation")

	// it resolves to the `name` parameter in the component signature
	require.Equal(t, nthIndex(src, "name", 0), def)
}

// TestLangsymParseGadt checks that a `.gadt` template is parsed in mixed mode,
// so the `{% … %}` code islands resolve (plain-Gad parsing chokes on the literal
// text). Go-to-definition on a value used in an output island resolves to its
// declaration in the code island.
func TestLangsymParseGadt(t *testing.T) {
	src := "{% x := 1 %}Hi {%= x %}\n"
	file, sf, err := langsymParse("t.gadt", []byte(src))
	require.NoError(t, err)

	def, ok := langsym.Definition(file, sf, nthIndex(src, "x", 1))
	require.True(t, ok, "definition should resolve for a .gadt code island")
	require.Equal(t, nthIndex(src, "x", 0), def)
}

// TestCompleteMidEditGadt reproduces completion inside a mid-edit `.gadt` where
// the caret sits in an empty expression slot (`for i, u in ‸ begin`): the buffer
// does not fully parse, but the tolerant parse still yields a partial AST so the
// in-scope variable is offered instead of "no suggestions".
func TestCompleteMidEditGadt(t *testing.T) {
	src := "{% users := [1, 2, 3] %}\n{%-- for i, u in  begin %}\n{%= u %}\n{%-- end %}\n"
	caret := strings.Index(src, "in  begin") + len("in ")

	file, sf, err := langsymParse("t.gadt", []byte(src))
	require.Error(t, err, "mid-edit source should not fully parse")
	require.NotNil(t, file, "a partial AST must still be returned")

	var labels []string
	for _, s := range completionItems(file, sf, caret) {
		labels = append(labels, s.Label)
	}
	require.Contains(t, labels, "users", "the in-scope variable must be a candidate")
}

// TestLangsymParseGad checks the plain-Gad path still works.
func TestLangsymParseGad(t *testing.T) {
	src := "x := 1\ny := x + 1\nprintln(y)\n"
	file, sf, err := langsymParse("t.gad", []byte(src))
	require.NoError(t, err)

	def, ok := langsym.Definition(file, sf, nthIndex(src, "x", 1))
	require.True(t, ok)
	require.Equal(t, 0, def)
}
