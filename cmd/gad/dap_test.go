//go:build !js
// +build !js

package main

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-dap"
)

// dapTestClient drives a dapServer over in-memory pipes.
type dapTestClient struct {
	t    *testing.T
	toS  io.Writer        // client -> server
	msgs chan dap.Message // server -> client
	seq  int
}

func startDAP(t *testing.T) *dapTestClient {
	t.Helper()
	cliR, srvW := io.Pipe() // server -> client
	srvR, cliW := io.Pipe() // client -> server

	s := &dapServer{
		in:   bufio.NewReader(srvR),
		out:  srvW,
		done: make(chan debugResult, 1),
	}
	go func() { _ = s.serve() }()

	c := &dapTestClient{t: t, toS: cliW, msgs: make(chan dap.Message, 64)}
	go func() {
		r := bufio.NewReader(cliR)
		for {
			m, err := dap.ReadProtocolMessage(r)
			if err != nil {
				close(c.msgs)
				return
			}
			c.msgs <- m
		}
	}()
	return c
}

func (c *dapTestClient) send(req dap.Message) {
	c.t.Helper()
	if err := dap.WriteProtocolMessage(c.toS, req); err != nil {
		c.t.Fatalf("write: %v", err)
	}
}

func (c *dapTestClient) header(command string) dap.Request {
	c.seq++
	return dap.Request{
		ProtocolMessage: dap.ProtocolMessage{Seq: c.seq, Type: "request"},
		Command:         command,
	}
}

// waitFor returns the next message satisfying pred, or fails on timeout.
func (c *dapTestClient) waitFor(pred func(dap.Message) bool) dap.Message {
	c.t.Helper()
	timeout := time.After(3 * time.Second)
	for {
		select {
		case m, ok := <-c.msgs:
			if !ok {
				c.t.Fatal("connection closed before expected message")
			}
			if pred(m) {
				return m
			}
		case <-timeout:
			c.t.Fatal("timed out waiting for message")
		}
	}
}

func isEvent(name string) func(dap.Message) bool {
	return func(m dap.Message) bool {
		e, ok := m.(dap.EventMessage)
		return ok && e.GetEvent().Event == name
	}
}

func TestDAPSession(t *testing.T) {
	dir := t.TempDir()
	prog := filepath.Join(dir, "p.gad")
	if err := os.WriteFile(prog, []byte("a := 1\nb := 2\nreturn a + b\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := startDAP(t)

	// initialize -> capabilities + initialized event
	c.send(&dap.InitializeRequest{Request: c.header("initialize")})
	c.waitFor(isEvent("initialized"))

	// set a breakpoint on line 3
	sb := &dap.SetBreakpointsRequest{Request: c.header("setBreakpoints")}
	sb.Arguments = dap.SetBreakpointsArguments{
		Source:      dap.Source{Path: prog},
		Breakpoints: []dap.SourceBreakpoint{{Line: 3}},
	}
	c.send(sb)
	c.waitFor(func(m dap.Message) bool {
		r, ok := m.(*dap.SetBreakpointsResponse)
		return ok && len(r.Body.Breakpoints) == 1 && r.Body.Breakpoints[0].Verified
	})

	c.send(&dap.ConfigurationDoneRequest{Request: c.header("configurationDone")})

	// launch the program
	lr := &dap.LaunchRequest{Request: c.header("launch")}
	lr.Arguments = []byte(`{"program":"` + filepath.ToSlash(prog) + `"}`)
	c.send(lr)

	// expect a stopped(breakpoint) event
	stopped := c.waitFor(isEvent("stopped")).(*dap.StoppedEvent)
	if stopped.Body.Reason != "breakpoint" {
		t.Fatalf("expected breakpoint stop, got %q", stopped.Body.Reason)
	}

	// stackTrace -> a frame at line 3
	c.send(&dap.StackTraceRequest{Request: c.header("stackTrace")})
	st := c.waitFor(func(m dap.Message) bool {
		_, ok := m.(*dap.StackTraceResponse)
		return ok
	}).(*dap.StackTraceResponse)
	if len(st.Body.StackFrames) == 0 || st.Body.StackFrames[0].Line != 3 {
		t.Fatalf("expected top frame at line 3, got %+v", st.Body.StackFrames)
	}

	// variables (locals) -> a and b present
	vr := &dap.VariablesRequest{Request: c.header("variables")}
	vr.Arguments = dap.VariablesArguments{VariablesReference: localsRef}
	c.send(vr)
	vars := c.waitFor(func(m dap.Message) bool {
		_, ok := m.(*dap.VariablesResponse)
		return ok
	}).(*dap.VariablesResponse)
	values := map[string]bool{}
	for _, v := range vars.Body.Variables {
		values[v.Value] = true
	}
	if !values["1"] || !values["2"] {
		t.Fatalf("expected locals 1 and 2, got %+v", vars.Body.Variables)
	}

	// continue -> program finishes -> terminated
	c.send(&dap.ContinueRequest{Request: c.header("continue")})
	c.waitFor(isEvent("terminated"))
}
