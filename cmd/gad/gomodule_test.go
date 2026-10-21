package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gad-lang/gad/internal/gaddoc"
)

// `gad gomodule` of samples/modules/gomodule/shapes.gad (issue #17): the
// package it writes is the one in the repository (but the hand-written
// init.go), and its API stub is `gaddoc api`'s — the docs round-trip.
func TestGomoduleShapes(t *testing.T) {
	const sample = "../../samples/modules/gomodule"
	dir := t.TempDir()
	init, err := os.ReadFile(filepath.Join(sample, "shapes", "init.go"))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "init.go"), init, 0o644); err != nil {
		t.Fatal(err)
	}
	if err = runGomodule(filepath.Join(sample, "shapes.gad"), gomoduleOpts{out: dir}); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"shapes.gadb", "shapes_test.go", "shapes.gad"} {
		got, err := os.ReadFile(filepath.Join(dir, n))
		if err != nil {
			t.Fatal(err)
		}
		want, err := os.ReadFile(filepath.Join(sample, "shapes", n))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(want) {
			t.Errorf("%s differs from the repository's: run `go generate ./samples/modules/gomodule/shapes`", n)
		}
	}
	// shapes.go: but the go:generate line (the source's path from the output)
	got, _ := os.ReadFile(filepath.Join(dir, "shapes.go"))
	want, _ := os.ReadFile(filepath.Join(sample, "shapes", "shapes.go"))
	strip := func(b []byte) string {
		var out []string
		for _, l := range strings.Split(string(b), "\n") {
			if !strings.HasPrefix(l, "//go:generate") {
				out = append(out, l)
			}
		}
		return strings.Join(out, "\n")
	}
	if strip(got) != strip(want) {
		t.Errorf("shapes.go differs from the repository's: run `go generate ./samples/modules/gomodule/shapes`")
	}

	api, _, err := gaddoc.API(dir, "shapes")
	if err != nil {
		t.Fatal(err)
	}
	stub, _ := os.ReadFile(filepath.Join(dir, "shapes.gad"))
	if api != string(stub) {
		t.Fatalf("the API stub is not gaddoc api's")
	}
}

// A module that imports a file cannot be made a Go module: the host cannot
// resolve it.
func TestGomoduleFileImport(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "m.gad")
	if err := os.WriteFile(src, []byte(`x := import("./other.gad"); export y = 1`), 0o644); err != nil {
		t.Fatal(err)
	}
	err := runGomodule(src, gomoduleOpts{out: filepath.Join(dir, "m")})
	if err == nil || !strings.Contains(err.Error(), "other.gad") {
		t.Fatalf("want an error of the file import, got %v", err)
	}
}
