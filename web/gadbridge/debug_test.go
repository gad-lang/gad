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

// TestDebugManagerInspect starts a session, pauses, and inspects a container
// local — the tree-navigator flow the WASM bridge exposes as gadInspect.
func TestDebugManagerInspect(t *testing.T) {
	m := NewDebugManager()
	src := "d := {a: 1, b: [10, 20]}\nreturn d\n"

	r := m.Start(DebugStartRequest{Source: src, Path: "t.gad", Breakpoints: []int{2}})
	if r.State != "stopped" {
		t.Fatalf("start: got state=%q, want stopped", r.State)
	}

	insp, err := m.Inspect(r.Session, "d")
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if insp.Type != "dict" || !insp.Expandable || len(insp.Entries) != 2 {
		t.Fatalf("inspect d = %+v, want expandable dict with 2 entries", insp)
	}
	// The nested array child is itself expandable, reached via ["b"].
	var b *InspectEntry
	for i := range insp.Entries {
		if insp.Entries[i].Key == "b" {
			b = &insp.Entries[i]
		}
	}
	if b == nil || b.Accessor != `["b"]` || !b.Expandable {
		t.Fatalf("dict entry b = %+v, want expandable with accessor [\"b\"]", b)
	}

	// A missing session is an error, not a panic.
	if _, err := m.Inspect("nope", "d"); err == nil {
		t.Fatal("inspect on unknown session: want error")
	}
	m.Command(r.Session, "continue")
}

// TestInspectSource inspects a value evaluated fresh (no debug session) against a
// source prelude — the session-less path of gadInspect.
func TestInspectSource(t *testing.T) {
	insp, err := InspectSource("d := {a: 1, b: [10, 20]}", "d")
	if err != nil {
		t.Fatalf("InspectSource: %v", err)
	}
	if insp.Type != "dict" || !insp.Expandable || len(insp.Entries) != 2 {
		t.Fatalf("InspectSource d = %+v, want expandable dict with 2 entries", insp)
	}
	// Drill into b: [10, 20] is an array with two indexed entries.
	sub, err := InspectSource("d := {a: 1, b: [10, 20]}", `d["b"]`)
	if err != nil {
		t.Fatalf("InspectSource sub: %v", err)
	}
	if sub.Type != "array" || len(sub.Entries) != 2 || sub.Entries[0].Accessor != "[0]" {
		t.Fatalf("InspectSource d[\"b\"] = %+v, want array of 2", sub)
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
