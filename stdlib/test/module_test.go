package test_test

import (
	"strings"
	"testing"

	gad "github.com/gad-lang/gad"
	gadtest "github.com/gad-lang/gad/stdlib/test"
)

// call invokes method name on t with the given positional args.
func call(t *gadtest.T, name string, args ...gad.Object) (gad.Object, error) {
	return t.CallName(name, gad.Call{Args: gad.Args{gad.Array(args)}})
}

func TestEqualStringDiff(t *testing.T) {
	// two differing strings -> a unified line diff in the failure message.
	ctx := gadtest.NewT("ctx")
	if _, err := call(ctx, "equal", gad.Str("a\nb\nc\n"), gad.Str("a\nX\nc\n")); err == nil {
		t.Fatal("expected a failure")
	}
	if len(ctx.Failures()) != 1 {
		t.Fatalf("expected one failure, got %v", ctx.Failures())
	}
	msg := ctx.Failures()[0]
	for _, want := range []string{"strings not equal", "--- expected", "+++ actual", "-b", "+X"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("string diff missing %q in:\n%s", want, msg)
		}
	}

	// non-string operands keep the plain repr (no diff).
	nctx := gadtest.NewT("ctx")
	call(nctx, "equal", gad.Int(1), gad.Int(2))
	m := nctx.Failures()[0]
	if strings.Contains(m, "strings not equal") || !strings.Contains(m, "not equal") {
		t.Fatalf("non-string equal should use plain repr, got:\n%s", m)
	}

	// equal strings still pass.
	pctx := gadtest.NewT("ctx")
	mustPass(t, pctx, "equal", gad.Str("same"), gad.Str("same"))
}

// mustPass asserts the method returned Nil with no error and recorded no failure.
func mustPass(tb testing.TB, t *gadtest.T, name string, args ...gad.Object) {
	tb.Helper()
	ret, err := call(t, name, args...)
	if err != nil {
		tb.Fatalf("%s: unexpected error: %v", name, err)
	}
	if ret != gad.Nil {
		tb.Fatalf("%s: expected Nil, got %v", name, ret)
	}
	if t.Failure() {
		tb.Fatalf("%s: unexpected failure: %v", name, t.Failures())
	}
}

// mustFail asserts the method returned an abort error and recorded a failure.
func mustFail(tb testing.TB, name string, args ...gad.Object) {
	tb.Helper()
	t := gadtest.NewT(name)
	_, err := call(t, name, args...)
	if err == nil {
		tb.Fatalf("%s: expected an abort error, got nil", name)
	}
	if !t.Failure() {
		tb.Fatalf("%s: expected a recorded failure", name)
	}
}

func TestAssertionsPass(t *testing.T) {
	ctx := gadtest.NewT("ctx")
	mustPass(t, ctx, "equal", gad.Int(1), gad.Int(1))
	mustPass(t, ctx, "notEqual", gad.Int(1), gad.Int(2))
	mustPass(t, ctx, "true", gad.True)
	mustPass(t, ctx, "false", gad.False)
	mustPass(t, ctx, "nil", gad.Nil)
	mustPass(t, ctx, "notNil", gad.Int(0))
	mustPass(t, ctx, "contains", gad.Str("hello world"), gad.Str("world"))
	if ctx.Failure() {
		t.Fatalf("context should have no failures, got %v", ctx.Failures())
	}
}

func TestAssertionsFail(t *testing.T) {
	mustFail(t, "equal", gad.Int(1), gad.Int(2))
	mustFail(t, "notEqual", gad.Int(1), gad.Int(1))
	mustFail(t, "true", gad.False)
	mustFail(t, "false", gad.True)
	mustFail(t, "nil", gad.Int(1))
	mustFail(t, "notNil", gad.Nil)
	mustFail(t, "contains", gad.Str("hello"), gad.Str("bye"))
}

func TestFailReturnsFailError(t *testing.T) {
	ctx := gadtest.NewT("ctx")
	_, err := call(ctx, "fail", gad.Str("boom"))
	fe, ok := err.(*gadtest.FailError)
	if !ok {
		t.Fatalf("expected *FailError, got %T", err)
	}
	if fe.Msg != "boom" {
		t.Fatalf("expected msg %q, got %q", "boom", fe.Msg)
	}
	if !ctx.Failure() {
		t.Fatal("expected recorded failure")
	}
}

func TestSkip(t *testing.T) {
	ctx := gadtest.NewT("ctx")
	_, err := call(ctx, "skip", gad.Str("later"))
	se, ok := err.(*gadtest.SkipError)
	if !ok {
		t.Fatalf("expected *SkipError, got %T", err)
	}
	if se.Msg != "later" {
		t.Fatalf("expected msg %q, got %q", "later", se.Msg)
	}
	skipped, msg := ctx.Skipped()
	if !skipped || msg != "later" {
		t.Fatalf("expected skipped with %q, got skipped=%v msg=%q", "later", skipped, msg)
	}
	if ctx.Failure() {
		t.Fatal("skip must not record a failure")
	}
}

func TestLogAndIndex(t *testing.T) {
	ctx := gadtest.NewT("myname")
	if _, err := call(ctx, "log", gad.Str("hi"), gad.Int(2)); err != nil {
		t.Fatal(err)
	}
	if got := ctx.Logs(); len(got) != 1 || got[0] != "hi 2" {
		t.Fatalf("unexpected logs: %v", got)
	}
	name, err := ctx.IndexGet(nil, gad.Str("name"))
	if err != nil || name.ToString() != "myname" {
		t.Fatalf("name index: %v %v", name, err)
	}
	failed, err := ctx.IndexGet(nil, gad.Str("failed"))
	if err != nil || failed != gad.False {
		t.Fatalf("failed index: %v %v", failed, err)
	}
}

func TestModuleHelperDelegates(t *testing.T) {
	fn, ok := gadtest.Module["equal"].(*gad.BuiltinFunction)
	if !ok {
		t.Fatalf("test.equal is not a BuiltinFunction: %T", gadtest.Module["equal"])
	}
	ctx := gadtest.NewT("ctx")
	// test.equal(t, 1, 1) passes.
	if _, err := fn.Value(gad.Call{Args: gad.Args{gad.Array{ctx, gad.Int(1), gad.Int(1)}}}); err != nil {
		t.Fatalf("passing helper returned error: %v", err)
	}
	if ctx.Failure() {
		t.Fatalf("passing helper recorded a failure: %v", ctx.Failures())
	}
	// test.equal(t, 1, 2) fails.
	if _, err := fn.Value(gad.Call{Args: gad.Args{gad.Array{ctx, gad.Int(1), gad.Int(2)}}}); err == nil {
		t.Fatal("failing helper returned no error")
	}
	if !ctx.Failure() {
		t.Fatal("failing helper recorded no failure")
	}
}

func TestHelperNoOp(t *testing.T) {
	ctx := gadtest.NewT("ctx")
	ret, err := call(ctx, "helper")
	if err != nil || ret != gad.Nil {
		t.Fatalf("helper: ret=%v err=%v", ret, err)
	}
	if ctx.Failure() {
		t.Fatal("helper must not record a failure")
	}
}

func TestFailurePropagatesFromSubs(t *testing.T) {
	parent := gadtest.NewT("p")
	if parent.Failure() || parent.SelfFailed() {
		t.Fatal("fresh parent should not be failed")
	}
	// simulate a failing subtest by recording a failure on a child collected
	// under the parent (mirrors what t.run does).
	child := gadtest.NewT("p/child")
	call(child, "fail", gad.Str("boom"))
	parent.AddSub(child)

	if !child.Failure() || !child.SelfFailed() {
		t.Fatal("child should be failed")
	}
	if parent.SelfFailed() {
		t.Fatal("parent has no own failure")
	}
	if !parent.Failure() {
		t.Fatal("parent.Failure() must be true when a subtest failed")
	}
	if subs := parent.Subs(); len(subs) != 1 || subs[0] != child {
		t.Fatalf("expected one sub == child, got %v", subs)
	}
}

func TestBenchN(t *testing.T) {
	ctx := gadtest.NewT("b")
	ctx.SetBenchN(42)
	n, err := ctx.IndexGet(nil, gad.Str("n"))
	if err != nil || n != gad.Int(42) {
		t.Fatalf("bench n: %v %v", n, err)
	}
}
