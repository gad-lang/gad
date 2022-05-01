package importers_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ozanh/ugo"
	"github.com/ozanh/ugo/importers"
)

func TestFileImporter(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	orig := ugo.PrintWriter
	ugo.PrintWriter = buf
	defer func() {
		ugo.PrintWriter = orig
	}()

	tempDir := t.TempDir()
	files := map[string]string{
		"./test1.ugo": `
import("./test2.ugo")
println("test1")
`,
		"./test2.ugo": `
import("./foo/test3.ugo")
println("test2")
`,
		"./foo/test3.ugo": `
import("./test4.ugo")
println("test3")
`,
		"./foo/test4.ugo": `
import("./bar/test5.ugo")
println("test4")
`,
		"./foo/bar/test5.ugo": `
import("../test6.ugo")
println("test5")
`,
		"./foo/test6.ugo": `
println("test6")
`,
	}
	createModules(t, tempDir, files)

	script := []byte(`
import("test1.ugo")
println("main")

// modules have been imported already, so these imports will not trigger a print.
import("test1.ugo")
import("test2.ugo")
import("foo/test3.ugo")
import("foo/test4.ugo")
import("foo/bar/test5.ugo")
import("foo/test6.ugo")

func() {
	import("test1.ugo")
	import("test2.ugo")
	import("foo/test3.ugo")
	import("foo/test4.ugo")
	import("foo/bar/test5.ugo")
	import("foo/test6.ugo")
}()

`)
	opts := ugo.DefaultCompilerOptions
	opts.ModuleMap = ugo.NewModuleMap().
		SetExtImporter(&importers.FileImporter{WorkDir: tempDir})
	bc, err := ugo.Compile(script, opts)
	require.NoError(t, err)
	ret, err := ugo.NewVM(bc).Run(nil)
	require.NoError(t, err)
	require.Equal(t, ugo.Undefined, ret)
	require.Equal(t,
		"test6\ntest5\ntest4\ntest3\ntest2\ntest1\nmain\n",
		strings.ReplaceAll(buf.String(), "\r", ""),
	)
}

func createModules(t *testing.T, baseDir string, files map[string]string) {
	for file, data := range files {
		path := filepath.Join(baseDir, file)
		err := os.MkdirAll(filepath.Dir(path), 0755)
		require.NoError(t, err)
		err = ioutil.WriteFile(path, []byte(data), 0644)
		require.NoError(t, err)
	}
}
