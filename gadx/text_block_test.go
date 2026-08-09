package gadx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gad-lang/gad"
)

// TestRenderTextBlock verifies that `@text` emits its indented body verbatim,
// joining lines with newlines and preserving blank lines as paragraph breaks.
func TestRenderTextBlock(t *testing.T) {
	dir := t.TempDir()
	src := "@main\n    @text\n        a\n        b\n\n        c\n"
	srcPath := filepath.Join(dir, "text.gadx")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	r := newTestRender(t, dir)
	out, err := renderString(r, srcPath, gad.Dict{})
	if err != nil {
		t.Fatal(err)
	}
	if out != "a\nb\n\nc" {
		t.Fatalf("expected %q, got %q", "a\nb\n\nc", out)
	}
}

// TestRenderTextBlockInterp verifies `{= … }` interpolation still works inside a
// `@text` body (source positions are preserved per line).
func TestRenderTextBlockInterp(t *testing.T) {
	dir := t.TempDir()
	src := "@main\n    @text\n        value: {= 1 + 1 }\n        done\n"
	srcPath := filepath.Join(dir, "text_interp.gadx")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	r := newTestRender(t, dir)
	out, err := renderString(r, srcPath, gad.Dict{})
	if err != nil {
		t.Fatal(err)
	}
	if out != "value: 2\ndone" {
		t.Fatalf("expected %q, got %q", "value: 2\ndone", out)
	}
}

// TestRenderParaBlock verifies `@p` groups non-blank lines into <p> paragraphs
// separated by blank lines.
func TestRenderParaBlock(t *testing.T) {
	dir := t.TempDir()
	src := "@main\n    @p\n        a\n        b\n\n        c\n"
	srcPath := filepath.Join(dir, "para.gadx")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	r := newTestRender(t, dir)
	out, err := renderString(r, srcPath, gad.Dict{})
	if err != nil {
		t.Fatal(err)
	}
	if out != "<p>a\nb</p><p>c</p>" {
		t.Fatalf("expected %q, got %q", "<p>a\nb</p><p>c</p>", out)
	}
}

// TestRenderMdBlock verifies `@md` renders its Markdown body to HTML, supports
// `{= … }` interpolation and nested `@` directives inline, keeps Markdown after a
// nested directive as its own block, and preserves indentation for code blocks.
func TestRenderMdBlock(t *testing.T) {
	dir := t.TempDir()
	src := "@main\n" +
		"    @md\n" +
		"        # Title {= 1 + 1 }\n" +
		"\n" +
		"        Some **bold** text.\n" +
		"\n" +
		"            code kept\n" +
		"\n" +
		"        @p\n" +
		"            nested para\n" +
		"\n" +
		"        > quote after directive\n"
	srcPath := filepath.Join(dir, "md.gadx")
	if err := os.WriteFile(srcPath, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	r := newTestRender(t, dir)
	out, err := renderString(r, srcPath, gad.Dict{})
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range []string{
		"<h1", "Title 2", "<strong>bold</strong>",
		"<pre><code>code kept", // 4-space indent preserved as a code block
		"<p>nested para</p>",    // nested @p rendered inline
		"<blockquote>",          // Markdown after the nested directive still converts
	} {
		if !strings.Contains(out, w) {
			t.Fatalf("md render missing %q:\n%s", w, out)
		}
	}
}
