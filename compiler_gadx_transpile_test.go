package gad_test

import (
	"strings"
	"testing"

	gad "github.com/gad-lang/gad"
	_ "github.com/gad-lang/gad/gadx" // installs the Markdown + HTML-parse hooks
)

// TestTranspileGadxHyphenAttr covers a tag attribute whose name is not a valid
// Gad identifier (e.g. `data-line`): the lowered named-argument key must be a
// quoted string, else the transpiled Gad re-parses as `data - line` and fails.
func TestTranspileGadxHyphenAttr(t *testing.T) {
	src := "@main\n    div.box[data-line=5, aria-label=\"hi\"]\n        p ok\n"
	out, err := gad.TranspileGadxSource("h.gadx", []byte(src))
	if err != nil {
		t.Fatalf("transpile: %v", err)
	}
	got := string(out)
	for _, w := range []string{`"data-line"=`, `"aria-label"=`} {
		if !strings.Contains(got, w) {
			t.Fatalf("hyphenated attribute key not quoted (%q missing):\n%s", w, got)
		}
	}
	// The bug emitted the key unquoted (`data-line=…`), which re-parses as the
	// expression `data - line` — so the unquoted form must NOT appear.
	if strings.Contains(got, "data-line=") && !strings.Contains(got, `"data-line"=`) {
		t.Fatalf("hyphenated attribute key emitted unquoted:\n%s", got)
	}
}

// TestTranspileGadxMarkdown covers the `@md` lowering: the Markdown is rendered
// to HTML at transpile time and parsed into gadx.Tag/gadx.Text nodes (not a
// runtime gadx.Md container), with interpolations preserved as dynamic values
// and inline HTML flowing through as tags.
func TestTranspileGadxMarkdown(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want []string // substrings the transpiled Gad must contain
	}{
		{
			name: "static",
			src:  "@main\n    @md\n        # Hello\n\n        A **static** line.\n",
			want: []string{`"h1"`, `"strong"`, `"static"`},
		},
		{
			name: "interpolation",
			src:  "@main\n    @md\n        # {= title }\n",
			want: []string{`"h1"`, "title"},
		},
		{
			name: "inline-html",
			src:  "@main\n    @md\n        A <span class=\"x\">tag</span> here.\n",
			want: []string{`"span"`, `class="x"`},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := gad.TranspileGadxSource(c.name+".gadx", []byte(c.src))
			if err != nil {
				t.Fatalf("transpile: %v", err)
			}
			got := string(out)
			if strings.Contains(got, "gadx.Md(") {
				t.Fatalf("@md must lower to gadx.Tag, not a runtime gadx.Md container:\n%s", got)
			}
			for _, w := range c.want {
				if !strings.Contains(got, w) {
					t.Fatalf("transpiled output missing %q:\n%s", w, got)
				}
			}
		})
	}
}
