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

// TestClassInstanceInterface checks the `classInstance` interface: it matches an
// instance of a user-defined class, and rejects the class itself and non-class
// values.
func TestClassInstanceInterface(t *testing.T) {
	ci := ClassInstanceInterface

	// A class instance satisfies it; the class object and other values do not.
	testExpectRun(t, `C := Class("C", (cls, def) => def(; fields=(; x=(=0))))
		f := func(o classInstance) => 1
		return f(C())`, nil, Int(1))
	testExpectRun(t, `C := Class("C", (cls, def) => def(; fields=(; x=(=0))))
		f := func(o classInstance) => 1
		try { f(C); return "accepted" } catch e { return "rejected" }`, nil, Str("rejected"))
	testExpectRun(t, `f := func(o classInstance) => 1
		try { f(5); return "accepted" } catch e { return "rejected" }`, nil, Str("rejected"))

	require.Same(t, ci, BuiltinObjects[BuiltinClassInstance])
	require.Equal(t, "classInstance", ci.Name())
}

// TestClassTypeInterface checks the `classType` interface: it matches a
// user-defined class object (rejecting its instances and non-class values), and
// calling such a class yields a classInstance.
func TestClassTypeInterface(t *testing.T) {
	ct := ClassTypeInterface

	testExpectRun(t, `C := Class("C", (cls, def) => def(; fields=(; x=(=0))))
		f := func(t classType) => 1
		return f(C)`, nil, Int(1))
	testExpectRun(t, `C := Class("C", (cls, def) => def(; fields=(; x=(=0))))
		f := func(t classType) => 1
		try { f(C()); return "accepted" } catch e { return "rejected" }`, nil, Str("rejected"))
	// A classType is callable and returns a classInstance.
	testExpectRun(t, `C := Class("C", (cls, def) => def(; fields=(; x=(=0))))
		make := func(t classType) <classInstance> => t()
		return typeName(make(C))`, nil, Str("C"))

	require.Same(t, ct, BuiltinObjects[BuiltinClassType])
	require.Equal(t, "classType", ct.Name())
}

// TestReadableWritableInterfaces checks the `readable`/`writable` interfaces:
// they match any value that can be read from / written to (a buffer), and reject
// non-I/O values.
func TestReadableWritableInterfaces(t *testing.T) {
	require.Same(t, ReadableInterface, BuiltinObjects[BuiltinReadable])
	require.Same(t, WritableInterface, BuiltinObjects[BuiltinWritable])
	require.Equal(t, "readable", ReadableInterface.Name())
	require.Equal(t, "writable", WritableInterface.Name())

	testExpectRun(t, `b := buffer(); f := func(w writable) { write(w, "hi"); return str(w) }
		return f(b)`, nil, Str("hi"))
	testExpectRun(t, `f := func(w writable) => 1
		try { f(5); return "accepted" } catch e { return "rejected" }`, nil, Str("rejected"))
	testExpectRun(t, `b := buffer(); write(b, "data")
		f := func(r readable) => str(read(r))
		return f(b)`, nil, Str("data"))
	testExpectRun(t, `f := func(r readable) => 1
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

// TestTypeUnionSyntax checks the `type <T1|T2|…>` first-class type-union value
// (expression form) and the `type NAME <…>` declaration (statement form).
func TestTypeUnionSyntax(t *testing.T) {
	// Expression form: build a union and use it as a parameter type and `::` cast.
	testExpectRun(t, `n := type <int|uint>; return typeName(n)`, nil, Str("typeUnion"))
	testExpectRun(t, `const num = type <int|uint|float>
		f := func(v num) => v + 1
		return [f(1), f(2u), f(3.5)]`, nil, Array{Int(2), Uint(3), Float(4.5)})
	testExpectRun(t, `const num = type <int|uint>; return 1 :: num`, nil, Int(1))
	testExpectRun(t, `const num = type <int|uint>
		try { "x" :: num; return "cast" } catch e { return "rejected" }`, nil, Str("rejected"))

	// Statement form: `type NAME <…>` is sugar for `const NAME = type <…>`.
	testExpectRun(t, `type num <int|uint|float|decimal>
		f := func(v num) => v + 1
		return f(4)`, nil, Int(5))

	// A named union nests inside an inline union in a parameter type.
	testExpectRun(t, `type num <int|uint>
		f := func(a str|num) => 1
		return [f("x"), f(5)]`, nil, Array{Int(1), Int(1)})

	// `type` remains an ordinary identifier when not followed by `<`.
	testExpectRun(t, `type := 5; return type + 1`, nil, Int(6))
}

// TestParamTypeFromEnclosingLocal covers a parameter type that names a local of
// an enclosing scope: the type must be captured into the function's closure so it
// resolves correctly even when the function runs in a different frame (regression
// for the ScopeLocal type-symbol read from the wrong frame).
func TestParamTypeFromEnclosingLocal(t *testing.T) {
	// A type union held in an enclosing local, used as a parameter type of a
	// returned closure: the closure is called from the top-level frame, unrelated
	// to the frame where the union local lives.
	testExpectRun(t, `outer := func() {
			MyInt := type <int|uint>
			return func(v MyInt) => v + 1
		}
		g := outer()
		return g(41)`, nil, Int(42))

	// The same closure invoked indirectly (through another function's frame).
	testExpectRun(t, `outer := func() {
			MyInt := type <int|uint>
			return func(v MyInt) => v + 1
		}
		apply := func(fn callable, x) => fn(x)
		return apply(outer(), 41)`, nil, Int(42))

	// The captured type still rejects a non-matching value from the wrong frame.
	testExpectRun(t, `outer := func() {
			MyInt := type <int|uint>
			return func(v MyInt) => 1
		}
		g := outer()
		try { g("x"); return "accepted" } catch e { return "rejected" }`,
		nil, Str("rejected"))

	// A class method whose injected `this cls` receiver names the enclosing
	// define-callback local stays registrable (not turned into an anonymous
	// closure) — the receiver type is left uncaptured.
	testExpectRun(t, `C := Class("C", (cls, def) => def(; fields=(; x=(=0)),
			methods=(; get=(self) => self.x)))
		return C(; x=7).get()`, nil, Int(7))
}
