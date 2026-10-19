package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// TestClassCastRecursive verifies that `dict ::> Class` builds the instance
// whole: a class field from its dict, an array of classes item by item (of
// any depth), a group (`{ … }`: its field $N, in the dict's path) from its
// dict, an enum
// from a member's name or value (in an array too), the defaults of what is
// not given.
func TestClassCastRecursive(t *testing.T) {
	testExpectRun(t, `
		enum Kind {
			Kitchen
			Bathroom
		}
		class Address {
			street str
			city str = "Boston"
		}
		class Link { href str }
		class Form {
			name str
			rooms int = 3
			home Address
			work? Address
			links []Link
			grid [][]Link
			kind Kind
			kinds []Kind
			{
				a int
				b? str = "bee"
			}
		}
		f := {name: "Ana", home: {street: "Main"}, links: [{href: "x"}, {href: "y"}],
			grid: [[{href: "g"}]], kind: "Bathroom", kinds: ["Kitchen", "Bathroom"], $1: {a: 1}} ::> Form
		return [
			f.home.city, f.home.street, f.links[1].href, f.grid[0][0].href,
			f.kind == Kind.Bathroom, f.kinds[0] == Kind.Kitchen,
			f.rooms, f.$1.a, f.$1.b, f.work,
		]`,
		nil, Array{Str("Boston"), Str("Main"), Str("y"), Str("g"), True, True, Int(3), Int(1), Str("bee"), Nil})
}

// An array item a class cannot be built from is refused.
func TestClassCastRecursiveRefused(t *testing.T) {
	testExpectRun(t, `
		class Link { href str }
		class Form { links []Link }
		try {
			x := {links: [1]} ::> Form
		} catch e {
			return "refused"
		}
		return "taken"`,
		nil, Str("refused"))
}

// [autoGroupKeys]: a group's keys given flat go to it (the constructor and
// `::>`), are read flat, and each group is an instance even when none of its
// fields is given (an optional one stays nil); a group in a group too.
// Without it, a group's keys are required under it ($1).
func TestClassAutoGroupKeys(t *testing.T) {
	testExpectRun(t, `
		[autoGroupKeys]
		class Form {
			name str
			{
				a int
				b? str = "bee"
				{ c int }
			}
			? { d str }
		}
		f := {name: "Ana", a: 1, c: 3} ::> Form
		g := Form(; name="x", a=2)
		g.b = "set"
		return [f.a, f.b, f.c, f.$1.a, f.$2, g.a, g.b, g.$1.$1 != nil]`,
		nil, Array{Int(1), Str("bee"), Int(3), Int(1), Nil, Int(2), Str("set"), True})

	// the default: required under $N — flat keys are no group's
	testExpectRun(t, `
		class Form {
			name str
			{ a int }
		}
		f := {name: "x", a: 1} ::> Form
		g := {name: "x", $1: {a: 2}} ::> Form
		return [f.$1, g.$1.a]`,
		nil, Array{Nil, Int(2)})
}
