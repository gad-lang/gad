package parser_test

import (
	"testing"

	"github.com/gad-lang/gad/parser/test"
)

// A keyword is a name where only a name may be: a field of an interface or a
// class, a member, a key of a contract's meta.
func TestKeywordAsMemberName(t *testing.T) {
	for src, want := range map[string]string{
		`interface I { if? any; for? int; return str }`: `interface I {for? int; if? any; return str; }`,
		`class C { if? any; nil? str }`:                 `class C {if? any; nil? str}`,
	} {
		test.New(t, src).Code(want)
	}
}

// `end` in the brackets of an index is the last index.
func TestIndexEnd(t *testing.T) {
	for src, want := range map[string]string{
		`a[end]`:     `a[end]`,
		`a[end-1]`:   `a[(end - 1)]`,
		`a[1:end]`:   `a[1:end]`,
		`a[b[end]]`:  `a[b[end]]`,
		`a[end] = 1`: `a[end] = 1`,
	} {
		test.New(t, src).Code(want)
	}
}
