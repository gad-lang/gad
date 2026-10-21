package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// `Class.@flat`: its fields and its parents', flat, by name. An `[ordered]`
// class's as its body declares them — several parents (`*P`, each with its
// own parents), mixins (`use`, one or several in a clause) and spreads
// (`**E`), in any order, each one's fields in its place —; its instances
// give them in that order. `[ordered]` or not: a class is always ordered.
func TestClassFlat(t *testing.T) {
	src := `
		class P1 { p1a str; p1b str }
		class P0 { p0 str }
		class P2 { *P0; p2 str }
		mixin MA { ma str }
		mixin MB { mb str }
		mixin MC { mc str }
		[ordered]
		class C {
			z str
			*P1
			a int
			use MA, MB
			**{fields: (;s1=str)}
			b int
			*P2
			use MC
			**{fields: (;s2=str, s3=str)}
			c int
			**{fields: (;s4=str)}
		}
		class D { z str; *P1; a int; use MA }
		`
	testExpectRun(t, src+`return collect(keys(C.@flat))`, nil, Array{
		Str("z"), Str("p1a"), Str("p1b"), Str("a"), Str("ma"), Str("mb"), Str("s1"), Str("b"),
		Str("p0"), Str("p2"), Str("mc"), Str("s2"), Str("s3"), Str("c"), Str("s4"),
	})
	testExpectRun(t, src+`return collect(keys(D.@flat))`, nil, Array{Str("z"), Str("p1a"), Str("p1b"), Str("a"), Str("ma")})
	// the fields themselves: a parent's, of its class
	testExpectRun(t, src+`f := C.@flat; return [typeName(f[1].v), f[1].v.@name, len(f)]`, nil, Array{Str("ClassField"), Str("p1a"), Int(15)})
	// an instance, in the same order
	testExpectRun(t, src+`return str(C(; z="z", a=1, b=2, c=3))`, nil,
		Str("‹class instance of ‹(main).C›: {z: \"z\", p1a: nil, p1b: nil, a: 1, ma: nil, mb: nil, s1: ‹builtin type ‹str› with 9 methods›, b: 2, p0: nil, p2: nil, mc: nil, s2: ‹builtin type ‹str› with 9 methods›, s3: ‹builtin type ‹str› with 9 methods›, c: 3, s4: ‹builtin type ‹str› with 9 methods›}›"))
	// a field declared again: once, where first declared, the nearest's
	testExpectRun(t, `
		class P { a str; b str }
		[ordered] class C { x str; *P; [label="mine"] a str }
		f := C.@flat
		return [collect(keys(f)), str(f[1].v.@meta)]`, nil,
		Array{Array{Str("x"), Str("a"), Str("b")}, Str("(;label=\"mine\")")})
}
