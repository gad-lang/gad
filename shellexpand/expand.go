// Package shellexpand expands shell-style variable references in a string,
// supporting bash-style parameter expansion. It is used to expand the `env`
// section of a Gad workspace config (.gad.yaml).
//
// A reference is either an environment variable (`$VAR`, `${VAR}`) or, when the
// name inside `${…}` begins with a dot, a path into the config document itself
// (`${.ide.a[1][2].g}`). Both forms accept the parameter-expansion operators:
//
//	${var:-default}   use default if var is unset or null (no assignment)
//	${var:=default}   assign default to var if it is unset or null
//	${var:+alternate} use alternate only if var is set and non-null, else empty
//	${var:offset:length}  substring starting at offset (length optional)
//	${var#pattern}    remove the shortest matching prefix
//	${var##pattern}   remove the longest matching prefix
//	${var%pattern}    remove the shortest matching suffix
//	${var%%pattern}   remove the longest matching suffix
//	${var/pattern/string}   replace the first match of pattern
//	${var//pattern/string}  replace all matches of pattern
//	${var/#pattern/string}  replace pattern only at the front
//	${var/%pattern/string}  replace pattern only at the end
//
// Patterns are shell globs (`*`, `?`, `[…]`); `/` is an ordinary character.
package shellexpand

import (
	"strconv"
	"strings"
)

// Env provides the variable sources for Expand.
type Env struct {
	// Get returns the value of an environment variable and whether it is set.
	Get func(name string) (string, bool)
	// Set assigns an environment variable, used by the `${var:=default}`
	// operator. When nil the assignment is skipped (the value is still expanded).
	Set func(name, value string)
	// Config is the config document (map[string]any / []any tree) that a
	// dot-prefixed reference (`${.a.b[0]}`) navigates. When nil such references
	// resolve to empty/unset.
	Config any
}

// lookup resolves a reference name to its value. A name beginning with "." is a
// config path; any other name is an environment variable.
func (e Env) lookup(name string) (string, bool) {
	if strings.HasPrefix(name, ".") {
		if e.Config == nil {
			return "", false
		}
		return lookupConfigPath(e.Config, name)
	}
	if e.Get == nil {
		return "", false
	}
	return e.Get(name)
}

// WalkExpand recursively expands every string value in a config document (a
// map[string]any / []any / scalar tree) using env, returning a new tree. Maps
// and slices are walked deeply. A string that contained a reference is, after
// expansion, coerced to a bool / int / float when it now holds one, so
// `port: "${PORT:-8080}"` yields the integer 8080 while a literal `"8080"`
// (with no reference) stays a string. Non-string scalars are returned unchanged.
func WalkExpand(v any, env Env) any {
	switch t := v.(type) {
	case string:
		out := Expand(t, env)
		if out != t {
			return coerceScalar(out)
		}
		return out
	case map[string]any:
		m := make(map[string]any, len(t))
		for k, val := range t {
			m[k] = WalkExpand(val, env)
		}
		return m
	case map[any]any:
		m := make(map[any]any, len(t))
		for k, val := range t {
			m[k] = WalkExpand(val, env)
		}
		return m
	case []any:
		s := make([]any, len(t))
		for i, val := range t {
			s[i] = WalkExpand(val, env)
		}
		return s
	default:
		return v
	}
}

// coerceScalar converts an expanded string to a bool / int / float when it holds
// exactly one; otherwise it returns the string unchanged. Booleans follow YAML,
// which also accepts yes/no (case-insensitively) in addition to true/false.
func coerceScalar(s string) any {
	switch strings.ToLower(s) {
	case "true", "yes":
		return true
	case "false", "no":
		return false
	}
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return int(i)
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return s
}

// Expand expands all `$VAR` / `${…}` references in s using env.
func Expand(s string, env Env) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		c := s[i]
		switch c {
		case '\\':
			// Backslash escapes the next byte (so `\$` is a literal `$`).
			if i+1 < len(s) {
				b.WriteByte(s[i+1])
				i += 2
			} else {
				b.WriteByte(c)
				i++
			}
		case '$':
			ref, next, ok := parseRef(s, i)
			if !ok {
				b.WriteByte(c)
				i++
				continue
			}
			b.WriteString(expandRef(ref, env))
			i = next
		default:
			b.WriteByte(c)
			i++
		}
	}
	return b.String()
}

// parseRef parses a reference starting at s[i] (which is '$'). It returns the
// reference body (without the leading `$`/braces), the index just past it, and
// whether a reference was found.
func parseRef(s string, i int) (body string, next int, ok bool) {
	i++ // past '$'
	if i >= len(s) {
		return "", 0, false
	}
	if s[i] == '{' {
		// Braced form: scan to the matching '}' (balanced, so nested `${…}` in a
		// default word is kept intact for the recursive expansion).
		depth := 1
		j := i + 1
		for j < len(s) && depth > 0 {
			switch s[j] {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					return s[i+1 : j], j + 1, true
				}
			}
			j++
		}
		return "", 0, false // unterminated
	}
	// Bare form: `$NAME`, NAME is [A-Za-z_][A-Za-z0-9_]*.
	if !isNameStart(s[i]) {
		return "", 0, false
	}
	j := i
	for j < len(s) && isNameByte(s[j]) {
		j++
	}
	return s[i:j], j, true
}

func isNameStart(c byte) bool {
	return c == '_' || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func isNameByte(c byte) bool {
	return isNameStart(c) || (c >= '0' && c <= '9')
}

// expandRef resolves and applies the operator of a single reference body (the
// text inside `${…}`, or a bare name).
func expandRef(body string, env Env) string {
	name, op := splitNameOp(body)
	val, set := env.lookup(name)

	if op == "" {
		return val
	}

	switch {
	case strings.HasPrefix(op, ":-"):
		if !set || val == "" {
			return Expand(op[2:], env)
		}
		return val
	case strings.HasPrefix(op, ":="):
		if !set || val == "" {
			w := Expand(op[2:], env)
			if env.Set != nil && !strings.HasPrefix(name, ".") {
				env.Set(name, w)
			}
			return w
		}
		return val
	case strings.HasPrefix(op, ":+"):
		if set && val != "" {
			return Expand(op[2:], env)
		}
		return ""
	case strings.HasPrefix(op, ":?"):
		// Error form: with no error channel here, fall back to the value (empty
		// when unset), which is the least surprising behaviour for a config.
		return val
	case strings.HasPrefix(op, ":"):
		return substring(val, op[1:])
	case strings.HasPrefix(op, "##"):
		return stripPrefix(val, op[2:], true)
	case strings.HasPrefix(op, "#"):
		return stripPrefix(val, op[1:], false)
	case strings.HasPrefix(op, "%%"):
		return stripSuffix(val, op[2:], true)
	case strings.HasPrefix(op, "%"):
		return stripSuffix(val, op[1:], false)
	case strings.HasPrefix(op, "//"):
		return replace(val, op[2:], env, replaceAll)
	case strings.HasPrefix(op, "/#"):
		return replace(val, op[2:], env, replaceFront)
	case strings.HasPrefix(op, "/%"):
		return replace(val, op[2:], env, replaceBack)
	case strings.HasPrefix(op, "/"):
		return replace(val, op[1:], env, replaceFirst)
	}
	return val
}

// splitNameOp splits a reference body into the variable name and the operator
// suffix (including its leading operator character). A dot-prefixed name keeps
// its `.field`/`[index]` path; an ordinary name is [A-Za-z0-9_]+.
func splitNameOp(body string) (name, op string) {
	if body == "" {
		return "", ""
	}
	i := 0
	if body[0] == '.' {
		// Config path: `.field`, `[index]`, chained. Stops at an operator char.
		i = 1
		for i < len(body) {
			c := body[i]
			if isNameByte(c) || c == '.' || c == '[' || c == ']' {
				i++
				continue
			}
			break
		}
	} else {
		for i < len(body) && isNameByte(body[i]) {
			i++
		}
	}
	return body[:i], body[i:]
}

// substring implements `${var:offset}` and `${var:offset:length}` with bash
// semantics: a negative offset counts from the end, and a negative length is an
// offset from the end.
func substring(val, spec string) string {
	offStr, lenStr, hasLen := spec, "", false
	if idx := strings.IndexByte(spec, ':'); idx >= 0 {
		offStr, lenStr, hasLen = spec[:idx], spec[idx+1:], true
	}
	n := len(val)
	off, err := strconv.Atoi(strings.TrimSpace(offStr))
	if err != nil {
		return val
	}
	if off < 0 {
		off = n + off
	}
	if off < 0 {
		off = 0
	}
	if off > n {
		return ""
	}
	if !hasLen {
		return val[off:]
	}
	l, err := strconv.Atoi(strings.TrimSpace(lenStr))
	if err != nil {
		return val[off:]
	}
	end := off + l
	if l < 0 {
		end = n + l // negative length: offset from the end
	}
	if end < off {
		return ""
	}
	if end > n {
		end = n
	}
	return val[off:end]
}
