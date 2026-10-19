package parser_test

import (
	"testing"

	"github.com/gad-lang/gad/parser/test"
)

// A field of an interface may have a default: `name? T = expr`.
func TestInterfaceFieldDefault(t *testing.T) {
	for src, want := range map[string]string{
		`interface P { x int = 1 }`:                             `interface P {x int = 1; }`,
		`interface P { layout? str = "table"; n? int = 1 + 2 }`: `interface P {layout? str = "table"; n? int = (1 + 2); }`,
		"interface P {\n  a? int = 1\n  b str\n}":               `interface P {a? int = 1; b str; }`,
		`interface P { mode? enum { a; b } = #b }`:              `interface P {mode? enum {a, b} = #b; }`,
		`interface P { box: { w int } = {w: 1} }`:               `interface P {box: {w int; } = {w: 1}; }`,
		`interface P { f? <(x int) <r int>> = (x) => x }`:       `interface P {f? <(x int) <r int>> = (x) => x; }`,
		`interface P { c? int = (= 1) }`:                        `interface P {c? int = (= 1); }`,
	} {
		test.New(t, src).Code(want)
	}
}

func TestInterfaceFieldDefaultIndented(t *testing.T) {
	test.New(t, "interface P {\n  a? int = 1\n  b str\n  c? str = \"x\"\n}").
		IndentedCode("interface P {\n\ta? int = 1\n\tb str\n\tc? str = \"x\"\n}")
}
