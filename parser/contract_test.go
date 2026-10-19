package parser_test

import (
	"testing"

	"github.com/gad-lang/gad/parser/test"
)

func TestContractStmt(t *testing.T) {
	for src, want := range map[string]string{
		`contract TField { target any; meta { label? str } }`:                                                `contract TField {target any; meta {label? str; }}`,
		`contract TSelect { *TField; target type<str|enum>; meta { options []str; layout? str = "table" } }`: `contract TSelect {*TField; target type<str|enum>; meta {layout? str = "table"; options []str; }}`,
		"contract T {\n  *A\n  *B\n  target int\n}":                                                          `contract T {*A; *B; target int}`,
		`contract E { meta { options []target = target.@members } }`:                                         `contract E {meta {options []target = target.@members; }}`,
		`[since=2] contract T { target any }`:                                                                `[since=2] contract T {target any}`,
		`export contract T { target any }`:                                                                   `export contract T {target any}`,
	} {
		test.New(t, src).Code(want)
	}
	// `contract` is a name elsewhere
	test.New(t, `contract := 1`).Code(`contract := 1`)
	test.New(t, `x := contract.a`).Code(`x := contract.a`)
}
