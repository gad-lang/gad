package parser_test

import (
	"testing"

	"github.com/gad-lang/gad/parser/test"
)

// Typed dicts (issue #19): `type NAME dict[K, V]` declared, `type dict[K, V]`
// and `type NAME dict[…]` as values; `dict` an ordinary name elsewhere.
func TestTypedDictType(t *testing.T) {
	for src, want := range map[string]string{
		"type scores dict[str, int]":    "type scores dict[str, int]",
		"type bag dict":                 "type bag dict",
		"x := type dict[int]":           "x := type dict[int]",
		"x := type myD dict[str, bool]": "x := type myD dict[str, bool]",
		"export type s dict[str, int]":  "export type s dict[str, int]",
		"d := dict(; a=1)":              "d := dict(; a=1)",
		"D := DictType[str, int]":       "D := DictType[str, int]",
	} {
		test.New(t, src).Code(want)
	}
}
