package gadbridge

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"sync"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/debug"
	"github.com/gad-lang/gad/gadx"
	"github.com/gad-lang/gad/parser"
)

// A self-contained, HTTP-free debug driver. It mirrors the request/response
// protocol web/ide serves over HTTP, but returns plain structs so any host — the
// WebAssembly bridge in particular — can drive stepping without a Go web server.
// It reuses the stable debug.Engine and the VM; the web/ide HTTP manager is left
// untouched.

// errDebugSessionNotFound is returned for an unknown or finished session.
var errDebugSessionNotFound = errors.New("unknown or finished session")

// DebugManager owns the live debug sessions.
type DebugManager struct {
	mu       sync.Mutex
	sessions map[string]*debugSession
	// ModuleMap, when set, resolves `import(...)` for debug sessions. Nil means no
	// importable modules (fine for self-contained scripts, and avoids pulling
	// OS/network stdlib into a WASM build).
	ModuleMap *gad.ModuleMap
}

// NewDebugManager returns an empty DebugManager.
func NewDebugManager() *DebugManager {
	return &DebugManager{sessions: map[string]*debugSession{}}
}

// DebugStartRequest launches a debug session. Path selects the dialect by
// extension (.gadx → gadx, .gadt → template, else plain Gad).
type DebugStartRequest struct {
	Source      string   `json:"source"`
	Path        string   `json:"path"`
	Breakpoints []int    `json:"breakpoints"`
	StopOnEntry bool     `json:"stopOnEntry"`
	Args        []string `json:"args"`
	// BreakpointSpecs, when present, take precedence over Breakpoints and carry
	// each breakpoint's disabled flag and optional condition expression.
	BreakpointSpecs []BreakpointSpec `json:"breakpointSpecs"`
}

// BreakpointSpec is a breakpoint with an optional disabled flag and a condition
// expression (evaluated in the paused frame; the breakpoint pauses when truthy).
type BreakpointSpec struct {
	Line      int    `json:"line"`
	Disabled  bool   `json:"disabled"`
	Condition string `json:"condition"`
}

// DebugVariable is a local variable observed at a stop.
type DebugVariable struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

// DebugFrame is one call-stack frame with its local variables.
type DebugFrame struct {
	Name   string          `json:"name"`
	File   string          `json:"file"`
	Line   int             `json:"line"`
	Column int             `json:"column"`
	Locals []DebugVariable `json:"locals"`
}

// DebugResponse is the result of a Start/Command call (JSON shape matches the
// app's DebugResponse in backends/debug.ts).
type DebugResponse struct {
	Session     string          `json:"session,omitempty"`
	State       string          `json:"state"` // "stopped" | "terminated" | "error"
	Reason      string          `json:"reason,omitempty"`
	File        string          `json:"file,omitempty"`
	Line        int             `json:"line,omitempty"`
	Column      int             `json:"column,omitempty"`
	Frames      []DebugFrame    `json:"frames,omitempty"`
	Locals      []DebugVariable `json:"locals,omitempty"`
	Output      string          `json:"output,omitempty"`
	Stdout      string          `json:"stdout,omitempty"`
	Stderr      string          `json:"stderr,omitempty"`
	Result      string          `json:"result,omitempty"`
	Error       string          `json:"error,omitempty"`
	Diagnostics []Diagnostic    `json:"diagnostics,omitempty"`
}

type debugSession struct {
	eng    *debug.Engine
	done   chan debugRunResult
	out    *syncBuffer
	outLen int
	err    *syncBuffer
	errLen int
}

type debugRunResult struct {
	result string
	err    error
}

// syncBuffer is a goroutine-safe buffer capturing stdout/stderr.
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

// Start compiles the source, starts a VM under the debugger and runs to the
// first stop (breakpoint / stop-on-entry) or to completion.
func (m *DebugManager) Start(req DebugStartRequest) DebugResponse {
	builtins := gadx.AppendBuiltins(gad.NewBuiltins())
	st := gad.NewSymbolTable(builtins.NameSet)

	opts := gad.CompileOptions{}
	if m.ModuleMap != nil {
		opts.CompilerOptions.ModuleMap = m.ModuleMap
	}
	switch {
	case strings.HasSuffix(req.Path, ".gadx"):
		// The .gadx ModuleFile selects gad's native Gadx front-end.
		opts.ModuleFile = req.Path
	case strings.HasSuffix(req.Path, ".gadt"):
		opts.ParserOptions.Mode |= parser.ParseMixed
		opts.ScannerOptions.Mode |= parser.ScanMixed | parser.ScanConfigDisabled
		opts.ScannerOptions.MixedDelimiter = parser.DefaultMixedDelimiter
	}

	cr, err := gad.Compile(st, []byte(req.Source), opts)
	if err != nil {
		return DebugResponse{State: "error", Diagnostics: Diagnose(req.Source)}
	}

	eng := debug.New(req.StopOnEntry)
	if len(req.BreakpointSpecs) > 0 {
		bps := make([]debug.Breakpoint, len(req.BreakpointSpecs))
		for i, s := range req.BreakpointSpecs {
			bps[i] = debug.Breakpoint{Line: s.Line, Disabled: s.Disabled, Condition: s.Condition}
		}
		eng.SetConditionalBreakpoints(bps)
	} else {
		eng.SetBreakpoints(req.Breakpoints)
	}

	out := &syncBuffer{}
	errBuf := &syncBuffer{}
	for _, w := range cr.Warnings {
		errBuf.Write([]byte(w.Error() + "\n"))
	}

	vm := gad.NewVM(builtins.Build(), cr.Bytecode).SetRecover(true)
	vm.SetDebugger(eng)

	// Parse args into positional + named values for the script's `param (…)`
	// (typed coercion included), the same as a plain run.
	runOpts := &gad.RunOpts{StdOut: out, StdErr: errBuf}
	if len(req.Args) > 0 {
		runOpts.Args, runOpts.NamedArgs = gad.ParseArgsToRunOpts(req.Args)
	}

	sess := &debugSession{eng: eng, done: make(chan debugRunResult, 1), out: out, err: errBuf}
	go func() {
		ret, rerr := vm.RunOpts(runOpts)
		res := ""
		if ret != nil && ret != gad.Nil {
			res = ret.ToString()
		}
		sess.done <- debugRunResult{res, rerr}
	}()

	id := newDebugSessionID()
	m.mu.Lock()
	m.sessions[id] = sess
	m.mu.Unlock()

	resp := sess.waitNext()
	resp.Session = id
	if resp.State == "terminated" {
		m.remove(id)
	}
	return resp
}

// Command resumes a session (continue/next/stepIn/stepOut/pause) to the next
// stop or completion.
func (m *DebugManager) Command(session, command string) DebugResponse {
	m.mu.Lock()
	sess := m.sessions[session]
	m.mu.Unlock()
	if sess == nil {
		return DebugResponse{State: "error", Error: errDebugSessionNotFound.Error()}
	}
	switch command {
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
		return DebugResponse{State: "error", Error: "unknown command " + command}
	}
	resp := sess.waitNext()
	resp.Session = session
	if resp.State == "terminated" {
		m.remove(session)
	}
	return resp
}

// Eval evaluates an expression against a paused session's current frame.
func (m *DebugManager) Eval(session, expr string, repr bool) (value string, err error) {
	m.mu.Lock()
	sess := m.sessions[session]
	m.mu.Unlock()
	if sess == nil {
		return "", errDebugSessionNotFound
	}
	return sess.eng.EvalInFrame(expr, repr)
}

// Inspect evaluates expr against a paused session's current frame and returns
// its tree-navigator description (type, value and, for containers, its immediate
// children with Gad accessors). Valid only while the session is parked at a stop.
func (m *DebugManager) Inspect(session, expr string) (InspectResult, error) {
	m.mu.Lock()
	sess := m.sessions[session]
	m.mu.Unlock()
	if sess == nil {
		return InspectResult{}, errDebugSessionNotFound
	}
	obj, err := sess.eng.EvalObject(expr)
	if err != nil {
		return InspectResult{}, err
	}
	return InspectObject(nil, obj), nil
}

// Stop discards a session (the VM goroutine is left to unwind on the next
// resume; callers that abort a runaway program should terminate the host, e.g.
// a Web Worker).
func (m *DebugManager) Stop(session string) {
	m.remove(session)
}

func (m *DebugManager) remove(id string) {
	m.mu.Lock()
	delete(m.sessions, id)
	m.mu.Unlock()
}

func (s *debugSession) drainOutput() (stdout, stderr string) {
	stdout, s.outLen = s.out.since(s.outLen)
	stderr, s.errLen = s.err.since(s.errLen)
	return stdout, stderr
}

// waitNext resumes until the next stop or completion and builds the response.
func (s *debugSession) waitNext() DebugResponse {
	select {
	case ev := <-s.eng.Stops():
		stdout, stderr := s.drainOutput()
		return DebugResponse{
			State:  "stopped",
			Reason: string(ev.Reason),
			File:   ev.File,
			Line:   ev.Line,
			Column: ev.Column,
			Frames: debugFramesOf(s.eng),
			Locals: debugLocalsOf(s.eng),
			Output: stdout + stderr,
			Stdout: stdout,
			Stderr: stderr,
		}
	case r := <-s.done:
		stdout, stderr := s.drainOutput()
		if r.err != nil && stderr == "" {
			stderr = r.err.Error()
		}
		resp := DebugResponse{
			State: "terminated", Output: stdout + stderr,
			Stdout: stdout, Stderr: stderr, Result: r.result,
		}
		if r.err != nil {
			resp.Error = r.err.Error()
		}
		return resp
	}
}

func debugFramesOf(eng *debug.Engine) []DebugFrame {
	src := eng.Frames()
	out := make([]DebugFrame, 0, len(src))
	for i := len(src) - 1; i >= 0; i-- { // innermost first
		f := src[i]
		locals := make([]DebugVariable, len(f.Locals))
		for j, v := range f.Locals {
			locals[j] = DebugVariable{Name: v.Name, Type: v.Type, Value: v.Value}
		}
		out = append(out, DebugFrame{Name: f.FuncName, File: f.File, Line: f.Line, Column: f.Column, Locals: locals})
	}
	return out
}

func debugLocalsOf(eng *debug.Engine) []DebugVariable {
	src := eng.Locals()
	out := make([]DebugVariable, len(src))
	for i, v := range src {
		out[i] = DebugVariable{Name: v.Name, Type: v.Type, Value: v.Value}
	}
	return out
}

func newDebugSessionID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
