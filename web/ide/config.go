package ide

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gad-lang/gad/gadconfig"
	"gopkg.in/yaml.v3"
)

// handleConfig reads (GET) or writes (PUT) the workspace configuration. The API
// contract is a single merged document: the project config (.gad/gad.yaml)
// provides everything except the `ide` key, which comes from (and is written to)
// .gad/ide.yaml. On read, a legacy `ide` key inside gad.yaml is used when
// ide.yaml is absent.
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		doc, err := s.readMergedConfig()
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
		if err := s.writeSplitConfig(doc); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, doc)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// readMergedConfig returns the workspace config as one document: the gad.yaml
// keys plus an `ide` key sourced from ide.yaml (falling back to a legacy `ide`
// key already present in gad.yaml).
func (s *Server) readMergedConfig() (map[string]any, error) {
	doc, err := readConfig(gadconfig.File(s.Root))
	if err != nil {
		return nil, err
	}
	ide, err := readConfig(gadconfig.IDEFile(s.Root))
	if err != nil {
		return nil, err
	}
	if len(ide) > 0 {
		doc["ide"] = ide // ide.yaml wins over any legacy inline `ide`
	}
	return doc, nil
}

// writeSplitConfig writes the `ide` key to ide.yaml and the remaining keys to
// gad.yaml, keeping the two files separate (creating the config dir as needed).
func (s *Server) writeSplitConfig(doc map[string]any) error {
	ide, _ := doc["ide"].(map[string]any)

	rest := make(map[string]any, len(doc))
	for k, v := range doc {
		if k == "ide" {
			continue
		}
		rest[k] = v
	}
	if err := writeYAMLFile(gadconfig.File(s.Root), rest); err != nil {
		return err
	}
	return writeYAMLFile(gadconfig.IDEFile(s.Root), ide)
}

// writeYAMLFile marshals doc and writes it to path (a nil/empty doc writes an
// empty file rather than "null"), creating the parent directory if needed.
func writeYAMLFile(path string, doc map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if len(doc) == 0 {
		return os.WriteFile(path, nil, 0o644)
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}

// readConfig loads a YAML config file as a generic document. A missing file yields an
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
