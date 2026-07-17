package ide

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/debug"
	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/stdlib/helper"
	"github.com/gad-lang/gad/web/gadbridge"
)

// errSessionNotFound is returned when a debug session id is unknown or finished.
var errSessionNotFound = errors.New("unknown or finished session")

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
	// NormalizeFile, when set, maps a debugger source file name (the main file's
	// "(main)" sentinel, or a "file:<abs>" imported-module path) to a
	// workspace-relative path the UI can open. mainPath is the session's entry
	// file. When nil, file names are passed through unchanged.
	NormalizeFile func(mainPath, engineFile string) string
	// RelativizeValue, when set, rewrites absolute workspace paths embedded in a
	// rendered variable value (module / function ToString) to a relative form.
	RelativizeValue func(value string) string
	// TemplateDelimiter, when set, supplies the mixed-mode start/end delimiter
	// used to compile `.gadt` template sessions (from the workspace config). When
	// nil the parser default (`{%` / `%}`) is used.
	TemplateDelimiter func() parser.MixedDelimiter
}

// NewDebugManager returns an empty DebugManager.
func NewDebugManager() *DebugManager {
	return &DebugManager{sessions: map[string]*debugSession{}}
}

type debugSession struct {
	eng    *debug.Engine
	done   chan debugRunResult
	out    *syncBuffer // stdout
	outLen int         // stdout bytes already reported
	err    *syncBuffer // stderr
	errLen int         // stderr bytes already reported
	ended  bool
	// normalize maps a debugger file name to a workspace-relative path; identity
	// when the manager has no NormalizeFile hook.
	normalize func(engineFile string) string
	// relativize rewrites absolute workspace paths in a rendered value; identity
	// when the manager has no RelativizeValue hook.
	relativize func(value string) string
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

// BreakpointSpec is a breakpoint with optional disabled flag and condition,
// sent by the IDE. A bare line in StartRequest.Breakpoints is an enabled,
// unconditional breakpoint; BreakpointSpecs (when present) take precedence.
type BreakpointSpec struct {
	Line      int    `json:"line"`
	Disabled  bool   `json:"disabled"`
	Condition string `json:"condition"`
}

// StartRequest launches a debug session.
type StartRequest struct {
	Source          string           `json:"source"`
	Breakpoints     []int            `json:"breakpoints"`
	BreakpointSpecs []BreakpointSpec `json:"breakpointSpecs"`
	StopOnEntry     bool             `json:"stopOnEntry"`
	Path            string           `json:"path"`     // workspace-relative file, for imports
	Args            []string         `json:"args"`     // CLI-style positional arguments
	Disabled        []string         `json:"disabled"` // builtin modules to disable
	Safe            bool             `json:"safe"`     // disable all unsafe modules
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
	File        string                 `json:"file,omitempty"` // workspace-relative file of the stop
	Line        int                    `json:"line,omitempty"`
	Column      int                    `json:"column,omitempty"`
	Frames      []DebugFrame           `json:"frames,omitempty"`
	Locals      []DebugVariable        `json:"locals,omitempty"`
	Output      string                 `json:"output,omitempty"` // combined stdout+stderr delta (compat)
	Stdout      string                 `json:"stdout,omitempty"` // stdout delta since the last stop
	Stderr      string                 `json:"stderr,omitempty"` // stderr delta since the last stop
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

	builtins := newBuiltins(req.Path)
	st := gad.NewSymbolTable(builtins.NameSet)

	var mm *gad.ModuleMap
	if m.BuildModuleMap != nil {
		mm = m.BuildModuleMap(req)
	} else {
		// Default: builtin stdlib modules so imports like time/strings resolve.
		mm = helper.NewModuleMapBuilder().Build()
	}
	opts := gad.CompileOptions{
		CompilerOptions: gad.CompilerOptions{ModuleMap: mm},
	}
	// `.gadt` files are templates: compile in mixed (template) mode so debugging
	// a template steps through its transpiled Gad rather than failing to compile
	// the literal template text (mirrors run and cmd/gad). Delimiters come from
	// the workspace config via TemplateDelimiter, defaulting to `{%` / `%}`.
	if strings.HasSuffix(req.Path, ".gadt") {
		opts.ParserOptions.Mode |= parser.ParseMixed
		opts.ScannerOptions.Mode |= parser.ScanMixed | parser.ScanConfigDisabled
		delim := parser.DefaultMixedDelimiter
		if m.TemplateDelimiter != nil {
			delim = m.TemplateDelimiter()
		}
		opts.ScannerOptions.MixedDelimiter = delim
	}
	bc, err := compileFor(st, []byte(req.Source), req.Path, opts)
	if err != nil {
		writeJSON(w, DebugResponse{State: "error", Diagnostics: gadbridge.Diagnose(req.Source)})
		return
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

	normalize := func(f string) string { return f }
	if m.NormalizeFile != nil {
		normalize = func(f string) string { return m.NormalizeFile(req.Path, f) }
	}
	relativize := func(v string) string { return v }
	if m.RelativizeValue != nil {
		relativize = m.RelativizeValue
	}
	sess := &debugSession{
		eng: eng, done: make(chan debugRunResult, 1), out: out, err: errBuf,
		normalize: normalize, relativize: relativize,
	}
	go func() {
		ret, rerr := vm.RunOpts(&gad.RunOpts{Args: args, StdOut: out, StdErr: errBuf})
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

// EvalRequest evaluates an expression in a paused session's current frame.
type EvalRequest struct {
	Session string `json:"session"`
	Expr    string `json:"expr"`
	Repr    bool   `json:"repr"`
}

// HandleEval evaluates an expression against the paused frame's locals, so the
// Evaluate panel reflects the live debug state (not a fresh standalone VM).
func (m *DebugManager) HandleEval(w http.ResponseWriter, r *http.Request) {
	var req EvalRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	m.mu.Lock()
	sess := m.sessions[req.Session]
	m.mu.Unlock()
	if sess == nil {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown or finished session"})
		return
	}
	value, err := sess.eng.EvalInFrame(req.Expr, req.Repr)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "value": sess.relativize(value)})
}

// evalObject evaluates expr in a paused session's current frame and returns the
// resulting object (for the inspect/tree-navigator endpoint).
func (m *DebugManager) evalObject(session, expr string) (gad.Object, error) {
	m.mu.Lock()
	sess := m.sessions[session]
	m.mu.Unlock()
	if sess == nil {
		return nil, errSessionNotFound
	}
	return sess.eng.EvalObject(expr)
}

func (m *DebugManager) remove(id string) {
	m.mu.Lock()
	delete(m.sessions, id)
	m.mu.Unlock()
}

// drainOutput returns the new stdout and stderr produced since the last report.
func (s *debugSession) drainOutput() (stdout, stderr string) {
	stdout, s.outLen = s.out.since(s.outLen)
	stderr, s.errLen = s.err.since(s.errLen)
	return stdout, stderr
}

// waitNext resumes the session until the next stop or program completion and
// builds the response (including the call stack, locals and new output).
func (s *debugSession) waitNext() DebugResponse {
	select {
	case ev := <-s.eng.Stops():
		stdout, stderr := s.drainOutput()
		return DebugResponse{
			State:  "stopped",
			Reason: string(ev.Reason),
			File:   s.normalize(ev.File),
			Line:   ev.Line,
			Column: ev.Column,
			Frames: framesOf(s.eng, s.normalize, s.relativize),
			Locals: localsOf(s.eng, s.relativize),
			Output: stdout + stderr,
			Stdout: stdout,
			Stderr: stderr,
		}
	case r := <-s.done:
		s.ended = true
		stdout, stderr := s.drainOutput()
		// An uncaught error is returned (not written to the stderr buffer), so
		// surface it on the stderr stream too — like the run path.
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

func framesOf(eng *debug.Engine, normalize, relativize func(string) string) []DebugFrame {
	src := eng.Frames()
	out := make([]DebugFrame, 0, len(src))
	// Innermost first.
	for i := len(src) - 1; i >= 0; i-- {
		f := src[i]
		locals := make([]DebugVariable, len(f.Locals))
		for j, v := range f.Locals {
			locals[j] = DebugVariable{Name: v.Name, Type: v.Type, Value: relativize(v.Value)}
		}
		out = append(out, DebugFrame{Name: relativize(f.FuncName), File: normalize(f.File), Line: f.Line, Column: f.Column, Locals: locals})
	}
	return out
}

func localsOf(eng *debug.Engine, relativize func(string) string) []DebugVariable {
	src := eng.Locals()
	out := make([]DebugVariable, len(src))
	for i, v := range src {
		out[i] = DebugVariable{Name: v.Name, Type: v.Type, Value: relativize(v.Value)}
	}
	return out
}

func newSessionID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
