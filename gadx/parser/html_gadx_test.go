package parser

import (
	"bytes"
	"strings"
	"testing"

	gadxnode "github.com/gad-lang/gad/gadx/node"
	"github.com/gad-lang/gad/parser/source"
)

// transpileGadx parses gadx source and re-emits it as pug-style gadx via
// WriteGadx, for asserting the HTML → pug transpilation.
func transpileGadx(t *testing.T, src string) string {
	t.Helper()
	fs := source.NewFileSet()
	f := fs.AddFileData("test.gadx", -1, []byte(src))
	file, err := NewParser(f).ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var buf bytes.Buffer
	file.WriteGadx(gadxnode.NewGadxCodeContext(&buf))
	return buf.String()
}

// TestHtmlWriteGadx checks that an HTML region transpiles to pug-style gadx
// (tags, `.class`/`#id`, `[attr]` groups, `| text`) rather than raw HTML.
func TestHtmlWriteGadx(t *testing.T) {
	out := transpileGadx(t, "@main\n    <a href=\"/x\" title=\"hi\">Home</a>\n")
	// Attributes merge into one group and a single text body is inlined on the tag.
	for _, want := range []string{`a[href="/x", title="hi"] Home`} {
		if !strings.Contains(out, want) {
			t.Fatalf("transpiled gadx missing %q:\n%s", want, out)
		}
	}
	// It must not fall back to raw HTML markup.
	if strings.Contains(out, "<a") || strings.Contains(out, "</a>") {
		t.Fatalf("transpiled gadx still contains raw HTML:\n%s", out)
	}
}

// TestHtmlWriteGadxInterleave checks that a gadx statement interleaved inside an
// HTML region transpiles back as a gadx directive (not raw HTML).
func TestHtmlWriteGadxInterleave(t *testing.T) {
	out := transpileGadx(t, "@main\n    <ul>\n        @for x in [1, 2]\n            <li>{x}</li>\n    </ul>\n")
	// An HTML-region `{x}` interpolation outputs its value, so it transpiles to
	// the pug output form `{= x }` (with `=`), not a no-output `{x}`.
	for _, want := range []string{"ul", "@for x in [1, 2]", "li", "{= x }"} {
		if !strings.Contains(out, want) {
			t.Fatalf("transpiled gadx missing %q:\n%s", want, out)
		}
	}
}

// TestHtmlWriteGadxNested checks nested elements and an interpolated attribute.
func TestHtmlWriteGadxNested(t *testing.T) {
	out := transpileGadx(t, "@global u\n@main\n    <ul><li><a href={u}>x</a></li></ul>\n")
	// The innermost `<a href={u}>x</a>` inlines its single text: `a[href=u] x`.
	for _, want := range []string{"ul", "li", "a[href=u] x"} {
		if !strings.Contains(out, want) {
			t.Fatalf("transpiled gadx missing %q:\n%s", want, out)
		}
	}
}

// TestTagInlineTextThenChildren covers a tag whose body is inline text FOLLOWED by
// an indented block: `h2 Example —\n    code file` — the leading text and the child
// tags are both children of the tag (previously an "unexpected INDENT" parse error).
func TestTagInlineTextThenChildren(t *testing.T) {
	out := transpileGadx(t, "@main\n    h2 Example —\n        code file.gad\n")
	// The inline text and the code child both become children of the h2 (the
	// transpiler normalises the inline text to a `| ` line under the tag).
	for _, want := range []string{"h2", "| Example —", "code file.gad"} {
		if !strings.Contains(out, want) {
			t.Fatalf("transpiled gadx missing %q:\n%s", want, out)
		}
	}
}
