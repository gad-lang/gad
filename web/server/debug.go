package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"sync"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/debug"
	"github.com/gad-lang/gad/web/gadbridge"
)

// A request/response debugging protocol that the web "Run & Debug" page drives:
// /api/debug/start launches a session and runs to the first stop (or end);
// /api/debug/command resumes (continue/step) to the next stop (or end). Each
// response carries the stop event, the call stack, locals and any new output.

type debugManager struct {
	mu       sync.Mutex
	sessions map[string]*debugSession
}

func newDebugManager() *debugManager {
	return &debugManager{sessions: map[string]*debugSession{}}
}

type debugSession struct {
	eng    *debug.Engine
	done   chan debugRunResult
	out    *syncBuffer
	outLen int // bytes of output already reported
	ended  bool
}

type debugRunResult struct {
	result string
	err    error
}

// syncBuffer is a goroutine-safe buffer capturing the program's stdout/stderr.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) since(n int) (string, int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	s := b.buf.String()
	if n > len(s) {
		n = len(s)
	}
	return s[n:], len(s)
}

// Wire structures.
type startRequest struct {
	Source      string `json:"source"`
	Breakpoints []int  `json:"breakpoints"`
	StopOnEntry bool   `json:"stopOnEntry"`
}

type commandRequest struct {
	Session string `json:"session"`
	Command string `json:"command"`
}

type debugVariable struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

type debugFrame struct {
	Name   string `json:"name"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

type debugResponse struct {
	Session     string                 `json:"session,omitempty"`
	State       string                 `json:"state"` // "stopped" | "terminated" | "error"
	Reason      string                 `json:"reason,omitempty"`
	Line        int                    `json:"line,omitempty"`
	Column      int                    `json:"column,omitempty"`
	Frames      []debugFrame           `json:"frames,omitempty"`
	Locals      []debugVariable        `json:"locals,omitempty"`
	Output      string                 `json:"output,omitempty"`
	Result      string                 `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Diagnostics []gadbridge.Diagnostic `json:"diagnostics,omitempty"`
}

func (m *debugManager) handleStart(w http.ResponseWriter, r *http.Request) {
	var req startRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	builtins := gad.NewBuiltins()
	st := gad.NewSymbolTable(builtins.NameSet)
	_, bc, err := gad.Compile(st, []byte(req.Source), gad.CompileOptions{})
	if err != nil {
		writeJSON(w, debugResponse{State: "error", Diagnostics: gadbridge.Diagnose(req.Source)})
		return
	}

	eng := debug.New(req.StopOnEntry)
	eng.SetBreakpoints(req.Breakpoints)
	out := &syncBuffer{}
	vm := gad.NewVM(builtins.Build(), bc).SetRecover(true)
	vm.SetDebugger(eng)

	sess := &debugSession{eng: eng, done: make(chan debugRunResult, 1), out: out}
	go func() {
		ret, rerr := vm.RunOpts(&gad.RunOpts{StdOut: out, StdErr: out})
		res := ""
		if ret != nil && ret != gad.Nil {
			res = ret.ToString()
		}
		sess.done <- debugRunResult{res, rerr}
	}()

	id := newSessionID()
	m.mu.Lock()
	m.sessions[id] = sess
	m.mu.Unlock()

	resp := sess.waitNext()
	resp.Session = id
	if resp.State == "terminated" {
		m.remove(id)
	}
	writeJSON(w, resp)
}

func (m *debugManager) handleCommand(w http.ResponseWriter, r *http.Request) {
	var req commandRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	m.mu.Lock()
	sess := m.sessions[req.Session]
	m.mu.Unlock()
	if sess == nil {
		writeJSON(w, debugResponse{State: "error", Error: "unknown or finished session"})
		return
	}

	switch req.Command {
	case "continue":
		sess.eng.Continue()
	case "next", "stepOver":
		sess.eng.StepOver()
	case "stepIn", "step":
		sess.eng.StepInto()
	case "stepOut", "out":
		sess.eng.StepOut()
	case "pause":
		sess.eng.Pause()
	default:
		writeJSON(w, debugResponse{State: "error", Error: "unknown command " + req.Command})
		return
	}

	resp := sess.waitNext()
	resp.Session = req.Session
	if resp.State == "terminated" {
		m.remove(req.Session)
	}
	writeJSON(w, resp)
}

func (m *debugManager) remove(id string) {
	m.mu.Lock()
	delete(m.sessions, id)
	m.mu.Unlock()
}

// waitNext resumes the session until the next stop or program completion and
// builds the response (including the call stack, locals and new output).
func (s *debugSession) waitNext() debugResponse {
	select {
	case ev := <-s.eng.Stops():
		out, n := s.out.since(s.outLen)
		s.outLen = n
		return debugResponse{
			State:  "stopped",
			Reason: string(ev.Reason),
			Line:   ev.Line,
			Column: ev.Column,
			Frames: framesOf(s.eng),
			Locals: localsOf(s.eng),
			Output: out,
		}
	case r := <-s.done:
		s.ended = true
		out, n := s.out.since(s.outLen)
		s.outLen = n
		resp := debugResponse{State: "terminated", Output: out, Result: r.result}
		if r.err != nil {
			resp.Error = r.err.Error()
		}
		return resp
	}
}

func framesOf(eng *debug.Engine) []debugFrame {
	src := eng.Frames()
	out := make([]debugFrame, 0, len(src))
	// Innermost first.
	for i := len(src) - 1; i >= 0; i-- {
		f := src[i]
		out = append(out, debugFrame{Name: f.FuncName, Line: f.Line, Column: f.Column})
	}
	return out
}

func localsOf(eng *debug.Engine) []debugVariable {
	src := eng.Locals()
	out := make([]debugVariable, len(src))
	for i, v := range src {
		out[i] = debugVariable{Name: v.Name, Type: v.Type, Value: v.Value}
	}
	return out
}

func newSessionID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
