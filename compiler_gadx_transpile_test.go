package gad_test

import (
	"strings"
	"testing"

	gad "github.com/gad-lang/gad"
	_ "github.com/gad-lang/gad/gadx" // registers the transpile-time Markdown renderer hook
)

// TestTranspileGadxStaticMarkdown covers transpile-time `@md` handling: a fully
// static block is pre-rendered to raw HTML, while a block with interpolation or
// a nested directive keeps the runtime gadx.Md container.
func TestTranspileGadxStaticMarkdown(t *testing.T) {
	static := "@main\n" +
		"    @md\n" +
		"        # Hello\n" +
		"\n" +
		"        A **static** line.\n"
	out, err := gad.TranspileGadxSource("static.gadx", []byte(static))
	if err != nil {
		t.Fatalf("transpile static: %v", err)
	}
	got := string(out)
	if strings.Contains(got, "gadx.Md(") {
		t.Fatalf("static @md should not emit a runtime gadx.Md container:\n%s", got)
	}
	if !strings.Contains(got, "<h1") || !strings.Contains(got, "<strong>static</strong>") {
		t.Fatalf("static @md should be pre-rendered to HTML:\n%s", got)
	}
	if !strings.Contains(got, `raw "`) {
		t.Fatalf("pre-rendered HTML must be written raw (unescaped):\n%s", got)
	}

	// Interpolation -> dynamic -> keep the runtime container.
	dynamic := "@main\n" +
		"    @md\n" +
		"        # {= title }\n"
	out, err = gad.TranspileGadxSource("dyn.gadx", []byte(dynamic))
	if err != nil {
		t.Fatalf("transpile dynamic: %v", err)
	}
	if !strings.Contains(string(out), "gadx.Md(") {
		t.Fatalf("interpolated @md must keep the runtime gadx.Md container:\n%s", out)
	}

	// A nested `@` directive -> dynamic -> keep the runtime container.
	nested := "@main\n" +
		"    @md\n" +
		"        # Title\n" +
		"\n" +
		"        @p\n" +
		"            inner\n"
	out, err = gad.TranspileGadxSource("nested.gadx", []byte(nested))
	if err != nil {
		t.Fatalf("transpile nested: %v", err)
	}
	if !strings.Contains(string(out), "gadx.Md(") {
		t.Fatalf("@md with a nested directive must keep the runtime gadx.Md container:\n%s", out)
	}
}
