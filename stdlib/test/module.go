// Package test provides the `test` builtin module and the `T` context passed to
// test/benchmark functions run by `gad test`. Its assertions mirror Go's
// testing + testify/require: a failed assertion records the failure and aborts
// the current test (require semantics).
package test

import (
	"strings"

	gad "github.com/gad-lang/gad"
)

// ModuleName is the import/builtin name of the module.
const ModuleName = "test"

// TT is the object type of the test context `T`.
var TT = gad.NewBuiltinObjType("T")

// FailError is returned by an assertion (or t.fatal) to abort the running test;
// the runner catches it and records the failure via the test context state.
type FailError struct{ Msg string }

func (e *FailError) Error() string { return e.Msg }

// SkipError is returned by t.skip to stop and mark the test skipped.
type SkipError struct{ Msg string }

func (e *SkipError) Error() string { return e.Msg }

// T is the per-test context. Assertion methods record a failure and, being
// require-style, abort by returning a FailError. Subtests started with `t.run`
// (or nested `test NAME { … }` statements) are recorded in subs.
type T struct {
	name     string
	failures []string
	logs     []string
	skipMsg  string
	skipped  bool
	benchN   int  // benchmark iteration count (0 for tests)
	subs     []*T // subtests started with t.run
}

// NewT returns a fresh test context named name.
func NewT(name string) *T { return &T{name: name} }

// Name is the test's name.
func (t *T) Name() string { return t.name }

// SelfFailed reports whether this test recorded a failure of its own (not
// counting subtests).
func (t *T) SelfFailed() bool { return len(t.failures) > 0 }

// Failure reports whether this test or any of its subtests failed (like Go's
// testing.T.Failed).
func (t *T) Failure() bool {
	if len(t.failures) > 0 {
		return true
	}
	for _, s := range t.subs {
		if s.Failure() {
			return true
		}
	}
	return false
}

// Failures are the failure messages recorded by this test itself.
func (t *T) Failures() []string { return t.failures }

// Subs are the subtests started with t.run, in run order.
func (t *T) Subs() []*T { return t.subs }

// AddSub records child as a subtest of t.
func (t *T) AddSub(child *T) { t.subs = append(t.subs, child) }

// Logs are the messages written with t.log.
func (t *T) Logs() []string { return t.logs }

// Skipped reports whether the test was skipped, with its message.
func (t *T) Skipped() (bool, string) { return t.skipped, t.skipMsg }

// SetBenchN sets the benchmark iteration count returned by t.n.
func (t *T) SetBenchN(n int) { t.benchN = n }

var (
	_ gad.Object           = (*T)(nil)
	_ gad.NameCallerObject = (*T)(nil)
	_ gad.IndexGetter      = (*T)(nil)
)

func (t *T) Type() gad.ObjectType { return TT }
func (t *T) ToString() string     { return gad.ReprQuote("T " + t.name) }
func (t *T) IsFalsy() bool        { return false }
func (t *T) Equal(right gad.Object) bool {
	o, _ := right.(*T)
	return o == t
}

// IndexGet exposes read-only fields: .name, .failed, .n (benchmark count).
func (t *T) IndexGet(_ *gad.VM, index gad.Object) (gad.Object, error) {
	switch index.ToString() {
	case "name":
		return gad.Str(t.name), nil
	case "failed":
		return gad.Bool(t.Failure()), nil
	case "n":
		return gad.Int(t.benchN), nil
	}
	return nil, gad.ErrInvalidIndex.NewError(index.ToString())
}

// fail records msg and returns a FailError so the test aborts.
func (t *T) fail(msg string) (gad.Object, error) {
	t.failures = append(t.failures, msg)
	return nil, &FailError{Msg: msg}
}

// run runs a subtest `t.run(name, fn)` (also what nested `test NAME { … }`
// statements lower to): it invokes fn with a fresh child context named
// parent/name, records it under subs, and returns whether the subtest passed.
// A subtest failure or skip does not abort the parent (like Go's t.Run).
func (t *T) run(c gad.Call) (gad.Object, error) {
	name := c.Args.Get(0).ToString()
	fn, _ := c.Args.Get(1).(gad.CallerObject)
	if fn == nil {
		return nil, gad.ErrType.NewError("run: second argument must be a function")
	}
	child := &T{name: t.name + "/" + name}
	_, err := runFn(c.VM, fn, child)
	// A require-style abort already recorded the failure in child; a skip set its
	// flag. Any other error is an unexpected runtime failure of the subtest.
	if err != nil && !child.SelfFailed() {
		if skipped, _ := child.Skipped(); !skipped {
			child.failures = append(child.failures, err.Error())
		}
	}
	t.AddSub(child)
	return gad.Bool(!child.Failure()), nil
}

// runFn invokes fn(child) on a forked VM.
func runFn(vm *gad.VM, fn gad.CallerObject, child *T) (gad.Object, error) {
	inv := gad.NewInvoker(vm, fn)
	inv.Acquire()
	defer inv.Release()
	return inv.Invoke(gad.Args{gad.Array{child}}, nil)
}

// CallName dispatches t's methods (assertions and controls).
func (t *T) CallName(name string, c gad.Call) (gad.Object, error) {
	switch name {
	case "name":
		return gad.Str(t.name), nil
	case "failed":
		return gad.Bool(t.Failure()), nil
	case "log":
		t.logs = append(t.logs, joinArgs(c.Args))
		return gad.Nil, nil
	case "fail":
		return t.fail(argMsg(c.Args, "failed"))
	case "fatal":
		return t.fail(argMsg(c.Args, "failed"))
	case "skip":
		t.skipped, t.skipMsg = true, argMsg(c.Args, "")
		return nil, &SkipError{Msg: t.skipMsg}
	case "helper":
		// Marks the caller as a test helper. Gad does not track caller source
		// positions for failure attribution, so this is a no-op accepted for
		// parity with Go's t.Helper().
		return gad.Nil, nil
	case "run":
		return t.run(c)
	case "true":
		if v := c.Args.Get(0); v.IsFalsy() {
			return t.fail(withMsg(c, "expected true, got "+repr(v)))
		}
		return gad.Nil, nil
	case "false":
		if v := c.Args.Get(0); !v.IsFalsy() {
			return t.fail(withMsg(c, "expected false, got "+repr(v)))
		}
		return gad.Nil, nil
	case "nil":
		if v := c.Args.Get(0); v != gad.Nil {
			return t.fail(withMsg(c, "expected nil, got "+repr(v)))
		}
		return gad.Nil, nil
	case "notNil":
		if v := c.Args.Get(0); v == gad.Nil {
			return t.fail(withMsg(c, "expected non-nil"))
		}
		return gad.Nil, nil
	case "equal":
		a, b := c.Args.Get(0), c.Args.Get(1)
		if !a.Equal(b) {
			return t.fail(withMsg(c, "not equal:\n  expected: "+repr(a)+"\n  actual:   "+repr(b)))
		}
		return gad.Nil, nil
	case "notEqual":
		a, b := c.Args.Get(0), c.Args.Get(1)
		if a.Equal(b) {
			return t.fail(withMsg(c, "should not be equal: "+repr(a)))
		}
		return gad.Nil, nil
	case "contains":
		s, sub := c.Args.Get(0).ToString(), c.Args.Get(1).ToString()
		if !strings.Contains(s, sub) {
			return t.fail(withMsg(c, "expected "+repr(gad.Str(s))+" to contain "+repr(gad.Str(sub))))
		}
		return gad.Nil, nil
	case "error":
		if err := callFn(c); err == nil {
			return t.fail(withMsg(c, "expected an error, got none"))
		}
		return gad.Nil, nil
	case "noError":
		if err := callFn(c); err != nil {
			return t.fail(withMsg(c, "unexpected error: "+err.Error()))
		}
		return gad.Nil, nil
	}
	return nil, gad.ErrInvalidIndex.NewError(name)
}

// callFn calls the first argument (a callable) with no args, returning its error.
func callFn(c gad.Call) error {
	fn, _ := c.Args.Get(0).(gad.CallerObject)
	if fn == nil {
		return nil
	}
	_, err := gad.DoCall(fn, gad.Call{VM: c.VM})
	return err
}

func repr(o gad.Object) string {
	if s, err := gad.ToRepr(nil, o, gad.PrinterStateOptions{}); err == nil {
		return s.ToString()
	}
	return o.ToString()
}

func joinArgs(args gad.Args) string {
	var b strings.Builder
	args.Walk(func(i int, arg gad.Object) any {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(arg.ToString())
		return nil
	})
	return b.String()
}

// argMsg returns the first arg as a message, or def when none was passed.
func argMsg(args gad.Args, def string) string {
	if args.Length() > 0 {
		return joinArgs(args)
	}
	return def
}

// withMsg appends a `msg=` named argument (or a trailing positional message) to
// the default failure text.
func withMsg(c gad.Call, def string) string {
	if m := c.NamedArgs.GetValueOrNil("msg"); m != nil && m != gad.Nil {
		return m.ToString() + ": " + def
	}
	return def
}

// Module is the `test` builtin namespace.
var Module = gad.Dict{
	// T is the test-context type.
	"T": TT,
	// require-style helpers that take the test context as the first argument,
	// mirroring Go's testify/require (`test.equal(t, a, b)`); they delegate to
	// the matching t method.
	"equal":    reqFn("equal"),
	"notEqual": reqFn("notEqual"),
	"true":     reqFn("true"),
	"false":    reqFn("false"),
	"nil":      reqFn("nil"),
	"notNil":   reqFn("notNil"),
	"contains": reqFn("contains"),
	"error":    reqFn("error"),
	"noError":  reqFn("noError"),
	"fail":     reqFn("fail"),
	"fatal":    reqFn("fatal"),
}

// reqFn builds a `test.NAME(t, args…)` helper delegating to t.NAME(args…).
func reqFn(name string) *gad.BuiltinFunction {
	return &gad.BuiltinFunction{
		FuncName: name,
		Value: func(c gad.Call) (gad.Object, error) {
			t, _ := c.Args.Get(0).(*T)
			if t == nil {
				return nil, gad.ErrType.NewError("first argument must be a test context (T)")
			}
			rest := c.Args
			rest.Shift()
			return t.CallName(name, gad.Call{VM: c.VM, Args: rest, NamedArgs: c.NamedArgs})
		},
	}
}

// ModuleInit registers the module data.
var ModuleInit = gad.ModuleInitFunc(func(module *gad.Module, c gad.Call) (err error) {
	module.Data = Module
	return
})
