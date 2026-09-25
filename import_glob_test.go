package gad_test

import (
	"os"
	"path/filepath"
	"testing"

	gad "github.com/gad-lang/gad"
	"github.com/gad-lang/gad/importers"
	"github.com/stretchr/testify/require"
)

// TestMatchGlob covers segment globs and `**` spanning zero or more segments.
func TestMatchGlob(t *testing.T) {
	for _, c := range []struct {
		pattern, rel string
		want         bool
	}{
		{"*.gad", "a.gad", true},
		{"*.gad", "x/a.gad", false},
		{"*/*.gad", "x/a.gad", true},
		{"**/*.gad", "a.gad", true},
		{"**/*.gad", "x/y/a.gad", true},
		{"x/**", "x/y/z", true},
		{"x/**/a.gad", "x/a.gad", true},
		{"x/**/a.gad", "y/a.gad", false},
		{"[ab].gad", "b.gad", true},
		{"?.gad", "ab.gad", false},
	} {
		require.Equal(t, c.want, gad.MatchGlob(c.pattern, c.rel), "%s ~ %s", c.pattern, c.rel)
	}

	base, glob := gad.SplitGlobPattern("./plugins/**/*.gad")
	require.Equal(t, "./plugins", base)
	require.Equal(t, "**/*.gad", glob)
	require.Equal(t, -1, gad.GlobDepth(glob))
	require.Equal(t, 2, gad.GlobDepth("*/*.gad"))
}

// TestPathFilters covers the embed-style filters: globs match the base name or
// the relative path, regexps the relative path; excludes win.
func TestPathFilters(t *testing.T) {
	f := gad.PathFilters{Includes: []string{"*.gad"}, Excludes: []string{"*_test.gad"}}
	require.True(t, f.Match("a.gad"))
	require.True(t, f.Match("sub/a.gad"))
	require.False(t, f.Match("a_test.gad"))
	require.False(t, f.Match("a.txt"))

	f = gad.PathFilters{IncludesRe: []string{`^sub/`}, ExcludesRe: []string{`skip`}}
	require.True(t, f.Match("sub/a.gad"))
	require.False(t, f.Match("a.gad"))
	require.False(t, f.Match("sub/skip.gad"))

	f = gad.PathFilters{Excludes: []string{"sub/**"}}
	require.False(t, f.Match("sub/x/a.gad"))
	require.True(t, f.Match("a.gad"))
	require.True(t, (&gad.PathFilters{}).IsZero())

	// MatchModule: test files only when an include names `_test`.
	require.True(t, gad.IsTestFile("x/a_test.gadx"))
	require.False(t, gad.IsTestFile("a_testing.gad"))
	require.False(t, (&gad.PathFilters{}).MatchModule("a_test.gad"))
	require.False(t, (&gad.PathFilters{Includes: []string{"*.gad"}}).MatchModule("a_test.gad"))
	require.True(t, (&gad.PathFilters{Includes: []string{"*_test.gad"}}).MatchModule("a_test.gad"))
	require.True(t, (&gad.PathFilters{}).MatchModule("a.gad"))
}

// globTree writes a small module tree and returns its directory.
func globTree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"plugins/b.gad":      "param (; tag = \"-\")\nexport name = \"b\"\nexport label = tag\n",
		"plugins/a.gad":      "param (; tag = \"-\")\nexport name = \"a\"\nexport label = tag\n",
		"plugins/a_test.gad": "export name = \"a_test\"\n",
		"plugins/sub/c.gad":  "export name = \"c\"\n",
		"parts/00_init.gad":  "parts := []\n",
		"parts/10_one.gad":   "parts += \"one\"\n",
		"parts/20_two.gad":   "parts += \"two\"\n",
		"parts/sub/30_x.gad": "parts += \"x\"\n",
		"parts/40_test.gad":  "parts += \"test\"\n",
	}
	for name, src := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte(src), 0o644))
	}
	return dir
}

// runGlob compiles and runs src as dir/main.gad with a file importer rooted at
// dir.
func runGlob(t *testing.T, dir, src string) (gad.Object, error) {
	t.Helper()
	main := filepath.Join(dir, "main.gad")
	require.NoError(t, os.WriteFile(main, []byte(src), 0o644))
	builtins := gad.NewBuiltins()
	opts := gad.CompileOptions{}
	opts.ModuleMap = gad.NewModuleMap().SetExtImporter(&importers.FileImporter{WorkDir: dir})
	opts.ModuleFile = main
	cr, err := gad.Compile(gad.NewSymbolTable(builtins.NameSet), []byte(src), opts)
	if err != nil {
		return nil, err
	}
	return gad.NewVM(builtins.Build(), cr.Bytecode).RunOpts(&gad.RunOpts{})
}

// TestImportGlob covers `import(pattern)`: an array of the matching modules,
// sorted by path; `**`; the @includes/@excludes(_re) filters; module params
// applied to each match; no match -> empty array; the importing file skipped.
func TestImportGlob(t *testing.T) {
	dir := globTree(t)
	names := `names := func(ms) => [m.name for m in ms]` + "\n"

	for _, c := range []struct {
		src  string
		want gad.Object
	}{
		// test files (`*_test.gad`) are skipped by default…
		{`return names(import("./plugins/*.gad"))`, gad.Array{gad.Str("a"), gad.Str("b")}},
		{`return names(import("./plugins/**/*.gad"))`, gad.Array{gad.Str("a"), gad.Str("b"), gad.Str("c")}},
		// …a generic include does not bring them back…
		{`return names(import("./plugins/*.gad"; @includes="a*"))`, gad.Array{gad.Str("a")}},
		// …an include naming `_test` does (and the excludes still apply).
		{`return names(import("./plugins/*.gad"; @includes=["*_test.gad"]))`, gad.Array{gad.Str("a_test")}},
		{`return names(import("./plugins/*.gad"; @includes=["*.gad", "*_test.gad"]))`,
			gad.Array{gad.Str("a"), gad.Str("a_test"), gad.Str("b")}},
		{`return names(import("./plugins/*.gad"; @includes_re=["_test"]))`, gad.Array{gad.Str("a_test")}},
		{`return names(import("./plugins/*.gad"; @includes=["*_test.gad"], @excludes=["a*"]))`, gad.Array{}},
		{`return names(import("./plugins/**/*.gad"; @excludes=["b.gad"]))`, gad.Array{gad.Str("a"), gad.Str("c")}},
		{`return names(import("./plugins/**/*.gad"; @includes_re=["^sub/"]))`, gad.Array{gad.Str("c")}},
		{`return [m.label for m in import("./plugins/[ab].gad"; tag="T")]`, gad.Array{gad.Str("T"), gad.Str("T")}},
		{`return import("./nothing/*.gad")`, gad.Array{}},
		{`return len(import("./*.gad"))`, gad.Int(0)}, // main.gad itself is skipped
	} {
		got, err := runGlob(t, dir, names+c.src)
		require.NoError(t, err, c.src)
		require.Equal(t, c.want, got, c.src)
	}

	// Filters need a glob, and must be literal strings / valid regexps.
	for _, src := range []string{
		`import("./plugins/a.gad"; @includes=["x"])`,
		`import("./plugins/*.gad"; @includes=[1])`,
		`import("./plugins/*.gad"; @includes_re=["("])`,
	} {
		_, err := runGlob(t, dir, src)
		require.Error(t, err, src)
	}
}

// TestIncludeGlob covers `include (pattern)`: every matching file inline, in
// path order, narrowed by the unprefixed includes/excludes(_re) filters (include
// takes no params, so they need no `@`).
func TestIncludeGlob(t *testing.T) {
	dir := globTree(t)
	for _, c := range []struct {
		src  string
		want gad.Object
	}{
		{"include (\"./parts/*.gad\")\nreturn parts", gad.Array{gad.Str("one"), gad.Str("two")}},
		{"include (\"./parts/*.gad\"; includes=[\"*.gad\", \"*_test.gad\"])\nreturn parts",
			gad.Array{gad.Str("one"), gad.Str("two"), gad.Str("test")}},
		{"include (\"./parts/**/*.gad\"; excludes=[\"2*\"])\nreturn parts", gad.Array{gad.Str("one"), gad.Str("x")}},
		{"include (\"./parts/00_init.gad\", \"./parts/[12]*.gad\"; includes_re=[\"two\"])\nreturn parts",
			gad.Array{gad.Str("two")}},
	} {
		got, err := runGlob(t, dir, c.src)
		require.NoError(t, err, c.src)
		require.Equal(t, c.want, got, c.src)
	}

	for _, src := range []string{
		`include ("./parts/*.gad"; @excludes=["x"])`,      // import's `@` form is not include's
		`include ("./parts/00_init.gad"; excludes=["x"])`, // filters need a glob
		`include ("./parts/*.gad"; x=1)`,                  // no other named args
	} {
		_, err := runGlob(t, dir, src)
		require.Error(t, err, src)
	}
}
