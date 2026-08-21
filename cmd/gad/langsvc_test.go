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

// TestLangsymParseGad checks the plain-Gad path still works.
func TestLangsymParseGad(t *testing.T) {
	src := "x := 1\ny := x + 1\nprintln(y)\n"
	file, sf, err := langsymParse("t.gad", []byte(src))
	require.NoError(t, err)

	def, ok := langsym.Definition(file, sf, nthIndex(src, "x", 1))
	require.True(t, ok)
	require.Equal(t, 0, def)
}
