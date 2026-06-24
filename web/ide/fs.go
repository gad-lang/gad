package ide

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// errOutsideRoot is returned when a request path escapes the workspace.
var errOutsideRoot = errors.New("path escapes the workspace root")

// resolve maps a workspace-relative path to an absolute path inside Root,
// rejecting traversal outside the workspace. An empty path resolves to Root.
func (s *Server) resolve(rel string) (string, error) {
	clean := filepath.Clean("/" + filepath.ToSlash(rel)) // force absolute, strip ..
	abs := filepath.Join(s.Root, filepath.FromSlash(clean))
	if abs != s.Root && !strings.HasPrefix(abs, s.Root+string(os.PathSeparator)) {
		return "", errOutsideRoot
	}
	return abs, nil
}

// rel returns the workspace-relative, slash-separated form of abs.
func (s *Server) rel(abs string) string {
	r, err := filepath.Rel(s.Root, abs)
	if err != nil {
		return abs
	}
	return filepath.ToSlash(r)
}

// treeNode is one entry in the workspace file tree.
type treeNode struct {
	Name     string      `json:"name"`
	Path     string      `json:"path"`
	Dir      bool        `json:"dir"`
	Children []*treeNode `json:"children,omitempty"`
}

// handleTree returns the workspace file tree. Build/vendor directories are
// always skipped; dot-prefixed (hidden) entries are skipped unless the request
// passes ?hidden=true.
func (s *Server) handleTree(w http.ResponseWriter, r *http.Request) {
	showHidden := r.URL.Query().Get("hidden") == "true"
	root, err := s.buildTree(s.Root, showHidden)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, root)
}

// alwaysIgnoredDir reports whether a directory is always omitted from the tree
// (heavy build/vendor/VCS dirs), regardless of the show-hidden toggle.
func alwaysIgnoredDir(name string) bool {
	switch name {
	case "node_modules", "dist", ".git", ".__tmp", "vendor":
		return true
	}
	return false
}

func (s *Server) buildTree(dir string, showHidden bool) (*treeNode, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	node := &treeNode{Name: filepath.Base(dir), Path: s.rel(dir), Dir: true}
	if node.Path == "." {
		node.Name = filepath.Base(s.Root)
	}
	for _, e := range entries {
		name := e.Name()
		hidden := strings.HasPrefix(name, ".")
		if e.IsDir() {
			if alwaysIgnoredDir(name) || (hidden && !showHidden) {
				continue
			}
			child, err := s.buildTree(filepath.Join(dir, name), showHidden)
			if err != nil {
				return nil, err
			}
			node.Children = append(node.Children, child)
			continue
		}
		if hidden && !showHidden {
			continue
		}
		node.Children = append(node.Children, &treeNode{
			Name: name, Path: s.rel(filepath.Join(dir, name)),
		})
	}
	sort.Slice(node.Children, func(i, j int) bool {
		a, b := node.Children[i], node.Children[j]
		if a.Dir != b.Dir {
			return a.Dir // directories first
		}
		return a.Name < b.Name
	})
	return node, nil
}

// fileRequest is the body for file/mkdir/delete/rename writes.
type fileRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	To      string `json:"to"` // rename target
}

// handleFile reads (GET ?path=) or writes (PUT) a workspace file.
func (s *Server) handleFile(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		abs, err := s.resolve(r.URL.Query().Get("path"))
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			writeError(w, statusForFS(err), err.Error())
			return
		}
		writeJSON(w, map[string]string{"path": s.rel(abs), "content": string(data)})
	case http.MethodPut:
		var req fileRequest
		if err := decodeBody(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		abs, err := s.resolve(req.Path)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := os.WriteFile(abs, []byte(req.Content), 0o644); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, map[string]string{"path": s.rel(abs)})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleMkdir(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req fileRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	abs, err := s.resolve(req.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]string{"path": s.rel(abs)})
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req fileRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	abs, err := s.resolve(req.Path)
	if err != nil || abs == s.Root {
		writeError(w, http.StatusBadRequest, "invalid path")
		return
	}
	if err := os.RemoveAll(abs); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]string{"path": s.rel(abs)})
}

func (s *Server) handleRename(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req fileRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	from, err1 := s.resolve(req.Path)
	to, err2 := s.resolve(req.To)
	if err1 != nil || err2 != nil || from == s.Root {
		writeError(w, http.StatusBadRequest, "invalid path")
		return
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := os.Rename(from, to); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]string{"path": s.rel(to)})
}

func statusForFS(err error) int {
	if os.IsNotExist(err) {
		return http.StatusNotFound
	}
	return http.StatusInternalServerError
}
