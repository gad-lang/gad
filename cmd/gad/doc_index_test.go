// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	cc "github.com/moisespsena-go/command-context"
	"github.com/stretchr/testify/require"
)

func TestBuildIndexTree(t *testing.T) {
	root := filepath.FromSlash("/out")
	paths := []string{
		filepath.FromSlash("/out/a.md"),
		filepath.FromSlash("/out/sub/b.md"),
		filepath.FromSlash("/out/sub/deep/c.md"),
	}
	tree := buildIndexTree(paths, []string{root})

	// Root has file a.md and subdir sub.
	require.Len(t, tree[root].files, 1)
	_, ok := tree[root].dirs[filepath.FromSlash("/out/sub")]
	require.True(t, ok, "root should link the sub subdirectory")

	// sub has file b.md and subdir deep.
	sub := filepath.FromSlash("/out/sub")
	require.Len(t, tree[sub].files, 1)
	_, ok = tree[sub].dirs[filepath.FromSlash("/out/sub/deep")]
	require.True(t, ok, "sub should link the deep subdirectory")
}

// TestDocCommandGeneratesIndexes runs `gad doc` on a nested tree and checks a
// README.md is generated per directory with file entries and subdirectory links.
func TestDocCommandGeneratesIndexes(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "src", "sub"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "src", "a.gad"),
		[]byte("/*** a module. ***/\nexport A = 1\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "src", "sub", "b.gad"),
		[]byte("/*** b module. ***/\nexport B = 2\n"), 0o644))

	orig, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(orig) }()

	var out, errBuf bytes.Buffer
	inCtx := &cc.CommandContext{Out: &out, Err: &errBuf, InputArgs: cc.Args{
		"--no-config", "--no-doctest", "--out", "doc", filepath.Join("src", "..."),
	}}
	runCtx, err := docCommand().Parse(inCtx)
	require.NoError(t, err)
	require.NoError(t, runCtx.Run())

	// The tree is mirrored under doc/ from the workspace, so sources live at
	// doc/src/…: the root index links the src subdirectory; doc/src lists a.md
	// and links sub; doc/src/sub lists b.md.
	rootIdx, err := os.ReadFile(filepath.Join(dir, "doc", "README.md"))
	require.NoError(t, err)
	require.Contains(t, string(rootIdx), "[src](src/README.md)")

	srcIdx, err := os.ReadFile(filepath.Join(dir, "doc", "src", "README.md"))
	require.NoError(t, err)
	require.Contains(t, string(srcIdx), "[a](a.md)")
	require.Contains(t, string(srcIdx), "[sub](sub/README.md)")

	subIdx, err := os.ReadFile(filepath.Join(dir, "doc", "src", "sub", "README.md"))
	require.NoError(t, err)
	require.Contains(t, string(subIdx), "[b](b.md)")
}
