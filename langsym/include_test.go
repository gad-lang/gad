package langsym_test

import (
	"testing"

	"github.com/gad-lang/gad/langsym"
	"github.com/stretchr/testify/require"
)

// hasSymbol reports whether a completion with the given label is present.
func hasSymbol(syms []langsym.Symbol, label string) *langsym.Symbol {
	for i := range syms {
		if syms[i].Label == label {
			return &syms[i]
		}
	}
	return nil
}

// TestCompletionsFollowsInclude verifies that with an IncludeResolver installed,
// an included file's top-level declarations appear in the includer's completions
// from the include line onward.
func TestCompletionsFollowsInclude(t *testing.T) {
	prev := langsym.IncludeResolver
	defer func() { langsym.IncludeResolver = prev }()
	langsym.IncludeResolver = func(fromFile, path string) ([]byte, string, bool) {
		if path == "config.gad" {
			return []byte("/// the app name\nappName := \"Gadapp\"\nconst Version = 2\n"), "config.gad", true
		}
		return nil, "", false
	}

	src := "" +
		"x := 1\n" +
		"include (\"config.gad\")\n" +
		"println(x)\n"
	f, sf := parse(t, src)

	// After the include line, appName and Version are in scope.
	syms := langsym.Completions(f, sf, nth(src, "println", 0))
	app := hasSymbol(syms, "appName")
	require.NotNil(t, app, "appName from the included file should complete")
	require.Equal(t, "the app name", app.Doc, "the included decl's doc is carried over")
	require.NotNil(t, hasSymbol(syms, "Version"), "Version from the included file should complete")
	require.NotNil(t, hasSymbol(syms, "x"), "local x is still there")
}

// TestCompletionsIncludeBeforeSite verifies included symbols are NOT offered
// before the include statement (they are declared at the include line).
func TestCompletionsIncludeBeforeSite(t *testing.T) {
	prev := langsym.IncludeResolver
	defer func() { langsym.IncludeResolver = prev }()
	langsym.IncludeResolver = func(fromFile, path string) ([]byte, string, bool) {
		return []byte("appName := \"Gadapp\"\n"), "config.gad", true
	}

	src := "" +
		"before := 1\n" +
		"include (\"config.gad\")\n"
	f, sf := parse(t, src)

	// Caret on the first line (before the include): appName not yet in scope.
	syms := langsym.Completions(f, sf, nth(src, "before", 0)+2)
	require.Nil(t, hasSymbol(syms, "appName"), "included symbol must not leak above the include")
}

// TestCompletionsIncludeNilResolver verifies that without a resolver includes
// are ignored (no crash, no cross-file symbols).
func TestCompletionsIncludeNilResolver(t *testing.T) {
	prev := langsym.IncludeResolver
	defer func() { langsym.IncludeResolver = prev }()
	langsym.IncludeResolver = nil

	src := "include (\"config.gad\")\nx := 1\n"
	f, sf := parse(t, src)
	syms := langsym.Completions(f, sf, nth(src, "x", 0))
	require.NotNil(t, hasSymbol(syms, "x"))
	require.Nil(t, hasSymbol(syms, "appName"))
}
