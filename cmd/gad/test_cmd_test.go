package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	cc "github.com/moisespsena-go/command-context"
	"github.com/stretchr/testify/require"
)

func TestIsPrefixFunc(t *testing.T) {
	require.True(t, isPrefixFunc("testFoo", "test"))
	require.True(t, isPrefixFunc("TestFoo", "test")) // case-insensitive
	require.True(t, isPrefixFunc("benchBar", "bench"))
	require.False(t, isPrefixFunc("test", "test"))   // bare prefix rejected
	require.False(t, isPrefixFunc("helper", "test")) // no prefix
	require.False(t, isPrefixFunc("tes", "test"))    // shorter than prefix
	require.False(t, isPrefixFunc("attest", "test")) // prefix must be at start
}

func TestTestFilesDiscovery(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(p, []byte(body), 0o644))
		return p
	}
	write("a_test.gad", "func testA(t) {}")
	write("b.gad", "func notATest() {}")
	write("c_test.gad", "func testC(t) {}")

	files, err := testFiles(dir)
	require.NoError(t, err)
	require.Len(t, files, 2)
	for _, f := range files {
		require.True(t, filepath.Base(f) == "a_test.gad" || filepath.Base(f) == "c_test.gad", f)
	}
}

// runFileOn writes src to a temp *_test.gad file and runs it, returning the
// pass/fail/skip counts and the combined output.
func runFileOn(t *testing.T, o *testOptions, src string) (pass, fail, skip int, out string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "x_test.gad")
	require.NoError(t, os.WriteFile(path, []byte(src), 0o644))
	require.NoError(t, o.compile())
	var buf bytes.Buffer
	ctx := &cc.CommandContext{Out: &buf, Err: &buf}
	pass, fail, skip = o.runFile(ctx, path)
	return pass, fail, skip, buf.String()
}

func TestRunFileReports(t *testing.T) {
	src := `test := import("test")
func testPass(t) { t.equal(1, 1); t.true(true) }
func testFailAssert(t) { t.equal(1, 2) }
func testSkip(t) { t.skip("nope") }
func testHelper(t) { test.contains(t, "abc", "b") }
func benchIgnored(t) { for i := 0; i < t.n; i++ {} }
`
	o := &testOptions{verbose: true}
	pass, fail, skip, out := runFileOn(t, o, src)
	require.Equal(t, 2, pass, out) // testPass, testHelper
	require.Equal(t, 1, fail, out) // testFailAssert
	require.Equal(t, 1, skip, out) // testSkip
	require.Contains(t, out, "FAIL")
	require.Contains(t, out, "not equal")
}

func TestRunFileRunFilter(t *testing.T) {
	src := `func testOne(t) { t.true(true) }
func testTwo(t) { t.equal(1, 2) }
`
	o := &testOptions{runPat: "One"} // only testOne
	pass, fail, _, out := runFileOn(t, o, src)
	require.Equal(t, 1, pass, out)
	require.Equal(t, 0, fail, out)
}

func TestRunFileBench(t *testing.T) {
	src := `func benchNoop(t) { for i := 0; i < t.n; i++ {} }`
	o := &testOptions{benchPat: ".", benchtime: time.Millisecond}
	pass, fail, skip, out := runFileOn(t, o, src)
	require.Equal(t, 0, pass)
	require.Equal(t, 0, fail)
	require.Equal(t, 0, skip)
	require.Contains(t, out, "BENCH")
	require.Contains(t, out, "ns/op")
}

func TestRunFileStatementForm(t *testing.T) {
	src := `test := import("test")
test onePass { t.equal(1, 1) }
test "two words" { t.true(true) }
test threeFail { t.equal(1, 2) }
test fourSkip { t.skip("later") }
`
	o := &testOptions{verbose: true}
	pass, fail, skip, out := runFileOn(t, o, src)
	require.Equal(t, 2, pass, out)        // onePass, "two words"
	require.Equal(t, 1, fail, out)        // threeFail
	require.Equal(t, 1, skip, out)        // fourSkip
	require.Contains(t, out, "two words") // string name preserved
	require.Contains(t, out, "FAIL")
}

func TestRunFileNestedSubtests(t *testing.T) {
	src := `test := import("test")
test parent {
	t.helper()
	test childOk { t.equal(1, 1) }
	test childBad { t.equal(1, 2) }
	test group {
		test deep { t.true(true) }
	}
}
`
	o := &testOptions{verbose: true}
	pass, fail, skip, out := runFileOn(t, o, src)
	require.Equal(t, 0, skip, out)
	// pass: childOk, group, group/deep  (3); fail: parent, childBad (2)
	require.Equal(t, 3, pass, out)
	require.Equal(t, 2, fail, out)
	require.Contains(t, out, "parent/childOk")
	require.Contains(t, out, "parent/childBad")
	require.Contains(t, out, "parent/group/deep") // deep nesting
	require.Contains(t, out, "FAIL")              // parent fails because a subtest failed
}

func TestRunFileStatementBench(t *testing.T) {
	src := `bench "the loop" { for i := 0; i < t.n; i++ {} }`
	o := &testOptions{benchPat: ".", benchtime: time.Millisecond}
	pass, fail, _, out := runFileOn(t, o, src)
	require.Equal(t, 0, pass)
	require.Equal(t, 0, fail)
	require.Contains(t, out, "BENCH")
	require.Contains(t, out, "the loop")
}

func TestRunFileCompileError(t *testing.T) {
	o := &testOptions{}
	_, fail, _, out := runFileOn(t, o, "func testX(t) { this is not valid")
	require.Equal(t, 1, fail)
	require.Contains(t, out, "FAIL")
}

func TestRunFileRuntimeError(t *testing.T) {
	src := `func testBoom(t) { undefinedFn() }`
	o := &testOptions{}
	_, fail, _, out := runFileOn(t, o, src)
	require.Equal(t, 1, fail)
	require.Contains(t, out, "FAIL")
}

// TestRunFileRaises covers t.raises: it asserts a callee raises, with optional
// eq/contains/prefix/suffix message matchers. A callee that raises at run time
// must be caught cleanly (not panic) and its bare message matched (no `ErrCall:`
// wrapper).
func TestRunFileRaises(t *testing.T) {
	src := `
func testRaisesBare(t)     { a := []; t.raises(() => a[5]) }
func testRaisesContains(t) { a := []; t.raises(() => a[5]; contains="IndexOutOfBounds") }
func testRaisesEq(t)       { a := []; t.raises(() => a[5]; eq="IndexOutOfBoundsError: 5") }
func testRaisesAffix(t)    { a := []; t.raises(() => a[5]; prefix="IndexOutOfBounds", suffix="5") }
func testRaisesNoErrFails(t) { t.raises(() => 42) }
func testRaisesBadMatchFails(t) { a := []; t.raises(() => a[5]; contains="nope") }
`
	o := &testOptions{}
	pass, fail, _, out := runFileOn(t, o, src)
	require.Equal(t, 4, pass, out) // bare, contains, eq, affix
	require.Equal(t, 2, fail, out) // no-error, bad-match
}

// The inline checks (issue #22) run in a test: at the top level and in a test
// function, a failure failing it.
func TestRunFileInlineChecks(t *testing.T) {
	src := "x := 10\n= 10\nprintln(\"top\")\n<<< top\n" +
		"func testOk(t) {\n\tprintln(1)\n\t<<< 1\n\tt.equal(1, 1)\n\t2 * 2\n\t= 4\n}\n" +
		"func testBad(t) {\n\t2 * 2\n\t= 5\n}\n"
	o := &testOptions{verbose: true}
	pass, fail, _, out := runFileOn(t, o, src)
	require.Equal(t, 1, pass, out)
	require.Equal(t, 1, fail, out)
	require.Contains(t, out, "got 4, want 5")

	// a failing check at the top level fails the file
	_, fail, _, out = runFileOn(t, &testOptions{}, "x := 1\n= 2\nfunc testA(t) {}\n")
	require.Equal(t, 1, fail, out)
	require.Contains(t, out, "got 1, want 2")
}

// In a _test.gadx file: `<<<` after markup checks what was rendered into the
// current node since the previous check (one line, or the lines indented
// under it, no closer); `~ = EXPR` checks a value.
func TestRunFileInlineChecksGadx(t *testing.T) {
	run := func(src string) (int, int, string) {
		dir := t.TempDir()
		path := filepath.Join(dir, "x_test.gadx")
		require.NoError(t, os.WriteFile(path, []byte(src), 0o644))
		o := &testOptions{}
		require.NoError(t, o.compile())
		var buf bytes.Buffer
		pass, fail, _ := o.runFile(&cc.CommandContext{Out: &buf, Err: &buf}, path)
		return pass, fail, buf.String()
	}
	pass, fail, out := run("div hello\n\n<<< <div>hello</div>\np\n\tspan a\nul\n\tli 1\n\n<<<\n\t<p><span>a</span></p>\n\t<ul><li>1</li></ul>\n" +
		"~ x := 10\n~ = 10\n\n@test ok\n\t! t.true yes\n")
	require.Equal(t, 1, pass, out)
	require.Equal(t, 0, fail, out)

	_, fail, out = run("div hello\n<<< <div>bye</div>\n@test ok\n\t! t.true yes\n")
	require.Equal(t, 1, fail, out)
	require.Contains(t, out, "x_test.gadx:2:1: CheckError: rendered: got \"<div>hello</div>\", want \"<div>bye</div>\"")
}
