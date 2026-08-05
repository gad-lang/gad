//go:build !js
// +build !js

package main

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/gad-lang/gad"
)

// runScriptCapture compiles+runs src as a script with modulePath and args,
// capturing os.Stdout (where println writes).
func runScriptCapture(t *testing.T, modulePath, src string, args []string) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	builtins := gad.NewBuiltins().Build()
	s := newScript(builtins, context.Background(), modulePath, ".", []byte(src), io.Discard)
	s.args = args
	runErr := s.execute()

	_ = w.Close()
	os.Stdout = orig
	out, _ := io.ReadAll(r)
	if runErr != nil {
		t.Fatalf("execute: %v\noutput: %s", runErr, out)
	}
	return string(out)
}

// TestRawArgvPassthrough checks that `param (*argv)` receives every CLI arg
// unparsed with argv[0] = the module path used to invoke it. The script runs as
// the main module, so the bare `--` options terminator is dropped.
func TestRawArgvPassthrough(t *testing.T) {
	out := runScriptCapture(t, "a/b/script.gad", "param (*argv)\nprintln(argv)",
		[]string{"x", "--y=1", "--", "z"})
	want := `["a/b/script.gad", "x", "--y=1", "z"]`
	if !strings.Contains(out, want) {
		t.Fatalf("raw argv passthrough:\n got %q\nwant contains %q", out, want)
	}
}

// TestDropFirstOptionsTerminator checks the `--` removal helper: only the first
// bare `--` is dropped; `--flag` tokens are untouched.
func TestDropFirstOptionsTerminator(t *testing.T) {
	got := dropFirstOptionsTerminator([]string{"x", "--y=1", "--", "z", "--"})
	want := []string{"x", "--y=1", "z", "--"}
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("got %v, want %v", got, want)
	}
	// No terminator: unchanged.
	same := dropFirstOptionsTerminator([]string{"a", "b"})
	if strings.Join(same, "\x00") != "a\x00b" {
		t.Fatalf("unchanged case: got %v", same)
	}
}

// TestNonRawArgvStillParses checks that a normal `param (...)` keeps the typed
// `--name=value` parsing (no passthrough): --count=5 becomes the int 5.
func TestNonRawArgvStillParses(t *testing.T) {
	out := runScriptCapture(t, "n.gad", "param (name; count=0)\nprintln([name, count])",
		[]string{"alice", "--count=5"})
	if !strings.Contains(out, `["alice", 5]`) {
		t.Fatalf("normal parsing: got %q, want contains [\"alice\", 5]", out)
	}
}
