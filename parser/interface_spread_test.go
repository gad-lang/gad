package parser_test

import (
	"testing"

	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
	"github.com/gad-lang/gad/parser/test"
)

func parseIface(t *testing.T, src string) *node.InterfaceExpr {
	t.Helper()
	f, err := parser.NewParserWithOptions(source.NewFileSet().AppendFileData("t.gad", []byte(src)), nil, nil).ParseFile()
	if err != nil {
		t.Fatal(err)
	}
	st, ok := f.Stmts[0].(*node.InterfaceStmt)
	if !ok {
		t.Fatalf("not an interface declaration: %T", f.Stmts[0])
	}
	return &st.InterfaceExpr
}

// In an interface body, `**<name>` is the rest capture, and `**Expr` — any
// expression, a bare name included — a spread of members given at run time.
func TestParseInterfaceSpreadAndRest(t *testing.T) {
	iface := parseIface(t, "interface I { a int; **extra; ** {fields: {b: int}}; **<rest> }")
	if iface.Rest == nil || iface.Rest.Name != "rest" {
		t.Errorf("rest: %#v", iface.Rest)
	}
	if len(iface.Spreads) != 2 {
		t.Fatalf("spreads: %d", len(iface.Spreads))
	}
	if id, ok := iface.Spreads[0].(*node.IdentExpr); !ok || id.Name != "extra" {
		t.Errorf("a bare name is a spread: %#v", iface.Spreads[0])
	}
	if _, ok := iface.Spreads[1].(*node.DictExpr); !ok {
		t.Errorf("a dict spread: %#v", iface.Spreads[1])
	}

	// `**(extra)` is a spread too
	iface = parseIface(t, "interface I { **(extra) }")
	if len(iface.Spreads) != 1 || iface.Rest != nil {
		t.Errorf("parenthesized: spreads %d, rest %v", len(iface.Spreads), iface.Rest)
	}

	// a rest capture not closed by `>` is an error
	if _, err := parser.NewParserWithOptions(source.NewFileSet().AppendFileData("t.gad", []byte("interface I { **<rest }")), nil, nil).ParseFile(); err == nil {
		t.Error("`**<rest` unclosed: want an error")
	}
}

// Written back: the spreads right after the fields, the rest capture as
// `**<name>`, after the methods.
func TestFormatInterfaceSpreadAndRest(t *testing.T) {
	test.New(t, "interface I { area() <float>; **<rest>; get label str; ** extra; name str }").
		Code("interface I {name str; **extra; get label str; area() <float>; **<rest>; }").
		FormattedCode("interface I {\n\tname str\n\t**extra\n\tget label str\n\tarea() <float>\n\t**<rest>\n}")
}
