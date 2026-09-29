package parser_test

import (
	"testing"

	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/parser/test"
)

// parseClass parses src, a class statement, and returns its literal.
func parseClass(t *testing.T, src string) *node.TypeLitExpr {
	t.Helper()
	f, err := parser.NewParserWithOptions(source.NewFileSet().AppendFileData("t.gad", []byte(src)), nil, nil).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	decl, ok := f.Stmts[0].(*node.TypeDeclStmt)
	if !ok {
		t.Fatalf("not a class declaration: %T", f.Stmts[0])
	}
	return &decl.TypeLitExpr
}

// `a class { … }` types a field by a class declared right there: the field's
// type is the class literal, anonymous, with its own fields — nested, with
// metadata, nullable.
func TestParseFieldClass(t *testing.T) {
	cls := parseClass(t, `class O {
    [label="A"] a? class {
        b class { [hint="C"] c int = 3 }
    }
    d str
}`)
	if len(cls.Fields) != 2 {
		t.Fatalf("fields: %d", len(cls.Fields))
	}
	a := cls.Fields[0]
	if a.Name.Ident.Name != "a" || !a.Name.Nullable || a.Meta == nil {
		t.Errorf("a: %+v", a.Name)
	}
	if len(a.Name.Type) != 1 {
		t.Fatalf("a has one type: %d", len(a.Name.Type))
	}
	inner, ok := a.Name.Type[0].Expr.(*node.TypeLitExpr)
	if !ok || inner.NameExpr != nil || inner.ImpliedName != "" {
		t.Fatalf("a's type is an anonymous class literal (named by the compiler): %#v", a.Name.Type[0].Expr)
	}
	b := inner.Fields[0]
	deeper, ok := b.Name.Type[0].Expr.(*node.TypeLitExpr)
	if !ok || len(deeper.Fields) != 1 || deeper.Fields[0].Meta == nil || deeper.Fields[0].Value == nil {
		t.Errorf("b's class, with c and its metadata and default: %#v", b.Name.Type[0].Expr)
	}

	// `class` without a body stays a name: a type called class
	cls = parseClass(t, "class O { a class }")
	if _, isLit := cls.Fields[0].Name.Type[0].Expr.(*node.TypeLitExpr); isLit {
		t.Error("`a class` with no body is a type by name")
	}
}

// `**Expr` in a class body is a spread item: any expression, several, beside
// the fields.
func TestParseClassSpread(t *testing.T) {
	cls := parseClass(t, "class O { a int; **extra; ** {fields: {b: 1}} + more\n c str }")
	if len(cls.Spreads) != 2 || len(cls.Fields) != 2 {
		t.Fatalf("spreads %d, fields %d", len(cls.Spreads), len(cls.Fields))
	}
	if id, ok := cls.Spreads[0].(*node.IdentExpr); !ok || id.Name != "extra" {
		t.Errorf("the first spread: %#v", cls.Spreads[0])
	}
	if _, ok := cls.Spreads[1].(*node.BinaryExpr); !ok {
		t.Errorf("the second spread is the whole expression: %#v", cls.Spreads[1])
	}

	// `*Parent` is still a parent
	cls = parseClass(t, "class O { *P; **x }")
	if len(cls.Parents) != 1 || len(cls.Spreads) != 1 {
		t.Errorf("parents %d, spreads %d", len(cls.Parents), len(cls.Spreads))
	}

	// `**` with nothing after it is an error
	if _, err := parser.NewParserWithOptions(source.NewFileSet().AppendFileData("t.gad", []byte("class O { ** }")), nil, nil).ParseFile(); err == nil {
		t.Error("`**` alone: want an error")
	}
}

// A spread is written back as it was, after the fields; the formatting is
// stable.
func TestFormatClassSpread(t *testing.T) {
	test.New(t, `class O { a str; **extra }`).
		Code(`class O {a str; **extra}`).
		FormattedCode("class O {\n\ta str\n\t**extra\n}")
	test.New(t, `class O { **{fields: {b: 1}} }`).
		Code(`class O {**{ fields: { b: 1 } }}`)
}
