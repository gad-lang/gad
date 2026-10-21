package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// Arrays and key-value arrays order as Python's lists (of tuples): by the
// first items that differ, else the shorter first; nil before them.
func TestSequenceOrdering(t *testing.T) {
	for _, c := range []struct {
		src  string
		want Object
	}{
		{`return [1, 2] < [1, 3]`, True},
		{`return [1, 3] < [1, 2]`, False},
		{`return [1] < [1, 2]`, True},
		{`return [1, 2] <= [1, 2]`, True},
		{`return [1, 2] < [1, 2]`, False},
		{`return [2] > [1, 9]`, True},
		{`return [1, 2] >= [1, 3]`, False},
		{`return [[1, 2], [0]] < [[1, 2], [1]]`, True},
		{`return nil < [1]`, True},
		{`return [1] > nil`, True},
		{`return [] < []`, False},
		{`return [] <= []`, True},
		{`return (;a=1, b=2) < (;a=1, b=3)`, True},
		{`return (;a=1) < (;b=0)`, True},
		{`return (;a=1) < (;a=1, b=0)`, True},
		{`return (;b=1) > (;a=9)`, True},
		{`return sort([[2], [1, 5], [], [1]])`, Array{Array{}, Array{Int(1)}, Array{Int(1), Int(5)}, Array{Int(2)}}},
	} {
		t.Run(c.src, func(t *testing.T) {
			testExpectRun(t, c.src, nil, c.want)
		})
	}
	expectErrIs(t, `x := ["a"]; return [1] < x`, nil, ErrType)
}
