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
// its start — past the comment, which went down a line. The parentheses
// are the source's, once each: none added around a binary, a unary or a
// conditional (`if x > 0`, not `if (x > 0)`), none dropped that groups.
func TestFormatNilCommentAndParens(t *testing.T) {
	src := "a := nil // nothing yet\nb := 1\nc := (-b) + 2\nd := (!a) && b\ne := (a ? 1 : 2) + 3\n"
	want := "a := nil // nothing yet\n\nvar (b = 1, c = (-b) + 2, d = (!a) && b, e = (a ? 1 : 2) + 3)\n"
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

// Formatted, a source keeps its parentheses and gets no others: what groups
// is as written, and reads back the same (the safety net checks the meaning
// too). A `for` writes a space before its body.
func TestFormatMinimalParens(t *testing.T) {
	src := "for i := 0; i < len(v); i++ {\n\tif c >= '0' && c <= '9' {\n\t\td = append(d, int(c) - int('0'))\n\t}\n}\n" +
		"x := (a + b) * c\ny := a - (b - c)\nz := m ? m[0] : \"\"\nw := !(a && b)\nu := a == nil || b\nfor q {\n\tbreak\n}\n"
	want := "for i := 0; i < len(v); i++ {\n\tif c >= '0' && c <= '9' {\n\t\td = append(d, int(c) - int('0'))\n\t}\n}\n\n" +
		"var (\n\tu = a == nil || b\n\tw = !(a && b)\n\tx = (a + b) * c\n\ty = a - (b - c)\n\tz = m ? m[0] : \"\"\n)\n\nfor q {\n\tbreak\n}\n"
	o := &fmtOptions{codeFlags: fmtFormatFlag()}
	out, err := o.formatSource("x.gad", []byte(src), false)
	require.NoError(t, err)
	require.Equal(t, want, out)
}

// A union of types as a type argument — `Range[int|float]` — is written
// without parentheses: in them it is an operation on two values
// (`Range[(int | float)]`), which the program fails at.
func TestFormatTypeArgUnion(t *testing.T) {
	src := "class F {\n\tb? Range[int|float]\n\tc? int|float\n}\n"
	want := "class F {\n\tb? Range[int | float]\n\tc? int | float\n}\n"
	o := &fmtOptions{codeFlags: fmtFormatFlag()}
	out, err := o.formatSource("x.gad", []byte(src), false)
	require.NoError(t, err)
	require.Equal(t, want, out)
}

// An `@import` is written back as the directive it was — not the
// `name := import("m")` it lowers to —, a run of them one block, none merged
// into a `var (…)`.
func TestFormatImportDirective(t *testing.T) {
	src := "@import \"time\" as tm\n@import \"sys\"\n@import { a, b: c } from \"m\"\nx := tm.now()\n"
	want := "@import \"time\" as tm\n@import \"sys\"\n@import { a, b:c } from \"m\"\n\nx := tm.now()\n"
	o := &fmtOptions{codeFlags: fmtFormatFlag()}
	out, err := o.formatSource("x.gad", []byte(src), false)
	require.NoError(t, err)
	require.Equal(t, want, out)
}

// The comments of a spec of a `const (…)`/`var (…)` — above it, at the end
// of its line — go with it wherever the group's order puts it, and the
// group is then one spec a line: they were left after the group.
func TestFormatDeclGroupComments(t *testing.T) {
	src := "const (\n\tsys = import(\"sys\")\n\t// the options\n\tlc = import(\"lc\")\n\tzz = 1 // trailing\n)\n"
	want := "const (\n\tzz = 1 // trailing\n\t// the options\n\tlc = import(\"lc\")\n\tsys = import(\"sys\")\n)\n"
	o := &fmtOptions{codeFlags: fmtFormatFlag()}
	out, err := o.formatSource("x.gad", []byte(src), false)
	require.NoError(t, err)
	require.Equal(t, want, out)
	again, err := o.formatSource("x.gad", []byte(out), false)
	require.NoError(t, err)
	require.Equal(t, out, again, "formatting must be idempotent")
}
