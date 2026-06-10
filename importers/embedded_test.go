package importers_test

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/importers"
	gadtime "github.com/gad-lang/gad/stdlib/time"
	"github.com/stretchr/testify/require"
)

func TestEmbeddedFileImporter_Import(t *testing.T) {
	tempDir0 := t.TempDir()
	tempDir0Files := map[string]string{
		"d0/test1.txt":    `test1`,
		"d0/a/a1.txt":     `a1`,
		"d0/a/a2.txt":     `a2`,
		"d0/b/b1.txt":     `b1`,
		"d0/b/b2.txt":     `b2`,
		"d0/c/d/d1.txt":   `d1`,
		"d0/c/d/d2.txt":   `d2`,
		"d0/c/d/e/e1.txt": `e1`,
		"d0/f/df1.txt":    `f1`,
	}
	createFiles(t, tempDir0, tempDir0Files)
	tempDir1 := t.TempDir()
	tempDir1Files := map[string]string{
		"d1/test2.txt":    `test2`,
		"d1/f/df1.txt":    `d1f1`,
		"d1/f/g/h/h1.txt": `h1`,
		"d1/a/a3.txt":     `a3`,
	}
	createFiles(t, tempDir1, tempDir1Files)

	var buf bytes.Buffer
	opts := gad.DefaultCompilerOptions
	opts.ModuleMap = gad.NewModuleMap()
	opts.ModuleMap.AddBuiltinModuleInit(gadtime.ModuleName, gadtime.ModuleInit)
	opts.EmbededdMap = gad.NewEmbedMap()
	opts.EmbededdMap.Add("f1", gad.EmbeddedFileData(`f1data`))
	opts.EmbededdMap.Add("f2", gad.EmbeddedFile(gad.Embedded{
		Name:          "f2",
		ModTime:       time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
		ReaderFactory: gad.EmbeddedBytesReaderFactory(`f2 data`),
	}))

	opts.EmbededdMap.Add("f2", gad.EmbeddedFile(gad.Embedded{
		Name:          "f2",
		ModTime:       time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
		ReaderFactory: gad.EmbeddedBytesReaderFactory(`f2 data`),
	}))

	opts.EmbededdMap.SetExtImporter(&importers.EmbeddedFileImporter{
		WorkDirs: []string{tempDir0, tempDir1},
	})

	var (
		ret gad.Object
		err error
	)

	t.Run("dir", func(t *testing.T) {
		var (
			ret gad.Object
			err error
		)

		ret, err = run(&buf, []byte(`e := embed("d"; tree, sources=["d0", "d1"]); return str(e), e`), opts)
		require.NoError(t, err)
		retArr, _ := ret.(gad.Array)
		require.NotNil(t, retArr)
		require.Len(t, retArr, 2)
		require.Equal(t, gad.Str("‹Embedded: dir ‹d› with 12 files›"), retArr[0])
		retE, _ := retArr[1].(*gad.Embedded)
		require.NotNil(t, retE)
		files := retE.Files(true)
		// the 1 is d0/f/df1.txt overrided by d1/f/df1.txt
		require.Equal(t, len(files), len(tempDir0Files)+len(tempDir1Files)-1)

		for _, file := range files {
			rel, _ := filepath.Rel(filepath.Dir(tempDir0), file.AbsPath)
			rel = filepath.Join(strings.Split(rel, string(filepath.Separator))[1:]...)
			data, err := file.Read()
			require.NoError(t, err)

			d := tempDir0Files[rel]
			if len(d) == 0 {
				d = tempDir1Files[rel]
			}
			require.Equal(t, d, string(data), rel)
		}
	})

	ret, err = run(&buf, []byte(`return str(embed("df1.txt"; sources=["d0/f"]))`), opts)
	require.NoError(t, err)
	require.Equal(t, gad.Str(fmt.Sprintf("‹Embedded: file ‹df1.txt› 2 B \"%s/d0/f/df1.txt\"›", tempDir0)), ret)

	ret, err = run(&buf, []byte(`return str(embed("d1"))`), opts)
	require.NoError(t, err)
	require.Equal(t, gad.Str(fmt.Sprintf("‹Embedded: dir ‹d1› \"%s/d1\" with 4 files›", tempDir1)), ret)

	ret, err = run(&buf, []byte(`return str(embed("df1.txt"; sources=["d1/f"]))`), opts)
	require.NoError(t, err)
	require.Equal(t, gad.Str(fmt.Sprintf("‹Embedded: file ‹df1.txt› 4 B \"%s/d1/f/df1.txt\"›", tempDir1)), ret)

	ret, err = run(&buf, []byte(`f := embed("f1"); return str([f, f.size, int(f.modTime)])`), opts)
	require.NoError(t, err)
	require.Equal(t, gad.Str("[‹Embedded: file ‹f1› 6 B›, 6, -62135596800]"), ret)

	ret, err = run(&buf, []byte(`f := embed("f2"); return str([f, f.size, typeof(f.modTime), int(f.modTime)])`), opts)
	require.NoError(t, err)
	require.Equal(t, gad.Str("[‹Embedded: file ‹f2› 7 B›, 7, ‹reflect type ‹time.Time››, 1257894000]"), ret)

	ret, err = run(&buf, []byte(`import("time"); f := embed("f2"); return str([f, f.size, typeof(f.modTime), int(f.modTime)])`), opts)
	require.NoError(t, err)
	require.Equal(t, gad.Str("[‹Embedded: file ‹f2› 7 B›, 7, ‹builtin type ‹time››, 1257894000]"), ret)
}
