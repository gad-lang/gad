package parser_test

import (
	"testing"

	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/parser/test"
)

// parseFile parses src.
func parseFile(t *testing.T, src string) *parser.File {
	t.Helper()
	f, err := parser.NewParserWithOptions(source.NewFileSet().AppendFileData("t.gad", []byte(src)), nil, nil).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	return f
}

// A group — `{ … }` in a class body — is the field `$N`, N its count in the
// body, at the place it is written, typed by the anonymous class of the block;
// `? { … }` makes it nullable; the metadata before it is its own; groups nest
// without limit, each body counting its own.
func TestParseClassGroups(t *testing.T) {
	cls := parseClass(t, `class O {
    a str
    [width=2] { b str; { c int; { d bool } } }
    ? { e str }
}`)
	if len(cls.Fields) != 3 {
		t.Fatalf("fields: %d", len(cls.Fields))
	}
	g1, g2 := cls.Fields[1], cls.Fields[2]
	if !g1.Group || g1.Name.Ident.Name != "$1" || g1.Name.Nullable || g1.Meta == nil {
		t.Errorf("$1: %+v %+v", g1, g1.Name)
	}
	if !g2.Group || g2.Name.Ident.Name != "$2" || !g2.Name.Nullable || g2.Meta != nil {
		t.Errorf("$2: %+v %+v", g2, g2.Name)
	}
	body, ok := g1.Name.Type[0].Expr.(*node.TypeLitExpr)
	if !ok || body.Mixin || !body.Group || len(body.Fields) != 2 || body.Fields[0].Name.Ident.Name != "b" {
		t.Fatalf("$1's class: %#v", g1.Name.Type[0].Expr)
	}
	// nested: the group's body counts its own groups
	inner := body.Fields[1]
	if !inner.Group || inner.Name.Ident.Name != "$1" {
		t.Errorf("the group in $1: %+v", inner.Name)
	}
	deepest := inner.Name.Type[0].Expr.(*node.TypeLitExpr).Fields[1]
	if !deepest.Group || deepest.Name.Ident.Name != "$1" {
		t.Errorf("the group in the group: %+v", deepest.Name)
	}
}

// In a mixin a group is a class too, never a mixin.
func TestParseMixinGroups(t *testing.T) {
	f := parseFile(t, "mixin M { { x int }; y str }")
	var cls *node.TypeLitExpr
	switch s := f.Stmts[0].(type) {
	case *node.TypeDeclStmt:
		cls = &s.TypeLitExpr
	default:
		t.Fatalf("not a declaration: %T", f.Stmts[0])
	}
	if !cls.Mixin || !cls.Fields[0].Group || cls.Fields[0].Name.Ident.Name != "$1" {
		t.Fatalf("the mixin's group: %+v", cls.Fields[0])
	}
	if g := cls.Fields[0].Name.Type[0].Expr.(*node.TypeLitExpr); g.Mixin {
		t.Error("a mixin's group is a mixin: want a class")
	}
}

// In an interface a group is an interface: `$N interface { … }`.
func TestParseInterfaceGroups(t *testing.T) {
	f := parseFile(t, "interface I { a str; [width=1] { b int; { c str } }; ? { d str } }")
	stmt, ok := f.Stmts[0].(*node.InterfaceStmt)
	if !ok {
		t.Fatalf("not an interface: %T", f.Stmts[0])
	}
	var groups []*node.InterfaceMemberExpr
	for _, m := range stmt.Members {
		if m.Group {
			groups = append(groups, m)
		}
	}
	if len(groups) != 2 || groups[0].Name.Ident.Name != "$1" || groups[0].Meta == nil ||
		groups[1].Name.Ident.Name != "$2" || !groups[1].Name.Nullable {
		t.Fatalf("groups: %+v", groups)
	}
	inner, ok := groups[0].Name.Type[0].Expr.(*node.InterfaceExpr)
	if !ok {
		t.Fatalf("$1's type: %#v", groups[0].Name.Type[0].Expr)
	}
	var nested bool
	for _, m := range inner.Members {
		nested = nested || (m.Group && m.Name.Ident.Name == "$1")
	}
	if !nested {
		t.Error("the group in $1 is not its $1")
	}
}

// A group is written back as the block it was — never as `$N class { … }`,
// which does not parse —, its fields in their order (its columns), and the
// fields of a class with groups too: a group's name is its place.
func TestFormatGroups(t *testing.T) {
	test.New(t, `class O { z str; [width=2] { y int; x int }; ? { w str }; a str }`).
		Code(`class O {z str; [width=2] {y int; x int}; ? {w str}; a str}`).
		FormattedCode("class O {\n\tz str\n\t[width=2]\n\t{\n\t\ty int\n\t\tx int\n\t}\n\t? {\n\t\tw str\n\t}\n\ta str\n}")
	test.New(t, `interface I { a str; { b int }; ? { c str } }`).
		Code(`interface I {a str; {b int; }; ? {c str; }; }`)
	test.New(t, `mixin M { { x int } }`).
		Code(`mixin M {{x int}}`)
	// `[ordered]`: the fields as declared
	test.New(t, "[ordered] class O { b str; a str }").
		FormattedCode("[ordered]\nclass O {\n\tb str\n\ta str\n}")
	// otherwise the canonical order, as always
	test.New(t, "class O { b str; a str }").
		FormattedCode("class O {\n\ta str\n\tb str\n}")
}
