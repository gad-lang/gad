package giom

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gad-lang/gad"
)

// renderGiom writes src to a temp .giom file and renders it.
func renderGiom(t *testing.T, src string, globals gad.Dict) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "t.giom")
	if err := os.WriteFile(p, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	r := newTestRender(t, dir)
	out, err := renderString(r, p, globals)
	if err != nil {
		t.Fatalf("render: %v\nsrc:\n%s", err, src)
	}
	return out
}

// TestAttributeGroupRendering verifies that single, multi-value and multi-line
// attribute groups all render, and that separators inside values are ignored.
func TestAttributeGroupRendering(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "single",
			src:  "@main\n    div[class=\"a\"] hi\n",
			want: `<div class="a">hi</div>`,
		},
		{
			name: "multi-value comma",
			src:  "@main\n    a[href=\"/x\", title=\"go\"] link\n",
			want: `<a href="/x" title="go">link</a>`,
		},
		{
			name: "multi-line",
			src: "@main\n" +
				"    a[\n" +
				"        href=\"/x\"\n" +
				"        title=\"go\"\n" +
				"    ] link\n",
			want: `<a href="/x" title="go">link</a>`,
		},
		{
			name: "flag and expression",
			src:  "@main\n    input[type=\"text\", disabled, value=1+2]\n",
			want: `<input type="text" disabled="disabled" value="3" />`,
		},
		{
			name: "comma inside value not split",
			src:  "@main\n    div[title=[1, 2][0], class=\"c\"] x\n",
			want: `<div title="1" class="c">x</div>`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := renderGiom(t, tc.src, gad.Dict{})
			if got != tc.want {
				t.Fatalf("render mismatch\n got: %s\nwant: %s", got, tc.want)
			}
		})
	}
}

// TestAttributeExpressionPosition verifies a runtime error inside a multi-line
// attribute value reports the correct source line and column.
func TestAttributeExpressionPosition(t *testing.T) {
	src := "@global bad\n" +
		"@main\n" +
		"    div[\n" +
		"        class=\"a\"\n" +
		"        title=bad()\n" + // line 5, col 18 (the call `(`)
		"    ] hi\n"
	re := runForError(t, src)
	line, col := firstTraceLineCol(re)
	if line != 5 || col != 18 {
		t.Fatalf("attribute nil-call resolved to %d:%d, want 5:18\ntrace:\n%+v", line, col, re.StackTrace())
	}
}
