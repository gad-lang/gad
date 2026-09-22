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

// TestParseMetadataClass verifies metadata parses on a class and its fields,
// props and methods and round-trips through the formatter.
func TestParseMetadataClass(t *testing.T) {
	test.ExpectParseString(t,
		"[a=1]\nclass C { [b=2]\n x = 0 }",
		`[a=1] class C {[b=2] x = 0}`)

	test.ExpectParseString(t,
		"class C { methods { [route=\"/s\"]\n save() { return 1 } } }",
		`class C {methods {[route="/s"] save() {return 1}}}`)

	// marker type and mixin.
	test.ExpectParseString(t,
		"[kind=\"m\"]\ntype T { [n=1]\n x = 0 }",
		`[kind="m"] type T {[n=1] x = 0}`)
	test.ExpectParseString(t,
		"[role=\"r\"]\nmixin M { x = 0 }",
		`[role="r"] mixin M {x = 0}`)
}
