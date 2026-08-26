package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// TestTransformCastRestCapture covers the `**name` rest-capture field of an
// interface used with the transforming cast `:::`: keys not named by the
// interface are gathered into a dict bound to that name.
func TestTransformCastRestCapture(t *testing.T) {
	testExpectRun(t, `
		d := {a: 1, b: 2, c: 3}
		r := d ::: interface { a int, **rest }
		return [r.a, r.rest.b, r.rest.c]`, nil, Array{Int(1), Int(2), Int(3)})

	// With no extra keys the rest dict is empty.
	testExpectRun(t, `
		d := {a: 1}
		return len((d ::: interface { a int, **rest }).rest)`, nil, Int(0))
}

// TestTransformCastNestedClass covers github.com/gad-lang/gad issue #5: a dict
// cast to an interface whose field is a class type is coerced into an instance of
// that class, recursively.
func TestTransformCastNestedClass(t *testing.T) {
	base := "class Point { x int; y int }\nclass Rect { a Point; b Point }\n"

	// A class-typed field is built from its nested dict.
	testExpectRun(t, base+`
		d := {rect: {a: {x: 1, y: 2}, b: {x: 5, y: 6}}, z: "z"}
		r := d ::: interface { rect Rect, **other }
		return [typeName(r.rect), typeName(r.rect.a), r.rect.a.x, r.rect.b.y, r.other.z]`,
		nil, Array{Str("Rect"), Str("Point"), Int(1), Int(6), Str("z")})

	// The exact repr from the issue: `rect` becomes a Rect, `z` moves into `other`.
	testExpectRun(t, base+`
		d := {rect: {a: {x: 1, y: 2}, b: {x: 5, y: 6}}, z: "z value"}
		got := repr(d ::: interface { rect Rect, **other })
		want := repr({rect: Rect(; a=Point(;x=1,y=2), b=Point(;x=5,y=6)), other: {z: "z value"}})
		return got == want`, nil, True)
}

// TestTransformCastNestedInterface covers a field typed by an inline interface:
// the field's value stays a dict but ITS fields are coerced (here a/b -> Point).
func TestTransformCastNestedInterface(t *testing.T) {
	base := "class Point { x int; y int }\n"
	testExpectRun(t, base+`
		d := {rect: {a: {x: 1, y: 2}, b: {x: 5, y: 6}}, z: "z value"}
		got := repr(d ::: interface { rect interface{ a Point, b Point }, **otherValue })
		want := repr({rect: {a: Point(;x=1,y=2), b: Point(;x=5,y=6)}, otherValue: {z: "z value"}})
		return got == want`, nil, True)
}

// TestTransformCastNonDictSource covers "any item getter": the transform accepts
// a key-value array and a class instance as the source, not only a dict literal.
func TestTransformCastNonDictSource(t *testing.T) {
	base := "class Point { x int; y int }\nclass Rect { a Point; b Point }\n"

	// A key-value array source, nested all the way down.
	testExpectRun(t, base+`
		kv := (; rect=(; a=(; x=1, y=2), b=(; x=5, y=6)), z="z")
		r := kv ::: interface { rect Rect, **other }
		return [typeName(r.rect), r.rect.a.x, r.other.z]`,
		nil, Array{Str("Rect"), Int(1), Str("z")})

	// A class instance source: its fields feed the transform.
	testExpectRun(t, base+`
		class Wrapper { p Point; extra str }
		w := Wrapper(; p = {x: 1, y: 2}, extra = "hi")
		r := w ::: interface { p Point, **rest }
		return [typeName(r.p), r.p.y, r.rest.extra]`,
		nil, Array{Str("Point"), Int(2), Str("hi")})
}

// TestTransformCastVsCheck contrasts `:::` (transform) with `::` (check): `::`
// leaves the value unchanged and fails when a nested dict is not already the
// declared type, while `:::` coerces it.
func TestTransformCastVsCheck(t *testing.T) {
	base := "class Point { x int; y int }\nclass Rect { a Point; b Point }\n"

	// `::` does not coerce: the dict's `rect` is a dict, not a Rect -> not
	// assignable -> error (caught).
	testExpectRun(t, base+`
		d := {rect: {a: {x: 1, y: 2}, b: {x: 5, y: 6}}}
		ok := false
		try { d :: interface { rect Rect } } catch { ok = true }
		return ok`, nil, True)

	// `:::` coerces and succeeds.
	testExpectRun(t, base+`
		d := {rect: {a: {x: 1, y: 2}, b: {x: 5, y: 6}}}
		return typeName((d ::: interface { rect Rect }).rect)`, nil, Str("Rect"))
}

// TestTransformCastMissingField: a required (non-nullable) field absent from the
// source makes the transform fail; a nullable field may be absent.
func TestTransformCastMissingField(t *testing.T) {
	expectErrHas(t, `
		d := {b: 2}
		return d ::: interface { a int, **rest }`, nil, `field "a" is required`)

	testExpectRun(t, `
		d := {b: 2}
		r := d ::: interface { a? int, **rest }
		return [r.rest.b, r.a == nil]`, nil, Array{Int(2), True})
}

// TestTransformCastBool covers `expr ::: bool`: a fast truthiness conversion that
// yields the same value as the bool() builtin, but as a direct cast (no call).
// Unlike the checked cast `::`, it converts a non-bool instead of erroring.
func TestTransformCastBool(t *testing.T) {
	// Per-shape results, converting (not checking) by truthiness.
	testExpectRun(t, `return [
		5 ::: bool, 0 ::: bool,
		"x" ::: bool, "" ::: bool,
		[1] ::: bool, [] ::: bool,
		{a: 1} ::: bool, {} ::: bool,
		nil ::: bool, true ::: bool, false ::: bool]`,
		nil, Array{True, False, True, False, True, False, True, False, False, True, False})

	// Exact parity with bool() for every value.
	testExpectRun(t, `
		ok := true
		for v in [5, 0, "x", "", [1], [], {a: 1}, {}, nil, true, false, 0.0, 3.14] {
			if bool(v) != (v ::: bool) { ok = false }
		}
		return ok`, nil, True)

	// Floats follow IEEE-754: 0.0 is truthy, only NaN is falsy.
	testExpectRun(t, `return [0.0 ::: bool, 3.14 ::: bool, float("nan") ::: bool]`,
		nil, Array{True, True, False})

	// `:::` converts a non-bool; the checked cast `::` would reject it.
	testExpectRun(t, `return 5 ::: bool`, nil, True)
	expectErrHas(t, `return 5 :: bool`, nil, `not assignable to`)
}

// TestTransformCastClassToClass covers github.com/gad-lang/gad issue #7: `:::` to
// a class builds an instance of that class from the source's members, keeping
// only the fields the target declares — a conversion between class shapes.
func TestTransformCastClassToClass(t *testing.T) {
	base := "class User { name str; isAdmin? bool }\nclass Tag { name str }\n"

	// User -> Tag keeps `name`, drops the undeclared `isAdmin`.
	testExpectRun(t, base+`
		u := User(; name = "Jonh", isAdmin = true)
		return repr(u ::: Tag) == repr(Tag(; name = "Jonh"))`, nil, True)

	// Chained: User -> Tag -> User; the round-trip drops isAdmin (nullable -> nil).
	testExpectRun(t, base+`
		u := User(; name = "Jonh", isAdmin = true)
		return repr(u ::: Tag ::: User) == repr(User(; name = "Jonh"))`, nil, True)

	// A dict source works too.
	testExpectRun(t, base+`
		return repr({name: "Ann", extra: 1} ::: Tag) == repr(Tag(; name = "Ann"))`, nil, True)

	// An instance already of the target class is returned unchanged.
	testExpectRun(t, base+`
		tg := Tag(; name = "x")
		return (tg ::: Tag).name`, nil, Str("x"))

	// A missing required (non-nullable) target field is an error.
	expectErrHas(t, base+`
		return Tag(; name = "n") ::: interface { z int }`, nil, `field "z"`)
}
