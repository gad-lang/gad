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

// The comments of a member of a body — above it, at the end of its line —
// go with it wherever the formatter puts it: a class's fields sorted, kept
// in order by `[ordered]` (before `export` too), an interface's members, an
// enum's items. None is left behind the body.
func TestFormatMemberComments(t *testing.T) {
	src := `export class Form {
	name str           // required
	company? str       // optional
}

[ordered]
export class F2 {
	name str // required
	// the company
	company? str // optional
}

interface I {
	zeta str // last
	// alpha doc
	alpha int // first
}

enum E {
	a // the a
	b // the b
}

class C {
	// doc of z
	[label="Z"]
	z str // z trail
	/* a block */ a int
}
`
	want := `export class Form {
	company? str // optional
	name str // required
}

[ordered]
export class F2 {
	name str // required
	// the company
	company? str // optional
}

interface I {
	// alpha doc
	alpha int // first
	zeta str // last
}

enum E {
	a // the a
	b // the b
}

class C {
	/* a block */
	a int
	// doc of z
	[label="Z"]
	z str // z trail
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

// A comment after `nil` stays on its line: nil ended 9 characters after
// its start — past the comment, which went down a line. A unary or a
// ternary in parentheses is written in one pair of them, not two (each
// writes its own): formatting it again changed it.
func TestFormatNilCommentAndParens(t *testing.T) {
	src := "a := nil // nothing yet\nb := 1\nc := (-b) + 2\nd := (!a) && b\ne := (a ? 1 : 2) + 3\n"
	want := "a := nil // nothing yet\n\nvar (b = 1, c = ((-b) + 2), d = ((!a) && b), e = ((a ? 1 : 2) + 3))\n"
	o := &fmtOptions{codeFlags: fmtFormatFlag()}
	out, err := o.formatSource("x.gad", []byte(src), false)
	require.NoError(t, err)
	require.Equal(t, want, out)
	again, err := o.formatSource("x.gad", []byte(out), false)
	require.NoError(t, err)
	require.Equal(t, out, again, "formatting must be idempotent")
}

// An anonymous class — the type of a field, written in its place — keeps
// the order of its fields, as a group's body does: it has no `[ordered]` of
// its own. A declared class is sorted.
func TestFormatAnonymousClassKeepsOrder(t *testing.T) {
	src := "class F { x class { b int; a int }; c str; a str }\n"
	want := "class F {\n\ta str\n\tc str\n\tx class {\n\t\tb int\n\t\ta int\n\t}\n}\n"
	o := &fmtOptions{codeFlags: fmtFormatFlag()}
	out, err := o.formatSource("x.gad", []byte(src), false)
	require.NoError(t, err)
	require.Equal(t, want, out)
}

// A one-line doc block closed by a plain `*/` — `/** text */` — is a doc
// of one line: its `*/` was kept in it (`/// text */`).
func TestFormatDocClosedByStarSlash(t *testing.T) {
	o := &fmtOptions{codeFlags: fmtFormatFlag()}
	out, err := o.formatSource("x.gad", []byte("/** What a page tells. */\nclass P { a int }\n"), false)
	require.NoError(t, err)
	require.Equal(t, "/// What a page tells.\nclass P {\n\ta int\n}\n", out)
}
