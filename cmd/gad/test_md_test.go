package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	cc "github.com/moisespsena-go/command-context"
	"github.com/stretchr/testify/require"
)

// `gad test FILE.md` runs the gad/gadt/gadx fences (issue #21).
func TestMarkdownFences(t *testing.T) {
	md := "# t\n\n```gad\nx := 1\n= 1\n```\n\n```gad ignore\nnot gad\n```\n\n```go\nfunc main() {}\n```\n\n" +
		"```gadx\ndiv hi\n<<< <div>hi</div>\n```\n\n```gadt\na{%\nx := 2\n= 2\n%}\n```\n\n````gad\nwider: not run\n````\n"
	fs := markdownFences(md)
	require.Len(t, fs, 3)
	require.Equal(t, []string{"gad", "gadx", "gadt"}, []string{fs[0].lang, fs[1].lang, fs[2].lang})
	require.Equal(t, 3, fs[0].line)

	run := func(md string) (int, int, string) {
		dir := t.TempDir()
		path := filepath.Join(dir, "doc.md")
		require.NoError(t, os.WriteFile(path, []byte(md), 0o644))
		var buf bytes.Buffer
		o := &testOptions{}
		require.NoError(t, o.compile())
		pass, fail, _ := o.runFile(&cc.CommandContext{Out: &buf, Err: &buf}, path)
		return pass, fail, buf.String()
	}
	pass, fail, out := run(md)
	require.Equal(t, 3, pass, out)
	require.Equal(t, 0, fail, out)

	pass, fail, out = run("```gad\n1 + 1\n= 3\n```\n\n```gad\nprintln(1)\n<<< 1\n```\n")
	require.Equal(t, 1, pass, out)
	require.Equal(t, 1, fail, out)
	require.Contains(t, out, "doc.md:1 (gad): CheckError: got 2, want 3")
}

// A directory: its test files and its Markdown files.
func TestTestFilesMarkdown(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"a_test.gad", "b.gad", "c.md"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, n), []byte(""), 0o644))
	}
	files, err := testFiles(dir)
	require.NoError(t, err)
	var names []string
	for _, f := range files {
		names = append(names, filepath.Base(f))
	}
	require.ElementsMatch(t, []string{"a_test.gad", "c.md"}, names)
}
