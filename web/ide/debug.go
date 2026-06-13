package ide

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
	"github.com/gad-lang/gad/stdlib/helper"
	"github.com/gad-lang/gad/web/gadbridge"
)

// A request/response debugging protocol that the web "Run & Debug" page (and the
// IDE) drive: HandleStart launches a session and runs to the first stop (or
// end); HandleCommand resumes (continue/step) to the next stop (or end). Each
// response carries the stop event, the call stack, locals and any new output.
// It is exported so both web/server and web/ide share one implementation.

// DebugManager owns the live debug sessions.
type DebugManager struct {
	mu       sync.Mutex
	sessions map[string]*debugSession
	// BuildModuleMap, when set, builds the module map for a debug session from
	// its start request (e.g. rooted at the file's directory with the requested
	// module toggles). When nil, a default stdlib module map is used so builtin
	// imports such as `import("time")` still resolve.
	BuildModuleMap func(req StartRequest) *gad.ModuleMap
}

// NewDebugManager returns an empty DebugManager.
func NewDebugManager() *DebugManager {
	return &DebugManager{sessions: map[string]*debugSession{}}
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

// StartRequest launches a debug session.
type StartRequest struct {
	Source      string   `json:"source"`
	Breakpoints []int    `json:"breakpoints"`
	StopOnEntry bool     `json:"stopOnEntry"`
	Path        string   `json:"path"`     // workspace-relative file, for imports
	Args        []string `json:"args"`     // CLI-style positional arguments
	Disabled    []string `json:"disabled"` // builtin modules to disable
	Safe        bool     `json:"safe"`     // disable all unsafe modules
}

// CommandRequest resumes a session (continue/next/stepIn/stepOut/pause).
type CommandRequest struct {
	Session string `json:"session"`
	Command string `json:"command"`
}

// DebugVariable is a local variable observed at a stop.
type DebugVariable struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

// DebugFrame is one call-stack frame, including its own local variables.
type DebugFrame struct {
	Name   string          `json:"name"`
	File   string          `json:"file"`
	Line   int             `json:"line"`
	Column int             `json:"column"`
	Locals []DebugVariable `json:"locals"`
}

// DebugResponse is the result of a start/command call.
type DebugResponse struct {
	Session     string                 `json:"session,omitempty"`
	State       string                 `json:"state"` // "stopped" | "terminated" | "error"
	Reason      string                 `json:"reason,omitempty"`
	Line        int                    `json:"line,omitempty"`
	Column      int                    `json:"column,omitempty"`
	Frames      []DebugFrame           `json:"frames,omitempty"`
	Locals      []DebugVariable        `json:"locals,omitempty"`
	Output      string                 `json:"output,omitempty"`
	Result      string                 `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Diagnostics []gadbridge.Diagnostic `json:"diagnostics,omitempty"`
}

// HandleStart compiles the source, starts a VM under the debugger and runs to
// the first stop (breakpoint, stop-on-entry) or to completion.
func (m *DebugManager) HandleStart(w http.ResponseWriter, r *http.Request) {
	var req StartRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	builtins := gad.NewBuiltins()
	st := gad.NewSymbolTable(builtins.NameSet)

	var mm *gad.ModuleMap
	if m.BuildModuleMap != nil {
		mm = m.BuildModuleMap(req)
	} else {
		// Default: builtin stdlib modules so imports like time/strings resolve.
		mm = helper.NewModuleMapBuilder().Build()
	}
	_, bc, err := gad.Compile(st, []byte(req.Source), gad.CompileOptions{
		CompilerOptions: gad.CompilerOptions{ModuleMap: mm},
	})
	if err != nil {
		writeJSON(w, DebugResponse{State: "error", Diagnostics: gadbridge.Diagnose(req.Source)})
		return
	}

	eng := debug.New(req.StopOnEntry)
	eng.SetBreakpoints(req.Breakpoints)
	out := &syncBuffer{}
	vm := gad.NewVM(builtins.Build(), bc).SetRecover(true)
	vm.SetDebugger(eng)

	args := gad.Args{}
	if len(req.Args) > 0 {
		arr := make(gad.Array, len(req.Args))
		for i, a := range req.Args {
			arr[i] = gad.Str(a)
		}
		args = append(args, arr)
	}

	sess := &debugSession{eng: eng, done: make(chan debugRunResult, 1), out: out}
	go func() {
		ret, rerr := vm.RunOpts(&gad.RunOpts{Args: args, StdOut: out, StdErr: out})
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

// HandleCommand resumes an existing session to the next stop or completion.
func (m *DebugManager) HandleCommand(w http.ResponseWriter, r *http.Request) {
	var req CommandRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	m.mu.Lock()
	sess := m.sessions[req.Session]
	m.mu.Unlock()
	if sess == nil {
		writeJSON(w, DebugResponse{State: "error", Error: "unknown or finished session"})
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
		writeJSON(w, DebugResponse{State: "error", Error: "unknown command " + req.Command})
		return
	}

	resp := sess.waitNext()
	resp.Session = req.Session
	if resp.State == "terminated" {
		m.remove(req.Session)
	}
	writeJSON(w, resp)
}

func (m *DebugManager) remove(id string) {
	m.mu.Lock()
	delete(m.sessions, id)
	m.mu.Unlock()
}

// waitNext resumes the session until the next stop or program completion and
// builds the response (including the call stack, locals and new output).
func (s *debugSession) waitNext() DebugResponse {
	select {
	case ev := <-s.eng.Stops():
		out, n := s.out.since(s.outLen)
		s.outLen = n
		return DebugResponse{
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
		resp := DebugResponse{State: "terminated", Output: out, Result: r.result}
		if r.err != nil {
			resp.Error = r.err.Error()
		}
		return resp
	}
}

func framesOf(eng *debug.Engine) []DebugFrame {
	src := eng.Frames()
	out := make([]DebugFrame, 0, len(src))
	// Innermost first.
	for i := len(src) - 1; i >= 0; i-- {
		f := src[i]
		locals := make([]DebugVariable, len(f.Locals))
		for j, v := range f.Locals {
			locals[j] = DebugVariable{Name: v.Name, Type: v.Type, Value: v.Value}
		}
		out = append(out, DebugFrame{Name: f.FuncName, File: f.File, Line: f.Line, Column: f.Column, Locals: locals})
	}
	return out
}

func localsOf(eng *debug.Engine) []DebugVariable {
	src := eng.Locals()
	out := make([]DebugVariable, len(src))
	for i, v := range src {
		out[i] = DebugVariable{Name: v.Name, Type: v.Type, Value: v.Value}
	}
	return out
}

func newSessionID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
