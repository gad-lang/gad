package gadx

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gad-lang/gad"
)

// dirModule is a module the application makes of a directory of its own (a
// SourceFiles): the names of the .gad files in it, and their contents.
type dirModule struct{ dir string }

func (m *dirModule) SourceFiles() []string { return []string{m.dir} }

func (m *dirModule) Import(context.Context, *gad.ModuleSpec) (any, string, error) {
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		return nil, "", err
	}
	var parts []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".gad") {
			data, err := os.ReadFile(filepath.Join(m.dir, e.Name()))
			if err != nil {
				return nil, "", err
			}
			parts = append(parts, fmt.Sprintf("%q", e.Name()+"="+strings.TrimSpace(string(data))))
		}
	}
	sort.Strings(parts)
	src := "export names = [" + strings.Join(parts, ", ") + "]"
	return gad.SourceCode{Data: []byte(src), Kind: gad.SourceKindGad}, m.dir, nil
}

// rerender renders tpl twice — the first stamps the change, the second, past
// the delay, compiles it — and returns the second's output.
func rerender(t *testing.T, r *Render, tpl string) string {
	t.Helper()
	_, _ = renderString(r, tpl, gad.Dict{})
	time.Sleep(15 * time.Millisecond)
	out, err := renderString(r, tpl, gad.Dict{})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// A module made of a directory (a SourceFiles naming it): a template that
// imports it recompiles when a file is added to the directory, written, or
// taken from it — or from a directory below it.
func TestRenderRecompilesOnSourceFilesDirChange(t *testing.T) {
	dir, other := t.TempDir(), t.TempDir()
	base := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	writeFileWithMtime(t, filepath.Join(other, "a.gad"), `1`, base)

	tpl := filepath.Join(dir, "page.gadx")
	writeFileWithMtime(t, tpl, `@import "cfg" as cfg
@main
    p {= str(cfg.names) }`, base)

	r := newTestRender(t, dir)
	r.ModuleMapFunc = func(mm *gad.ModuleMap) *gad.ModuleMap {
		return mm.Add("cfg", &dirModule{dir: other})
	}
	out, err := renderString(r, tpl, gad.Dict{})
	if err != nil || !strings.Contains(out, "a.gad=1") {
		t.Fatalf("first: %q %v", out, err)
	}

	// added
	writeFileWithMtime(t, filepath.Join(other, "b.gad"), `2`, time.Now())
	if out = rerender(t, r, tpl); !strings.Contains(out, "b.gad=2") {
		t.Fatalf("a file added, not recompiled: %q", out)
	}
	// written (its size the same: by its time)
	writeFileWithMtime(t, filepath.Join(other, "b.gad"), `3`, time.Now().Add(time.Second))
	if out = rerender(t, r, tpl); !strings.Contains(out, "b.gad=3") {
		t.Fatalf("a file written, not recompiled: %q", out)
	}
	// taken
	if err := os.Remove(filepath.Join(other, "a.gad")); err != nil {
		t.Fatal(err)
	}
	if out = rerender(t, r, tpl); strings.Contains(out, "a.gad") {
		t.Fatalf("a file taken, not recompiled: %q", out)
	}
	// below it: the whole tree
	if err := os.Mkdir(filepath.Join(other, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	before, _ := dirSignature(other, -1)
	writeFileWithMtime(t, filepath.Join(other, "sub", "c.gad"), `4`, time.Now())
	if after, _ := dirSignature(other, -1); after == before {
		t.Fatal("a file in a directory below: the signature did not change")
	}
}

// A glob import depends on its directory's listing, not only on the files it
// read: a module added to the directory recompiles the template.
func TestRenderGlobImportRecompilesOnNewFile(t *testing.T) {
	dir := t.TempDir()
	base := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Mkdir(filepath.Join(dir, "parts"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFileWithMtime(t, filepath.Join(dir, "parts", "a.gad"), `export name = "a"`, base)

	tpl := filepath.Join(dir, "page.gadx")
	writeFileWithMtime(t, tpl, `~ parts := import("parts/*.gad")::dict
@main
    p {= str(sort(collect(keys(parts)))) }`, base)

	r := newTestRender(t, dir)
	out, err := renderString(r, tpl, gad.Dict{})
	if err != nil || !strings.Contains(out, "a") {
		t.Fatalf("first: %q %v", out, err)
	}
	writeFileWithMtime(t, filepath.Join(dir, "parts", "b.gad"), `export name = "b"`, time.Now())
	if out = rerender(t, r, tpl); !strings.Contains(out, "&#34;b&#34;") {
		t.Fatalf("a module added to the glob's directory, not recompiled: %q", out)
	}
	if err := os.Remove(filepath.Join(dir, "parts", "b.gad")); err != nil {
		t.Fatal(err)
	}
	if out = rerender(t, r, tpl); strings.Contains(out, "&#34;b&#34;") {
		t.Fatalf("a module taken from the glob's directory, not recompiled: %q", out)
	}
}
