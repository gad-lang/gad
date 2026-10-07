package gadbridge

import (
	"strings"
	"testing"
)

// formatted is src formatted with 4-space indentation; it must format to
// itself again.
func formatted(t *testing.T, src string) string {
	t.Helper()
	opts := GadxFormatOptions{Indent: "    "}
	res := FormatGadx(src, opts)
	if !res.OK {
		t.Fatalf("format failed: %v", res.Diagnostics)
	}
	if again := FormatGadx(res.Source, opts); again.Source != res.Source {
		t.Fatalf("not idempotent:\n%s\n---\n%s", res.Source, again.Source)
	}
	return res.Source
}

// What the formatter keeps of a template as it was written: its @imports
// (not the `~ x := import(…)` they lower to), a `~~` block with its doc and
// comments, the blank lines between statements, the doc of an @const, the
// names of an @global, a call's body under it (no synthetic `@slot #main`),
// a script of one interpolation on its line; and what it writes canonically:
// no `()` with no arguments, `///` for a doc of one line, no parentheses the
// source does not have.
func TestFormatGadxKeepsTheSource(t *testing.T) {
	src := `@global Context
@import "models" as models
@import { main: form_layout } from "form/layout.gadx"

~~
/**
What a post tells.
**/
export func opts(post) {
    a := post.A

    // the b of it
    b := post.B
    return a + b
}
~~

/** The layouts. **/
@const (L = {a: 1})

@comp item
    p x

    p y

@main
    ~ n := !!Context.N
    +item
        span body
    script[type="application/javascript"] {=raw Context.JS}
    p {= n ? "a" + "b" : "" }
`
	out := formatted(t, src)
	for _, want := range []string{
		"@global Context\n",
		"@import \"models\" as models\n@import { main: form_layout } from \"form/layout.gadx\"\n",
		"~~\n/**\nWhat a post tells.\n**/\nexport func opts(post) {",
		"    a := post.A\n\n    // the b of it\n    b := post.B\n",
		"/// The layouts.\n@const (L = { a: 1 })\n",
		"@comp item\n    p x\n\n    p y\n",
		"~ n := !!Context.N\n",
		"    +item\n        span body\n",
		"script[type=\"application/javascript\"] {= raw Context.JS }\n",
		"p {= n ? \"a\" + \"b\" : \"\" }\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	for _, not := range []string{"@slot #main", ":= import(", "@comp item()", "+item()", "(n ?"} {
		if strings.Contains(out, not) {
			t.Errorf("has %q:\n%s", not, out)
		}
	}
}
