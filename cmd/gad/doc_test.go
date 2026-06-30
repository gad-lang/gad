package main

import (
	"bytes"
	"path/filepath"
	"testing"

	cc "github.com/moisespsena-go/command-context"
	"github.com/stretchr/testify/require"
)

func TestDocGeneratorFromContentAndFromFile(t *testing.T) {
	var out, errBuf bytes.Buffer
	ctx := &cc.CommandContext{Out: &out, Err: &errBuf}
	gen := &DocGenerator{Context: ctx, MustExported: true, NoTest: true}

	src := []byte("/// pi\nexport Pi = 3.14\n")

	// FromContent renders the Markdown (must-exported flat layout).
	md, err := gen.FromContent("m.gad", src)
	require.NoError(t, err)
	require.Contains(t, md, "# m\n")
	require.Contains(t, md, "### const **Pi**")

	// FromFile returns the same Markdown plus the mirrored output path, without
	// touching the filesystem.
	md2, outPath, failed, err := gen.FromFile(src, filepath.Join("src", "m.gad"), "doc", "src")
	require.NoError(t, err)
	require.Equal(t, md, md2)
	require.Equal(t, filepath.Join("doc", "m.md"), outPath)
	require.Equal(t, 0, failed)
}

func TestDocGeneratorFromFileReportsFailingExample(t *testing.T) {
	var out, errBuf bytes.Buffer
	ctx := &cc.CommandContext{Out: &out, Err: &errBuf}

	// A doc-comment example whose result does not match its `>>>` assertion.
	bad := []byte("/**\n```gad\n1\n>>> 2\n```\n**/\nexport A = 1\n")

	gen := &DocGenerator{Context: ctx}
	_, _, failed, err := gen.FromFile(bad, "b.gad", "doc", ".")
	require.NoError(t, err)
	require.Equal(t, 1, failed)
	require.Contains(t, errBuf.String(), "example failed")

	// With NoTest, the failing example is neither run nor reported.
	errBuf.Reset()
	genNoTest := &DocGenerator{Context: ctx, NoTest: true}
	_, _, failed, err = genNoTest.FromFile(bad, "b.gad", "doc", ".")
	require.NoError(t, err)
	require.Equal(t, 0, failed)
	require.Empty(t, errBuf.String())
}
