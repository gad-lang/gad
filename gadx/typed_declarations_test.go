package gadx

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gad-lang/gad"
)

// renderErr renders src (no globals) and returns the output and any error.
func renderErr(t *testing.T, src string, globals gad.Dict) (string, error) {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "t.gadx")
	if err := os.WriteFile(p, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	err := newTestRender(t, dir).Render(&buf, p, globals)
	return buf.String(), err
}

// TestTypedParamDirective covers `@param` with typed names — it lowers to Gad's
// `param`, which accepts a type after each name (positional, and named after
// `;`, with defaults). Type unions are allowed.
func TestTypedParamDirective(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "named typed default",
			src:  "@param (; n int = 4)\n@main\n    p {=n}\n",
			want: "<p>4</p>",
		},
		{
			name: "positional typed and named typed default",
			src:  "@param (a int; b int = 2)\n@main\n    p {=b}\n",
			want: "<p>2</p>",
		},
		{
			name: "named typed union default",
			src:  "@param (; n int|uint = 4)\n@main\n    p {=n}\n",
			want: "<p>4</p>",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := strings.TrimSpace(renderGadx(t, tc.src, nil)); got != tc.want {
				t.Fatalf("render = %q, want %q\nsrc:\n%s", got, tc.want, tc.src)
			}
		})
	}
}

// TestTypedGlobalDirective covers `@global` with typed names. `@global` lowers
// to Gad's `global` (typed, with `=` nil-or-absent and `!?=` absent-only
// defaults). Because the bare form keeps its legacy space-separated-names
// meaning (`@global a b` = two globals), a typed global uses the parenthesized
// form: `@global (x int)`.
func TestTypedGlobalDirective(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		globals gad.Dict
		want    string
	}{
		{
			name:    "parenthesized typed",
			src:     "@global (x int)\n@main\n    p {=x}\n",
			globals: gad.Dict{"x": gad.Int(9)},
			want:    "<p>9</p>",
		},
		{
			name:    "parenthesized typed union",
			src:     "@global (x int|str)\n@main\n    p {=x}\n",
			globals: gad.Dict{"x": gad.Str("hi")},
			want:    "<p>hi</p>",
		},
		{
			name:    "typed nil-default applies",
			src:     "@global (x int = 3)\n@main\n    p {=x}\n",
			globals: gad.Dict{},
			want:    "<p>3</p>",
		},
		{
			name:    "typed absent-default applies",
			src:     "@global (x int !?= 4)\n@main\n    p {=x}\n",
			globals: gad.Dict{},
			want:    "<p>4</p>",
		},
		{
			// The legacy bare space-separated form is unchanged: two globals.
			name:    "legacy bare space-separated names",
			src:     "@global a b\n@main\n    p {= a + b }\n",
			globals: gad.Dict{"a": gad.Int(2), "b": gad.Int(3)},
			want:    "<p>5</p>",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, err := renderErr(t, tc.src, tc.globals)
			if err != nil {
				t.Fatalf("render error: %v\nsrc:\n%s", err, tc.src)
			}
			if got := strings.TrimSpace(out); got != tc.want {
				t.Fatalf("render = %q, want %q\nsrc:\n%s", got, tc.want, tc.src)
			}
		})
	}
}

// TestVarConstRejectTypes documents that `@var`/`@const` do NOT accept typed
// names: Gad's `var`/`const` have no type syntax (only `param`/`global` do). The
// declaration fails to compile.
func TestVarConstRejectTypes(t *testing.T) {
	for _, src := range []string{
		"@var x int = 1\n@main\n    p {=x}\n",
		"@var (x int = 1)\n@main\n    p {=x}\n",
		"@const x int = 1\n@main\n    p {=x}\n",
	} {
		if _, err := renderErr(t, src, nil); err == nil {
			t.Fatalf("expected a compile error for typed var/const, got none\nsrc:\n%s", src)
		}
	}
}

// TestTypeUnionsInSignatures verifies type unions (`a|b`) are accepted in every
// typed position of the `@func`/`@comp`/`@main` directives — parameter types,
// type-parameter constraints and return types (the return-type union is an
// unnamed union `<int|str>`).
func TestTypeUnionsInSignatures(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		globals gad.Dict
		want    string
	}{
		{
			name: "func param and return union",
			src:  "@func f(x int|str) <int|str>\n    | {=x}\n@main\n    +f(3)\n",
			want: "3",
		},
		{
			name: "comp param union",
			src:  "@comp c(v int|uint)\n    span {=v}\n@main\n    +c(2)\n",
			want: "<span>2</span>",
		},
		{
			name: "func type-parameter constraint union",
			src:  "@func id[T int|uint](v T) <T>\n    | {=v}\n@main\n    +id(5)\n",
			want: "5",
		},
		{
			name:    "main param union",
			src:     "@main(v int|str)\n    p {=v}\n",
			globals: gad.Dict{"v": gad.Str("ok")},
			want:    "<p>ok</p>",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := strings.TrimSpace(renderGadx(t, tc.src, tc.globals)); got != tc.want {
				t.Fatalf("render = %q, want %q\nsrc:\n%s", got, tc.want, tc.src)
			}
		})
	}
}
