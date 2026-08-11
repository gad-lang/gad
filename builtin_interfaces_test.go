package gad_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/gad-lang/gad"
)

// TestIterableInterface checks the builtin `iterable` interface: its native
// satisfaction accepts iterables (array/dict/str/iterator) and rejects
// non-iterables, both directly (no VM) and as a parameter type at run time.
func TestIterableInterface(t *testing.T) {
	// Direct, VM-less structural check (Go Iterabler / Iterator).
	for _, it := range []Object{Array{Int(1)}, Dict{"a": Int(1)}} {
		ok, err := IterableInterface.CanAssign(it)
		require.NoError(t, err)
		require.True(t, ok, it.Type().Name())
	}
	for _, notIt := range []Object{Int(5), Bool(true), Nil} {
		ok, err := IterableInterface.CanAssign(notIt)
		require.NoError(t, err)
		require.False(t, ok, notIt.Type().Name())
	}

	// The builtin resolves to the iterable interface value.
	require.Same(t, IterableInterface, BuiltinObjects[BuiltinIterable])

	// As a parameter type through the VM: every iterable is accepted, and a
	// non-iterable is rejected.
	testExpectRun(t, `f := func(x iterable) => len(x); return f([1, 2, 3])`, nil, Int(3))
	testExpectRun(t, `f := func(x iterable) => 1; return [f({a: 1}), f("hi"), f(iterate([1]))]`,
		nil, Array{Int(1), Int(1), Int(1)})
	testExpectRun(t, `f := func(x iterable) => 1
try { f(5); return "accepted" } catch e { return "rejected" }`, nil, Str("rejected"))
}

// TestBehaviouralInterfaces checks the builtin behavioural interfaces: callable,
// lengther, indexable, indexAssignable and indexDeletable, each backed by its
// Go predicate/interface.
func TestBehaviouralInterfaces(t *testing.T) {
	fn := &Function{FuncName: "f", Value: func(Call) (Object, error) { return Nil, nil }}
	cases := []struct {
		iface *Interface
		yes   []Object
		no    []Object
	}{
		{CallableInterface, []Object{fn}, []Object{Int(5), Str("x")}},
		{LengtherInterface, []Object{Array{Int(1)}, Str("hi"), Dict{}}, []Object{Int(5), Bool(true)}},
		{IndexableInterface, []Object{Array{Int(1)}, Dict{}}, []Object{Int(5)}},
		{IndexAssignableInterface, []Object{Array{Int(1)}, Dict{}}, []Object{Int(5), Str("x")}},
		{IndexDeletableInterface, []Object{Dict{}}, []Object{Int(5)}},
	}
	for _, c := range cases {
		for _, o := range c.yes {
			ok, err := c.iface.CanAssign(o)
			require.NoError(t, err)
			require.True(t, ok, "%s should accept %s", c.iface.Name(), o.Type().Name())
		}
		for _, o := range c.no {
			ok, err := c.iface.CanAssign(o)
			require.NoError(t, err)
			require.False(t, ok, "%s should reject %s", c.iface.Name(), o.Type().Name())
		}
	}

	// Usable as parameter types through the VM.
	testExpectRun(t, `f := func(x callable) => 1; return f(func() => 0)`, nil, Int(1))
	testExpectRun(t, `f := func(x lengther) => len(x); return f("abc")`, nil, Int(3))
	testExpectRun(t, `f := func(x indexable) => x[0]; return f([7, 8])`, nil, Int(7))
	testExpectRun(t, `f := func(x callable) => 1
try { f(5); return "accepted" } catch e { return "rejected" }`, nil, Str("rejected"))
}

// TestNumberTypeUnion checks the builtin `number` type union (int|uint|float|
// decimal): direct assignability, use as a parameter/return type, the `::` cast,
// and nesting inside another union (`str|number`).
func TestNumberTypeUnion(t *testing.T) {
	for _, n := range []Object{Int(1), Uint(2), Float(3.5), DecimalFromInt(4)} {
		ok, err := NumberTypeUnion.CanAssign(n)
		require.NoError(t, err)
		require.True(t, ok, n.Type().Name())
	}
	for _, notN := range []Object{Str("x"), Bool(true), Array{}} {
		ok, err := NumberTypeUnion.CanAssign(notN)
		require.NoError(t, err)
		require.False(t, ok, notN.Type().Name())
	}
	require.Same(t, NumberTypeUnion, BuiltinObjects[BuiltinNumberTypeUnion])
	require.Equal(t, "number", NumberTypeUnion.ToString())

	// As a parameter type, a return type and the `::` cast.
	testExpectRun(t, `f := func(v number) => v + 1; return [f(1), f(2.5), f(3u)]`,
		nil, Array{Int(2), Float(3.5), Uint(4)})
	testExpectRun(t, `f := func(v number) <number> => v + 1; return f(4)`, nil, Int(5))
	testExpectRun(t, `return 1 :: number`, nil, Int(1))
	testExpectRun(t, `try { "x" :: number; return "cast" } catch e { return "rejected" }`,
		nil, Str("rejected"))

	// A union nests inside another union: `str|number` accepts strings and numbers.
	testExpectRun(t, `f := func(a str|number) => 1; return [f("x"), f(5), f(1.5)]`,
		nil, Array{Int(1), Int(1), Int(1)})
	testExpectRun(t, `f := func(v number) => 1
try { f("x"); return "accepted" } catch e { return "rejected" }`, nil, Str("rejected"))
}
