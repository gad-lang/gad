package ide

import (
	"net/http"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// configFile is the project configuration file name, shared with `gad fmt`.
const configFile = ".gad.yaml"

// ideConfigFile holds the IDE layout/editor state, split out from .gad.yaml so
// project settings and IDE state live in separate files. Its whole document is
// the content of the config's `ide` key.
const ideConfigFile = ".gadide.yaml"

// handleConfig reads (GET) or writes (PUT) the workspace configuration. The API
// contract is a single merged document: `.gad.yaml` provides everything except
// the `ide` key, which comes from (and is written to) `.gadide.yaml`. On read,
// a legacy `ide` key inside `.gad.yaml` is used when `.gadide.yaml` is absent.
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

// readMergedConfig returns the workspace config as one document: the `.gad.yaml`
// keys plus an `ide` key sourced from `.gadide.yaml` (falling back to a legacy
// `ide` key already present in `.gad.yaml`).
func (s *Server) readMergedConfig() (map[string]any, error) {
	doc, err := readConfig(filepath.Join(s.Root, configFile))
	if err != nil {
		return nil, err
	}
	ide, err := readConfig(filepath.Join(s.Root, ideConfigFile))
	if err != nil {
		return nil, err
	}
	if len(ide) > 0 {
		doc["ide"] = ide // .gadide.yaml wins over any legacy inline `ide`
	}
	return doc, nil
}

// writeSplitConfig writes the `ide` key to `.gadide.yaml` and the remaining keys
// to `.gad.yaml`, keeping the two files separate.
func (s *Server) writeSplitConfig(doc map[string]any) error {
	ide, _ := doc["ide"].(map[string]any)

	rest := make(map[string]any, len(doc))
	for k, v := range doc {
		if k == "ide" {
			continue
		}
		rest[k] = v
	}
	if err := writeYAMLFile(filepath.Join(s.Root, configFile), rest); err != nil {
		return err
	}
	return writeYAMLFile(filepath.Join(s.Root, ideConfigFile), ide)
}

// writeYAMLFile marshals doc and writes it to path (a nil/empty doc writes an
// empty file rather than "null").
func writeYAMLFile(path string, doc map[string]any) error {
	if len(doc) == 0 {
		return os.WriteFile(path, nil, 0o644)
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
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
