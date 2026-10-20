package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// An array type is a constructor; `arrayType` builds one; `array` is `type []any`
// (issue #18).
func TestVMArrayTypeConstructor(t *testing.T) {
	// from its items, checked
	testExpectRun(t, `return (type [2]str)("a", "b")`, nil, Array{Str("a"), Str("b")})
	testExpectRun(t, `return (type [][]int)([1], [2, 3])`, nil, Array{Array{Int(1)}, Array{Int(2), Int(3)}})
	testExpectRun(t, `xs := [1, 2]; return (type []int)(*xs)`, nil, Array{Int(1), Int(2)})
	testExpectRun(t, `return (type []int)()`, nil, Array{})
	expectErrHas(t, `return (type [2]str)("a")`, newOpts(), "level 1 has 1 items, wants exactly 2")
	expectErrHas(t, `return (type [][]int)([1], [2, "a"])`, newOpts(), "item 1.1: ‹str› is not of the element type")
	expectErrHas(t, `return (type []int)(1; fill=0)`, newOpts(), "items, or size and fill: not both")
	expectErrHas(t, `return (type []int)(; color=1)`, newOpts(), `"color"`)

	// by its dimensions
	testExpectRun(t, `return (type [2][3]int)(; fill=0)`, nil,
		Array{Array{Int(0), Int(0), Int(0)}, Array{Int(0), Int(0), Int(0)}})
	testExpectRun(t, `return (type [][]int)(; size=[2, 1], fill=1)`, nil, Array{Array{Int(1)}, Array{Int(1)}})
	testExpectRun(t, `return (type [2][]int)(; size=[1], fill=1)`, nil, Array{Array{Int(1)}, Array{Int(1)}})
	testExpectRun(t, `return (type [3][2]int)(; fill=(i, j) => i * j)`, nil,
		Array{Array{Int(0), Int(0)}, Array{Int(0), Int(1)}, Array{Int(0), Int(2)}})
	testExpectRun(t, `return (type [2]str)(; fill=(i) => str(i))`, nil, Array{Str("0"), Str("1")})
	testExpectRun(t, `return (type [2]any)(; size=[2])`, nil, Array{Nil, Nil})
	testExpectRun(t, `n := 2; return (type [n]int)(; fill=5)`, nil, Array{Int(5), Int(5)})
	expectErrHas(t, `return (type [][]int)(; fill=0)`, newOpts(), "level 1 has no size")
	expectErrHas(t, `return (type [][]int)(; size=[2], fill=0)`, newOpts(), "1 sizes, wants 2")
	expectErrHas(t, `return (type [2]int)(; size=[3], fill=0)`, newOpts(), "level 1: size 3, wants exactly 2")
	expectErrHas(t, `return (type [2..4]int)(; size=[5], fill=0)`, newOpts(), "level 1: size 5, wants 2 to 4")
	expectErrHas(t, `return (type [2]int)(; fill="x")`, newOpts(), "fill: ‹str› is not of the element type")
	expectErrHas(t, `return (type [2]int)(; size=[2])`, newOpts(), "fill required")

	// a declared typed array, by its dimensions too (when no field is so named)
	testExpectRun(t, `type grid [2][2]int; g := grid(; fill=7); return [typeName(g), g[1]]`, nil,
		Array{Str("grid"), Array{Int(7), Int(7)}})
	testExpectRun(t, `type g [2]int { size? int }; return g(1, 2; size=3).size`, nil, Int(3))

	// `::>` still converts each leaf (the constructor checks only)
	testExpectRun(t, `return [1, 2] ::> []str`, nil, Array{Str("1"), Str("2")})
}

func TestVMArrayTypeBuiltin(t *testing.T) {
	testExpectRun(t, `return [str(arrayType(int)), str(arrayType(int; depth=2)), str(arrayType(str; size=[2])),
		str(arrayType(int; size=[[2, nil, nil], [nil, 1, 3]])), str(arrayType([int, str]))]`, nil,
		Array{Str("[]int"), Str("[][]int"), Str("[2]str"), Str("[2][1..3]int"), Str("[]<int|str>")})
	testExpectRun(t, `return arrayType(int; size=[3])(; fill=0)`, nil, Array{Int(0), Int(0), Int(0)})
	// structural equality, and the round trip of the reflection keys
	testExpectRun(t, `return [arrayType(int) == type []int, arrayType(int) == type []str, type [2]int == type [2]int,
		type [2]int == type [3]int, type []int == type [][]int]`, nil, Array{True, False, True, False, False})
	testExpectRun(t, `T := type [2][1..3]int; return arrayType(T.@elem; depth=T.@depth, size=T.@size) == T`, nil, True)
	testExpectRun(t, `class C {}; U := type [][]C; return arrayType(U.@elem; depth=U.@depth, size=U.@size) == U`, nil, True)
	// the type of the array types; overloads as any type with a constructor
	testExpectRun(t, `return [typeof(type []int) === arrayType, arrayType(int) :: arrayType == type []int, typeof(array) === arrayType]`,
		nil, Array{True, True, True})
	testExpectRun(t, `class C {}; met arrayType(x C) => "C"; return [arrayType(C()), str(arrayType(int))]`, nil,
		Array{Str("C"), Str("[]int")})
	expectErrHas(t, `return arrayType(1)`, newOpts(), "element 0: ‹int› is not a type")
	expectErrHas(t, `return arrayType(int; depth=2, size=[1])`, newOpts(), "depth 2, but 1 sizes")
	expectErrHas(t, `return arrayType(int; size=[[nil, 3, 1]])`, newOpts(), "the minimum 3 is greater than the maximum 1")
}

func TestVMArrayIsAnyArrayType(t *testing.T) {
	testExpectRun(t, `return [array === type []any, typeof([1, 2]) === type []any, arrayType(any) === array,
		typeof([1]) === array]`, nil, Array{True, True, True, True})
	testExpectRun(t, `return [array.@elem == [any], array.@depth, array.@size == [[nil, nil, nil]]]`, nil,
		Array{True, Int(1), True})
	testExpectRun(t, `f := func(x []any) => x; return [f([1]), [1] :: type []any, isArray([1])]`, nil,
		Array{Array{Int(1)}, Array{Int(1)}, True})
	// array keeps what it does
	testExpectRun(t, `return [array(1, 2), array(), array(; size=[3], fill=0), typeName([1])]`, nil,
		Array{Array{Int(1), Int(2)}, Array{}, Array{Int(0), Int(0), Int(0)}, Str("array")})
	expectErrHas(t, `type ints []int; return ints(1) :: array`, newOpts(), "not assignable")
}
