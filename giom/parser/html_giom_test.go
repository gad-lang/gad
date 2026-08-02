package parser

import (
	"bytes"
	"strings"
	"testing"

	giomnode "github.com/gad-lang/gad/giom/node"
	"github.com/gad-lang/gad/parser/source"
)

// transpileGiom parses giom source and re-emits it as pug-style giom via
// WriteGiom, for asserting the HTML → pug transpilation.
func transpileGiom(t *testing.T, src string) string {
	t.Helper()
	fs := source.NewFileSet()
	f := fs.AddFileData("test.giom", -1, []byte(src))
	file, err := NewParser(f).ParseFile()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var buf bytes.Buffer
	file.WriteGiom(giomnode.NewGiomCodeContext(&buf))
	return buf.String()
}

// TestHtmlWriteGiom checks that an HTML region transpiles to pug-style giom
// (tags, `.class`/`#id`, `[attr]` groups, `| text`) rather than raw HTML.
func TestHtmlWriteGiom(t *testing.T) {
	out := transpileGiom(t, "@main\n    <a href=\"/x\" title=\"hi\">Home</a>\n")
	for _, want := range []string{"a", `[href="/x"]`, `[title="hi"]`, "| Home"} {
		if !strings.Contains(out, want) {
			t.Fatalf("transpiled giom missing %q:\n%s", want, out)
		}
	}
	// It must not fall back to raw HTML markup.
	if strings.Contains(out, "<a") || strings.Contains(out, "</a>") {
		t.Fatalf("transpiled giom still contains raw HTML:\n%s", out)
	}
}

// TestHtmlWriteGiomInterleave checks that a giom statement interleaved inside an
// HTML region transpiles back as a giom directive (not raw HTML).
func TestHtmlWriteGiomInterleave(t *testing.T) {
	out := transpileGiom(t, "@main\n    <ul>\n        @for x in [1, 2]\n            <li>{x}</li>\n    </ul>\n")
	for _, want := range []string{"ul", "@for (x in [1, 2])", "li", "{x}"} {
		if !strings.Contains(out, want) {
			t.Fatalf("transpiled giom missing %q:\n%s", want, out)
		}
	}
}

// TestHtmlWriteGiomNested checks nested elements and an interpolated attribute.
func TestHtmlWriteGiomNested(t *testing.T) {
	out := transpileGiom(t, "@global u\n@main\n    <ul><li><a href={u}>x</a></li></ul>\n")
	for _, want := range []string{"ul", "li", "a", "[href=u]", "| x"} {
		if !strings.Contains(out, want) {
			t.Fatalf("transpiled giom missing %q:\n%s", want, out)
		}
	}
}
