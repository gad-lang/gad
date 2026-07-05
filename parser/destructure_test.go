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
