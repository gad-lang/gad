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
