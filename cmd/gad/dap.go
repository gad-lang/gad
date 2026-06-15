// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

//go:build !js && !nodebug
// +build !js,!nodebug

package main

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"sync"
	"sync/atomic"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/debug"
	"github.com/google/go-dap"
	cc "github.com/moisespsena-go/command-context"
)

// The single (conceptual) thread and the locals scope handle exposed to the
// client. The Gad VM is single-threaded.
const (
	dapThreadID = 1
	localsRef   = 1000
)

// dapServer implements a minimal Debug Adapter Protocol server over a stream,
// bridging an editor (e.g. VS Code) to the debug.Engine.
type dapServer struct {
	in  *bufio.Reader
	out io.Writer

	wmu sync.Mutex   // serializes writes
	seq atomic.Int64 // message sequence

	eng     *debug.Engine
	vm      *gad.VM
	lines   []string
	program string
	done    chan debugResult
}

// serveDAP runs a DAP session on stdio (the usual transport for an editor-
// launched debug adapter).
func serveDAP(ctx *cc.CommandContext) error {
	s := &dapServer{
		in:   bufio.NewReader(os.Stdin),
		out:  os.Stdout,
		done: make(chan debugResult, 1),
	}
	if len(ctx.Args) == 1 {
		s.program = ctx.Args[0]
	}
	return s.serve()
}

func (s *dapServer) serve() error {
	for {
		msg, err := dap.ReadProtocolMessage(s.in)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if stop := s.handle(msg); stop {
			return nil
		}
	}
}

func (s *dapServer) nextSeq() int { return int(s.seq.Add(1)) }

func (s *dapServer) send(m dap.Message) {
	s.wmu.Lock()
	defer s.wmu.Unlock()
	_ = dap.WriteProtocolMessage(s.out, m)
}

// resp builds a successful response header for req.
func (s *dapServer) resp(req *dap.Request) dap.Response {
	return dap.Response{
		ProtocolMessage: dap.ProtocolMessage{Seq: s.nextSeq(), Type: "response"},
		RequestSeq:      req.Seq,
		Success:         true,
		Command:         req.Command,
	}
}

func (s *dapServer) event(name string) dap.Event {
	return dap.Event{
		ProtocolMessage: dap.ProtocolMessage{Seq: s.nextSeq(), Type: "event"},
		Event:           name,
	}
}

// handle dispatches one incoming message. It returns true to end the session.
func (s *dapServer) handle(msg dap.Message) bool {
	switch m := msg.(type) {
	case *dap.InitializeRequest:
		r := &dap.InitializeResponse{Response: s.resp(&m.Request)}
		r.Body = dap.Capabilities{
			SupportsConfigurationDoneRequest: true,
			SupportsTerminateRequest:         true,
		}
		s.send(r)
		s.send(&dap.InitializedEvent{Event: s.event("initialized")})

	case *dap.SetBreakpointsRequest:
		s.send(s.handleSetBreakpoints(m))

	case *dap.ConfigurationDoneRequest:
		s.send(&dap.ConfigurationDoneResponse{Response: s.resp(&m.Request)})

	case *dap.LaunchRequest:
		s.handleLaunch(m)

	case *dap.ThreadsRequest:
		r := &dap.ThreadsResponse{Response: s.resp(&m.Request)}
		r.Body.Threads = []dap.Thread{{Id: dapThreadID, Name: "main"}}
		s.send(r)

	case *dap.StackTraceRequest:
		s.send(s.handleStackTrace(m))

	case *dap.ScopesRequest:
		r := &dap.ScopesResponse{Response: s.resp(&m.Request)}
		r.Body.Scopes = []dap.Scope{{Name: "Locals", VariablesReference: localsRef}}
		s.send(r)

	case *dap.VariablesRequest:
		s.send(s.handleVariables(m))

	case *dap.ContinueRequest:
		s.eng.Continue()
		r := &dap.ContinueResponse{Response: s.resp(&m.Request)}
		r.Body.AllThreadsContinued = true
		s.send(r)

	case *dap.NextRequest:
		s.eng.StepOver()
		s.send(&dap.NextResponse{Response: s.resp(&m.Request)})

	case *dap.StepInRequest:
		s.eng.StepInto()
		s.send(&dap.StepInResponse{Response: s.resp(&m.Request)})

	case *dap.StepOutRequest:
		s.eng.StepOut()
		s.send(&dap.StepOutResponse{Response: s.resp(&m.Request)})

	case *dap.PauseRequest:
		if s.eng != nil {
			s.eng.Pause()
		}
		s.send(&dap.PauseResponse{Response: s.resp(&m.Request)})

	case *dap.DisconnectRequest:
		// Abort the VM and end the session; as a stdio adapter the process then
		// exits, releasing any parked goroutine.
		if s.vm != nil {
			s.vm.Abort()
		}
		s.send(&dap.DisconnectResponse{Response: s.resp(&m.Request)})
		return true

	case *dap.TerminateRequest:
		if s.vm != nil {
			s.vm.Abort()
		}
		s.send(&dap.TerminateResponse{Response: s.resp(&m.Request)})

	case dap.RequestMessage:
		// Unsupported request: acknowledge so the client is not left waiting.
		s.send(&dap.ErrorResponse{
			Response: dap.Response{
				ProtocolMessage: dap.ProtocolMessage{Seq: s.nextSeq(), Type: "response"},
				RequestSeq:      m.GetRequest().Seq,
				Success:         false,
				Command:         m.GetRequest().Command,
				Message:         "unsupported request: " + m.GetRequest().Command,
			},
		})
	}
	return false
}

// launchArgs is the subset of launch configuration we read.
type launchArgs struct {
	Program     string `json:"program"`
	StopOnEntry bool   `json:"stopOnEntry"`
}

func (s *dapServer) handleLaunch(m *dap.LaunchRequest) {
	var args launchArgs
	_ = json.Unmarshal(m.Arguments, &args)
	if args.Program != "" {
		s.program = args.Program
	}

	if s.program == "" {
		s.send(&dap.ErrorResponse{Response: dap.Response{
			ProtocolMessage: dap.ProtocolMessage{Seq: s.nextSeq(), Type: "response"},
			RequestSeq:      m.Seq, Success: false, Command: m.Command,
			Message: "no program to launch",
		}})
		return
	}

	bc, builtins, lines, err := loadProgram(s.program)
	if err != nil {
		s.send(&dap.ErrorResponse{Response: dap.Response{
			ProtocolMessage: dap.ProtocolMessage{Seq: s.nextSeq(), Type: "response"},
			RequestSeq:      m.Seq, Success: false, Command: m.Command,
			Message: err.Error(),
		}})
		return
	}
	s.lines = lines

	if s.eng == nil {
		s.eng = debug.New(args.StopOnEntry)
	}
	s.vm = gad.NewVM(builtins.Build(), bc).SetRecover(true)
	s.vm.SetDebugger(s.eng)

	go func() {
		ret, rerr := s.vm.RunOpts(&gad.RunOpts{
			StdOut: &dapWriter{s: s, category: "stdout"},
			StdErr: &dapWriter{s: s, category: "stderr"},
		})
		s.done <- debugResult{ret, rerr}
	}()
	go s.forwardEvents()

	s.send(&dap.LaunchResponse{Response: s.resp(&m.Request)})
}

// forwardEvents turns engine stop events into DAP `stopped` events and the
// program's completion into `terminated`/`exited`.
func (s *dapServer) forwardEvents() {
	for {
		select {
		case ev := <-s.eng.Stops():
			body := dap.StoppedEventBody{
				Reason:            mapStopReason(ev.Reason),
				ThreadId:          dapThreadID,
				AllThreadsStopped: true,
			}
			e := &dap.StoppedEvent{Event: s.event("stopped")}
			e.Body = body
			s.send(e)
		case r := <-s.done:
			if r.err != nil {
				s.sendOutput("stderr", r.err.Error()+"\n")
			}
			ex := &dap.ExitedEvent{Event: s.event("exited")}
			if r.err != nil {
				ex.Body.ExitCode = 1
			}
			s.send(ex)
			s.send(&dap.TerminatedEvent{Event: s.event("terminated")})
			return
		}
	}
}

func (s *dapServer) handleSetBreakpoints(m *dap.SetBreakpointsRequest) *dap.SetBreakpointsResponse {
	if s.eng == nil {
		s.eng = debug.New(false)
	}
	var lines []int
	bps := make([]dap.Breakpoint, 0, len(m.Arguments.Breakpoints))
	for _, b := range m.Arguments.Breakpoints {
		lines = append(lines, b.Line)
		bps = append(bps, dap.Breakpoint{Verified: true, Line: b.Line})
	}
	s.eng.SetBreakpoints(lines)
	r := &dap.SetBreakpointsResponse{Response: s.resp(&m.Request)}
	r.Body.Breakpoints = bps
	return r
}

func (s *dapServer) handleStackTrace(m *dap.StackTraceRequest) *dap.StackTraceResponse {
	r := &dap.StackTraceResponse{Response: s.resp(&m.Request)}
	if s.eng == nil {
		return r
	}
	frames := s.eng.Frames()
	src := &dap.Source{Name: baseName(s.program), Path: s.program}
	// Innermost frame first (DAP convention).
	id := 0
	for i := len(frames) - 1; i >= 0; i-- {
		f := frames[i]
		r.Body.StackFrames = append(r.Body.StackFrames, dap.StackFrame{
			Id:     id,
			Name:   f.FuncName,
			Source: src,
			Line:   f.Line,
			Column: f.Column,
		})
		id++
	}
	r.Body.TotalFrames = len(r.Body.StackFrames)
	return r
}

func (s *dapServer) handleVariables(m *dap.VariablesRequest) *dap.VariablesResponse {
	r := &dap.VariablesResponse{Response: s.resp(&m.Request)}
	if s.eng == nil || m.Arguments.VariablesReference != localsRef {
		return r
	}
	for _, v := range s.eng.Locals() {
		r.Body.Variables = append(r.Body.Variables, dap.Variable{
			Name:  v.Name,
			Value: v.Value,
			Type:  v.Type,
		})
	}
	return r
}

func (s *dapServer) sendOutput(category, text string) {
	e := &dap.OutputEvent{Event: s.event("output")}
	e.Body = dap.OutputEventBody{Category: category, Output: text}
	s.send(e)
}

// dapWriter forwards VM stdout/stderr to the client as `output` events.
type dapWriter struct {
	s        *dapServer
	category string
}

func (w *dapWriter) Write(p []byte) (int, error) {
	w.s.sendOutput(w.category, string(p))
	return len(p), nil
}

func mapStopReason(r debug.StopReason) string {
	switch r {
	case debug.StopBreakpoint:
		return "breakpoint"
	case debug.StopEntry:
		return "entry"
	case debug.StopPause:
		return "pause"
	default:
		return "step"
	}
}

func baseName(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' || p[i] == '\\' {
			return p[i+1:]
		}
	}
	return p
}
