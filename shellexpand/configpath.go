package shellexpand

import (
	"fmt"
	"strconv"
	"strings"
)

// lookupConfigPath navigates a dot/bracket path (e.g. ".ide.a[1][2].g") into a
// config document (a map[string]any / []any tree) and returns the leaf as a
// string. Any scalar leaf (string, int, uint, float, bool) is converted to its
// string form. It reports whether the path resolved.
func lookupConfigPath(doc any, path string) (string, bool) {
	cur := doc
	i := 0
	if i < len(path) && path[i] == '.' {
		i++ // an initial '.' before the first field
	}
	for i < len(path) {
		switch path[i] {
		case '.':
			i++
		case '[':
			j := strings.IndexByte(path[i:], ']')
			if j < 0 {
				return "", false
			}
			idxStr := path[i+1 : i+j]
			idx, err := strconv.Atoi(strings.TrimSpace(idxStr))
			if err != nil {
				return "", false
			}
			arr, ok := toSlice(cur)
			if !ok || idx < 0 || idx >= len(arr) {
				return "", false
			}
			cur = arr[idx]
			i += j + 1
		default:
			// A field name: run of name bytes.
			start := i
			for i < len(path) && (isNameByte(path[i])) {
				i++
			}
			if i == start {
				return "", false
			}
			field := path[start:i]
			m, ok := toStringMap(cur)
			if !ok {
				return "", false
			}
			v, ok := m[field]
			if !ok {
				return "", false
			}
			cur = v
		}
	}
	return scalarString(cur)
}

// toStringMap adapts the map shapes yaml.v3 may produce (map[string]any or
// map[any]any) to a map keyed by string.
func toStringMap(v any) (map[string]any, bool) {
	switch m := v.(type) {
	case map[string]any:
		return m, true
	case map[any]any:
		out := make(map[string]any, len(m))
		for k, val := range m {
			out[fmt.Sprint(k)] = val
		}
		return out, true
	}
	return nil, false
}

func toSlice(v any) ([]any, bool) {
	if s, ok := v.([]any); ok {
		return s, true
	}
	return nil, false
}

// scalarString converts a scalar config leaf to its string form.
func scalarString(v any) (string, bool) {
	switch t := v.(type) {
	case string:
		return t, true
	case bool:
		return strconv.FormatBool(t), true
	case int:
		return strconv.Itoa(t), true
	case int64:
		return strconv.FormatInt(t, 10), true
	case uint:
		return strconv.FormatUint(uint64(t), 10), true
	case uint64:
		return strconv.FormatUint(t, 10), true
	case float64:
		return strconv.FormatFloat(t, 'g', -1, 64), true
	case float32:
		return strconv.FormatFloat(float64(t), 'g', -1, 32), true
	case nil:
		return "", false
	}
	return "", false
}
