package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cc "github.com/moisespsena-go/command-context"
	"github.com/stretchr/testify/require"
)

func newDocCtx(args ...string) (*cc.CommandContext, *bytes.Buffer, *bytes.Buffer) {
	var out, errBuf bytes.Buffer
	ctx := &cc.CommandContext{Out: &out, Err: &errBuf, Args: cc.Args(args)}
	return ctx, &out, &errBuf
}

func TestGenerateDocModuleHeadingAndErrors(t *testing.T) {
	// No exports and no documented internals: just the heading.
	md, err := generateDoc("foo/bar.gad", []byte("x := 1\n"), false)
	require.NoError(t, err)
	require.Equal(t, "# bar\n", md)

	_, err = generateDoc("bad.gad", []byte("const = \n"), false)
	require.Error(t, err)
}

func TestGenerateDocSections(t *testing.T) {
	src := "/***\nmodule overview.\n***/\n\n" +
		"/// the pi value\nexport Pi = 3.14\n\n" +
		"/// returns a + b\nexport func sum(a, b) { return a + b }\n"
	md, err := generateDoc("m.gad", []byte(src), true)
	require.NoError(t, err)

	require.Contains(t, md, "# m\n")
	require.Contains(t, md, "module overview.")
	require.Contains(t, md, "## Table of Contents")
	require.Contains(t, md, "## Constants")
	require.Contains(t, md, "### const **Pi**")
	require.Contains(t, md, "```gad\nconst Pi = 3.14\n```")
	require.Contains(t, md, "the pi value")
	require.Contains(t, md, "## Types")
	require.Contains(t, md, "### Functions")
	require.Contains(t, md, "#### func **sum**")
	require.Contains(t, md, "```gad\nsum(a, b)\n```")
	require.Contains(t, md, "returns a + b")
}

func TestGenerateDocFuncWithMethods(t *testing.T) {
	src := "/// a difference calculator\n" +
		"export func diff {\n" +
		"\t/// difference of two ints\n\t(a int, b int) => b - a\n\n" +
		"\t/// difference of two floats\n\t(a float, b float) => b - a\n}\n"
	md, err := generateDoc("m.gad", []byte(src), true)
	require.NoError(t, err)

	require.Contains(t, md, "#### func **diff**")
	require.Contains(t, md, "a difference calculator")
	require.Contains(t, md, "```gad\n(a int, b int)\n```")
	require.Contains(t, md, "difference of two ints")
	require.Contains(t, md, "**other methods**")
	require.Contains(t, md, "```gad\n(a float, b float)\n```")
	require.Contains(t, md, "difference of two floats")
}

func TestGenerateDocDictExport(t *testing.T) {
	src := "/// public API\nexport {\n" +
		"\t/// the max retries\n\tmaxRetries: 3,\n" +
		"\t/// compute the area\n\tarea: func(r) { return r * r },\n}\n"
	md, err := generateDoc("m.gad", []byte(src), true)
	require.NoError(t, err)

	require.Contains(t, md, "### const **maxRetries**")
	require.Contains(t, md, "```gad\nconst maxRetries = 3\n```")
	require.Contains(t, md, "the max retries")
	require.Contains(t, md, "#### func **area**")
	require.Contains(t, md, "```gad\narea(r)\n```")
	require.Contains(t, md, "compute the area")
}

func TestDocWritesTree(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "src", "sub"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "src", "a.gad"), []byte("const A = 1\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "src", "sub", "b.gad"), []byte("const B = 2\n"), 0o644))

	o := &docOptions{out: filepath.Join(dir, "doc"), workspace: dir, dstSet: true}
	ctx, _, _ := newDocCtx()
	require.NoError(t, o.processArg(ctx, filepath.Join(dir, "src", "..."), o.out, dir))

	for _, rel := range []string{"doc/src/a.md", "doc/src/sub/b.md"} {
		_, err := os.Stat(filepath.Join(dir, rel))
		require.NoError(t, err, rel)
	}
}

func TestDocNoSaveReportsOnly(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.gad"), []byte("const A = 1\n"), 0o644))

	o := &docOptions{out: filepath.Join(dir, "doc"), workspace: dir, dstSet: true, noSave: true}
	ctx, out, _ := newDocCtx()
	require.NoError(t, o.processFile(ctx, filepath.Join(dir, "a.gad"), o.out, dir))

	require.Contains(t, out.String(), filepath.Join("doc", "a.md"))
	_, err := os.Stat(filepath.Join(dir, "doc"))
	require.True(t, os.IsNotExist(err), "no output dir should be created")
}

func TestDocDefaultsToCurrentDirRecursive(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "sub"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.gad"), []byte("/// a value\nexport A = 1\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sub", "b.gad"), []byte("/// b value\nexport B = 2\n"), 0o644))

	orig, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(orig) }()

	var out, errBuf bytes.Buffer
	// InputArgs is empty (no PATH) + --no-config so it won't look for .gad.yaml.
	// ParseArgs should default to "..." and run() should produce doc/a.md + doc/sub/b.md.
	inCtx := &cc.CommandContext{Out: &out, Err: &errBuf, InputArgs: cc.Args{"--no-config", "--no-doctest"}}
	runCtx, err := docCommand().Parse(inCtx)
	require.NoError(t, err)
	require.NoError(t, runCtx.Run())

	for _, rel := range []string{"doc/a.md", "doc/sub/b.md"} {
		_, statErr := os.Stat(filepath.Join(dir, rel))
		require.NoError(t, statErr, rel)
	}
}

// TestDocPreservesTreeWithConfig reproduces the flatten bug: a workspace
// .gad.yaml makes o.workspace absolute, while a recursive "." scan yields
// cwd-relative paths. processFile must still mirror subdirectories under doc/
// instead of collapsing them to base names.
func TestDocPreservesTreeWithConfig(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "sub"), 0o755))
	// A config without a doc: section: enough to make the workspace absolute.
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gad.yaml"), []byte("fmt: {}\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.gad"), []byte("/// a value\nexport A = 1\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sub", "b.gad"), []byte("/// b value\nexport B = 2\n"), 0o644))

	orig, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(orig) }()

	var out, errBuf bytes.Buffer
	inCtx := &cc.CommandContext{Out: &out, Err: &errBuf, InputArgs: cc.Args{"--no-doctest"}}
	runCtx, err := docCommand().Parse(inCtx)
	require.NoError(t, err)
	require.NoError(t, runCtx.Run())

	_, err = os.Stat(filepath.Join(dir, "doc", "a.md"))
	require.NoError(t, err, "doc/a.md")
	_, err = os.Stat(filepath.Join(dir, "doc", "sub", "b.md"))
	require.NoError(t, err, "doc/sub/b.md should preserve the subdir, not flatten to doc/b.md")
	// The flattened path must NOT exist.
	_, err = os.Stat(filepath.Join(dir, "doc", "b.md"))
	require.True(t, os.IsNotExist(err), "doc/b.md (flattened) must not exist")
}

func TestGenerateDocSingleLineBlockTrimmed(t *testing.T) {
	// A single-line /*** … ***/ root block renders flush-left (the padding
	// spaces around the content are dropped); a multi-line block is untouched.
	md, err := generateDoc("m.gad", []byte("/*** module overview. ***/\nexport A = 1\n"), true)
	require.NoError(t, err)
	require.Contains(t, md, "\nmodule overview.\n")
	require.NotContains(t, md, " module overview. ")
}

func TestGenerateDocExportedAndInternal(t *testing.T) {
	src := "/// the pi value\nexport Pi = 3.14\n\n" +
		"/// double a number\ndbl := (x) => x * 2\n\n" +
		"/// the retry budget\nmaxRetries := 5\n\n" +
		"/// internal sum\nfunc isum(a, b) { return a + b }\n"

	// Default (must-exported false): two root sections, both populated.
	md, err := generateDoc("m.gad", []byte(src), false)
	require.NoError(t, err)
	require.Contains(t, md, "## Exported")
	require.Contains(t, md, "## Internal")
	// Exported entry stays under Exported; internals are documented too.
	require.Contains(t, md, "#### const **Pi**")
	require.Contains(t, md, "func **dbl**")       // closure -> Type/Functions
	require.Contains(t, md, "var **maxRetries**") // value -> Variables
	require.Contains(t, md, "### Variables")      // internal :=-value section
	require.Contains(t, md, "func **isum**")      // func statement -> Type/Functions
	require.Contains(t, md, "#### Functions")     // Types kind subsection
	require.Contains(t, md, "double a number")
	require.Contains(t, md, "internal sum")

	// must-exported true: no Internal section, flat layout.
	md2, err := generateDoc("m.gad", []byte(src), true)
	require.NoError(t, err)
	require.NotContains(t, md2, "## Internal")
	require.NotContains(t, md2, "## Exported")
	require.Contains(t, md2, "## Constants")
	require.Contains(t, md2, "### const **Pi**")
	require.NotContains(t, md2, "maxRetries") // internal omitted
	require.NotContains(t, md2, "isum")
}

func TestGenerateDocVariablesSection(t *testing.T) {
	// const + export are Constants; `:=` values and `var` are Variables.
	src := "/// the answer\nexport Answer = 42\n\n" +
		"/// an error value\nerr := error(\"oops\")\n\n" +
		"/// a counter\nvar count\n"
	md, err := generateDoc("m.gad", []byte(src), false)
	require.NoError(t, err)

	require.Contains(t, md, "### Constants")
	require.Contains(t, md, "#### const **Answer**")
	require.Contains(t, md, "### Variables")
	require.Contains(t, md, "#### var **err**")
	require.Contains(t, md, "#### var **count**")
	// A variable must not be rendered under Constants: Constants precedes
	// Variables, and err/count appear after the Variables header.
	require.Less(t, strings.Index(md, "### Variables"), strings.Index(md, "var **err**"))
	require.Less(t, strings.Index(md, "### Variables"), strings.Index(md, "var **count**"))
}

func TestGenerateDocTypesGroupedByKind(t *testing.T) {
	src := "/// a func\nfn := func() { return 1 }\n\n" +
		"/// a class\nclass Cls { x = 0 }\n\n" +
		"/// an enum\nenum En { A, B }\n\n" +
		"/// an interface\nmeti Iface { () <str> }\n"
	md, err := generateDoc("m.gad", []byte(src), false)
	require.NoError(t, err)

	require.Contains(t, md, "### Types")
	require.Contains(t, md, "#### Functions")
	require.Contains(t, md, "#### Classes")
	require.Contains(t, md, "#### Enums")
	require.Contains(t, md, "#### Interfaces")
	// Each entry sits under its kind subsection (one level deeper).
	require.Contains(t, md, "##### func **fn**")
	require.Contains(t, md, "##### class **Cls**")
	require.Contains(t, md, "##### enum **En**")
	require.Contains(t, md, "##### meti **Iface**")
	// Subsections render in declared order: Functions before Interfaces.
	require.Less(t, strings.Index(md, "#### Functions"), strings.Index(md, "#### Interfaces"))
}

func TestDocMustExportedFlagAndConfig(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "m.gad"),
		[]byte("/// exported\nexport A = 1\n\n/// internal\nb := 2\n"), 0o644))

	orig, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(orig) }()

	run := func(args ...string) {
		var out, errBuf bytes.Buffer
		in := append([]string{"--no-doctest"}, append(args, "m.gad")...)
		inCtx := &cc.CommandContext{Out: &out, Err: &errBuf, InputArgs: cc.Args(in)}
		runCtx, err := docCommand().Parse(inCtx)
		require.NoError(t, err)
		require.NoError(t, runCtx.Run())
	}
	read := func() string {
		b, err := os.ReadFile(filepath.Join(dir, "doc", "m.md"))
		require.NoError(t, err)
		return string(b)
	}

	run("--no-config")
	require.Contains(t, read(), "## Internal")

	run("--no-config", "--must-exported")
	require.NotContains(t, read(), "## Internal")

	// Config key must_exported: true is honoured when the flag is absent.
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gad.yaml"),
		[]byte("doc:\n  must_exported: true\n"), 0o644))
	run()
	require.NotContains(t, read(), "## Internal")
}

func TestDocResolveDirDst(t *testing.T) {
	// per-dir dst is relative to the input dir path.
	o := &docOptions{out: "/ws/doc", workspace: "/ws", dstSet: true}
	d := docInputDir{Path: "src/...", Dst: "api"}
	o.resolveDir(&d)
	require.True(t, d.dstSet)
	// resolveDir absolutizes via filepath.Abs; on Windows "/ws" is not absolute
	// and gains a drive letter, so absolutize the expected value the same way.
	wantDst, _ := filepath.Abs(filepath.Join("/ws", "src", "api"))
	require.Equal(t, wantDst, filepath.Clean(d.dst))

	// no per-dir dst, root dst is relative -> inherits root dst.
	o2 := &docOptions{out: "/ws/doc", workspace: "/ws", dstSet: true}
	d2 := docInputDir{Path: "src"}
	o2.resolveDir(&d2)
	require.True(t, d2.dstSet)
	require.Equal(t, "/ws/doc", d2.dst)

	// skip defaults to root skip; --no-skip clears it.
	o3 := &docOptions{out: "/ws/doc", workspace: "/ws", skip: true, noSkip: true, dstSet: true}
	d3 := docInputDir{Path: "src", Dst: "api"}
	o3.resolveDir(&d3)
	require.False(t, d3.skip)
}
