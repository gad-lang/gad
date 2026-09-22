package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// TestMetadataInterface verifies the `[k=v, …]` metadata block on an interface
// and its members is compiled into a KeyValueArray reachable via `@meta`.
func TestMetadataInterface(t *testing.T) {
	src := `
		[metakey="value", version=2]
		interface User {
			[db=(;primary_key)]
			id uint

			[route="/n"]
			get n int

			[rpc]
			save() <bool>
		}
		return [
			str(User.@meta),
			str(User.id.@meta),
			str(User.n.@meta),
			str(User.save.@meta),
		]`
	testExpectRun(t, src, nil, Array{
		Str(`(;metakey="value", version=2)`),
		Str(`(;db=(;primary_key))`),
		Str(`(;route="/n")`),
		Str(`(;rpc)`),
	})

	// An element with no metadata yields an empty key-value array (never errors).
	testExpectRun(t, `interface A { x int }; return str(A.@meta)`, nil, Str("(;)"))
	testExpectRun(t, `interface A { x int }; return str(A.x.@meta)`, nil, Str("(;)"))
}

// TestMetadataEnum verifies metadata on an enum and its items.
func TestMetadataEnum(t *testing.T) {
	src := `
		[category="acl"]
		enum Perm {
			[bit_pos=0]
			Read
			[bit_pos=1]
			Write
		}
		return [str(Perm.@meta), str(Perm.Read.@meta), str(Perm.Write.@meta)]`
	testExpectRun(t, src, nil, Array{
		Str(`(;category="acl")`),
		Str(`(;bit_pos=0)`),
		Str(`(;bit_pos=1)`),
	})
}

// TestMetadataRealExpressions verifies metadata values are real constant
// expressions (nested key-value arrays, arrays, dicts, flags), not only scalars.
func TestMetadataRealExpressions(t *testing.T) {
	src := `
		[tags=["a", "b"], opts=(;primary_key, order=3)]
		interface T { x int }
		m := T.@meta
		return [len(m), str(m)]`
	testExpectRun(t, src, nil, Array{
		Int(2),
		Str(`(;tags=["a", "b"], opts=(;primary_key, order=3))`),
	})

	// A dict value works too (a Dict is a real key-value expression); its keys
	// print in map order, so assert its length rather than its string.
	testExpectRun(t, `[shape={w: 1, h: 2}] interface T { x int }; return len(T.@meta[0].v)`,
		nil, Int(2))
}

// TestMetadataFunc verifies function metadata and the function reflection keys.
func TestMetadataFunc(t *testing.T) {
	// Named func: @meta plus @name/@args/@nargs/@ret reflection.
	testExpectRun(t, `
		[route="/save", auth]
		func save(a, b; opt=1) <int> { return a }
		return [str(save.@meta), save.@name, save.@args, save.@nargs, save.@ret, len(save.@methods)]`,
		nil, Array{
			Str(`(;route="/save", auth)`),
			Str("save"),
			Array{Str("a"), Str("b")},
			Array{Str("opt")},
			Str("<int>"),
			Int(1),
		})

	// An anonymous func with no metadata reports an empty key-value array.
	testExpectRun(t, `f := func(x) => x; return str(f.@meta)`, nil, Str("(;)"))
}

// TestMetadataClass verifies metadata on a class and its fields, methods and
// properties, plus a class field default is preserved alongside its metadata.
func TestMetadataClass(t *testing.T) {
	src := `
		[table="users", version=2]
		class User {
			[db=(;primary_key)]
			id = 0
			[db=(;column="full_name")]
			label = "anon"
			props {
				[computed]
				display => this.label
			}
			methods {
				[route="/save", method="POST"]
				save() { return this.id }
			}
		}
		u := User()
		return [
			str(User.@meta),
			str(User.id.@meta),
			str(User.label.@meta),
			str(User.display.@meta),
			str(User.save.@meta),
			u.id, u.label,   // defaults preserved
		]`
	testExpectRun(t, src, nil, Array{
		Str(`(;table="users", version=2)`),
		Str(`(;db=(;primary_key))`),
		Str(`(;db=(;column="full_name"))`),
		Str(`(;computed)`),
		Str(`(;route="/save", method="POST")`),
		Int(0), Str("anon"),
	})

	// A class/member with no metadata reports an empty key-value array.
	testExpectRun(t, `class C { x = 1 }; return str(C.@meta)`, nil, Str("(;)"))
	testExpectRun(t, `class C { x = 1 }; return str(C.x.@meta)`, nil, Str("(;)"))
}

// TestMetadataMarkerAndMixin verifies metadata on a marker `type` and a `mixin`.
func TestMetadataMarkerAndMixin(t *testing.T) {
	testExpectRun(t, `
		[kind="marker"]
		type Color { call() { return "c" } }
		return str(Color.@meta)`, nil, Str(`(;kind="marker")`))

	testExpectRun(t, `
		[role="mix"]
		mixin Timestamped { created = 0 }
		return str(Timestamped.@meta)`, nil, Str(`(;role="mix")`))
}
