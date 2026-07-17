package giom

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gad-lang/gad"
)

// TestVarConstDeclarations covers `@var`/`@const` in bare (single and
// comma-separated), single-line parenthesized, and multi-line parenthesized
// forms.
func TestVarConstDeclarations(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "var bare single",
			src:  "@main\n    @var a\n    ~ a = 5\n    p {= a }\n",
			want: "<p>5</p>",
		},
		{
			name: "var bare multi",
			src:  "@main\n    @var a, b\n    ~ a = 1\n    ~ b = 2\n    p {= a + b }\n",
			want: "<p>3</p>",
		},
		{
			name: "var bare multi with init",
			src:  "@main\n    @var a, b, c = 3\n    ~ a = 1\n    p {= a + c }\n",
			want: "<p>4</p>",
		},
		{
			name: "var paren single line",
			src:  "@main\n    @var (x = 10, y = 20)\n    p {= x + y }\n",
			want: "<p>30</p>",
		},
		{
			name: "var paren multi line",
			src:  "@main\n    @var (\n        m = 100\n        n, o = 7\n    )\n    p {= m + o }\n",
			want: "<p>107</p>",
		},
		{
			name: "const bare single",
			src:  "@main\n    @const k = 9\n    p {= k }\n",
			want: "<p>9</p>",
		},
		{
			name: "const bare multi",
			src:  "@main\n    @const a = 1, b = 2\n    p {= a + b }\n",
			want: "<p>3</p>",
		},
		{
			name: "const paren multi line",
			src:  "@main\n    @const (\n        p1 = 3\n        p2 = 4\n    )\n    p {= p1 * p2 }\n",
			want: "<p>12</p>",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := renderGiom(t, tc.src, gad.Dict{}); got != tc.want {
				t.Fatalf("render mismatch\n got: %s\nwant: %s", got, tc.want)
			}
		})
	}
}

// TestConstRequiresValue verifies that a const without an initializer is a
// compile error, and the error resolves to the giom source line.
func TestConstRequiresValue(t *testing.T) {
	dir := t.TempDir()
	src := "@main\n    @const missing\n    p x\n"
	p := filepath.Join(dir, "t.giom")
	if err := os.WriteFile(p, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	r := newTestRender(t, dir)
	_, err := renderString(r, p, gad.Dict{})
	if err == nil {
		t.Fatal("expected a compile error for const without value")
	}
	if !strings.Contains(err.Error(), "initializer") {
		t.Fatalf("expected missing-initializer error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "t.giom:2") {
		t.Fatalf("expected error at t.giom:2, got: %v", err)
	}
}
