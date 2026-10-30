package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// TestCondNilCompare checks a condition comparing with nil (`x == nil`,
// `x != nil`) in a conditional expression and in an if: each way leaves the
// stack as it found it — before, `??`'s jumps left x there on one of them, and
// an operation around the conditional took it (`1 + (x == nil ? 1 : x)` was
// 1 + x + x), an array literal after `+` met an int, and an if in a loop grew
// the stack each turn.
func TestCondNilCompare(t *testing.T) {
	for _, tc := range []struct {
		src  string
		want Object
	}{
		{`o := 3; return 1 + (o == nil ? 1 : o)`, Int(4)},
		{`o := nil; return 1 + (o == nil ? 1 : o)`, Int(2)},
		{`o := 3; return 1 + (o != nil ? o : 1)`, Int(4)},
		{`o := nil; return 1 + (o != nil ? o : 1)`, Int(2)},
		{`o := 3; found := []; found += [o == nil ? 1 : o, 2]; return found`, Array{Array{Int(3), Int(2)}}},
		{`o := 3; return [] + [o == nil ? 1 : 5, 2]`, Array{Int(5), Int(2)}},
		{`o := nil; return [] + [o != nil ? o : 1, 2]`, Array{Int(1), Int(2)}},
		{`o := 3; return [(o == nil) ? "nil" : "set", (o != nil) ? "set" : "nil"]`, Array{Str("set"), Str("set")}},
		// an if: the value after it is the one before it
		{`o := 3; x := 10; if o == nil { x = 1 }; return x + 1`, Int(11)},
		{`o := nil; x := 10; if o != nil { x = 1 } else { x = 2 }; return x + 1`, Int(3)},
		// a loop of ifs: the stack does not grow
		{`n := 0
		for i := 0; i < 1000; i++ {
			v := i % 2 == 0 ? nil : i
			if v == nil { n++ }
			if v != nil { n += 0 }
		}
		return n + (nil == nil ? 0 : 1)`, Int(500)},
	} {
		testExpectRun(t, tc.src, nil, tc.want)
	}
}
