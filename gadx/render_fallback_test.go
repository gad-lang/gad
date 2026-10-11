package gadx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/gad-lang/gad"
)

// fallbackTree is a library's templates: a page, the components it imports
// (one by a name without an extension), the parts of a glob.
func fallbackTree() fstest.MapFS {
	return fstest.MapFS{
		"page.gadx": {Data: []byte(`@import { main: box } from "parts/box.gadx"
@import { main: tag } from "parts/tag"
@main
    div
        +box
        +tag`)},
		"parts/box.gadx": {Data: []byte(`@export comp main
    p.box lib`)},
		"parts/tag.gadx": {Data: []byte(`@export comp main
    span.tag lib`)},
		"list/a.gadx": {Data: []byte(`@export comp main
    i a-lib`)},
		"list/b.gadx": {Data: []byte(`@export comp main
    i b-lib`)},
	}
}

// What the templates' tree lacks is read from the Fallback, at the same path:
// the rendered file, what it imports; a file of the tree wins, file by file —
// one imported by a file of the Fallback too.
func TestRenderFallback(t *testing.T) {
	dir := t.TempDir()
	r := newTestRender(t, dir)
	r.Fallback = fallbackTree()

	out, err := renderString(r, filepath.Join(dir, "page.gadx"), gad.Dict{})
	if err != nil {
		t.Fatal(err)
	}
	if want := `<div><p class="box">lib</p><span class="tag">lib</span></div>`; out != want {
		t.Fatalf("all of the Fallback: %q, want %q", out, want)
	}

	// the application's own box: the Fallback's page imports it
	dir2 := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir2, "parts"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFileWithMtime(t, filepath.Join(dir2, "parts", "box.gadx"), `@export comp main
    p.box app`, time.Now())
	r2 := newTestRender(t, dir2)
	r2.Fallback = fallbackTree()
	out, err = renderString(r2, filepath.Join(dir2, "page.gadx"), gad.Dict{})
	if err != nil {
		t.Fatal(err)
	}
	if want := `<div><p class="box">app</p><span class="tag">lib</span></div>`; out != want {
		t.Fatalf("the application's file wins: %q, want %q", out, want)
	}

	// none in either: not found
	if _, err := renderString(r, filepath.Join(dir, "none.gadx"), gad.Dict{}); err == nil {
		t.Fatal("a template in neither tree rendered")
	}
	// none without a Fallback
	if _, err := renderString(newTestRender(t, dir), filepath.Join(dir, "page.gadx"), gad.Dict{}); err == nil {
		t.Fatal("rendered from a Fallback it was not given")
	}
}

// A glob matches the files of both trees — a name in both, the tree's.
func TestRenderFallbackGlob(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "list"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFileWithMtime(t, filepath.Join(dir, "list", "b.gadx"), `@export comp main
    i b-app`, time.Now())
	writeFileWithMtime(t, filepath.Join(dir, "list", "c.gadx"), `@export comp main
    i c-app`, time.Now())
	writeFileWithMtime(t, filepath.Join(dir, "glob.gadx"), `~ parts := import("list/*.gadx")::dict
@main
    div
        @for k in sort(collect(keys(parts)))
            ~ part := parts[k].main
            +part`, time.Now())
	r := newTestRender(t, dir)
	r.Fallback = fallbackTree()
	out, err := renderString(r, filepath.Join(dir, "glob.gadx"), gad.Dict{})
	if err != nil {
		t.Fatal(err)
	}
	if want := `<div><i>a-lib</i><i>b-app</i><i>c-app</i></div>`; out != want {
		t.Fatalf("the glob: %q, want %q", out, want)
	}
}

// A file of the Fallback never changes; one the tree writes in its place
// recompiles what read it.
func TestRenderFallbackOverrideRecompiles(t *testing.T) {
	dir := t.TempDir()
	r := newTestRender(t, dir)
	r.Fallback = fallbackTree()
	page := filepath.Join(dir, "page.gadx")
	if _, err := renderString(r, page, gad.Dict{}); err != nil {
		t.Fatal(err)
	}
	// rendered again: nothing changed (the Fallback's are not "gone")
	var compiles int
	r.OnRender(func(bool, string, []string, time.Time, error) { compiles++ })
	time.Sleep(20 * time.Millisecond)
	if _, err := renderString(r, page, gad.Dict{}); err != nil {
		t.Fatal(err)
	}
	if compiles != 0 {
		t.Fatalf("recompiled with nothing changed: %d", compiles)
	}
	if err := os.MkdirAll(filepath.Join(dir, "parts"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFileWithMtime(t, filepath.Join(dir, "parts", "tag.gadx"), `@export comp main
    span.tag app`, time.Now().Add(-time.Second))
	var out string
	for i := 0; i < 3 && !strings.Contains(out, "app"); i++ {
		time.Sleep(20 * time.Millisecond)
		var err error
		if out, err = renderString(r, page, gad.Dict{}); err != nil {
			t.Fatal(err)
		}
	}
	if !strings.Contains(out, `<span class="tag">app</span>`) {
		t.Fatalf("the override not taken: %q", out)
	}
}
