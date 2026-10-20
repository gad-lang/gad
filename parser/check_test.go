package parser_test

import (
	"testing"

	"github.com/gad-lang/gad/parser/test"
)

// The inline checks (issue #22): `= EXPR` and `<<<` where a statement starts;
// the operators elsewhere unchanged.
func TestInlineChecks(t *testing.T) {
	for src, want := range map[string]string{
		"x := 10\n= 10":               "x := 10; = 10",
		"x := 10; = 10":               "x := 10; = 10",
		"f()\n<<< a\\nb":              "f(); <<< a\\nb\n",
		"f();<<< a\\tb\\\\c":          "f(); <<< a\\tb\\\\c\n",
		"f()\n<<<\na\nb\n>>>":         "f(); <<< a\\nb\n",
		"func g() {\n  f()\n  = 1\n}": "func g() {f(); = 1}",
		"x := 1 <<< 2":                "x := (1 <<< 2)",
		"x := 1 >>> 2":                "x := (1 >>> 2)",
		"v := (= 1 + 2)":              "v := (= 1 + 2)",
		"a = 1":                       "a = 1",
	} {
		test.New(t, src).Code(want)
	}
}

func TestInlineChecksIndented(t *testing.T) {
	test.New(t, "func g() {\n  f()\n  <<<\n  a\n    b\n  >>>\n  = 1\n}").
		IndentedCode("func g() {\n\tf()\n\t<<<\n\ta\n\t  b\n\t>>>\n\t= 1\n}")
}

// `example [NAME] { … }` (issue #22): a contextual keyword, before `{` or
// `NAME {`; anywhere else a name.
func TestExampleStmt(t *testing.T) {
	for src, want := range map[string]string{
		"example greet {\n  f()\n  = 1\n}\n<<< x": "example greet {f(); = 1}; <<< x\n",
		"example {\n  1\n}":                       "example {1}",
		"example in {\n  1\n}":                    "example in {1}",
		"example := 1":                            "example := 1",
		"x := {example: 1}":                       "x := { example: 1 }",
		"y := x.example":                          "y := x.example",
	} {
		test.New(t, src).Code(want)
	}
}
