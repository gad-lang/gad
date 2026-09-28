package importers_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gad-lang/gad"
	"github.com/stretchr/testify/require"

	"github.com/gad-lang/gad/importers"
)

// TestFileImporterSourceCode verifies Import returns a gad.SourceCode whose Kind
// is chosen from the file extension (.gad / .gadt / .gadx).
func TestFileImporterSourceCode(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) string {
		p := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
		return p
	}
	cases := []struct {
		file string
		kind gad.SourceKind
	}{
		{write("a.gad", "return 1"), gad.SourceKindGad},
		{write("b.gadt", "{%= 1 %}"), gad.SourceKindGadt},
		{write("c.gadx", "span Hi"), gad.SourceKindGadx},
	}
	imp := &importers.FileImporter{WorkDir: dir}
	for _, c := range cases {
		ei := imp.Get(c.file)
		require.NotNil(t, ei)
		data, uri, err := ei.Import(context.Background(),
			&gad.ModuleSpec{ModuleInfo: gad.ModuleInfo{Name: c.file}})
		require.NoError(t, err)
		sc, ok := data.(gad.SourceCode)
		require.Truef(t, ok, "want gad.SourceCode, got %T", data)
		require.Equal(t, c.kind, sc.Kind)
		require.NotEmpty(t, uri)
	}
}

func TestFileImporter(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	files := map[string]string{
		"./test1.gad": `
import("./test2.gad")
println("test1")
`,
		"./test2.gad": `
import("./foo/test3.gad")
println("test2")
`,
		"./foo/test3.gad": `
import("./test4.gad")
println("test3")
`,
		"./foo/test4.gad": `
import("./bar/test5.gad")
println("test4")
`,
		"./foo/bar/test5.gad": `
import("../test6.gad")
println("test5")
`,
		"./foo/test6.gad": `
import("sourcemod")
println("test6")
`,
		"./test7.gad": `
println("test7")
`,
	}

	script := `
import("test1.gad")
println("main")

// modules have been imported already, so these imports will not trigger a print.
import("test1.gad")
import("test2.gad")
import("foo/test3.gad")
import("foo/test4.gad")
import("foo/bar/test5.gad")
import("foo/test6.gad")

func() {
	import("test1.gad")
	import("test2.gad")
	import("foo/test3.gad")
	import("foo/test4.gad")
	import("foo/bar/test5.gad")
	import("foo/test6.gad")
}()

`
	moduleMap := gad.NewModuleMap().
		AddSourceModule("sourcemod", []byte(`
import("./test7.gad")
println("sourcemod")`))

	t.Run("default", func(t *testing.T) {
		buf.Reset()

		tempDir := t.TempDir()

		createFiles(t, tempDir, files)

		opts := gad.DefaultCompilerOptions
		opts.ModuleMap = moduleMap.Copy()
		opts.ModuleMap.SetExtImporter(&importers.FileImporter{WorkDir: tempDir})

		ret, err := run(buf, []byte(script), opts)
		require.NoError(t, err)
		require.Equal(t, gad.Nil, ret)
		require.Equal(t,
			"test7\nsourcemod\ntest6\ntest5\ntest4\ntest3\ntest2\ntest1\nmain\n",
			strings.ReplaceAll(buf.String(), "\r", ""),
		)
	})

	t.Run("default_dirs", func(t *testing.T) {
		buf.Reset()

		tempDir := t.TempDir()
		createFiles(t, tempDir, files)

		tempDir2 := t.TempDir()
		createFiles(t, tempDir2, map[string]string{
			"./test8.gad": `
import("./test1.gad")
println("test8")
`,
		})

		opts := gad.DefaultCompilerOptions
		opts.ModuleMap = moduleMap.Copy()
		opts.ModuleMap.SetExtImporter(&importers.FileImporter{
			WorkDir:      tempDir,
			NameResolver: importers.OsDirsNameResolver([]string{tempDir, tempDir2}),
		})

		script := script
		script += "\n" + `import("test8.gad")`
		ret, err := run(buf, []byte(script), opts)
		require.NoError(t, err)
		require.Equal(t, gad.Nil, ret)
		require.Equal(t,
			"test7\nsourcemod\ntest6\ntest5\ntest4\ntest3\ntest2\ntest1\nmain\ntest8\n",
			strings.ReplaceAll(buf.String(), "\r", ""),
		)
	})

	t.Run("shebang", func(t *testing.T) {
		buf.Reset()

		const shebangline = "#!/usr/bin/gad\n"

		mfiles := make(map[string]string)
		for k, v := range files {
			mfiles[k] = shebangline + v
		}

		tempDir := t.TempDir()

		createFiles(t, tempDir, mfiles)

		opts := gad.DefaultCompilerOptions
		opts.ModuleMap = moduleMap.Copy()
		opts.ModuleMap.SetExtImporter(
			&importers.FileImporter{
				WorkDir:    tempDir,
				FileReader: importers.ShebangReadFile,
			},
		)

		script := append([]byte(shebangline), script...)
		importers.Shebang2Slashes(script)

		ret, err := run(buf, script, opts)
		require.NoError(t, err)
		require.Equal(t, gad.Nil, ret)
		require.Equal(t,
			"test7\nsourcemod\ntest6\ntest5\ntest4\ntest3\ntest2\ntest1\nmain\n",
			strings.ReplaceAll(buf.String(), "\r", ""),
		)
	})

}

func createFiles(t *testing.T, baseDir string, files map[string]string) {
	for file, data := range files {
		path := filepath.Join(baseDir, file)
		err := os.MkdirAll(filepath.Dir(path), 0755)
		require.NoError(t, err)
		err = os.WriteFile(path, []byte(data), 0644)
		require.NoError(t, err)
	}
}

func run(w io.Writer, script []byte, opts gad.CompilerOptions) (ret gad.Object, err error) {
	builtins := gad.NewBuiltins().Build()
	cr1, err := gad.Compile(gad.NewSymbolTable(builtins.Builtins().NameSet), script, gad.CompileOptions{CompilerOptions: opts})
	bc := cr1.BC()
	if err != nil {
		return
	}
	return gad.NewVM(builtins, bc).RunOpts(&gad.RunOpts{
		StdOut: gad.NewWriter(w),
	})
}

// TestOsDirsNameResolverEmptyResolvesAgainstCwd verifies that with no search
// path a relative module name still resolves against the importing module's
// directory (so `gad run dir/main.gad` can import "./sibling.gad" from outside
// dir), while an absolute name and an empty cwd are left as they are.
func TestOsDirsNameResolverEmptyResolvesAgainstCwd(t *testing.T) {
	resolve := importers.OsDirsNameResolverPtr(&importers.PathList{})

	got, err := resolve(filepath.Join("samples", "modules"), "./mathx.gad")
	require.NoError(t, err)
	require.Equal(t, filepath.Join("samples", "modules", "mathx.gad"), got)

	abs := filepath.Join(string(filepath.Separator), "x", "y.gad")
	got, err = resolve("samples", abs)
	require.NoError(t, err)
	require.Equal(t, abs, got)

	got, err = resolve("", "./y.gad")
	require.NoError(t, err)
	require.Equal(t, "./y.gad", got)
}

// TestFileImporterGlob verifies Glob expands a pattern into regular files sorted
// by path, relative to the pattern's static base, honouring `**` and a
// fixed-depth pattern, and yields nothing for a missing base directory.
func TestFileImporterGlob(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"m/b.gad", "m/a.gad", "m/x/c.gad", "m/x/y/d.gad", "m/note.txt"} {
		p := filepath.Join(dir, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, nil, 0o644))
	}
	imp := &importers.FileImporter{WorkDir: dir}
	rels := func(pattern string) (out []string) {
		g := imp.Get(pattern).(gad.GlobExtImporter)
		ms, err := g.Glob()
		require.NoError(t, err)
		for _, m := range ms {
			require.True(t, filepath.IsAbs(m.Name))
			out = append(out, m.Rel)
		}
		return
	}
	require.Equal(t, []string{"a.gad", "b.gad"}, rels("./m/*.gad"))
	require.Equal(t, []string{"a.gad", "b.gad", "x/c.gad", "x/y/d.gad"}, rels("./m/**/*.gad"))
	require.Equal(t, []string{"x/c.gad"}, rels("m/*/*.gad"))
	require.Empty(t, rels("./missing/*.gad"))
}

// TestFileImporterSourceExtensionOrder verifies a name written without an
// extension resolves to the first existing source in the importing file's
// dialect order — .gad, .gadt, .gadx; from a .gadx file .gadx, .gad, .gadt —
// and that an extension-less glob matches modules by name, one per name.
func TestFileImporterSourceExtensionOrder(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"m.gad", "m.gadt", "m.gadx", "only.gadt", "x_cfg.gad", "y_cfg.gadx", "note.txt"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), nil, 0o644))
	}
	resolve := func(from, name string) string {
		imp := &importers.FileImporter{WorkDir: dir, SourceKind: gad.SourceKindForExt(from)}
		got, err := imp.Get(name).Name()
		require.NoError(t, err)
		return filepath.Base(got)
	}
	require.Equal(t, "m.gad", resolve("main.gad", "./m"))
	require.Equal(t, "m.gad", resolve("", "./m"))
	require.Equal(t, "m.gadx", resolve("page.gadx", "./m"))
	require.Equal(t, "only.gadt", resolve("page.gadx", "./only"))
	require.Equal(t, "m.gadt", resolve("main.gad", "./m.gadt")) // an explicit extension is kept

	rels := func(from, pattern string) (out []string) {
		imp := &importers.FileImporter{WorkDir: dir, SourceKind: gad.SourceKindForExt(from)}
		ms, err := imp.Get(pattern).(gad.GlobExtImporter).Glob()
		require.NoError(t, err)
		for _, m := range ms {
			out = append(out, m.Rel)
		}
		return
	}
	// `*` without an extension: modules only (not note.txt), one per name
	require.Equal(t, []string{"m.gad", "only.gadt", "x_cfg.gad", "y_cfg.gadx"}, rels("main.gad", "./*"))
	require.Equal(t, []string{"m.gadx", "only.gadt", "x_cfg.gad", "y_cfg.gadx"}, rels("page.gadx", "./*"))
	require.Equal(t, []string{"x_cfg.gad", "y_cfg.gadx"}, rels("main.gad", "./*_cfg"))
	// with an extension the pattern matches file names as written
	require.Equal(t, []string{"note.txt"}, rels("main.gad", "./*.txt"))
}
