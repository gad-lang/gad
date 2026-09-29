package parser_test

import (
	"testing"

	"github.com/gad-lang/gad/parser/test"
)

// TestParseGenerics checks the type parameters of a class and an interface —
// each a name and, optionally, its alias —, and the type arguments of a
// generic's use: one is an index (`Box[int]`), several a TypeArgsExpr
// (`Pair[str, int]`), in a type and in a value.
func TestParseGenerics(t *testing.T) {
	// a class: the parameters, with their aliases, a space before or none
	test.ExpectParseString(t, `class P [X int, Y float] { x X; y Y }`,
		`class P[X int, Y float] {x X; y Y}`)
	test.ExpectParseString(t, `class Box[T] { item T }`, `class Box[T] {item T}`)
	test.ExpectParseString(t, `class U[T int|str] { v T }`, `class U[T int|str] {v T}`)

	// an interface
	test.ExpectParseString(t, `interface Named[T str] { name T }`,
		`interface Named[T str] {name T; }`)
	// `[]` after an interface's name is still its array depth
	test.ExpectParseString(t, `interface A [] { x int }`, `interface A []{x int; }`)

	// the uses: in a field's type, and as a value
	test.ExpectParseString(t, `class H { b Box[int]; p? Pair[str, int|float]; s models.Page }`,
		`class H {b Box[int]; p? Pair[str, int|float]; s models.Page}`)
	test.ExpectParseString(t, `x := Pair[str, int](; key="k")`, `x := Pair[str, int](; key="k")`)
	test.ExpectParseString(t, `x := a[1]`, `x := a[1]`)
}
