package parser_test

import (
	"testing"

	"github.com/gad-lang/gad/parser/test"
)

// TestParseTypeParams checks that a type-parameter list `[T constraint, …]`
// parses in every function-signature context and round-trips through the AST
// String() rendering.
func TestParseTypeParams(t *testing.T) {
	// Named function.
	test.ExpectParseString(t,
		`func mySet[T indexable, K int|uint, V number](target T, k K, v V) <T> { return target }`,
		`func mySet[T indexable, K int|uint, V number](target T, k K, v V) <T> { return target }`)

	// Anonymous function, no space before the brackets.
	test.ExpectParseString(t, `x := func[T number](v T) => v`, `x := func[T number](v T) => v`)

	// Anonymous function, space before the brackets, block body.
	test.ExpectParseString(t, `x := func [T number](v T) <T> { return v }`,
		`x := func[T number](v T) <T> { return v }`)

	// Dict method shorthand, closure form.
	test.ExpectParseString(t, `d := { twice[T number](v T) <T>: v * 2 }`,
		`d := {twice[T number](v T) <T> : (v * 2)}`)

	// Func-header value.
	test.ExpectParseString(t, `h := <[T number](v T) <T>>`, `h := <[T number](v T) <T>>`)

	// Method interface header.
	test.ExpectParseString(t, `m := meti { [T number](v T) <T> }`,
		`m := meti {[T number](v T) <T>; }`)
}
