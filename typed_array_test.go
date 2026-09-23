package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
	"github.com/stretchr/testify/require"
)

// TestTypedArrayConstruct covers building a typed array: items are the
// positional arguments, each checked against the element type.
func TestTypedArrayConstruct(t *testing.T) {
	testExpectRun(t, `
		type numerics []<int|uint|float>
		n := numerics(1, 2.5, 3u)
		return [typeName(n), len(n), n[0], n[-1], str(n)]`,
		nil, Array{Str("numerics"), Int(3), Int(1), Uint(3), Str("numerics[1, 2.5, 3]")})

	// A wrong item is rejected by the constructor.
	testExpectRun(t, `
		type numerics []<int|float>
		try { numerics(1, "x"); return "ok" } catch { return "no" }`, nil, Str("no"))

	// Deeper nesting: an item of a depth-2 type is an array of the element type.
	testExpectRun(t, `
		type grid [][]int
		g := grid([1, 2], [3])
		try { grid([1, "x"]); return "ok" } catch { return [str(g), grid.@depth] }`,
		nil, Array{Str("grid[[1, 2], [3]]"), Int(2)})

	// An inline interface element (short and long forms).
	testExpectRun(t, `
		type users []{ name; id }
		type users2 [] interface { name; id }
		u := users({name: "a", id: 1})
		try { users2({name: "a"}); return "ok" } catch { return [len(u), u[0].name] }`,
		nil, Array{Int(1), Str("a")})
}

// TestTypedArrayBehavesAsArray covers the Array-like behaviour of a typed array:
// index get/set/delete, slicing, iteration, len, sort, `in`, copy, `+`/`++`/`+=`.
func TestTypedArrayBehavesAsArray(t *testing.T) {
	testExpectRun(t, `
		type nums []int
		n := nums(3, 1, 2)
		n[0] = 30
		delete n[1]
		s := n[0:1]
		n += 5
		m := n ++ [6, 7]
		sum := 0
		for v in n { sum += v }
		return [str(n), str(s), typeName(s), str(m), sum, 5 in n, copy(n) == n, str(sort(nums(3, 1, 2)))]`,
		nil, Array{
			Str("nums[30, 2, 5]"), Str("nums[30]"), Str("nums"), Str("nums[30, 2, 5, 6, 7]"),
			Int(37), True, True, Str("nums[1, 2, 3]"),
		})

	// Every write is checked: an index set, `+=`, `+` and `++`.
	for _, bad := range []string{`n[0] = "x"`, `n += "x"`, `n + "x"`, `n ++ ["x"]`} {
		testExpectRun(t, `type nums []int; n := nums(1)
			try { `+bad+`; return "ok" } catch { return "no" }`, nil, Str("no"))
	}
}

// TestTypedArrayCasts covers `::` (nominal: only a value of the type) and the
// transforming `:::` both ways — array ⇄ typed array ⇄ interface array — with
// the items validated.
func TestTypedArrayCasts(t *testing.T) {
	testExpectRun(t, `
		type numerics []<int|float>
		interface nums [] <int|float>
		sat := func(v, T) { try { v :: T; return true } catch { return false } }
		n := [1, 2.5] ::: numerics         // array -> typed array
		a := n ::: array                   // typed array -> array
		i := n ::: nums                    // typed array -> interface array
		back := i ::: numerics             // interface array -> typed array
		return [typeName(n), typeName(a), str(a), typeName(i), sat(n, nums),
			typeName(back), sat([1], numerics), sat(n, numerics)]`,
		nil, Array{Str("numerics"), Str("array"), Str("[1, 2.5]"), Str("array"), True,
			Str("numerics"), False, True})

	// Conversion validates each item.
	testExpectRun(t, `type numerics []<int|float>
		try { [1, "x"] ::: numerics; return "ok" } catch { return "no" }`, nil, Str("no"))
	testExpectRun(t, `type numerics []<int|float>; interface pts [] { x int }
		try { numerics(1) ::: pts; return "ok" } catch { return "no" }`, nil, Str("no"))

	// A typed-array parameter accepts only a value of the type (convert with :::).
	testExpectRun(t, `type numerics []<int|float>
		f := func(xs numerics) => len(xs)
		try { f([1, 2]); return "ok" } catch { return f([1, 2] ::: numerics) }`, nil, Int(2))
}

// TestTypedArrayReflection covers the type's reflection keys and metadata.
func TestTypedArrayReflection(t *testing.T) {
	testExpectRun(t, `
		[unit="m", version=2]
		type numerics []<int|float>
		return [numerics.@name, str(numerics.@meta), numerics.@depth, str(numerics.@elem),
			typeName(numerics), str(numerics.@fields), numerics.@new]`,
		nil, Array{Str("numerics"), Str(`(;unit="m", version=2)`), Int(1), Str("int|float"),
			Str("typedArrayType"), Str("{}"), Nil})
}

// TestTypedArrayMembers covers the optional class-like body: fields (defaults,
// types, metadata), properties, methods and `new` constructor overloads, and the
// instance access rule (int key -> item, other keys -> members).
func TestTypedArrayMembers(t *testing.T) {
	src := `
		type numerics []<int|float> {
			[db=(;col="lbl")]
			label = "none"
			scale int = 1
			new(n int) {
				items := []
				for i := 0; i < n; i++ { items += 0 }
				return new(*items; label="zeros")
			}
			props {
				[computed]
				total => this.sum() * this.scale
			}
			methods {
				[route="/sum"]
				sum() { s := 0; for v in this { s += v }; return s }
				add(v int|float) { this += v; return this }
			}
		}
		a := numerics(1, 2.5; scale=2)
		a.label = "hi"
		a.add(3)
		z := numerics(2)
		return [str(a), a.total, a[1], z.label, len(z),
			str(numerics.label.@meta), str(numerics.total.@meta), str(numerics.sum.@meta)]`
	testExpectRun(t, src, nil, Array{
		Str(`numerics[1, 2.5, 3]{label: "hi", scale: 2}`), Float(13), Float(2.5), Str("zeros"), Int(2),
		Str(`(;db=(;col="lbl"))`), Str(`(;computed)`), Str(`(;route="/sum")`),
	})

	// A field value is checked against its declared type; an unknown member is
	// rejected.
	testExpectRun(t, `type nums []int { scale int = 1 }; n := nums(1)
		try { n.scale = "x"; return "ok" } catch { return "no" }`, nil, Str("no"))
	testExpectRun(t, `type nums []int { scale int = 1 }; n := nums(1)
		try { n.nope = 1; return "ok" } catch { return "no" }`, nil, Str("no"))
	testExpectRun(t, `type nums []int; try { nums(1; x=1); return "ok" } catch { return "no" }`,
		nil, Str("no"))
}

// TestTypedArraySkipItemCheck verifies the item check is on by default and
// RunFlagSkipTypedArrayItemCheck disables it.
func TestTypedArraySkipItemCheck(t *testing.T) {
	src := []byte(`
		type nums []int
		n := nums(1)
		n += "x"
		return str(n)`)
	run := func(flags RunFlags) (Object, error) {
		builtins := NewBuiltins()
		cr, err := Compile(NewSymbolTable(builtins.NameSet), src, CompileOptions{})
		if err != nil {
			return nil, err
		}
		return NewVM(builtins.Build(), cr.Bytecode).RunOpts(&RunOpts{Flags: flags})
	}

	_, err := run(0)
	require.Error(t, err, "the item check is on by default")

	got, err := run(RunFlagSkipTypedArrayItemCheck)
	require.NoError(t, err)
	require.Equal(t, Str(`nums[1, "x"]`), got)
}

// An array type reflects itself with the keys an array-of-types interface
// answers to: `@depth` (how many `[]`) and `@elem` (the element types, resolved).
func TestArrayTypeReflection(t *testing.T) {
	testExpectRun(t, `
		interface Form { tags [][]<int|str>; names []str }
		a := Form.tags.types[0]
		b := Form.names.types[0]
		return [a.@depth, len(a.@elem), b.@depth, b.@elem[0] == str]`, nil,
		Array{Int(2), Int(2), Int(1), True})
}
