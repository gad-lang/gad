package gadbridge

import "testing"

// TestDebugManager drives a full session: start to a breakpoint, evaluate in the
// frame, step, and continue to termination — the same request/response flow the
// WASM bridge exposes.
func TestDebugManager(t *testing.T) {
	m := NewDebugManager()
	src := "a := 1\nb := 2\nc := a + b\nreturn c\n"

	r := m.Start(DebugStartRequest{Source: src, Path: "t.gad", Breakpoints: []int{3}})
	if r.State != "stopped" || r.Line != 3 {
		t.Fatalf("start: got state=%q line=%d, want stopped@3", r.State, r.Line)
	}
	if r.Session == "" {
		t.Fatal("start: empty session id")
	}
	if len(r.Locals) != 3 { // a, b, c (c still nil)
		t.Fatalf("start: got %d locals, want 3", len(r.Locals))
	}

	// Evaluate an expression in the paused frame.
	if v, err := m.Eval(r.Session, "a + b", false); err != nil || v != "3" {
		t.Fatalf("eval: got %q err=%v, want 3", v, err)
	}

	// Step over line 3 -> line 4.
	if r2 := m.Command(r.Session, "next"); r2.State != "stopped" || r2.Line != 4 {
		t.Fatalf("next: got state=%q line=%d, want stopped@4", r2.State, r2.Line)
	}

	// Continue to the end.
	r3 := m.Command(r.Session, "continue")
	if r3.State != "terminated" || r3.Result != "3" {
		t.Fatalf("continue: got state=%q result=%q, want terminated/3", r3.State, r3.Result)
	}

	// The session is gone after termination.
	if r4 := m.Command(r.Session, "continue"); r4.State != "error" {
		t.Fatalf("post-terminate command: got state=%q, want error", r4.State)
	}
}

// TestDebugManagerStopOnEntry stops before the first instruction.
func TestDebugManagerStopOnEntry(t *testing.T) {
	m := NewDebugManager()
	r := m.Start(DebugStartRequest{Source: "return 42\n", Path: "t.gad", StopOnEntry: true})
	if r.State != "stopped" || r.Reason != "entry" {
		t.Fatalf("got state=%q reason=%q, want stopped/entry", r.State, r.Reason)
	}
	if r2 := m.Command(r.Session, "continue"); r2.State != "terminated" || r2.Result != "42" {
		t.Fatalf("got state=%q result=%q, want terminated/42", r2.State, r2.Result)
	}
}
