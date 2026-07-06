package parser_test

import (
	"testing"

	"github.com/gad-lang/gad/parser/test"
)

func TestParseCurlyDestructure(t *testing.T) {
	// TypeScript-style `{ … }` named destructuring: key-on-the-left, `:` renames,
	// `=` is a fallback default, `**` is the optional rest target.
	test.ExpectParseString(t, `{ a, x: b, r = 2, **rest } := d`,
		`{ a, x:b, r=2, **rest } := d`)
	test.ExpectParseString(t, `{ a } = d`, `{ a } = d`)
	test.ExpectParseString(t, `{ x: b } := d`, `{ x:b } := d`)
	test.ExpectParseString(t, `{ a, **rest } := d`, `{ a, **rest } := d`)
	test.ExpectParseString(t, `{} := d`, `{} := d`)

	// a bare block is still a block, not a destructuring pattern.
	test.ExpectParseString(t, `{ x := 1 }`, `{ x := 1 }`)
}

func TestParseBracketDestructure(t *testing.T) {
	// `[a, b]` bracket array destructuring, statement level.
	test.ExpectParseString(t, `[a, b] := arr`, `[a, b] := arr`)
	test.ExpectParseString(t, `[a, b] = arr`, `[a, b] = arr`)
	// trailing `*rest`, both comma and bracket forms.
	test.ExpectParseString(t, `a, b, *rest := arr`, `a, b, *rest := arr`)
	test.ExpectParseString(t, `[a, b, *rest] := arr`, `[a, b, *rest] := arr`)
}

func TestParseConstDestructure(t *testing.T) {
	// const/var declarations with `{ … }` and `[ … ]` patterns.
	test.ExpectParseString(t, `const {x} = d`, `const { x } = d`)
	test.ExpectParseString(t, `const {k: v} = d`, `const { k:v } = d`)
	test.ExpectParseString(t, `var [a, b] = arr`, `var [a, b] = arr`)
	test.ExpectParseString(t, `const {a, **rest} = d`, `const { a, **rest } = d`)
}

// TestFormatMixedParen is a regression for the MultiParenExpr formatter, which
// used to duplicate the positional items and write them again in place of the
// named side (dropping it) on its multiline path. It renders inline in the
// canonical `(,` form.
func TestFormatMixedParen(t *testing.T) {
	// positional rest is a single `*rest`; named rest is `**rest`.
	test.New(t, `(a, b, *pos; c, d:p, r=2, **named) := mp`).
		FormattedCode(`(, a, b, *pos; c, d:p, r=2, **named) := mp`)
	test.New(t, `(1, 2; x=3) := mp`).FormattedCode(`(, 1, 2; x=3) := mp`)
	// `**pos` still parses (lenient alias) and round-trips.
	test.New(t, `(a, **pos; c) := mp`).FormattedCode(`(, a, **pos; c) := mp`)
}
