// Package ide implements the backend for `gad ide`: a small HTTP server that
// turns a workspace directory into a multi-file editing environment. On top of
// the shared language operations (format / diagnose / run / debug) it adds a
// sandboxed filesystem API rooted at the workspace and read/write access to the
// project's .gad.yaml (the `fmt` formatter settings and the `ide` layout key).
//
// The server is transport-agnostic: New returns a *Server whose Handler can be
// mounted by `gad ide` (serving the bundled React app) or used directly in
// tests.
package ide

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// Server serves the IDE API for a single workspace directory.
type Server struct {
	// Root is the absolute path of the workspace directory. All filesystem
	// operations are confined to this subtree.
	Root string
	// OpenFile, when set, is a workspace-relative path the UI should open first
	// (used when `gad ide FILE` targets a single file).
	OpenFile string
	// Static, when non-empty, is a directory with the built React app served at
	// the site root.
	Static string

	dbg *DebugManager
}

// New creates a Server for the given workspace path. If path points at a file,
// the workspace root becomes its directory and the file is reported as the
// initial file to open. A relative path is resolved against the process CWD.
func New(path string) (*Server, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	s := &Server{dbg: NewDebugManager()}
	if info.IsDir() {
		s.Root = abs
	} else {
		s.Root = filepath.Dir(abs)
		s.OpenFile = filepath.Base(abs)
	}
	return s, nil
}

// Handler returns the HTTP handler exposing the IDE API (and, if Static is set,
// the bundled web app).
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Workspace metadata.
	mux.HandleFunc("/api/ide/workspace", s.handleWorkspace)

	// Filesystem.
	mux.HandleFunc("/api/ide/tree", s.handleTree)
	mux.HandleFunc("/api/ide/file", s.handleFile)     // GET ?path / PUT
	mux.HandleFunc("/api/ide/mkdir", s.handleMkdir)   // POST
	mux.HandleFunc("/api/ide/delete", s.handleDelete) // POST
	mux.HandleFunc("/api/ide/rename", s.handleRename) // POST

	// Config (.gad.yaml).
	mux.HandleFunc("/api/ide/config", s.handleConfig) // GET / PUT

	// Language ops.
	mux.HandleFunc("/api/ide/format", s.handleFormat)     // POST
	mux.HandleFunc("/api/ide/diagnose", s.handleDiagnose) // POST
	mux.HandleFunc("/api/ide/run", s.handleRun)           // POST
	mux.HandleFunc("/api/ide/modules", s.handleModules)   // GET

	// Debug (shares the request/response protocol used by web/server).
	mux.HandleFunc("/api/ide/debug/start", postOnly(s.dbg.HandleStart))
	mux.HandleFunc("/api/ide/debug/command", postOnly(s.dbg.HandleCommand))

	if s.Static != "" {
		mux.Handle("/", spaFileServer(s.Static))
	}
	return withCORS(mux)
}

// --- HTTP helpers -----------------------------------------------------------

func (s *Server) handleWorkspace(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"root":     s.Root,
		"name":     filepath.Base(s.Root),
		"openFile": s.OpenFile,
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// decodeBody decodes a JSON request body (capped at 8 MiB) into v.
func decodeBody(r *http.Request, v any) error {
	return json.NewDecoder(io.LimitReader(r.Body, 8<<20)).Decode(v)
}

func postOnly(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		h(w, r)
	}
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// spaFileServer serves dir, falling back to index.html for client routes.
func spaFileServer(dir string) http.Handler {
	fileServer := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := filepath.Join(dir, filepath.Clean(r.URL.Path))
		if info, err := os.Stat(p); err != nil || info.IsDir() {
			if r.URL.Path != "/" {
				if _, err := os.Stat(filepath.Join(dir, "index.html")); err == nil {
					http.ServeFile(w, r, filepath.Join(dir, "index.html"))
					return
				}
			}
		}
		fileServer.ServeHTTP(w, r)
	})
}
