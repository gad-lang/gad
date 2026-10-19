package parser_test

import (
	"testing"

	"github.com/gad-lang/gad/parser/test"
)

// The sizes of the levels of an array type: `[N]`, `[..B]`, `[A..]`, `[A..B]`,
// with expressions, beside `[]` — in parameters, fields, unions, `type`
// declarations, casts and `type […]T` values.
func TestSizedArrayTypes(t *testing.T) {
	for src, want := range map[string]string{
		"func(v [2]int) {}":                     "func(v [2]int) {}",
		"func(v [..3]str) {}":                   "func(v [..3]str) {}",
		"func(v [2..]str) {}":                   "func(v [2..]str) {}",
		"func(v [2..5]str) {}":                  "func(v [2..5]str) {}",
		"func(v [][2]str) {}":                   "func(v [][2]str) {}",
		"func(v [count]str) {}":                 "func(v [count]str) {}",
		"func(v [n-1..n+1]int) {}":              "func(v [(n - 1)..(n + 1)]int) {}",
		"func(v [2]<int|str>) {}":               "func(v [2]<int | str>) {}",
		"type options [][2]str":                 "type options [][2]str",
		"type pair [2]str":                      "type pair [2]str",
		"type grid [2..3][2]int":                "type grid [2..3][2]int",
		"x := [1, 2, 3]::[2..3]int":             "x := [1, 2, 3]::[2..3]int",
		"x := [1, 2] :> [2]int":                 "x := [1, 2] :> [2]int",
		"x := [1, 2] ::> [][2]str":              "x := [1, 2] ::> [][2]str",
		"const t = type [2..3][2]str":           "const t = type [2..3][2]str",
		"x := typeName(type []int)":             "x := typeName(type []int)",
		"class Address { city []str|[][2]str }": "class Address {city []str | [][2]str}",
		"x := (type []int).@depth":              "x := (type []int).@depth",
		"x := [(type []int), 1]":                "x := [(type []int), 1]",
		"x := [type []int, 1]":                  "x := [type []int, 1]",
	} {
		test.New(t, src).Code(want)
	}
	// `type[0]` stays indexing a variable named type
	test.New(t, "x := type[0]").Code("x := type[0]")
}

func TestSizedArrayTypeErrors(t *testing.T) {
	test.New(t, "func(v [..]int) {}").Error()
}
