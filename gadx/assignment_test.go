package gadx

import (
	"testing"

	"github.com/gad-lang/gad"
)

// TestAssignmentForms verifies bare `IDENT := EXPR` / `IDENT = EXPR` statements
// (and the compound forms) at the template body level: IDENT is a plain GAD
// identifier (optional `$`, no `-`), and the value may span several lines while
// its brackets are unbalanced.
func TestAssignmentForms(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "define",
			src:  "@main\n    x := 42\n    p {= x }",
			want: `<p>42</p>`,
		},
		{
			name: "define then assign",
			src:  "@main\n    x := 1\n    x = 7\n    p {= x }",
			want: `<p>7</p>`,
		},
		{
			name: "compound add-assign",
			src:  "@main\n    x := 40\n    x += 2\n    p {= x }",
			want: `<p>42</p>`,
		},
		{
			name: "dollar identifier",
			src:  "@main\n    $x := \"hi\"\n    p {= $x }",
			want: `<p>hi</p>`,
		},
		{
			name: "value uses builtins and locals",
			src:  "@main\n    a := 3\n    b := a * a\n    p {= b }",
			want: `<p>9</p>`,
		},
		{
			name: "multi-line array value",
			src: "@main\n" +
				"    items := [\n" +
				"        \"a\",\n" +
				"        \"b\",\n" +
				"        \"c\"]\n" +
				"    p {= strings.join(items, \",\") }",
			want: `<p>a,b,c</p>`,
		},
		{
			name: "multi-line comprehension value",
			src: "@main\n" +
				"    evens := [\n" +
				"        n\n" +
				"        for n in [1, 2, 3, 4]\n" +
				"        if n % 2 == 0]\n" +
				"    p {= strings.join([str(n) for n in evens], \",\") }",
			want: `<p>2,4</p>`,
		},
		{
			name: "value drives a tag body",
			src:  "@main\n    n := 2\n    @for i in [1, 2, 3]\n        li {= i * n }",
			want: `<li>2</li><li>4</li><li>6</li>`,
		},
		{
			// A tag name may contain `-`; it is never an assignment (no bare `=`).
			name: "dashed tag is not an assignment",
			src:  "@main\n    my-el hi",
			want: `<my-el>hi</my-el>`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := renderGadx(t, tc.src, gad.Dict{})
			if got != tc.want {
				t.Fatalf("render mismatch\n got: %s\nwant: %s", got, tc.want)
			}
		})
	}
}
