package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// A type written as a selector or an index — `time.CalendarDate`,
// `Range[int]` — is the type it names where a type goes: a parameter's, an
// interface's field's, a class's field's; not the module it starts with.
func TestComputedTypes(t *testing.T) {
	testExpectRun(t, okFn+`
		f := func(x time.CalendarDate) { return x }
		interface I { d time.CalendarDate }
		return [
			str(f(2026-01-30D)),
			ok({d: 2026-01-30D}, I),
			ok({d: "2026-01-30"}, I),
			I.fields[0].types[0] == time.CalendarDate,
		]`,
		nil, Array{Str("2026-01-30"), True, False, True})

	// the message names the type as written
	expectErrHas(t, `f := func(x time.CalendarDate) { return x }; f("a")`,
		newOpts(), "expected time.CalendarDate")
}

// Range[T] is the type of a range whose bounds are of T: a class's field, an
// interface's — kept once the function writing it returned —, reflected
// (@elem), a union of types (type <…>) too.
func TestTypedRange(t *testing.T) {
	testExpectRun(t, okFn+`
		mk := func() {
			class Form { period Range[time.CalendarDate]; rooms? Range[int] }
			interface I { p Range[time.CalendarDate]; n? Range[type <int|float>] }
			return [Form, I]
		}
		[Form, I] := mk()
		z := 1
		T := Form.@fields["period"].@types[0]
		return [
			str(T),
			T.@elem == time.CalendarDate,
			ok(2026-01-30D .. 2026-02-02D, T),
			ok(1 .. 3, T),
			ok({p: 2026-01-30D .. 2026-02-02D}, I),
			ok({p: 1 .. 3}, I),
			ok({p: 2026-01-30D .. 2026-02-02D, n: 1.5 .. 3.0}, I),
			ok({p: 2026-01-30D .. 2026-02-02D, n: 1 .. 2}, I),
		]`,
		nil, Array{Str("Range[calendarDate]"), True, True, False, True, False, True, True})
}

// In a type, `Range[int|float]` is `Range[type <int|float>]`: the index's
// `|` joins types; out of a type, it is the bitwise or it is.
func TestTypedRangeUnion(t *testing.T) {
	testExpectRun(t, okFn+`
		class C { n? Range[int|float] }
		interface I { n Range[int|float] }
		f := func(x Range[int|float]) { return str(x) }
		return [str(C.@fields["n"].@types[0]), ok({n: 1.5 .. 3.0}, I), ok({n: 1 .. 2}, I), ok({n: "a" .. "b"}, I), f(1 .. 3), 5|3]`,
		nil, Array{Str("Range[int|float]"), True, True, False, Str("1 .. 3"), Int(7)})
}

