package langsym_test

import (
	"testing"

	"github.com/gad-lang/gad/langsym"
	"github.com/stretchr/testify/require"
)

// flatten renders the outline tree as "kind name" lines, indented by depth, for
// compact assertions.
func flatten(syms []langsym.OutlineSym, depth int, out *[]string) {
	for _, s := range syms {
		line := ""
		for i := 0; i < depth; i++ {
			line += "  "
		}
		*out = append(*out, line+s.Kind+" "+s.Name)
		flatten(s.Children, depth+1, out)
	}
}

func TestOutline(t *testing.T) {
	f, sf := parse(t, `
const Pi = 3.14
var count
func greet(name) => "hi"
class Animal { name = "?"; props { kind => "a" }; new { (n) => this(; name=n) }; methods { speak() => 1 } }
mixin Counter { count = 0; methods { inc() => 1 } }
type Marker { tag = 1; call(n) => n; methods { describe() => 1 } }
interface Greeter { name str; greet() <str> }
enum Color { Red; Green }
met Animal.run(this) => 1
const Shape = class { sides = 3 }
`)
	var lines []string
	flatten(langsym.Outline(f, sf), 0, &lines)

	want := []string{
		"const Pi",
		"var count",
		"func greet",
		"class Animal",
		"  field name",
		"  property kind",
		"  new new",
		"  method speak",
		"mixin Counter",
		"  field count",
		"  method inc",
		"type Marker", // a marker `type Name { … }`
		"  field tag",
		"  new call", // the call(…) factory
		"  method describe",
		"interface Greeter",
		"  field name",
		"  method greet",
		"enum Color",
		"  value Red",
		"  value Green",
		"met Animal.run",
		"class Shape", // `const Shape = class { … }` surfaces as the class
		"  field sides",
	}
	require.Equal(t, want, lines)
}

// TestOutlinePositions checks that a symbol carries the 0-based offset and 1-based
// line/column of its declaration, for editor navigation.
func TestOutlinePositions(t *testing.T) {
	src := "const Pi = 3.14\nfunc f() => 1\n"
	f, sf := parse(t, src)
	syms := langsym.Outline(f, sf)
	require.Len(t, syms, 2)
	require.Equal(t, "Pi", syms[0].Name)
	require.Equal(t, 1, syms[0].Line)
	require.Equal(t, 7, syms[0].Column) // 1-based column of `Pi`
	require.Equal(t, 6, syms[0].Offset) // 0-based byte offset of `Pi`
	require.Equal(t, "f", syms[1].Name)
	require.Equal(t, 2, syms[1].Line)
}

// What a module exports is outlined as what it is: an exported class, type,
// enum or function as the declaration; an exported value by its name.
func TestOutlineExports(t *testing.T) {
	f, sf := parse(t, `
class Link { href str }
export class default { links []Link; title str }
export type Links []Link
export enum Kind { A; B }
export func greet(name) => "hi"
export const Max = 10
export x = 1
`)
	var lines []string
	flatten(langsym.Outline(f, sf), 0, &lines)
	require.Equal(t, []string{
		"class Link",
		"  field href",
		"class default",
		"  field links",
		"  field title",
		"type Links",
		"enum Kind",
		"  value A",
		"  value B",
		"func greet",
		"const Max",
		// (the parser declares an exported value a const in its prelude)
		"const x",
	}, lines)
}

// flattenDetail is flatten with each symbol's detail.
func flattenDetail(syms []langsym.OutlineSym, depth int, out *[]string) {
	for _, s := range syms {
		line := ""
		for i := 0; i < depth; i++ {
			line += "  "
		}
		line += s.Kind + " " + s.Name
		if s.Detail != "" {
			line += " : " + s.Detail
		}
		*out = append(*out, line)
		flattenDetail(s.Children, depth+1, out)
	}
}

// A field's type is its detail; a field typed by a class or an interface
// written in place is `class`/`interface`, its members under it; a function
// of one signature has it as its detail, of several each as a child.
func TestOutlineTypesAndOverloads(t *testing.T) {
	f, sf := parse(t, `
class Link { href str }
export class default {
	title str
	link? Link
	kind int|str
	lead class {
		name str
		phone? str
	}
	contact interface { email str }
	props { size => 1 }
	methods {
		m(a int) <str> => ""
		n { (a) => 1; (a, b) => 2 }
	}
}
func f(a, b) => 1
func g { (a) => 1; (a, b) => 2 }
`)
	var lines []string
	flattenDetail(langsym.Outline(f, sf), 0, &lines)
	require.Equal(t, []string{
		"class Link",
		"  field href : str",
		"class default",
		"  field title : str",
		"  field link : ?Link",
		"  field kind : int|str",
		"  field lead : class",
		"    field name : str",
		"    field phone : ?str",
		"  field contact : interface",
		"    field email : str",
		"  property size",
		"  method m : (a int) <str>",
		"  method n",
		"    overload (a)",
		"    overload (a, b)",
		"func f : (a, b)",
		"func g",
		"  overload (a)",
		"  overload (a, b)",
	}, lines)
}
