package gad_test

import (
	"strings"
	"testing"

	. "github.com/gad-lang/gad"
	"github.com/gad-lang/gad/parser"
)

// TestPropGetterForms covers the read-only prop getter forms: the parenless
// getter (`prop pi => e`) and its typed variant (`prop pi <float> => e`) are
// valid and callable, the empty-parens getter (`prop pi() => e`) is rejected, and
// each form round-trips through the formatter.
func TestPropGetterForms(t *testing.T) {
	testExpectRun(t, `prop pi => 3.14; return pi()`, nil, Float(3.14))
	testExpectRun(t, `prop pi <float> => 3.14; return pi()`, nil, Float(3.14))

	mustErr := func(src, want string) {
		_, err := Compile(NewSymbolTable(NewBuiltins().NameSet), []byte(src), CompileOptions{})
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("Compile(%q): err=%v, want containing %q", src, err, want)
		}
	}
	mustErr(`return prop pi() => 3.14`, "empty accessor parens")
	mustErr(`return prop pi() <float> => 3.14`, "empty accessor parens")

	fmtOf := func(src string) string {
		f, err := parser.NewSingleParser(src, "", nil, nil).ParseFile()
		if err != nil {
			t.Fatalf("parse %q: %v", src, err)
		}
		return f.String()
	}
	for src, want := range map[string]string{
		"return prop pi => 3.14":         "return prop pi => 3.14",
		"return prop pi <float> => 3.14": "return prop pi <float> => 3.14",
		"return prop => x":               "return prop => x",
		"return prop x() { return v }":   "return prop x() { return v }",
	} {
		if got := fmtOf(src); got != want {
			t.Errorf("format %q = %q, want %q", src, got, want)
		}
	}
}
