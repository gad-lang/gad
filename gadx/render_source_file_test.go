package gadx

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gad-lang/gad"
)

// fileModule is a module the application adds from a file of its own, out of
// the template tree.
type fileModule struct{ path string }

func (m *fileModule) SourceFile() string { return m.path }

func (m *fileModule) Import(context.Context, *gad.ModuleSpec) (any, string, error) {
	src, err := os.ReadFile(m.path)
	if err != nil {
		return nil, "", err
	}
	return gad.SourceCode{Data: src, Kind: gad.SourceKindGad}, m.path, nil
}

// A module read from a file (a SourceFile) out of the template tree: a
// template that imports it recompiles when the file changes, and does not
// compile while it is missing.
func TestRenderRecompilesOnSourceFileChange(t *testing.T) {
	dir, other := t.TempDir(), t.TempDir()
	base := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	cfg := filepath.Join(other, "cfg.gad")
	writeFileWithMtime(t, cfg, `export name = "one"`, base)

	tpl := filepath.Join(dir, "page.gadx")
	writeFileWithMtime(t, tpl, `@import "cfg" as cfg
@main
    p {= cfg.name }`, base)

	r := newTestRender(t, dir)
	r.ModuleMapFunc = func(mm *gad.ModuleMap) *gad.ModuleMap {
		return mm.Add("cfg", &fileModule{path: cfg})
	}

	out, err := renderString(r, tpl, gad.Dict{})
	if err != nil || !strings.Contains(out, "one") {
		t.Fatalf("first: %q %v", out, err)
	}

	writeFileWithMtime(t, cfg, `export name = "two"`, time.Now())
	_, _ = renderString(r, tpl, gad.Dict{}) // stamps the change
	time.Sleep(15 * time.Millisecond)
	if out, err = renderString(r, tpl, gad.Dict{}); err != nil || !strings.Contains(out, "two") {
		t.Fatalf("the file changed, not recompiled: %q %v", out, err)
	}

	// gone: the template that imports it does not compile
	if err := os.Remove(cfg); err != nil {
		t.Fatal(err)
	}
	_, _ = renderString(r, tpl, gad.Dict{})
	time.Sleep(15 * time.Millisecond)
	if _, err = renderString(r, tpl, gad.Dict{}); err == nil {
		t.Fatal("the file is gone, and the template still compiles")
	}
}
