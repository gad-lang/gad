package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/gadconfig"
	"github.com/gad-lang/gad/shellexpand"
	"gopkg.in/yaml.v3"
)

// processEnvMap reads the current process environment into a name->value map.
func processEnvMap() map[string]string {
	m := map[string]string{}
	for _, kv := range os.Environ() {
		if i := strings.IndexByte(kv, '='); i > 0 {
			m[kv[:i]] = kv[i+1:]
		}
	}
	return m
}

// envEntry is one `env` section entry: NAME plus either a raw scalar value or a
// list of raw path segments (joined with the OS path-list separator).
type envEntry struct {
	name  string
	value string   // scalar form (list == nil)
	list  []string // path-list form (joined per-OS); nil for scalar
}

// envSectionEntries returns the ordered entries of the config `env` section,
// which may be a mapping (sorted by key for determinism) or a list of
// `NAME=value` strings (order preserved). A value may itself be an array of path
// segments (e.g. `GADPATH: ["A", "B"]`), which is joined with the OS
// path-list separator so it stays portable across Linux/Windows.
func envSectionEntries(section any) []envEntry {
	switch t := section.(type) {
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := make([]envEntry, 0, len(keys))
		for _, k := range keys {
			if e, ok := makeEnvEntry(k, t[k]); ok {
				out = append(out, e)
			}
		}
		return out
	case map[any]any:
		conv := make(map[string]any, len(t))
		for k, v := range t {
			conv[fmt.Sprint(k)] = v
		}
		return envSectionEntries(conv)
	case []any:
		out := make([]envEntry, 0, len(t))
		for _, item := range t {
			s, ok := shellScalar(item)
			if !ok {
				continue
			}
			if i := strings.IndexByte(s, '='); i > 0 {
				out = append(out, envEntry{name: s[:i], value: s[i+1:]})
			}
		}
		return out
	}
	return nil
}

// splitTopLevelColon splits s on `:` characters that are outside a `${…}`
// expansion, so a list-separator `:` splits while an operator `:` (as in
// `${var:-default}`) does not. It is used to unpack path segments authored in
// canonical Unix form inside an array element.
func splitTopLevelColon(s string) []string {
	var (
		parts []string
		depth int
		start int
	)
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			if depth > 0 {
				depth--
			}
		case ':':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	return append(parts, s[start:])
}

// makeEnvEntry builds an entry from a mapping value: a scalar, or an array of
// scalars (a per-OS-joined path list).
func makeEnvEntry(name string, v any) (envEntry, bool) {
	if arr, ok := v.([]any); ok {
		parts := make([]string, 0, len(arr))
		for _, item := range arr {
			if s, ok := shellScalar(item); ok {
				parts = append(parts, s)
			}
		}
		return envEntry{name: name, list: parts}, true
	}
	if s, ok := shellScalar(v); ok {
		return envEntry{name: name, value: s}, true
	}
	return envEntry{}, false
}

// shellScalar renders a scalar config value to its string form.
func shellScalar(v any) (string, bool) {
	switch t := v.(type) {
	case string:
		return t, true
	case bool:
		if t {
			return "true", true
		}
		return "false", true
	case int:
		return fmt.Sprint(t), true
	case int64, uint, uint64, float64, float32:
		return fmt.Sprint(t), true
	}
	return "", false
}

// loadWorkspaceEnv reads the `env` section of the workspace config
// (gadconfig.File(dir)) and returns a gad.Env that extends the process
// environment with the (expanded) config entries. Later map entries — processed
// in sorted-key order, or list order for the list form — may reference earlier
// ones and the process environment via bash-style expansion; a dot-prefixed
// reference (`${.a.b}`) reads the config document. A missing file or section
// yields the plain process environment.
func loadWorkspaceEnv(dir string) *gad.Env {
	base := processEnvMap()

	data, err := os.ReadFile(gadconfig.File(dir))
	if err != nil {
		return gad.NewEnvFromMap(base)
	}
	var doc map[string]any
	if yaml.Unmarshal(data, &doc) != nil {
		return gad.NewEnvFromMap(base)
	}

	exp := shellexpand.Env{
		Get:    func(name string) (string, bool) { v, ok := base[name]; return v, ok },
		Set:    func(name, value string) { base[name] = value },
		Config: doc,
	}
	sep := string(os.PathListSeparator)
	for _, e := range envSectionEntries(doc["env"]) {
		if e.list != nil {
			// The array form is the portable path-list form: it is authored in
			// canonical Unix form (`/` directory separator, `:` list separator)
			// and converted to the OS form here. An element may itself pack
			// several entries with `:`; each entry's `/` becomes the OS separator
			// and all are joined with the OS path-list separator.
			var parts []string
			for _, raw := range e.list {
				for _, piece := range splitTopLevelColon(raw) {
					parts = append(parts, filepath.FromSlash(shellexpand.Expand(piece, exp)))
				}
			}
			base[e.name] = strings.Join(parts, sep)
		} else {
			// The string form is taken literally (only expanded), so values that
			// are not paths (URLs, messages, …) keep their `:` and `/`.
			base[e.name] = shellexpand.Expand(e.value, exp)
		}
	}
	return gad.NewEnvFromMap(base)
}

// gadPathFromEnv returns the GADPATH search directories carried by env (an
// OS-list-separated GADPATH entry), for extending the module resolver.
func gadPathFromEnv(env *gad.Env) []string {
	if env == nil {
		return nil
	}
	v, ok := env.Get("GADPATH")
	if !ok || v == "" {
		return nil
	}
	return filepath.SplitList(v)
}
