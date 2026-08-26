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

// TestTransformCastBuiltinTypes covers `expr ::: T` for builtin types other than
// bool: the cast converts by calling T's constructor, so the typed overloads
// registered with AddMethod apply (str/int/float/uint/char/decimal). `any` is
// identity and an invalid conversion surfaces the constructor's error.
func TestTransformCastBuiltinTypes(t *testing.T) {
	// str: each source shape hits a different str() overload.
	testExpectRun(t, `return [5 ::: str, 3.14 ::: str, true ::: str, 'A' ::: str]`,
		nil, Array{Str("5"), Str("3.14"), Str("true"), Str("A")})

	// int: string parse, float truncation and char code — three overloads.
	testExpectRun(t, `return ["5" ::: int, 3.9 ::: int, 'A' ::: int]`,
		nil, Array{Int(5), Int(3), Int(65)})

	// float / uint / char / decimal.
	testExpectRun(t, `return [5 ::: float, "3.14" ::: float]`, nil, Array{Float(5), Float(3.14)})
	testExpectRun(t, `return 5 ::: uint`, nil, Uint(5))
	testExpectRun(t, `return [65 ::: char, "A" ::: char]`, nil, Array{Char('A'), Char('A')})
	testExpectRun(t, `return ("1.5" ::: decimal) == decimal("1.5")`, nil, True)

	// The cast IS the constructor call: `x ::: T` equals `T(x)` for every T.
	testExpectRun(t, `return (5 ::: str) == str(5) &&
		("5" ::: int) == int("5") &&
		(65 ::: char) == char(65) &&
		(5 ::: float) == float(5)`, nil, True)

	// Already the target type -> unchanged; `any` is identity.
	testExpectRun(t, `return [5 ::: int, "x" ::: str, 5 ::: any]`,
		nil, Array{Int(5), Str("x"), Int(5)})

	// An invalid conversion surfaces the constructor's own error.
	expectErrHas(t, `return "abc" ::: int`, nil, `int`)
}

// TestTransformCastFunction covers `expr ::: fn`: a transformer function is
// applied to the value — fn(expr) becomes the result. The transformer always
// receives the value as its single argument, so it is written `(v) => …`.
func TestTransformCastFunction(t *testing.T) {
	// An inline lambda transforms the value.
	testExpectRun(t, `return 5 ::: ((v) => v * 10)`, nil, Int(50))
	testExpectRun(t, `return "hi" ::: ((v) => v + "!")`, nil, Str("hi!"))

	// A lambda may ignore the value but must still declare the parameter.
	testExpectRun(t, `return (5 ::: ((v) => "is 5")) == "is 5"`, nil, True)

	// A named function works the same; casts chain left-to-right.
	testExpectRun(t, `
		double := (v) => v * 2
		inc := (v) => v + 1
		return 5 ::: double ::: inc`, nil, Int(11))

	// The transformer can change the type (int -> str -> its length).
	testExpectRun(t, `return 12345 ::: ((v) => str(v)) ::: ((v) => len(v))`, nil, Int(5))
}

// TestTransformCastAny covers `value ::: any`: the `any` type is identity, so the
// value is returned unchanged.
func TestTransformCastAny(t *testing.T) {
	testExpectRun(t, `return 5 ::: any`, nil, Int(5))
	testExpectRun(t, `return "x" ::: any`, nil, Str("x"))
	testExpectRun(t, `d := {a: 1}; return (d ::: any) == d`, nil, True)
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
