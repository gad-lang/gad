package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// TestClosureBlockValue pins the Julia-style implicit value of a closure with a
// block body: a block is worth its last evaluated expression — recursing into a
// nested block, every if/else-if/else branch and the try/catch bodies — and an
// assignment to plain variables is worth the assigned value(s). A loop, an if
// whose taken branch is missing, or another statement in the last position is
// nil (Julia's `nothing`). A func block has no implicit value.
func TestClosureBlockValue(t *testing.T) {
	for _, c := range []struct {
		src  string
		want Object
	}{
		{`return (() => { 5 + 10 })()`, Int(15)},
		{`return (() => { a := 5; a + 1 })()`, Int(6)},
		{`return (() => { if true { 2 }; 3 })()`, Int(3)},
		{`return (() => { true ? 1 : 2 })()`, Int(1)},
		{`return (() => { len([1, 2]) })()`, Int(2)},
		{`return (() => { match 1 { 1: "one"; else: "?" } })()`, Str("one")},
		// sub-blocks
		{`return (() => { if true { 2 } })()`, Int(2)},
		{`return (() => { if false { 1 } else { 2 } })()`, Int(2)},
		{`return (() => { x := 1; if x > 0 { "pos" } })()`, Str("pos")},
		{`return (() => { if false { 1 } })()`, Nil},
		{`g := (n) => { if n > 0 { "pos" } else if n < 0 { "neg" } else { "zero" } }
		  return [g(1), g(-1), g(0)]`, Array{Str("pos"), Str("neg"), Str("zero")}},
		{`return (() => { { { 7 } } })()`, Int(7)},
		{`return (() => { try { 8 } catch { 9 } })()`, Int(8)},
		{`return (() => { try { throw "x"; 8 } catch { 9 } })()`, Int(9)},
		{`log := []; v := (() => { try { 1 } finally { log += 2 } })(); return [v, log]`,
			Array{Int(1), Array{Int(2)}}},
		// assignments
		{`return (() => { a := 5 })()`, Int(5)},
		{`return (() => { a := 1; a += 4 })()`, Int(5)},
		{`return (() => { a, b := 1, 2 })()`, Array{Int(1), Int(2)}},
		{`return (() => { if true { y := "set" } })()`, Str("set")},
		{`o := {}; return (() => { o.k = 1 })()`, Nil},
		// no value
		{`return (() => { for i := 0; i < 2; i++ { i } })()`, Nil},
		{`return (() => { for x in [1, 2] { x } })()`, Nil},
		{`return (() => {})()`, Nil},
		{`return (() => { return 4; 5 })()`, Int(4)},
		{`return func() { 5 + 10 }()`, Nil},
		{`return (func() => 5 + 10)()`, Int(15)},
	} {
		testExpectRun(t, c.src, nil, c.want)
	}
}
