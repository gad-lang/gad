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
	"gopkg.in/yaml.v3"
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

		ret, err = run(&buf, []byte(`e := embed("d"; tree, sources=["d0", "d1"]); return str(e;indent), e`), opts)
		require.NoError(t, err)
		retArr, _ := ret.(gad.Array)
		require.NotNil(t, retArr)
		require.Len(t, retArr, 2)
		require.Contains(t, string(retArr[0].(gad.Str)), "‹Embedded: dir ‹d›")
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
require.Contains(t, string(ret.(gad.Str)), "d1/test2.txt")

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

// makeEmbedDir creates a temp directory with a subdirectory containing test files
// and returns the dir name alongside compiler options with an EmbeddedFileImporter.
//
// Directory structure:
//
//	{root}/
//	  src/
//	    a.go
//	    b.go
//	    c_test.go
//	    d.txt
//	    e.md
//	    sub/
//	      f.go
//	      g_test.go
//	      h.txt
//	      sub2/
//	        i.go
func makeEmbedDir(t *testing.T) (dirName, root string, opts gad.CompilerOptions) {
	t.Helper()
	root = t.TempDir()
	createFiles(t, root, map[string]string{
		"src/a.go":           "package a",
		"src/b.go":           "package b",
		"src/c_test.go":      "package c",
		"src/d.txt":          "content d",
		"src/e.md":           "# readme",
		"src/sub/f.go":       "package f",
		"src/sub/g_test.go":  "package g",
		"src/sub/h.txt":      "content h",
		"src/sub/sub2/i.go":  "package i",
	})
	opts = gad.DefaultCompilerOptions
	opts.ModuleMap = gad.NewModuleMap()
	opts.EmbededdMap = gad.NewEmbedMap()
	opts.EmbededdMap.SetExtImporter(&importers.EmbeddedFileImporter{
		WorkDirs: []string{root},
	})
	dirName = "src"
	return
}

func TestEmbeddedFileImporter_Includes(t *testing.T) {
	_, root, opts := makeEmbedDir(t)
	tp := fmt.Sprintf("indent,trimEmbedPath=%q", root)

	t.Run("include *.go files", func(t *testing.T) {
		ret, err := runScript(fmt.Sprintf(`e := embed("src"; includes=["*.go"]); return str(e;%s)`, tp), opts)
		require.NoError(t, err)
		require.Contains(t, string(ret.(gad.Str)), "src/a.go")
		require.Contains(t, string(ret.(gad.Str)), "src/sub/f.go")
		require.NotContains(t, string(ret.(gad.Str)), "src/d.txt")
		require.NotContains(t, string(ret.(gad.Str)), "src/sub/h.txt")
	})

	t.Run("include *.txt files", func(t *testing.T) {
		ret, err := runScript(fmt.Sprintf(`e := embed("src"; includes=["*.txt"]); return str(e;%s)`, tp), opts)
		require.NoError(t, err)
		require.Contains(t, string(ret.(gad.Str)), "src/d.txt")
		require.Contains(t, string(ret.(gad.Str)), "src/sub/h.txt")
		require.NotContains(t, string(ret.(gad.Str)), "src/a.go")
	})

	t.Run("include multiple patterns", func(t *testing.T) {
		ret, err := runScript(fmt.Sprintf(`e := embed("src"; includes=["*.go", "*.txt"]); return str(e;%s)`, tp), opts)
		require.NoError(t, err)
		// Go files(5) + txt files(2) = 7, no subdirectories match so just files
		require.Contains(t, string(ret.(gad.Str)), "src/a.go")
		require.Contains(t, string(ret.(gad.Str)), "src/d.txt")
		require.Contains(t, string(ret.(gad.Str)), "src/sub/h.txt")
		require.NotContains(t, string(ret.(gad.Str)), "src/e.md")
	})

	t.Run("include no match", func(t *testing.T) {
		ret, err := runScript(fmt.Sprintf(`e := embed("src"; includes=["*.py"]); return str(e;%s)`, tp), opts)
		require.NoError(t, err)
		require.NotContains(t, string(ret.(gad.Str)), "src/a.go")
		require.NotContains(t, string(ret.(gad.Str)), "src/d.txt")
		require.Contains(t, string(ret.(gad.Str)), "src")
	})

	t.Run("include with tree flag", func(t *testing.T) {
		ret, err := runScript(fmt.Sprintf(`e := embed("src"; tree, includes=["*.go"]); return str(e;%s)`, tp), opts)
		require.NoError(t, err)
		require.Contains(t, string(ret.(gad.Str)), "src/a.go")
		require.NotContains(t, string(ret.(gad.Str)), "src/d.txt")
	})

	t.Run("include with tree flag shows dirs", func(t *testing.T) {
		ret, err := runScript(fmt.Sprintf(`e := embed("src"; tree, includes=["*"]); return str(e;%s)`, tp), opts)
		require.NoError(t, err)
		require.Contains(t, string(ret.(gad.Str)), "src/a.go")
		require.Contains(t, string(ret.(gad.Str)), "src/e.md")
		require.Contains(t, string(ret.(gad.Str)), "src/sub/f.go")
		require.Contains(t, string(ret.(gad.Str)), "src/sub/sub2/i.go")
	})
}

func TestEmbeddedFileImporter_Excludes(t *testing.T) {
	_, root, opts := makeEmbedDir(t)
	tp := fmt.Sprintf("indent,trimEmbedPath=%q", root)

	t.Run("exclude *_test.go", func(t *testing.T) {
		ret, err := runScript(fmt.Sprintf(`e := embed("src"; excludes=["*_test.go"]); return str(e;%s)`, tp), opts)
		require.NoError(t, err)
		require.Contains(t, string(ret.(gad.Str)), "src/a.go")
		require.Contains(t, string(ret.(gad.Str)), "src/e.md")
		require.NotContains(t, string(ret.(gad.Str)), "src/c_test.go")
		require.NotContains(t, string(ret.(gad.Str)), "src/sub/g_test.go")
	})

	t.Run("exclude *.txt", func(t *testing.T) {
		ret, err := runScript(fmt.Sprintf(`e := embed("src"; excludes=["*.txt"]); return str(e;%s)`, tp), opts)
		require.NoError(t, err)
		require.Contains(t, string(ret.(gad.Str)), "src/a.go")
		require.Contains(t, string(ret.(gad.Str)), "src/c_test.go")
	})

	t.Run("exclude multiple patterns", func(t *testing.T) {
		ret, err := runScript(fmt.Sprintf(`e := embed("src"; excludes=["*_test.go", "*.txt"]); return str(e;%s)`, tp), opts)
		require.NoError(t, err)
		require.Contains(t, string(ret.(gad.Str)), "src/a.go")
		require.Contains(t, string(ret.(gad.Str)), "src/e.md")
		require.NotContains(t, string(ret.(gad.Str)), "src/c_test.go")
		require.NotContains(t, string(ret.(gad.Str)), "src/sub/g_test.go")
		require.NotContains(t, string(ret.(gad.Str)), "src/d.txt")
		require.NotContains(t, string(ret.(gad.Str)), "src/sub/h.txt")
	})

	t.Run("exclude with tree flag", func(t *testing.T) {
		ret, err := runScript(fmt.Sprintf(`e := embed("src"; tree, excludes=["*_test.go"]); return str(e;%s)`, tp), opts)
		require.NoError(t, err)
		require.Contains(t, string(ret.(gad.Str)), "dir")
		require.Contains(t, string(ret.(gad.Str)), "src/sub")
	})

	t.Run("exclude nothing (all files)", func(t *testing.T) {
		ret, err := runScript(fmt.Sprintf(`e := embed("src"); return str(e;%s)`, tp), opts)
		require.NoError(t, err)
		require.Contains(t, string(ret.(gad.Str)), "src/a.go")
		require.Contains(t, string(ret.(gad.Str)), "src/e.md")
		require.Contains(t, string(ret.(gad.Str)), "src/sub/sub2/i.go")
	})
}

func TestEmbeddedFileImporter_IncludesExcludes(t *testing.T) {
	_, root, opts := makeEmbedDir(t)
	tp := fmt.Sprintf("indent,trimEmbedPath=%q", root)

	t.Run("include *.go exclude *_test.go", func(t *testing.T) {
		ret, err := runScript(fmt.Sprintf(`e := embed("src"; includes=["*.go"], excludes=["*_test.go"]); return str(e;%s)`, tp), opts)
		require.NoError(t, err)
		
		require.Contains(t, string(ret.(gad.Str)), "src/a.go")
		require.Contains(t, string(ret.(gad.Str)), "src/sub/f.go")
		require.NotContains(t, string(ret.(gad.Str)), "src/c_test.go")
		require.NotContains(t, string(ret.(gad.Str)), "src/d.txt")
	})

	t.Run("include *.go exclude *.txt", func(t *testing.T) {
		ret, err := runScript(fmt.Sprintf(`e := embed("src"; includes=["*.go"], excludes=["*.txt"]); return str(e;%s)`, tp), opts)
		require.NoError(t, err)
require.Contains(t, string(ret.(gad.Str)), "src/a.go")
		require.NotContains(t, string(ret.(gad.Str)), "src/d.txt")
	})
}

func TestEmbeddedFileImporter_IncludesRe(t *testing.T) {
	_, root, opts := makeEmbedDir(t)
	tp := fmt.Sprintf("indent,trimEmbedPath=%q", root)

	t.Run("include go extension", func(t *testing.T) {
		ret, err := runScript(fmt.Sprintf(`e := embed("src"; includes_re=["\\.go$"]); return str(e;%s)`, tp), opts)
		require.NoError(t, err)
		require.Contains(t, string(ret.(gad.Str)), "src/a.go")
		require.NotContains(t, string(ret.(gad.Str)), "src/d.txt")
	})

	t.Run("include txt extension", func(t *testing.T) {
		ret, err := runScript(fmt.Sprintf(`e := embed("src"; includes_re=["\\.txt$"]); return str(e;%s)`, tp), opts)
		require.NoError(t, err)
		require.Contains(t, string(ret.(gad.Str)), "src/d.txt")
		require.NotContains(t, string(ret.(gad.Str)), "src/a.go")
	})

	t.Run("include _test", func(t *testing.T) {
		ret, err := runScript(fmt.Sprintf(`e := embed("src"; includes_re=["_test"]); return str(e;%s)`, tp), opts)
		require.NoError(t, err)
		require.Contains(t, string(ret.(gad.Str)), "src/c_test.go")
		require.NotContains(t, string(ret.(gad.Str)), "src/a.go")
	})

	t.Run("include multiple regex patterns", func(t *testing.T) {
		ret, err := runScript(fmt.Sprintf(`e := embed("src"; includes_re=["\\.go$", "\\.txt$"]); return str(e;%s)`, tp), opts)
		require.NoError(t, err)
		require.Contains(t, string(ret.(gad.Str)), "src/a.go")
		require.Contains(t, string(ret.(gad.Str)), "src/d.txt")
		require.NotContains(t, string(ret.(gad.Str)), "src/e.md")
	})

	t.Run("include_re with tree", func(t *testing.T) {
		ret, err := runScript(fmt.Sprintf(`e := embed("src"; tree, includes_re=["\\.go$"]); return str(e;%s)`, tp), opts)
		require.NoError(t, err)
		require.Contains(t, string(ret.(gad.Str)), "src/a.go")
		require.NotContains(t, string(ret.(gad.Str)), "src/d.txt")
	})
}

func TestEmbeddedFileImporter_ExcludesRe(t *testing.T) {
	_, root, opts := makeEmbedDir(t)
	tp := fmt.Sprintf("indent,trimEmbedPath=%q", root)

	t.Run("exclude _test", func(t *testing.T) {
		ret, err := runScript(fmt.Sprintf(`e := embed("src"; excludes_re=["_test"]); return str(e;%s)`, tp), opts)
		require.NoError(t, err)
		require.Contains(t, string(ret.(gad.Str)), "src/a.go")
		require.NotContains(t, string(ret.(gad.Str)), "src/c_test.go")
	})

	t.Run("exclude txt extension", func(t *testing.T) {
		ret, err := runScript(fmt.Sprintf(`e := embed("src"; excludes_re=["\\.txt$"]); return str(e;%s)`, tp), opts)
		require.NoError(t, err)
		require.Contains(t, string(ret.(gad.Str)), "src/a.go")
		require.Contains(t, string(ret.(gad.Str)), "src/c_test.go")
	})

	t.Run("exclude multiple regex patterns", func(t *testing.T) {
		ret, err := runScript(fmt.Sprintf(`e := embed("src"; excludes_re=["_test", "\\.txt$"]); return str(e;%s)`, tp), opts)
		require.NoError(t, err)
		require.Contains(t, string(ret.(gad.Str)), "src/a.go")
		require.NotContains(t, string(ret.(gad.Str)), "src/c_test.go")
		require.NotContains(t, string(ret.(gad.Str)), "src/d.txt")
	})

	t.Run("exclude_re with tree", func(t *testing.T) {
		ret, err := runScript(fmt.Sprintf(`e := embed("src"; tree, excludes_re=["_test"]); return str(e;%s)`, tp), opts)
		require.NoError(t, err)
		require.Contains(t, string(ret.(gad.Str)), "src/sub")
		require.Contains(t, string(ret.(gad.Str)), "src/a.go")
	})
}

func TestEmbeddedFileImporter_IncludesExcludesRe(t *testing.T) {
	_, root, opts := makeEmbedDir(t)
	tp := fmt.Sprintf("indent,trimEmbedPath=%q", root)

	t.Run("include go extension exclude _test", func(t *testing.T) {
		ret, err := runScript(fmt.Sprintf(`e := embed("src"; includes_re=["\\.go$"], excludes_re=["_test"]); return str(e;%s)`, tp), opts)
		require.NoError(t, err)
		require.Contains(t, string(ret.(gad.Str)), "src/a.go")
		require.NotContains(t, string(ret.(gad.Str)), "src/c_test.go")
		require.NotContains(t, string(ret.(gad.Str)), "src/d.txt")
	})

	t.Run("include _test exclude txt", func(t *testing.T) {
		ret, err := runScript(fmt.Sprintf(`e := embed("src"; includes_re=["_test"], excludes_re=["\\.txt$"]); return str(e;%s)`, tp), opts)
		require.NoError(t, err)
		require.Contains(t, string(ret.(gad.Str)), "src/c_test.go")
		require.NotContains(t, string(ret.(gad.Str)), "src/a.go")
	})
}

func TestEmbeddedFileImporter_ConfigFile(t *testing.T) {
	dirName, root, opts := makeEmbedDir(t)
	tp := fmt.Sprintf("indent,trimEmbedPath=%q", root)

	yamlConfig := map[string]interface{}{
		"includes": []string{"*.go"},
	}
	yamlData, err := yaml.Marshal(yamlConfig)
	require.NoError(t, err)
	createFiles(t, root, map[string]string{
		"embed.yaml": string(yamlData),
	})

	t.Run("config_file includes", func(t *testing.T) {
		script := fmt.Sprintf(`e := embed(%[1]q; config_file="embed.yaml"); return str(e;%[2]s)`, dirName, tp)
		ret, err := runScript(script, opts)
		require.NoError(t, err)
		require.Contains(t, string(ret.(gad.Str)), "src/a.go")
		require.NotContains(t, string(ret.(gad.Str)), "src/e.md")
	})

	t.Run("config_file overridden by inline", func(t *testing.T) {
		// config has includes=["*.go"], but inline includes=["*.txt"] overrides
		script := fmt.Sprintf(`e := embed(%[1]q; config_file="embed.yaml", includes=["*.txt"]); return str(e;%[2]s)`, dirName, tp)
		ret, err := runScript(script, opts)
		require.NoError(t, err)
		require.Contains(t, string(ret.(gad.Str)), "src/d.txt")
		require.NotContains(t, string(ret.(gad.Str)), "src/a.go")
	})
}

func TestEmbeddedFileImporter_TreeOnly(t *testing.T) {
	_, root, opts := makeEmbedDir(t)
	tp := fmt.Sprintf("indent,trimEmbedPath=%q", root)

	ret, err := runScript(fmt.Sprintf(`e := embed("src"; tree); return str(e;%s)`, tp), opts)
	require.NoError(t, err)
		// 9 files total
		require.Contains(t, string(ret.(gad.Str)), "src/a.go")
		require.Contains(t, string(ret.(gad.Str)), "src/sub/sub2/i.go")
}

func TestEmbeddedFileImporter_SingleFile(t *testing.T) {
	_, root, opts := makeEmbedDir(t)
	tp := fmt.Sprintf("indent,trimEmbedPath=%q", root)

	ret, err := runScript(fmt.Sprintf(`return str(embed("src/a.go");%s)`, tp), opts)
	require.NoError(t, err)
	require.Contains(t, string(ret.(gad.Str)), "‹src/a.go›")
}

func TestEmbeddedFileImporter_WithSources(t *testing.T) {
	dir0 := t.TempDir()
	dir1 := t.TempDir()
	createFiles(t, dir0, map[string]string{
		"d0/a.go": "package a",
		"d0/b.go": "package b",
	})
	createFiles(t, dir1, map[string]string{
		"d0/d.go": "package d",
	})
	opts := gad.DefaultCompilerOptions
	opts.ModuleMap = gad.NewModuleMap()
	opts.EmbededdMap = gad.NewEmbedMap()
	opts.EmbededdMap.SetExtImporter(&importers.EmbeddedFileImporter{
		WorkDirs: []string{dir0, dir1},
	})

	t.Run("sources with includes filtering", func(t *testing.T) {
	tp := fmt.Sprintf("indent,trimEmbedPath=%q", dir0)
		ret, err := runScript(fmt.Sprintf(`e := embed("d0"; includes=["*.go"]); return str(e;%s)`, tp), opts)
		require.NoError(t, err)
		require.Contains(t, string(ret.(gad.Str)), "d0/a.go")
		require.NotContains(t, string(ret.(gad.Str)), ".md")
	})
}

func runScript(script string, opts gad.CompilerOptions) (gad.Object, error) {
	var buf bytes.Buffer
	return run(&buf, []byte(script), opts)
}

