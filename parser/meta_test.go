package parser_test

import (
	"testing"

	"github.com/gad-lang/gad/parser/test"
)

// TestParseMetadata verifies a `[k=v, …]` metadata block parses on a declaration
// and its members and round-trips through the formatter.
func TestParseMetadata(t *testing.T) {
	// interface + a member, each with metadata (the member's is rendered inline
	// before it; both round-trip).
	test.ExpectParseString(t,
		"[a=1]\ninterface I { [b=2]\n x int }",
		`[a=1] interface I {[b=2] x int; }`)

	// flag entry (`k` == `k=true`) and a nested key-value array value.
	test.ExpectParseString(t,
		`[readonly, db=(;primary_key)] interface I { x int }`,
		`[readonly, db=(;primary_key)] interface I {x int; }`)

	// enum + item.
	test.ExpectParseString(t,
		"[cat=\"acl\"]\nenum E { [bit=0]\n Read }",
		`[cat="acl"] enum E {[bit=0] Read}`)

	// a bare array before a NON-declaration is still an ordinary array statement.
	test.ExpectParseString(t, "[1, 2, 3]\nx := 1", `[1, 2, 3]; x := 1`)
}
