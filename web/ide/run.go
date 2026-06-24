package ide

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/importers"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/stdlib/helper"
	"github.com/gad-lang/gad/web/gadbridge"
)

// builtinModules lists the importable stdlib modules the IDE run/debug dialog
// can toggle. unsafe marks modules disabled when the workspace runs in safe
// mode (filesystem / network access).
var builtinModules = []moduleInfo{
	{Name: "time"}, {Name: "strings"}, {Name: "fmt"}, {Name: "json"},
	{Name: "path"}, {Name: "encoding/base64"}, {Name: "compress/flate"},
	{Name: "http", Unsafe: true}, {Name: "os", Unsafe: true},
	{Name: "filepath", Unsafe: true},
}

type moduleInfo struct {
	Name   string `json:"name"`
	Unsafe bool   `json:"unsafe"`
}

// buildModuleMap creates a module map with the stdlib builtin modules (honouring
// safe mode and per-module disables) and a file importer rooted at workdir so
// relative imports resolve. Shared by run and debug.
func buildModuleMap(workdir string, disabled []string, safe bool) *gad.ModuleMap {
	mb := helper.NewModuleMapBuilder()
	mb.Safe = safe
	mb.Disabled = make(map[string]bool, len(disabled))
	for _, n := range disabled {
		mb.Disabled[n] = true
	}
	mm := mb.Build()
	// helper only honours Disabled for the unsafe modules; remove any other
	// requested module from the built map so every toggle takes effect.
	for _, n := range disabled {
		mm.Remove(n)
	}
	mm.SetExtImporter(&importers.FileImporter{
		WorkDir:    workdir,
		FileReader: importers.ShebangReadFile,
	})
	return mm
}

// handleModules lists the toggleable builtin modules.
func (s *Server) handleModules(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, builtinModules)
}

// evalRequest evaluates a single expression for the Evaluate panel. Source, when
// set, is prepended as context (the file's top-level definitions); the result is
// rendered with repr() when Repr is set, otherwise str().
type evalRequest struct {
	Expr     string   `json:"expr"`
	Repr     bool     `json:"repr"`
	Source   string   `json:"source"`   // optional prelude (file context)
	Path     string   `json:"path"`     // for relative imports
	Disabled []string `json:"disabled"` // builtin modules to disable
	Safe     bool     `json:"safe"`
}

// evalResult is the outcome of evaluating an expression.
type evalResult struct {
	OK     bool   `json:"ok"`
	Value  string `json:"value"`
	Error  string `json:"error"`
	Stdout string `json:"stdout"` // output produced while evaluating (prelude side effects)
}

// handleEval evaluates an expression in a fresh VM (optionally seeded with the
// file as a prelude) and returns its str()/repr() rendering. It does not run in a
// paused debug frame — the panel re-requests on step to refresh.
func (s *Server) handleEval(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req evalRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(req.Expr) == "" {
		writeJSON(w, evalResult{OK: false, Error: "empty expression"})
		return
	}

	render := "str"
	if req.Repr {
		render = "repr"
	}
	// Build the evaluation script: the file context's top-level returns are
	// stripped so its definitions are in scope but its return value does not mask
	// the expression result.
	script := gadbridge.EvalSource(req.Source, req.Expr, render)

	workdir := s.Root
	if req.Path != "" {
		if abs, err := s.resolve(req.Path); err == nil {
			workdir = filepath.Dir(abs)
		}
	}

	res := s.run(script, workdir, runRequest{Disabled: req.Disabled, Safe: req.Safe})
	out := evalResult{OK: res.OK, Stdout: res.Stdout}
	if res.OK {
		out.Value = s.relativizeValue(res.Result)
	} else {
		out.Error = res.Stderr
		if out.Error == "" && len(res.Diagnostics) > 0 {
			d := res.Diagnostics[0]
			out.Error = fmt.Sprintf("%d:%d %s", d.Line, d.Column, d.Message)
		}
	}
	writeJSON(w, out)
}

// formatRequest carries source for format/diagnose.
type formatRequest struct {
	Source string `json:"source"`
}

func (s *Server) handleFormat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req formatRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	writeJSON(w, gadbridge.Format(req.Source))
}

// transpileRequest carries source to transpile. Mixed selects template
// (`.gadt`) parsing; the override fields fall back to the workspace `transpile`
// config and then the built-in defaults.
type transpileRequest struct {
	Source          string `json:"source"`
	Path            string `json:"path"` // used only to infer mixed mode from .gadt
	Mixed           bool   `json:"mixed"`
	RawStrFuncStart string `json:"rawStrFuncStart"`
	RawStrFuncEnd   string `json:"rawStrFuncEnd"`
	WriteFunc       string `json:"writeFunc"`
}

// transpileOptions builds the effective node.TranspileOptions from the request,
// the workspace `transpile` config key and the built-in defaults (in that order
// of precedence).
func (s *Server) transpileOptions(req transpileRequest) *node.TranspileOptions {
	opts := gad.TranspileOptions() // defaults

	// Workspace config: transpile.{rawStrFuncStart,rawStrFuncEnd,writeFunc}.
	if doc, err := readConfig(filepath.Join(s.Root, configFile)); err == nil {
		if cfg, ok := doc["transpile"].(map[string]any); ok {
			if v, ok := cfg["rawStrFuncStart"].(string); ok {
				opts.RawStrFuncStart = v
			}
			if v, ok := cfg["rawStrFuncEnd"].(string); ok {
				opts.RawStrFuncEnd = v
			}
			if v, ok := cfg["writeFunc"].(string); ok {
				opts.WriteFunc = v
			}
		}
	}

	// Per-request overrides win.
	if req.RawStrFuncStart != "" {
		opts.RawStrFuncStart = req.RawStrFuncStart
	}
	if req.RawStrFuncEnd != "" {
		opts.RawStrFuncEnd = req.RawStrFuncEnd
	}
	if req.WriteFunc != "" {
		opts.WriteFunc = req.WriteFunc
	}
	return opts
}

// handleTranspile transpiles a Gad/template source to plain Gad write(...) calls.
func (s *Server) handleTranspile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req transpileRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	mixed := req.Mixed || strings.HasSuffix(req.Path, ".gadt")
	writeJSON(w, gadbridge.Transpile(req.Source, mixed, s.transpileOptions(req)))
}

// handleDoc returns the doc comments (`/?`, `/??`, `/???`) of a source, for the
// doc-comment side panel.
// inspectRequest inspects the value of an expression for the tree navigator.
// When Session is set the expression is evaluated in the paused debug frame;
// otherwise it runs standalone with Source as a prelude.
type inspectRequest struct {
	Session  string   `json:"session"`
	Expr     string   `json:"expr"`
	Source   string   `json:"source"`
	Path     string   `json:"path"`
	Disabled []string `json:"disabled"`
	Safe     bool     `json:"safe"`
}

// handleInspect evaluates Expr and returns its tree-navigator description (type,
// value and, for containers, its immediate children with Gad accessors).
func (s *Server) handleInspect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req inspectRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(req.Expr) == "" {
		writeJSON(w, map[string]any{"ok": false, "error": "empty expression"})
		return
	}

	var (
		obj gad.Object
		err error
	)
	if req.Session != "" {
		obj, err = s.dbg.evalObject(req.Session, req.Expr)
	} else {
		workdir := s.Root
		if req.Path != "" {
			if abs, e := s.resolve(req.Path); e == nil {
				workdir = filepath.Dir(abs)
			}
		}
		obj, err = s.evalObject(req.Source, req.Expr, workdir, runRequest{Disabled: req.Disabled, Safe: req.Safe})
	}
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	insp := gadbridge.InspectObject(nil, obj)
	// Show workspace-relative paths in module/function renderings.
	insp.Value = s.relativizeValue(insp.Value)
	for i := range insp.Entries {
		insp.Entries[i].Value = s.relativizeValue(insp.Entries[i].Value)
	}
	writeJSON(w, map[string]any{"ok": true, "inspect": insp})
}

// evalObject compiles `<prelude>\nreturn (expr)` (prelude returns stripped) and
// runs it, returning the resulting object for inspection.
func (s *Server) evalObject(source, expr, workdir string, req runRequest) (gad.Object, error) {
	script := gadbridge.EvalSource(source, expr, "")
	builtins := gad.NewBuiltins()
	st := gad.NewSymbolTable(builtins.NameSet)
	mm := buildModuleMap(workdir, req.Disabled, req.Safe)
	_, bc, err := gad.Compile(st, []byte(script), gad.CompileOptions{
		CompilerOptions: gad.CompilerOptions{ModuleMap: mm},
	})
	if err != nil {
		return nil, err
	}
	var stdout, stderr bytes.Buffer
	return gad.NewVM(builtins.Build(), bc).SetRecover(true).RunOpts(&gad.RunOpts{
		StdOut: &stdout, StdErr: &stderr,
	})
}

func (s *Server) handleDoc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req formatRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	writeJSON(w, map[string]any{"docs": gadbridge.DocComments(req.Source)})
}

func (s *Server) handleDiagnose(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req formatRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	writeJSON(w, map[string]any{"diagnostics": gadbridge.Diagnose(req.Source)})
}

// runRequest configures a run. Source defaults to the named file's content when
// empty so the UI can run a saved file directly.
type runRequest struct {
	Path     string   `json:"path"`     // workspace-relative, for imports + saved source
	Source   string   `json:"source"`   // overrides on-disk content when set
	Args     []string `json:"args"`     // CLI-style positional arguments
	Disabled []string `json:"disabled"` // builtin modules to disable
	Safe     bool     `json:"safe"`     // disable all unsafe modules
	// Output capture. SaveStdout / SaveStderr name workspace-relative files for
	// each stream; when Combine is set both streams are written (interleaved as
	// stdout then stderr) to SaveStdout only. SaveOut is the legacy combined
	// field, kept for compatibility (treated as SaveStdout + Combine).
	SaveStdout string `json:"saveStdout"`
	SaveStderr string `json:"saveStderr"`
	Combine    bool   `json:"combine"`
	SaveOut    string `json:"saveOut"`
}

// saveOutputs persists a run's stdout/stderr to the requested workspace files.
func (s *Server) saveOutputs(req runRequest, res gadbridge.RunResult) {
	stdoutFile, stderrFile, combine := req.SaveStdout, req.SaveStderr, req.Combine
	if req.SaveOut != "" && stdoutFile == "" {
		stdoutFile, combine = req.SaveOut, true // legacy combined field
	}
	write := func(rel, content string) {
		if rel == "" {
			return
		}
		if abs, err := s.resolve(rel); err == nil {
			_ = os.MkdirAll(filepath.Dir(abs), 0o755)
			_ = os.WriteFile(abs, []byte(content), 0o644)
		}
	}
	if combine {
		write(stdoutFile, res.Stdout+res.Stderr)
		return
	}
	write(stdoutFile, res.Stdout)
	write(stderrFile, res.Stderr)
}

// handleRun compiles and runs source with the requested module map and
// arguments, optionally persisting the combined output to a workspace file.
func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req runRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	src := req.Source
	workdir := s.Root
	if req.Path != "" {
		abs, err := s.resolve(req.Path)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		workdir = filepath.Dir(abs)
		if src == "" {
			data, err := os.ReadFile(abs)
			if err != nil {
				writeError(w, statusForFS(err), err.Error())
				return
			}
			src = string(data)
		}
	}

	res := s.run(src, workdir, req)
	s.saveOutputs(req, res)
	writeJSON(w, res)
}

// run compiles and executes src, mirroring gadbridge.RunResult but honouring the
// IDE's module map, arguments and safe-mode toggles.
func (s *Server) run(src, workdir string, req runRequest) gadbridge.RunResult {
	builtins := gad.NewBuiltins()
	st := gad.NewSymbolTable(builtins.NameSet)

	mm := buildModuleMap(workdir, req.Disabled, req.Safe)

	_, bc, err := gad.Compile(st, []byte(src), gad.CompileOptions{
		CompilerOptions: gad.CompilerOptions{ModuleMap: mm},
	})
	if err != nil {
		return gadbridge.RunResult{OK: false, Diagnostics: gadbridge.ErrorDiagnostics(err)}
	}

	args := gad.Args{}
	if len(req.Args) > 0 {
		arr := make(gad.Array, len(req.Args))
		for i, a := range req.Args {
			arr[i] = gad.Str(a)
		}
		args = append(args, arr)
	}

	var stdout, stderr bytes.Buffer
	ret, runErr := gad.NewVM(builtins.Build(), bc).SetRecover(true).RunOpts(&gad.RunOpts{
		Args:   args,
		StdOut: &stdout,
		StdErr: &stderr,
	})
	res := gadbridge.RunResult{OK: runErr == nil, Stdout: stdout.String(), Stderr: stderr.String()}
	if runErr != nil {
		res.Diagnostics = gadbridge.ErrorDiagnostics(runErr)
		if res.Stderr == "" {
			res.Stderr = runErr.Error()
		}
		return res
	}
	if ret != nil && ret != gad.Nil {
		res.Result = ret.ToString()
	}
	return res
}
