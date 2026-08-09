package gadbridge

import "testing"

// TestTemplateBraceEscape verifies that `\{%` escapes the template code-island
// delimiter to literal text in a `.gadt` template, while an unescaped `{%= … %}`
// output island still evaluates. Braces inside the expression's own string need
// no escaping.
func TestTemplateBraceEscape(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"escaped", "a \\{% not code %} b\n", "a {% not code %} b\n"},
		{"escaped-and-real", "val {%= 1 + 1 %} lit \\{%= x %}\n", "val 2 lit {%= x %}\n"},
		{"braces-in-expr", "a{%= \"cd{e}f\" %}g\n", "acd{e}fg\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := RunSource(c.src, "gadTemplate")
			if r.Stderr != "" {
				t.Fatalf("run error: %s", r.Stderr)
			}
			if r.Stdout != c.want {
				t.Fatalf("stdout = %q, want %q", r.Stdout, c.want)
			}
		})
	}
}
