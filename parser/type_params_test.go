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

// Type parameters (issue #23) on every function form: the shorthand and
// export, arrows, a function's methods, class methods and properties, and
// indexed return types.
func TestFuncTypeParamsForms(t *testing.T) {
	for src, want := range map[string]string{
		"g[K, V](k K, v V) => [k, v]":                        "g[K, V](k K, v V) => [k, v]",
		"export g[K, V](k K) <K> => nil":                     "export g[K, V](k K) <K> => nil",
		"a := [T](x T) => x":                                 "a := [T](x T) => x",
		"func fm { [T](x T) => x; (x, y) => x + y }":         "func fm {[T](x T) => x; (x, y) => (x + y)}",
		"class B { methods { conv[R](f) => R(f(this.v)) } }": "class B {methods {conv[R](f) => R(f(this.v))}}",
		"class B { props { as[R]() => R(this.v) } }":         "class B {props {as[R]() => R(this.v)}}",
		"func f[K](x K) <iterable[K, K]> => x":               "func f[K](x K) <iterable[K, K]> => x",
		"x := f[int|float](1)":                               "x := f[(int | float)](1)",
	} {
		test.New(t, src).Code(want)
	}
}
