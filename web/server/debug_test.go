package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func callStart(t *testing.T, m *debugManager, req startRequest) debugResponse {
	t.Helper()
	body, _ := json.Marshal(req)
	r := httptest.NewRequest("POST", "/api/debug/start", bytes.NewReader(body))
	w := httptest.NewRecorder()
	m.handleStart(w, r)
	var resp debugResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode start: %v (%s)", err, w.Body.String())
	}
	return resp
}

func callCommand(t *testing.T, m *debugManager, session, command string) debugResponse {
	t.Helper()
	body, _ := json.Marshal(commandRequest{Session: session, Command: command})
	r := httptest.NewRequest("POST", "/api/debug/command", bytes.NewReader(body))
	w := httptest.NewRecorder()
	m.handleCommand(w, r)
	var resp debugResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode command: %v (%s)", err, w.Body.String())
	}
	return resp
}

func TestDebugHTTPSession(t *testing.T) {
	m := newDebugManager()

	start := callStart(t, m, startRequest{
		Source:      "a := 1\nb := 2\nprintln(\"sum\", a+b)\nreturn a + b\n",
		Breakpoints: []int{3},
	})
	if start.State != "stopped" || start.Line != 3 {
		t.Fatalf("expected stopped at line 3, got %+v", start)
	}
	if start.Session == "" {
		t.Fatal("expected a session id")
	}
	if len(start.Frames) == 0 {
		t.Fatal("expected at least one frame")
	}
	values := map[string]bool{}
	for _, v := range start.Locals {
		values[v.Value] = true
	}
	if !values["1"] || !values["2"] {
		t.Fatalf("expected locals 1 and 2, got %+v", start.Locals)
	}

	// Continue to completion.
	end := callCommand(t, m, start.Session, "continue")
	if end.State != "terminated" {
		t.Fatalf("expected terminated, got %+v", end)
	}
	if end.Result != "3" {
		t.Fatalf("expected result 3, got %q", end.Result)
	}
	if end.Output == "" {
		t.Fatalf("expected program output, got none")
	}
}

func TestDebugHTTPStepping(t *testing.T) {
	m := newDebugManager()
	start := callStart(t, m, startRequest{
		Source:      "a := 1\nb := 2\nc := a + b\nreturn c\n",
		StopOnEntry: true,
	})
	if start.State != "stopped" || start.Reason != "entry" {
		t.Fatalf("expected entry stop, got %+v", start)
	}

	// Step over a few lines.
	seenLine := 0
	for i := 0; i < 5; i++ {
		resp := callCommand(t, m, start.Session, "next")
		if resp.State == "terminated" {
			if resp.Result != "3" {
				t.Fatalf("expected result 3, got %q", resp.Result)
			}
			return
		}
		if resp.State != "stopped" {
			t.Fatalf("unexpected state %+v", resp)
		}
		if resp.Line < seenLine {
			t.Fatalf("expected non-decreasing line, got %d after %d", resp.Line, seenLine)
		}
		seenLine = resp.Line
	}
	t.Fatal("did not reach termination after stepping")
}

func TestDebugHTTPCompileError(t *testing.T) {
	m := newDebugManager()
	resp := callStart(t, m, startRequest{Source: "return missing\n"})
	if resp.State != "error" {
		t.Fatalf("expected error state, got %+v", resp)
	}
	if len(resp.Diagnostics) == 0 {
		t.Fatal("expected diagnostics for a compile error")
	}
}
