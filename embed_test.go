package gad_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/gad-lang/gad"
	"github.com/gad-lang/gad/importers"
)

// TestVMEmbed exercises `embed(...)` at run time: the Embedded value's members
// (name/data/size), constant reuse, missing-name errors, and directory / search
// -path embedding via the on-disk file importer.
func TestVMEmbed(t *testing.T) {
	// --- in-memory file embeds ---------------------------------------------
	em := NewEmbedMap()
	em.AddFile("greeting.txt", []byte("hello"))
	opts := DefaultCompileOptions
	opts.EmbededdMap = em

	run := func(src string) (Object, error) {
		ret, _, err := NewEval(nil, nil, opts).RunScript(context.Background(), []byte(src))
		return ret, err
	}

	ret, err := run(`e := embed("greeting.txt"); return [e.name, str(e.data), e.size]`)
	require.NoError(t, err)
	require.Equal(t, Array{Str("greeting.txt"), Str("hello"), Int(5)}, ret)

	ret, err = run(`return typeName(embed("greeting.txt"))`)
	require.NoError(t, err)
	require.Equal(t, Str("Embedded"), ret)

	// the same file embedded twice shares a single constant.
	_, bc, err := Compile(NewSymbolTable(NewBuiltins().NameSet),
		[]byte(`embed("greeting.txt"); embed("greeting.txt")`), opts)
	require.NoError(t, err)
	require.Len(t, bc.Constants, 1)

	// an unknown embed name is a compile-time error.
	_, err = run(`embed("missing.txt")`)
	require.Error(t, err)

	// --- directory + `sources` via the on-disk importer --------------------
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "assets"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "assets", "a.txt"), []byte("AAA"), 0o644))

	dopts := DefaultCompileOptions
	dopts.EmbededdMap = NewEmbedMap().
		SetExtImporter(&importers.EmbeddedFileImporter{WorkDirs: []string{dir}})
	drun := func(src string) (Object, error) {
		ret, _, err := NewEval(nil, nil, dopts).RunScript(context.Background(), []byte(src))
		return ret, err
	}

	// a directory is indexed by entry name.
	ret, err = drun(`d := embed("assets"); return [d.name, str(d["a.txt"].data)]`)
	require.NoError(t, err)
	require.Equal(t, Array{Str("assets"), Str("AAA")}, ret)

	// `sources` lists directories to resolve the name in.
	ret, err = drun(`return str(embed("a.txt"; sources = ["assets"]).data)`)
	require.NoError(t, err)
	require.Equal(t, Str("AAA"), ret)

	// `.isDir` distinguishes a directory from a file.
	ret, err = drun(`d := embed("assets"); return [d.isDir, d["a.txt"].isDir]`)
	require.NoError(t, err)
	require.Equal(t, Array{True, False}, ret)

	// a directory's `.fs` is iterable (sorted here), yielding name -> entry.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "assets", "b.txt"), []byte("BBB"), 0o644))
	ret, err = drun(`
	names := []
	for name, e in iterator(embed("assets").fs; sorted) { names = append(names, name) }
	return str(names)`)
	require.NoError(t, err)
	require.Equal(t, Str(`["a.txt", "b.txt"]`), ret)
}
