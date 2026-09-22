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
