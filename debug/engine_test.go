package debug_test

import (
	"testing"
	"time"

	gad "github.com/gad-lang/gad"
	"github.com/gad-lang/gad/debug"
)

func compile(t *testing.T, src string) *gad.Bytecode {
	t.Helper()
	b := gad.NewBuiltins()
	st := gad.NewSymbolTable(b.NameSet)
	_, bc, err := gad.Compile(st, []byte(src), gad.CompileOptions{})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	return bc
}

// run starts the VM with eng attached in a goroutine and returns a result
// channel.
func run(t *testing.T, bc *gad.Bytecode, eng *debug.Engine) <-chan gad.Object {
	t.Helper()
	out := make(chan gad.Object, 1)
	vm := gad.NewVM(gad.NewBuiltins().Build(), bc)
	vm.SetDebugger(eng)
	go func() {
		ret, err := vm.RunOpts(&gad.RunOpts{})
		if err != nil {
			t.Errorf("run: %v", err)
		}
		out <- ret
	}()
	return out
}

func waitStop(t *testing.T, eng *debug.Engine) debug.StopEvent {
	t.Helper()
	select {
	case ev := <-eng.Stops():
		return ev
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for a stop event")
		return debug.StopEvent{}
	}
}

func waitResult(t *testing.T, out <-chan gad.Object) gad.Object {
	t.Helper()
	select {
	case r := <-out:
		return r
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for run result")
		return nil
	}
}

func TestBreakpointInspectStep(t *testing.T) {
	const src = "a := 1\nb := 2\nc := a + b\nreturn c\n" // lines 1..4

	eng := debug.New(false)
	eng.SetBreakpoints([]int{3})
	out := run(t, compile(t, src), eng)

	ev := waitStop(t, eng)
	if ev.Reason != debug.StopBreakpoint || ev.Line != 3 {
		t.Fatalf("expected breakpoint at line 3, got %+v", ev)
	}

	// a and b are assigned by line 3; verify they are visible.
	vals := map[string]bool{}
	for _, v := range eng.Locals() {
		vals[v.Value] = true
	}
	if !vals["1"] || !vals["2"] {
		t.Fatalf("expected locals 1 and 2 at the breakpoint, got %v", eng.Locals())
	}

	// Step to the next line, then run to completion.
	eng.StepOver()
	ev = waitStop(t, eng)
	if ev.Line != 4 {
		t.Fatalf("expected to step to line 4, got %+v", ev)
	}

	eng.Continue()
	if r := waitResult(t, out); r.(gad.Int) != 3 {
		t.Fatalf("expected result 3, got %v", r)
	}
}

func TestStopOnEntry(t *testing.T) {
	eng := debug.New(true)
	out := run(t, compile(t, "return 42\n"), eng)

	ev := waitStop(t, eng)
	if ev.Reason != debug.StopEntry {
		t.Fatalf("expected entry stop, got %+v", ev)
	}
	eng.Continue()
	if r := waitResult(t, out); r.(gad.Int) != 42 {
		t.Fatalf("expected 42, got %v", r)
	}
}

func TestStepIntoAndOut(t *testing.T) {
	const src = "f := func(x) {\n\treturn x * 2\n}\nr := f(21)\nreturn r\n"
	// line 1: f := func   line 2: return x*2   line 4: r := f(21)   line 5: return r

	eng := debug.New(false)
	eng.SetBreakpoints([]int{4}) // at the call site
	out := run(t, compile(t, src), eng)

	ev := waitStop(t, eng)
	if ev.Line != 4 {
		t.Fatalf("expected stop at call site line 4, got %+v", ev)
	}
	depthBefore := len(eng.Frames())

	// Step into the function body (line 2), depth should increase.
	eng.StepInto()
	ev = waitStop(t, eng)
	if ev.Line != 2 {
		t.Fatalf("expected to step into the function (line 2), got %+v", ev)
	}
	if len(eng.Frames()) <= depthBefore {
		t.Fatalf("expected deeper stack after step-into (%d -> %d)", depthBefore, len(eng.Frames()))
	}

	// Step out back to the caller.
	eng.StepOut()
	ev = waitStop(t, eng)
	if len(eng.Frames()) > depthBefore {
		t.Fatalf("expected to return to the caller frame after step-out, got %+v", ev)
	}

	eng.Continue()
	if r := waitResult(t, out); r.(gad.Int) != 42 {
		t.Fatalf("expected 42, got %v", r)
	}
}

func TestContinueNoBreakpoints(t *testing.T) {
	eng := debug.New(false)
	out := run(t, compile(t, "return 7\n"), eng)
	// No breakpoints, no stop-on-entry: should run straight to completion.
	if r := waitResult(t, out); r.(gad.Int) != 7 {
		t.Fatalf("expected 7, got %v", r)
	}
}
