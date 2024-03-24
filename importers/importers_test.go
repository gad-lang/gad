package importers_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gad-lang/gad"
	"github.com/stretchr/testify/require"

	"github.com/gad-lang/gad/importers"
)

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

		createModules(t, tempDir, files)

		opts := gad.DefaultCompilerOptions
		opts.ModuleMap = moduleMap.Copy()
		opts.ModuleMap.SetExtImporter(&importers.FileImporter{WorkDir: tempDir})
		bc, err := gad.Compile([]byte(script), gad.CompileOptions{CompilerOptions: opts})
		require.NoError(t, err)
		ret, err := gad.NewVM(bc).RunOpts(&gad.RunOpts{
			StdOut: gad.NewWriter(buf),
		})
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
		createModules(t, tempDir, files)

		tempDir2 := t.TempDir()
		createModules(t, tempDir2, map[string]string{
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
		bc, err := gad.Compile([]byte(script), gad.CompileOptions{CompilerOptions: opts})
		require.NoError(t, err)
		ret, err := gad.NewVM(bc).RunOpts(&gad.RunOpts{
			StdOut: gad.NewWriter(buf),
		})
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

		createModules(t, tempDir, mfiles)

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

		bc, err := gad.Compile(script, gad.CompileOptions{CompilerOptions: opts})
		require.NoError(t, err)
		ret, err := gad.NewVM(bc).RunOpts(&gad.RunOpts{
			StdOut: gad.NewWriter(buf),
		})
		require.NoError(t, err)
		require.Equal(t, gad.Nil, ret)
		require.Equal(t,
			"test7\nsourcemod\ntest6\ntest5\ntest4\ntest3\ntest2\ntest1\nmain\n",
			strings.ReplaceAll(buf.String(), "\r", ""),
		)
	})

}

func createModules(t *testing.T, baseDir string, files map[string]string) {
	for file, data := range files {
		path := filepath.Join(baseDir, file)
		err := os.MkdirAll(filepath.Dir(path), 0755)
		require.NoError(t, err)
		err = os.WriteFile(path, []byte(data), 0644)
		require.NoError(t, err)
	}
}
