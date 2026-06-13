package ide

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/importers"
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

// handleModules lists the toggleable builtin modules.
func (s *Server) handleModules(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, builtinModules)
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
	SaveOut  string   `json:"saveOut"`  // workspace-relative file for stdout+stderr
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

	if req.SaveOut != "" {
		if abs, err := s.resolve(req.SaveOut); err == nil {
			_ = os.MkdirAll(filepath.Dir(abs), 0o755)
			_ = os.WriteFile(abs, []byte(res.Stdout+res.Stderr), 0o644)
		}
	}
	writeJSON(w, res)
}

// run compiles and executes src, mirroring gadbridge.RunResult but honouring the
// IDE's module map, arguments and safe-mode toggles.
func (s *Server) run(src, workdir string, req runRequest) gadbridge.RunResult {
	builtins := gad.NewBuiltins()
	st := gad.NewSymbolTable(builtins.NameSet)

	mb := helper.NewModuleMapBuilder()
	mb.Safe = req.Safe
	mb.Disabled = make(map[string]bool, len(req.Disabled))
	for _, n := range req.Disabled {
		mb.Disabled[n] = true
	}
	mm := mb.Build()
	// helper only honours Disabled for the unsafe modules; remove any other
	// requested module from the built map so every toggle takes effect.
	for _, n := range req.Disabled {
		mm.Remove(n)
	}
	mm.SetExtImporter(&importers.FileImporter{
		WorkDir:    workdir,
		FileReader: importers.ShebangReadFile,
	})

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
