package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// TestMetaTypeDispatch verifies a `type<X>` parameter dispatches on the TYPE VALUE
// X: passing the class Rect selects the type<Rect> overload, Point the type<Point>
// one, and the bound `t` is the class itself.
func TestMetaTypeDispatch(t *testing.T) {
	testExpectRun(t, `
		class Rect {}
		class Point {}
		func d {
			(t type<Rect>)  => [t == Rect, 1]
			(t type<Point>) => [t == Point, 2]
		}
		return [d(Rect), d(Point)]`,
		nil, Array{Array{True, Int(1)}, Array{True, Int(2)}})
}

// TestMetaTypeSeparatesInstanceAndType verifies Option A: an instance overload
// `(t X)` and a type-value overload `(t type<X>)` coexist and dispatch distinctly —
// an instance of X hits the first, the type value X the second.
func TestMetaTypeSeparatesInstanceAndType(t *testing.T) {
	testExpectRun(t, `
		class Rect { w = 7 }
		func k {
			(t Rect)       => ["instance", t.w]
			(t type<Rect>) => ["type", t == Rect]
		}
		return [k(Rect()), k(Rect)]`,
		nil, Array{Array{Str("instance"), Int(7)}, Array{Str("type"), True}})
}

// TestMetaTypeRejectsInstance verifies a `type<X>` parameter does NOT accept an
// instance of X (a type value is not an instance).
func TestMetaTypeRejectsInstance(t *testing.T) {
	testExpectRun(t, `
		class Rect {}
		func d { (t type<Rect>) => 1 }
		ok := false
		try { d(Rect()) } catch { ok = true }
		return ok`,
		nil, True)
}

// TestMetaTypeUsableAsType verifies the bound `t` (the type value) can be used as a
// type — e.g. to construct an instance.
func TestMetaTypeUsableAsType(t *testing.T) {
	testExpectRun(t, `
		class Rect { w = 9 }
		func make(t type<Rect>) => t()
		return make(Rect).w`,
		nil, Int(9))
}

// TestMetaTypeMethodForm verifies `met f(_ type<X>)` adds a type-value overload to
// an existing callable.
func TestMetaTypeMethodForm(t *testing.T) {
	testExpectRun(t, `
		class Rect {}
		func g(x) => "any"
		met g(_ type<Rect>) => "type<Rect>"
		return [g(Rect), g(123)]`,
		nil, Array{Str("type<Rect>"), Str("any")})
}

// TestMetaTypeUnion verifies a `type<X|Y>` parameter accepts the type value X or Y
// and rejects others.
func TestMetaTypeUnion(t *testing.T) {
	testExpectRun(t, `
		func a(t type<int|bool>) => 1
		ok := false
		try { a(str) } catch { ok = true }
		return [a(int), a(bool), ok]`,
		nil, Array{Int(1), Int(1), True})
}

// TestStaticTypeMembers verifies a marker type's static field, method (this = the
// type) and property.
func TestStaticTypeMembers(t *testing.T) {
	testExpectRun(t, `
		type Color {
			red = "#f00"
			methods { describe() => "red=" + this.red }
			props { count => 1 }
		}
		return [Color.red, Color.describe(), Color.count]`,
		nil, Array{Str("#f00"), Str("red=#f00"), Int(1)})
}

// TestStaticTypeCallFactory verifies the `call(…)` factory: it is invoked as
// Name(…) and returns an arbitrary value (not an instance).
func TestStaticTypeCallFactory(t *testing.T) {
	testExpectRun(t, `
		type Maker { call(n) => "made:" + n }
		return Maker("x")`,
		nil, Str("made:x"))
}

// TestStaticTypeExprForm verifies the anonymous `const X = type { … }` form.
func TestStaticTypeExprForm(t *testing.T) {
	testExpectRun(t, `
		const Dir = type { up = 1; down = 2 }
		return [Dir.up, Dir.down]`,
		nil, Array{Int(1), Int(2)})
}

// TestStaticTypeMetaDispatch verifies a marker type dispatches through type<Name>.
func TestStaticTypeMetaDispatch(t *testing.T) {
	testExpectRun(t, `
		type A {}
		type B {}
		func d {
			(t type<A>) => "a"
			(t type<B>) => "b"
		}
		return [d(A), d(B)]`,
		nil, Array{Str("a"), Str("b")})
}

// TestStaticTypeNoInstances verifies `x :: Name` is rejected — a marker has no
// instances.
func TestStaticTypeNoInstances(t *testing.T) {
	testExpectRun(t, `
		type Marker {}
		ok := false
		try { 5 :: Marker } catch { ok = true }
		return ok`,
		nil, True)
}
