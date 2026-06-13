package ide

import (
	"net/http"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// configFile is the project configuration file name, shared with `gad fmt`.
const configFile = ".gad.yaml"

// handleConfig reads (GET) or writes (PUT) the workspace .gad.yaml. The whole
// document is round-tripped as JSON so the UI can edit the `fmt` formatter
// settings and the `ide` layout key while preserving any other keys.
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(s.Root, configFile)
	switch r.Method {
	case http.MethodGet:
		doc, err := readConfig(path)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, doc)
	case http.MethodPut:
		var doc map[string]any
		if err := decodeBody(r, &doc); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		out, err := yaml.Marshal(doc)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := os.WriteFile(path, out, 0o644); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, doc)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// readConfig loads .gad.yaml as a generic document. A missing file yields an
// empty document so the UI starts with defaults.
func readConfig(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, err
	}
	doc := map[string]any{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return doc, nil
}
