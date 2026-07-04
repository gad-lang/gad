package parser_test

import (
	"testing"

	. "github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/test"
)

func TestParseTestStmt(t *testing.T) {
	// identifier name
	test.ExpectParseString(t, `test x { t.equal(1, 1) }`, `test x {t.equal(1, 1)}`)
	// string name (kept quoted)
	test.ExpectParseString(t, `test "x is one" { t.true(x == 1) }`, `test "x is one" {t.true((x == 1))}`)
	// bench form
	test.ExpectParseString(t, `bench loop { for i := 0; i < t.n; i++ {} }`,
		`bench loop {for i := 0; (i < t.n); i++{}}`)
	// an identifier name that is not a bare ident is round-tripped quoted anyway
	test.ExpectParseString(t, `bench "the loop" {}`, `bench "the loop" {}`)

	// `test`/`bench` remain ordinary identifiers when not followed by NAME `{`.
	test.ExpectParseString(t, `test := import("test")`, `test := import("test")`)
	test.ExpectParseString(t, `bench.run()`, `bench.run()`)
	test.ExpectParseString(t, `test(x)`, `test(x)`)
}

func TestParseTestStmtNode(t *testing.T) {
	stmts := test.New(t, `test hello { a := 1 }`).File().File().Stmts
	if len(stmts) != 1 {
		t.Fatalf("expected 1 stmt, got %d", len(stmts))
	}
	ts, ok := stmts[0].(*TestStmt)
	if !ok {
		t.Fatalf("expected *TestStmt, got %T", stmts[0])
	}
	if ts.Kind != TestKindTest {
		t.Fatalf("expected TestKindTest, got %v", ts.Kind)
	}
	if ts.Name != "hello" || ts.Quoted {
		t.Fatalf("name=%q quoted=%v", ts.Name, ts.Quoted)
	}

	ts = test.New(t, `bench "b one" {}`).File().File().Stmts[0].(*TestStmt)
	if ts.Kind != TestKindBench || ts.Name != "b one" || !ts.Quoted {
		t.Fatalf("bench: kind=%v name=%q quoted=%v", ts.Kind, ts.Name, ts.Quoted)
	}
}
