package gadx

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/node"
)

// Inspect is given every source file of a compilation — the rendered file, the
// modules it imports and the files they include, recursively — parsed, with
// the positions of the source; and it is given them again when the template
// recompiles.
func TestRenderInspect(t *testing.T) {
	dir := t.TempDir()
	write := func(name, src string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("parts/data.gad", "tr := func(s) { return s }\ntitle := tr(\"from_include\")\n")
	write("layouts/one.gadx", "@include \"parts/data.gad\"\n@main\n    p {tr(\"from_module\")}\n")
	write("main.gad", "layouts := import(\"layouts/*\")::dict\nreturn layouts.one.main\n")

	type call struct {
		file      string
		line, col int
	}
	var (
		calls []call
		files []string
	)
	r := newTestRender(t, dir)
	r.Inspect = func(srcPath string, file *parser.File) error {
		rel, _ := filepath.Rel(dir, srcPath)
		files = append(files, filepath.ToSlash(rel))
		node.Walk(file, func(n ast.Node) bool {
			if c, ok := n.(*node.CallExpr); ok {
				if id, ok := c.Func.(*node.IdentExpr); ok && id.Name == "tr" {
					p := file.InputFile.Set().Position(c.Pos())
					calls = append(calls, call{filepath.ToSlash(rel), p.Line, p.Column})
				}
			}
			return true
		})
		return nil
	}

	if _, err := renderString(r, filepath.Join(dir, "main.gad"), gad.Dict{}); err != nil {
		t.Fatal(err)
	}
	want := []call{{"layouts/one.gadx", 3, 8}, {"parts/data.gad", 2, 10}}
	slices.SortFunc(calls, func(a, b call) int { return compareStr(a.file, b.file) })
	if !slices.Equal(calls, want) {
		t.Errorf("calls %v, want %v", calls, want)
	}
	slices.Sort(files)
	if want := []string{"layouts/one.gadx", "main.gad", "parts/data.gad"}; !slices.Equal(files, want) {
		t.Errorf("inspected %v, want %v", files, want)
	}
}

func compareStr(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}
