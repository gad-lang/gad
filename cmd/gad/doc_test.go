package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDocGeneratorFromContentAndFromFile(t *testing.T) {
	gen := &DocGenerator{MustExported: true, NoTest: true}

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
	// A doc-comment example whose result does not match its `>>>` assertion.
	bad := []byte("/**\n```gad\n1\n>>> 2\n```\n**/\nexport A = 1\n")

	var errs []string
	gen := &DocGenerator{OnError: func(msg string) { errs = append(errs, msg) }}
	_, _, failed, err := gen.FromFile(bad, "b.gad", "doc", ".")
	require.NoError(t, err)
	require.Equal(t, 1, failed)
	require.Len(t, errs, 1)
	require.Contains(t, errs[0], "example failed")

	// With NoTest, the failing example is neither run nor reported.
	errs = nil
	genNoTest := &DocGenerator{NoTest: true, OnError: func(msg string) { errs = append(errs, msg) }}
	_, _, failed, err = genNoTest.FromFile(bad, "b.gad", "doc", ".")
	require.NoError(t, err)
	require.Equal(t, 0, failed)
	require.Empty(t, errs)
}
