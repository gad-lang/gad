package gadx

import (
	"strings"
	"testing"
)

// TestBlockComment covers `/* … */` block comments: silent, recognized only at
// line start, and spanning multiple lines.
func TestBlockComment(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "single line, before content",
			src:  "/* a comment */\n@main\n    p hi\n",
			want: `<p>hi</p>`,
		},
		{
			name: "multi-line",
			src:  "/*\n multi\n line\n*/\n@main\n    p hi\n",
			want: `<p>hi</p>`,
		},
		{
			name: "doc form is also silent",
			src:  "/** doc **/\n@main\n    p hi\n",
			want: `<p>hi</p>`,
		},
		{
			name: "between statements",
			src:  "@main\n    p a\n    /* mid */\n    p b\n",
			want: `<p>a</p><p>b</p>`,
		},
		{
			name: "empty block",
			src:  "/**/\n@main\n    p hi\n",
			want: `<p>hi</p>`,
		},
		{
			// `/*` mid-line is not a comment: it stays literal text.
			name: "mid-line is literal",
			src:  "@main\n    p a /* not a comment */\n",
			want: `<p>a /* not a comment */</p>`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := strings.TrimSpace(renderGadx(t, tc.src, nil))
			if got != tc.want {
				t.Fatalf("render mismatch\n got: %q\nwant: %q", got, tc.want)
			}
		})
	}
}
