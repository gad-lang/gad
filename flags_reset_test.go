package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// `yes` is a key given without a value; `no` resets the key — as if not given
// —: it removes the pairs of the key before it and stays in their place,
// until a pair of the key after it takes its place; `true`, `false`, `nil`
// and the other values are values given.
func TestFlagsYesNoReset(t *testing.T) {
	// the sequence: a reset removes the pairs of its key before it and stays,
	// until a pair of the key after it takes its place
	testExpectRun(t, `return (;a=1, a=2, a=no, a=3, a=4) == (;a=3, a=4)`, nil, True)
	testExpectRun(t, `return [str((;a=1, a=2, a=no, a=3, a=4)), str((;a=2, a=no)), str((;a=no, a=2)), str((;a=no)), str((;a=no, a=no))]`,
		nil, Array{Str("(;a=3, a=4)"), Str("(;a=no)"), Str("(;a=2)"), Str("(;a=no)"), Str("(;a=no)")})
	testExpectRun(t, `return [len((;a=2, a=no)), (;a=2, a=no) == (;a=no), (;a=2, a=no) == (;)]`, nil, Array{Int(1), True, False})
	// a reset resets what comes before it in the same sequence
	testExpectRun(t, `return [dict((;a, a=2)), dict((;a, a=2, a=no))]`, nil, Array{Dict{"a": Int(2)}, Dict{}})
	testExpectRun(t, `base := (;a=1, b=2); return [str((;**base, a=no)), str((;a=no, **base)), str(base + (;b=3))]`,
		nil, Array{Str("(;b=2, a=no)"), Str("(;a=1, b=2)"), Str("(;a=1, b=2, b=3)")})
	// a reset stays, so it resets the key of what it is added to
	testExpectRun(t, `return [str((;a) + (;a=2) + (;a=no)), dict((;a) + (;a=2) + (;a=no)), dict((;a=no))]`,
		nil, Array{Str("(;a=no)"), Dict{}, Dict{}})
	testExpectRun(t, `return [str({a: no} ::> keyValueArray), str(keyValueArray({a: no}))]`,
		nil, Array{Str("(;a=no)"), Str("(;a=no)")})
	testExpectRun(t, `k := (;a=no); return ["a" in k, k.get("a"), k.values("a"), k.flag("a")]`,
		nil, Array{False, Nil, Array(nil), False})
	// `+` leaves its operands as they were
	testExpectRun(t, `x := (;a=1); y := x + (;b=2); return [str(x), str(y)]`, nil, Array{Str("(;a=1)"), Str("(;a=1, b=2)")})
	// a value of no read at run time resets too
	testExpectRun(t, `v := no; return dict((;a=1, a=v))`, nil, Dict{})

	// given without a value, and values given
	testExpectRun(t, `kva := (;a, b=yes, c=true, d=nil); return [kva.get("a") == yes, kva.get("b") == yes, typeName(kva.get("c")), kva.get("d"), "d" in kva]`,
		nil, Array{True, True, Str("bool"), Nil, True})
	testExpectRun(t, `return [typeName((;a)[0].v), typeName((;a=true)[0].v)]`, nil, Array{Str("flag"), Str("bool")})

	// metadata: the same rule
	testExpectRun(t, "base := (;a=1, b=2)\n[a=1, a=no, b, c=no, **base, b=no]\nclass X {}\nreturn [str(X.@meta), \"c\" in X.@meta, \"b\" in X.@meta]",
		nil, Array{Str("(;c=no, a=1, b=no)"), False, False})

	// named arguments: one reset is not given — the default applies —, nil is
	// a value given
	testExpectRun(t, `f := func(; a=1) => a; return [f(), f(; a=no), f(; a=yes), f(; a=nil), f(; a=2, a=no), f(; a=2)]`,
		nil, Array{Int(1), Int(1), Yes, Nil, Int(1), Int(2)})
	testExpectRun(t, `g := func(; a) => a; return [g(), g(; a=nil), g(; a=no)]`, nil, Array{Nil, Nil, Nil})
	testExpectRun(t, `f := func(; a=1) => a; base := (;a=5); return [f(; **base), f(; **base, a=no)]`,
		nil, Array{Int(5), Int(1)})
	// the rest (`**na`) has no reset key
	testExpectRun(t, `f := func(; a=1, **rest) => [a, dict(rest)]; return f(; **(;a=5), a=no, b=2, b=no, c)`,
		nil, Array{Int(1), Dict{"c": Yes}})
	// resetting a name the function does not take is no error
	testExpectRun(t, `f := func(x; y=1) => y; return f(0; zz=no)`, nil, Int(1))
}

// `a ?! b` is b when a is `no`, else a; `a ?!= b` assigns b when a is `no`.
func TestNoCoalesce(t *testing.T) {
	testExpectRun(t, `x := no; return [x ?! 5, nil ?! 5, 0 ?! 5, false ?! 5, yes ?! 5, no ?! no ?! 6]`,
		nil, Array{Int(5), Nil, Int(0), False, Yes, Int(6)})
	testExpectRun(t, `y := no; y ?!= 7; z := nil; z ?!= 7; w := 1; w ?!= 7; return [y, z, w]`,
		nil, Array{Int(7), Nil, Int(1)})
	// lazy: the default is not evaluated for a value given
	testExpectRun(t, `n := 0; f := func() { n++; return 1 }; v := 2 ?! f(); return [v, n]`, nil, Array{Int(2), Int(0)})
	// a named param not given is `no` until its default
	testExpectRun(t, `f := func(; a=1) => a; return f(; a=no ?! 3)`, nil, Int(3))
}

// A key-value array built in Go is read with its resets applied: its items,
// its length, its lookups, its dict, its equality.
func TestKeyValueArrayFromGoNormalized(t *testing.T) {
	kva := KeyValueArray{
		&KeyValue{K: Str("a"), V: Int(1)},
		&KeyValue{K: Str("a"), V: No},
		&KeyValue{K: Str("b"), V: Int(2)},
		&KeyValue{K: Str("c"), V: Int(3)},
		&KeyValue{K: Str("c"), V: No},
	}
	want := KeyValueArray{
		&KeyValue{K: Str("a"), V: No},
		&KeyValue{K: Str("b"), V: Int(2)},
		&KeyValue{K: Str("c"), V: No},
	}
	if got := kva.Normalized(); !got.Equal(want) || len(got) != 3 {
		t.Fatalf("Normalized: %v", got)
	}
	if kva.Length() != 3 || !kva.Equal(want) {
		t.Fatalf("Length %d, Equal", kva.Length())
	}
	testExpectRun(t, `global (kva); return [str(kva), len(kva), "a" in kva, kva.get("c"), dict(kva), [k for k, v in kva]]`,
		newOpts().Globals(Dict{"kva": kva}),
		Array{Str("(;a=no, b=2, c=no)"), Int(3), False, Nil, Dict{"b": Int(2)}, Array{Str("a"), Str("b"), Str("c")}})
	// as named args: the default of a reset param applies
	testExpectRun(t, `global (kva); f := func(; a=9, b=0, c=8) => [a, b, c]; return f(; **kva)`,
		newOpts().Globals(Dict{"kva": kva}), Array{Int(9), Int(2), Int(8)})
}
