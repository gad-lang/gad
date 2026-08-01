package shellexpand

import (
	"regexp"
	"strings"
)

// globToRegex converts a shell glob (`*`, `?`, `[…]`, `[!…]`) to a regexp
// source. Unlike path.Match, `/` is treated as an ordinary character, matching
// bash parameter-expansion pattern semantics.
func globToRegex(glob string) string {
	var b strings.Builder
	for i := 0; i < len(glob); i++ {
		c := glob[i]
		switch c {
		case '\\':
			// Escape: the next byte is a literal (so `\/` is a plain '/', `\*` a
			// literal star), regexp-quoted when it is a regexp metacharacter.
			if i+1 < len(glob) {
				nc := glob[i+1]
				if strings.IndexByte(`.+()|^$\{}*?[]`, nc) >= 0 {
					b.WriteByte('\\')
				}
				b.WriteByte(nc)
				i++
			} else {
				b.WriteString(`\\`)
			}
		case '*':
			b.WriteString(".*")
		case '?':
			b.WriteByte('.')
		case '[':
			j := i + 1
			if j < len(glob) && (glob[j] == '!' || glob[j] == '^') {
				j++
			}
			if j < len(glob) && glob[j] == ']' { // a literal ] as the first member
				j++
			}
			for j < len(glob) && glob[j] != ']' {
				j++
			}
			if j >= len(glob) {
				b.WriteString(`\[`) // unterminated class → literal '['
			} else {
				cls := glob[i : j+1]
				if len(cls) > 1 && cls[1] == '!' {
					cls = "[^" + cls[2:]
				}
				b.WriteString(cls)
				i = j
			}
		default:
			if strings.IndexByte(`.+()|^$\{}`, c) >= 0 {
				b.WriteByte('\\')
			}
			b.WriteByte(c)
		}
	}
	return b.String()
}

// fullMatch reports whether s matches glob in its entirety.
func fullMatch(glob, s string) bool {
	re, err := regexp.Compile("^(?:" + globToRegex(glob) + ")$")
	if err != nil {
		return false
	}
	return re.MatchString(s)
}

// stripPrefix removes a prefix of val matching glob. With longest=false the
// shortest matching prefix is removed; with longest=true the longest one.
func stripPrefix(val, glob string, longest bool) string {
	if longest {
		for i := len(val); i >= 0; i-- {
			if fullMatch(glob, val[:i]) {
				return val[i:]
			}
		}
	} else {
		for i := 0; i <= len(val); i++ {
			if fullMatch(glob, val[:i]) {
				return val[i:]
			}
		}
	}
	return val
}

// stripSuffix removes a suffix of val matching glob. With longest=false the
// shortest matching suffix is removed; with longest=true the longest one.
func stripSuffix(val, glob string, longest bool) string {
	if longest {
		for i := 0; i <= len(val); i++ {
			if fullMatch(glob, val[i:]) {
				return val[:i]
			}
		}
	} else {
		for i := len(val); i >= 0; i-- {
			if fullMatch(glob, val[i:]) {
				return val[:i]
			}
		}
	}
	return val
}

type replaceMode int

const (
	replaceFirst replaceMode = iota
	replaceAll
	replaceFront
	replaceBack
)

// replace implements `${var/pat/str}` and its variants. spec is `pattern/string`
// (a single unescaped `/` separates them; the string part may be empty or
// omitted). The replacement string is itself expanded.
func replace(val, spec string, env Env, mode replaceMode) string {
	pat, repl := splitReplaceSpec(spec)
	if pat == "" {
		return val
	}
	repl = Expand(repl, env)

	src := globToRegex(pat)
	switch mode {
	case replaceFront:
		src = "^(?:" + src + ")"
	case replaceBack:
		src = "(?:" + src + ")$"
	default:
		src = "(?:" + src + ")"
	}
	re, err := regexp.Compile(src)
	if err != nil {
		return val
	}
	// Use a literal replacement (no $1 expansion from the replacement text).
	lit := func(string) string { return repl }
	if mode == replaceAll {
		return re.ReplaceAllStringFunc(val, lit)
	}
	// Replace only the first match.
	loc := re.FindStringIndex(val)
	if loc == nil {
		return val
	}
	return val[:loc[0]] + repl + val[loc[1]:]
}

// splitReplaceSpec splits `pattern/string` at the first unescaped `/`. A missing
// `/` means an empty replacement.
func splitReplaceSpec(spec string) (pat, repl string) {
	for i := 0; i < len(spec); i++ {
		if spec[i] == '\\' {
			i++
			continue
		}
		if spec[i] == '/' {
			return spec[:i], spec[i+1:]
		}
	}
	return spec, ""
}
