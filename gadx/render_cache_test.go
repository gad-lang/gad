package gadx

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gad-lang/gad"
)

func renderTo(t *testing.T, r *Render, file string) string {
	t.Helper()
	var buf bytes.Buffer
	if err := r.Render(&buf, file, gad.Dict{}); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

// writeTemplate writes the file and backdates it by age, standing in for an
// edit saved that long ago without the test having to sleep.
func writeTemplate(t *testing.T, path, body string, age time.Duration) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	when := time.Now().Add(-age)
	if err := os.Chtimes(path, when, when); err != nil {
		t.Fatal(err)
	}
}

// An edit saved longer ago than the delay is picked up by the very next render.
// The delay is there to skip a file still being written, so it is measured from
// the file's mtime; measuring it from the moment a render first noticed made
// the wait restart on each page load, and the first render after an edit always
// served the previous build.
func TestRenderRecompilesEditOlderThanDelay(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.gad")

	writeTemplate(t, file, `println("A")`, 0)

	r := NewRender(dir)
	r.TemplateDelay = 500 * time.Millisecond

	if got := renderTo(t, r, file); got != "A\n" {
		t.Fatalf("first render = %q, want %q", got, "A\n")
	}

	writeTemplate(t, file, `println("B")`, 3*time.Second)

	if got := renderTo(t, r, file); got != "B\n" {
		t.Errorf("render after a settled edit = %q, want %q", got, "B\n")
	}
}

// A file written just now is left alone until the delay passes, so a template
// caught mid-write is not compiled half-formed.
func TestRenderHoldsFreshEditUntilDelay(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "main.gad")

	writeTemplate(t, file, `println("A")`, 0)

	r := NewRender(dir)
	r.TemplateDelay = 500 * time.Millisecond

	renderTo(t, r, file)

	writeTemplate(t, file, `println("B")`, 0)

	if got := renderTo(t, r, file); got != "A\n" {
		t.Errorf("render right after the edit = %q, want the previous build %q", got, "A\n")
	}

	time.Sleep(600 * time.Millisecond)

	if got := renderTo(t, r, file); got != "B\n" {
		t.Errorf("render after the delay = %q, want %q", got, "B\n")
	}
}
