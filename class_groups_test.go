package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// A group of a class — `{ … }` in its body — compiles to the field `$N` at
// its place, typed by the group's own class: `@groups` lists them in order,
// the metadata written before the block is the field's, `? { … }` is
// nullable, groups nest; the field is reached by its name like any other.
func TestClassGroups(t *testing.T) {
	testExpectRun(t, `
		class C {
			a str
			[width=2] { b str; { c int } }
			? { d str }
		}
		names := []
		for f in C.@groups { names += f.@name }
		g := C.@fields["$1"]
		return [names, str(g.@meta), len(C.@fields), g.@nullable, C.@fields["$2"].@nullable]`,
		nil, Array{Array{Str("$1"), Str("$2")}, Str("(;width=2)"), Int(3), False, True})

	// the fields of an ordered class, the groups in their places
	testExpectRun(t, `
		[ordered] class C { z str; { y int }; a str; ? { x str } }
		c := C(; z="1", a="2")
		return [sort(collect(keys(C.@fields))), str(c)]`,
		nil, Array{
			Array{Str("$1"), Str("$2"), Str("a"), Str("z")},
			Str("‹class instance of ‹(main).C›: {z: 1, $1: nil, a: 2, $2: nil}›"),
		})

	// a group's field holds an instance of the group's class, by its name
	testExpectRun(t, `
		class C { { b str; { c int } } }
		G := C.@fields["$1"].@types[0]
		c := C(; $1=G(; b="x"))
		d := C(; $1={b: "y"})
		return [c.$1.b, d.$1.b, len(G.@groups)]`,
		nil, Array{Str("x"), Str("y"), Int(1)})

	// written by hand, `$1 class { … }` is the same field as a group
	testExpectRun(t, `
		class A { { b int } }
		class B { $1 class { b int } }
		return [len(A.@groups), len(B.@groups), A(; $1={b: 1}).$1.b, B(; $1={b: 2}).$1.b]`,
		nil, Array{Int(1), Int(1), Int(1), Int(2)})
}

// The class of a group is named as an anonymous field's, by its path below
// the class: `O.$1`, its group's `O.$1.$1`, a group of a field's class
// `O.a.$1`.
func TestClassGroupNames(t *testing.T) {
	testExpectRun(t, `
		class O { a class { { z int } }; { x int; { y int } } }
		cls := func(c, name) => c.@fields[name].@types[0]
		return [cls(O, "$1").@name, cls(cls(O, "$1"), "$1").@name, cls(cls(O, "a"), "$1").@name]`,
		nil, Array{Str("O.$1"), Str("O.$1.$1"), Str("O.a.$1")})
}

// In a mixin a group is a class too: the class that uses the mixin has the
// field `$N` of it, and the mixin lists it in `@groups`.
func TestMixinGroups(t *testing.T) {
	testExpectRun(t, `
		mixin M { { x int }; y str }
		class C { use M }
		return [M.@groups[0].@name, typeName(C.@fields["$1"].@types[0])]`,
		nil, Array{Str("$1"), Str("Class")})
}

// In an interface a group is an interface: `$N interface { … }`, listed in
// `groups` (as its `fields`, no `@`); a value satisfies it when its `$N` does — a nullable group may
// be nil.
func TestInterfaceGroups(t *testing.T) {
	testExpectRun(t, `
		interface I { a str; { b int }; ? { c str } }
		ok := func(v) { try { v :: I; return true } catch { return false } }
		return [len(I.groups), I.groups[0].name,
			ok({a: "x", "$1": {b: 1}}),
			ok({a: "x", "$1": {b: 1}, "$2": nil}),
			ok({a: "x"})]`,
		nil, Array{Int(2), Str("$1"), True, True, False})
}

// Only a group's name is one: `$` and digits, which no identifier is.
func TestIsGroupName(t *testing.T) {
	for name, want := range map[string]bool{"$1": true, "$12": true, "$": false, "$a": false, "a": false, "1": false} {
		if got := IsGroupName(name); got != want {
			t.Errorf("IsGroupName(%q) = %v", name, got)
		}
	}
}

// The compiler gives a group's field the name `$N` and its class the name of
// its path (`O.$1`): both among the constants of the bytecode; a group's
// metadata, its field's.
func TestCompileClassGroups(t *testing.T) {
	bi := NewBuiltins()
	res, err := Compile(NewSymbolTable(bi.NameSet), []byte(`class O { a str; [width=2] { b int }; ? { c str } }`), CompileOptions{})
	if err != nil {
		t.Fatal(err)
	}
	has := map[string]bool{}
	var walk func(o Object)
	walk = func(o Object) {
		switch v := o.(type) {
		case Str:
			has[string(v)] = true
		case KeyValueArray:
			for _, kv := range v {
				walk(kv.K)
				walk(kv.V)
			}
		case Array:
			for _, x := range v {
				walk(x)
			}
		case *TypedIdent:
			has[v.Name] = true
		}
	}
	for _, c := range res.Bytecode.Constants {
		walk(c)
	}
	for _, want := range []string{"$1", "$2", "O.$1", "O.$2", "width"} {
		if !has[want] {
			t.Errorf("%q not among the constants", want)
		}
	}
}
