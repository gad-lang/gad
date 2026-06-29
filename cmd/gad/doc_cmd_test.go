package main

import (
	"bytes"
	"os"
	"path/filepath"
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
	// No exports: just the heading.
	md, err := generateDoc("foo/bar.gad", []byte("x := 1\n"))
	require.NoError(t, err)
	require.Equal(t, "# bar\n", md)

	_, err = generateDoc("bad.gad", []byte("const = \n"))
	require.Error(t, err)
}

func TestGenerateDocSections(t *testing.T) {
	src := "/***\nmodule overview.\n***/\n\n" +
		"/// the pi value\nexport Pi = 3.14\n\n" +
		"/// returns a + b\nexport func sum(a, b) { return a + b }\n"
	md, err := generateDoc("m.gad", []byte(src))
	require.NoError(t, err)

	require.Contains(t, md, "# m\n")
	require.Contains(t, md, "module overview.")
	require.Contains(t, md, "## Table of Contents")
	require.Contains(t, md, "## Constants")
	require.Contains(t, md, "### const **Pi**")
	require.Contains(t, md, "    const Pi = 3.14")
	require.Contains(t, md, "the pi value")
	require.Contains(t, md, "## Types")
	require.Contains(t, md, "### func **sum**")
	require.Contains(t, md, "    sum(a, b)")
	require.Contains(t, md, "returns a + b")
}

func TestGenerateDocFuncWithMethods(t *testing.T) {
	src := "/// a difference calculator\n" +
		"export func diff {\n" +
		"\t/// difference of two ints\n\t(a int, b int) => b - a\n\n" +
		"\t/// difference of two floats\n\t(a float, b float) => b - a\n}\n"
	md, err := generateDoc("m.gad", []byte(src))
	require.NoError(t, err)

	require.Contains(t, md, "### func **diff**")
	require.Contains(t, md, "a difference calculator")
	require.Contains(t, md, "    (a int, b int)")
	require.Contains(t, md, "difference of two ints")
	require.Contains(t, md, "**other methods**")
	require.Contains(t, md, "    (a float, b float)")
	require.Contains(t, md, "difference of two floats")
}

func TestGenerateDocDictExport(t *testing.T) {
	src := "/// public API\nexport {\n" +
		"\t/// the max retries\n\tmaxRetries: 3,\n" +
		"\t/// compute the area\n\tarea: func(r) { return r * r },\n}\n"
	md, err := generateDoc("m.gad", []byte(src))
	require.NoError(t, err)

	require.Contains(t, md, "### const **maxRetries**")
	require.Contains(t, md, "    const maxRetries = 3")
	require.Contains(t, md, "the max retries")
	require.Contains(t, md, "### func **area**")
	require.Contains(t, md, "    area(r)")
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

func TestDocResolveDirDst(t *testing.T) {
	// per-dir dst is relative to the input dir path.
	o := &docOptions{out: "/ws/doc", workspace: "/ws", dstSet: true}
	d := docInputDir{Path: "src/...", Dst: "api"}
	o.resolveDir(&d)
	require.True(t, d.dstSet)
	require.Equal(t, filepath.Clean("/ws/src/api"), filepath.Clean(d.dst))

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
