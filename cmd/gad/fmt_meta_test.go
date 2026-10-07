package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// The metadata of a declaration is formatted on its own line, before it —
// a field, a class, an interface and its members, an enum and its items, a
// func, a func of several methods, a declared union, an export of each —,
// kept whole: a function as a value (`validation=func(v) => …`), not the
// key-value array's method shorthand, which metadata does not read.
// Formatting twice is formatting once.
func TestFormatMetadata(t *testing.T) {
	src := `[label="Contact"] class ContactForm { [label="Full Name"] name str; [validation=func(v) => v ? nil : "x", w=[f, func(r) { return nil }]] zip str }
[doc="i"] interface I { [label="x"] x str }
enum Size { [label="Small"] S, [label="Large"] L }
[deprecated] func f(x) { return x }
[m=1] func g { (x) => x; (x, y) => y }
[x] type T <str|int>
[m=1] export class D { a str }
[n=2] export type U <int|float>
[ordered] class F { z str; a int }
`
	want := `[label="Contact"]
class ContactForm {
	[label="Full Name"]
	name str
	[validation=func(v) => (v ? nil : "x"), w=[f, func(r) { return nil }]]
	zip str
}

[doc="i"]
interface I {
	[label="x"]
	x str
}

enum Size {
	[label="Small"]
	S
	[label="Large"]
	L
}

[deprecated]
func f(x) {
	return x
}

[m=1]
func g {
	(x) => x

	(x, y) => y
}

[x]
type T <str|int>

[m=1]
export class D {
	a str
}

[n=2]
export type U <int|float>

[ordered]
class F {
	z str
	a int
}
`
	o := &fmtOptions{codeFlags: fmtFormatFlag()}
	out, err := o.formatSource("x.gad", []byte(src), false)
	require.NoError(t, err)
	require.Equal(t, want, out)
	again, err := o.formatSource("x.gad", []byte(out), false)
	require.NoError(t, err)
	require.Equal(t, out, again, "formatting must be idempotent")
}
