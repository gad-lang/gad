package giom

import (
	"testing"

	"github.com/gad-lang/gad"
)

// TestGlobalDeclarations covers `@global` in its forms: legacy space-separated
// names, single, comma-separated with `=` (nil-or-absent) and `!?=`
// (absent-only) defaults, and the multi-line parenthesized form.
func TestGlobalDeclarations(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		globals gad.Dict
		want    string
	}{
		{
			name:    "legacy space-separated",
			src:     "@global a b\n@main\n    p {= a + b }\n",
			globals: gad.Dict{"a": gad.Int(3), "b": gad.Int(4)},
			want:    "<p>7</p>",
		},
		{
			name:    "single",
			src:     "@global x\n@main\n    p {= x }\n",
			globals: gad.Dict{"x": gad.Int(9)},
			want:    "<p>9</p>",
		},
		{
			name:    "comma with = default (absent filled)",
			src:     "@global a, b = 5\n@main\n    p {= (a ?? 0) + b }\n",
			globals: gad.Dict{"a": gad.Int(1)},
			want:    "<p>6</p>",
		},
		{
			name:    "!?= default (absent filled)",
			src:     "@global c !?= 7\n@main\n    p {= c }\n",
			globals: gad.Dict{},
			want:    "<p>7</p>",
		},
		{
			name:    "!?= default keeps host nil",
			src:     "@global c !?= 7\n@main\n    p {= c ?? \"nil\" }\n",
			globals: gad.Dict{"c": gad.Nil},
			want:    "<p>nil</p>",
		},
		{
			name:    "multi-line paren",
			src:     "@global (\n    m\n    n = 2\n)\n@main\n    p {= (m ?? 0) + n }\n",
			globals: gad.Dict{"m": gad.Int(1)},
			want:    "<p>3</p>",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := renderGiom(t, tc.src, tc.globals); got != tc.want {
				t.Fatalf("render mismatch\n got: %s\nwant: %s", got, tc.want)
			}
		})
	}
}
