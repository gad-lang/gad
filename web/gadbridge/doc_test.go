package gadbridge

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDocGad documents a .gad file: module heading, /*** root block, and the
// exported symbols with their doc comments.
func TestDocGad(t *testing.T) {
	src := "/*** greetings module. ***/\n\n" +
		"/** The greeting prefix. **/\nexport hello = \"hi\"\n\n" +
		"/** Adds two numbers. **/\nexport func add(a, b) { return a + b }\n"
	md, err := Doc(src, "gad")
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range []string{
		"greetings module.", "## Public API",
		"### hello", "= \"hi\"", "The greeting prefix.",
		"### add", "Adds two numbers.",
	} {
		if !strings.Contains(md, w) {
			t.Fatalf("gad doc missing %q:\n%s", w, md)
		}
	}
	// The rendered Markdown must be clean — no raw data-source-pos span or comment
	// markers.
	if strings.Contains(md, "data-source-pos") {
		t.Fatalf("gad doc leaked a raw data-source-pos span:\n%s", md)
	}
	if strings.Contains(md, "**/") || strings.Contains(md, "/**") {
		t.Fatalf("gad doc leaked comment markers:\n%s", md)
	}
}

// TestDocGadx documents a .gadx template: components with doc text.
func TestDocGadx(t *testing.T) {
	src := "/** Reusable widgets. **/\n@comp greeting(name)\n    p hi\n"
	md, err := Doc(src, "gadx")
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range []string{
		"## Components", "### greeting",
		"Reusable widgets.",
	} {
		if !strings.Contains(md, w) {
			t.Fatalf("gadx doc missing %q:\n%s", w, md)
		}
	}
}

// TestExtractDocGadtModuleProse verifies that a `.gadt` module doc placed inside
// the leading code island — a `{%-- … --%}` block wrapping a `/*** … ***/` root
// comment, a `/** … **/` block or a normal `/* … */` comment — is captured as
// prose and never leaks into the rendered template output.
func TestExtractDocGadtModuleProse(t *testing.T) {
	cases := []struct{ name, src, wantProse string }{
		{"root", "{%--\n/***\nRoot doc.\n***/\n--%}\n<h1>hi</h1>\n", "Root doc."},
		{"block", "{%--\n/**\nBlock doc.\n**/\n--%}\n<h1>hi</h1>\n", "Block doc."},
		{"normal", "{%-- /* Normal doc. */ --%}\n<h1>hi</h1>\n", "Normal doc."},
		{"shebang+root", "#!/usr/bin/env gad\n{%--\n/*** After shebang. ***/\n--%}\n<h1>hi</h1>\n", "After shebang."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d, err := ExtractDoc(c.src, "gadTemplate")
			if err != nil {
				t.Fatal(err)
			}
			if d.Prose != c.wantProse {
				t.Fatalf("prose = %q, want %q", d.Prose, c.wantProse)
			}
			// The doc comment must not be emitted as template text.
			if r := RunSource(c.src, "gadTemplate"); r.Stdout != "<h1>hi</h1>\n" {
				t.Fatalf("run leaked the doc: %q", r.Stdout)
			}
		})
	}

	// A plain .gad file's module prose is a DETACHED leading block (blank line
	// after); a block glued to a statement documents the statement instead.
	if d, _ := ExtractDoc("/** Plain gad. **/\n\nexport a = 1\n", "gad"); d.Prose != "Plain gad." {
		t.Fatalf("gad prose regressed: %q", d.Prose)
	}
}

// TestExtractDocModuleProseBlank verifies the module prose is a leading `/** … **/`
// block DETACHED from the code — followed by a blank line or at end of file — and
// that a block glued to a statement documents the statement, not the module.
func TestExtractDocModuleProseBlank(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"detached", "/**\n# Mod\nDoc.\n**/\n\nconst x = 1\n", "# Mod\nDoc."},
		{"eof", "/**\n# Mod\n**/\n", "# Mod"},
		{"three-star still works", "/***\n# Mod\n***/\n\nconst x = 1\n", "# Mod"},
		{"attached is not module", "/**\nDoc of x.\n**/\nconst x = 1\n", ""},
		{"none", "const x = 1\n", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d, err := ExtractDoc(c.src, "gad")
			if err != nil {
				t.Fatal(err)
			}
			if d.Prose != c.want {
				t.Fatalf("prose = %q, want %q", d.Prose, c.want)
			}
		})
	}
}

// TestDocMetadataAndMembers verifies exported declarations carry their `[k=v, …]`
// metadata tag and their members (each with its own tag and doc) — a typed array
// type, an array interface, a class and an enum — and that the Markdown renders
// them.
func TestDocMetadataAndMembers(t *testing.T) {
	src := `
/// Numbers.
[unit="m"]
export type numerics []<int|float> {
	/// a label
	[db=(;col="lbl")]
	label = "none"
	methods {
		/// sums
		[route="/sum"]
		sum() => 0
	}
}

/// Points.
[k=1]
export interface points [] {
	/// x coord
	[pk]
	x int
}

/// Colors.
[c=1]
export enum Color {
	/// red
	[hex="f00"]
	Red
}
`
	d, err := ExtractDoc(src, "gad")
	require.NoError(t, err)
	require.Len(t, d.Sections, 1)
	syms := d.Sections[0].Symbols
	require.Len(t, syms, 3)

	require.Equal(t, `[unit="m"]`, syms[0].Meta)
	require.Equal(t, " []<int | float>", syms[0].Signature)
	require.Equal(t, []DocMember{
		{Group: "Fields", Signature: `label = "none"`, Meta: `[db=(;col="lbl")]`, Doc: "a label"},
		{Group: "Methods", Signature: "sum()", Meta: `[route="/sum"]`, Doc: "sums"},
	}, syms[0].Members)

	require.Equal(t, `[k=1]`, syms[1].Meta)
	require.Equal(t, " interface []{…}", syms[1].Signature)
	require.Equal(t, []DocMember{{Group: "Fields", Signature: "x int", Meta: "[pk]", Doc: "x coord"}}, syms[1].Members)

	require.Equal(t, `[c=1]`, syms[2].Meta)
	require.Equal(t, []DocMember{{Group: "Variants", Signature: "Red", Meta: `[hex="f00"]`, Doc: "red"}}, syms[2].Members)

	md := RenderMarkdown(d)
	for _, want := range []string{"[unit=\"m\"]", "#### Fields", "[db=(;col=\"lbl\")]\nlabel = \"none\"", "[route=\"/sum\"]\nsum()", "[hex=\"f00\"]\nRed"} {
		require.Contains(t, md, want)
	}
}

// A class's parents, mixins and spreads, a mixin's `this` and an interface's
// parents, properties, methods and rest are documented, each in its group
// (#28); a parent or a mixin names what it is (Ref).
func TestDocClassStructure(t *testing.T) {
	src := `
/** named **/
export interface Named { name str }
/** with an id **/
export interface Ident {
	*Named
	id int
	get label str
	label() <str>
	**<rest>
}
/** moves **/
export mixin Mover {
	this { x int }
	methods { move(dx) { this.x += dx } }
}
/** a base **/
export class Base { name str }
/** a point **/
export class Point {
	*Base
	use Mover
	y int
	**{fields: {z: int}}
	methods { describe() => "point" }
}
`
	d, err := ExtractDoc(src, "gad")
	require.NoError(t, err)
	members := map[string][]string{}
	for _, sec := range d.Sections {
		for _, s := range sec.Symbols {
			for _, m := range s.Members {
				members[s.Name] = append(members[s.Name], m.Group+": "+m.Signature+" "+m.Ref)
			}
		}
	}
	require.Equal(t, []string{"Parents: *Named Named", "Fields: id int ", "Properties: get label str ",
		"Methods: label() <str> ", "Rest: **<rest> "}, members["Ident"])
	require.Equal(t, []string{"This: x int ", "Methods: move(dx) "}, members["Mover"])
	require.Equal(t, []string{"Parents: *Base Base", "Mixins: use Mover Mover", "Fields: y int ",
		"Spreads: **{fields: {z: int}} ", "Methods: describe() "}, members["Point"])
}

// Each documented declaration says what it is (#29: the structure view).
func TestDocSymbolKind(t *testing.T) {
	src := `
export interface I { x int }
export mixin M { x int }
export class C { x int }
export enum E { a, b }
export type T []int
export func f(a int) <int> => a
export func g { (a int) => a; (a str) => a }
export v = 1
`
	d, err := ExtractDoc(src, "gad")
	require.NoError(t, err)
	kinds := map[string]string{}
	for _, sec := range d.Sections {
		for _, s := range sec.Symbols {
			kinds[s.Name] = SymbolKind(s)
		}
	}
	require.Equal(t, map[string]string{"I": "interface", "M": "mixin", "C": "class", "E": "enum",
		"T": "type", "f": "func", "g": "func", "v": "value"}, kinds)
}
