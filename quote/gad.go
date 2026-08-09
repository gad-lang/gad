package quote

import (
	"fmt"
	"strconv"
	"strings"
)

// This file implements quote/unquote for every Gad string literal flavour, in
// four variations along two axes — raw vs cooked (escape-interpreted) and
// single-delimiter vs heredoc (a fence of three or more delimiters, PostgreSQL
// style):
//
//	          | single delimiter        | heredoc (3+ fence)
//	----------+-------------------------+---------------------------
//	 cooked   | "…"   QuoteString       | """…""" QuoteHeredoc
//	 raw      | `…`   QuoteRawString    | ```…``` QuoteRawHeredoc
//
// Every form may span multiple lines: a source line break is a real newline in
// the value. Cooked forms interpret escapes (\n, \t, \xNN, …) and the `\{` / `\}`
// interpolation-delimiter escape; raw forms are verbatim. Only heredocs strip the
// common leading indentation of their body.

// DefaultMaxLineWidth is the per-line width Quote targets when none is given.
const DefaultMaxLineWidth = 120

// Options controls Quote's choice of literal form.
type Options struct {
	// MaxLineWidth makes Quote prefer a multi-line (heredoc) form when a
	// single-line literal would exceed this many bytes on a source line and the
	// content actually spans lines. Zero selects DefaultMaxLineWidth; a negative
	// value disables the width heuristic (always single-line when possible).
	MaxLineWidth int
	// Raw prefers the verbatim (backtick) forms over the cooked (`"…"`) ones.
	Raw bool
}

func (o Options) maxWidth() int {
	if o.MaxLineWidth == 0 {
		return DefaultMaxLineWidth
	}
	return o.MaxLineWidth
}

// Quote encodes s as the most readable Gad string literal for opts: a
// single-delimiter form when it fits on one line, or a heredoc when the content
// spans lines and a single-line literal would exceed the line-width limit
// (default 120). The result always round-trips through Unquote.
func Quote(s string, opts ...Options) string {
	var o Options
	if len(opts) > 0 {
		o = opts[0]
	}
	width := o.maxWidth()
	if o.Raw {
		// A raw single-backtick string already spans lines verbatim and a heredoc
		// of the same body is no narrower, so QuoteRaw's choice is the best raw form
		// regardless of width.
		return QuoteRaw(s)
	}
	single := QuoteString(s)
	if fitsWidth(single, width) || !strings.Contains(s, "\n") {
		return single
	}
	return QuoteHeredoc(s)
}

// Unquote decodes any Gad string literal — cooked or raw, single-delimiter or
// heredoc — dispatching on its opening delimiter and fence width.
func Unquote(lit string) (string, error) {
	switch {
	case len(lit) >= 2 && lit[0] == '"':
		if fenceOf(lit, '"') >= 3 {
			return UnquoteHeredoc(lit)
		}
		return UnquoteString(lit)
	case len(lit) >= 2 && lit[0] == '`':
		return UnquoteRaw(lit)
	default:
		return "", fmt.Errorf("quote: %q is not a string literal", lit)
	}
}

// fitsWidth reports whether the single-line literal q fits in width. A negative
// width disables the check (always fits).
func fitsWidth(q string, width int) bool {
	return width < 0 || len(q) <= width
}

// =============================================================================
// Cooked string: "…"
// =============================================================================

// QuoteString encodes s as a Gad cooked string literal `"…"` using Go-compatible
// escape sequences.
func QuoteString(s string) string { return strconv.Quote(s) }

// UnquoteString decodes a cooked string literal `"…"` to its value: escape
// sequences are interpreted, raw line breaks are kept (a `"…"` may span lines)
// and `\{` / `\}` collapse to literal braces.
func UnquoteString(lit string) (string, error) {
	if !wrapped(lit, '"') {
		return "", fmt.Errorf("quote: %q is not a double-quoted string", lit)
	}
	return unquoteCooked(lit, false)
}

// InterpolationText decodes a cooked interpolated-string body `"…"` like
// UnquoteString but keeps `\{` / `\}` escaped (as `\{` / `\}`) so the
// interpolation parser treats those braces as literal text rather than
// delimiters.
func InterpolationText(lit string) (string, error) {
	if !wrapped(lit, '"') {
		return "", fmt.Errorf("quote: %q is not a double-quoted string", lit)
	}
	return unquoteCooked(lit, true)
}

// unquoteCooked resolves a `"…"` literal. When keepBraceEscape is true the
// `\{` / `\}` escapes are preserved for a later interpolation pass; otherwise
// they collapse to literal braces.
func unquoteCooked(lit string, keepBraceEscape bool) (string, error) {
	lit = rewriteBraceEscapes(lit, keepBraceEscape)
	lit = escapeRawNewlines(lit)
	return strconv.Unquote(lit)
}

// =============================================================================
// Raw string: `…`
// =============================================================================

// QuoteRawString encodes s as a raw string literal “ `…` “ (verbatim). It
// returns an error when s contains a backtick, since a single-backtick raw string
// cannot hold one — use QuoteRawHeredoc or QuoteRaw instead.
func QuoteRawString(s string) (string, error) {
	if strings.ContainsRune(s, '`') {
		return "", fmt.Errorf("quote: a raw string cannot contain a backtick; use a raw heredoc")
	}
	return "`" + s + "`", nil
}

// UnquoteRawString decodes a raw string literal “ `…` “ (verbatim, no escapes).
func UnquoteRawString(lit string) (string, error) {
	if !wrapped(lit, '`') {
		return "", fmt.Errorf("quote: %q is not a backtick string", lit)
	}
	return lit[1 : len(lit)-1], nil
}

// =============================================================================
// Cooked heredoc: """…"""
// =============================================================================

// QuoteHeredoc encodes s as a cooked heredoc `"""…"""`. Backslashes are escaped
// (the form interprets escapes) and the fence widens past any run of `"` in the
// body so it cannot close early. The single-line body form is used, so the value
// round-trips with no indentation stripping.
func QuoteHeredoc(s string) string {
	body := strings.ReplaceAll(s, `\`, `\\`)
	fence := strings.Repeat(`"`, fenceWidth(body, '"'))
	return fence + body + fence
}

// UnquoteHeredoc decodes a cooked heredoc `"""…"""`: the fence is removed, the
// common leading indentation of the body is stripped (multi-line form only) and
// escape sequences are interpreted.
func UnquoteHeredoc(lit string) (string, error) {
	n := fenceOf(lit, '"')
	if n < 3 || !wrappedFence(lit, '"', n) {
		return "", fmt.Errorf("quote: %q is not a \"\"\" heredoc", lit)
	}
	body := stripHeredocIndent(heredocBody(lit, n), heredocStripCount(lit, n))
	return unescapeHeredoc(body), nil
}

// =============================================================================
// Raw heredoc: ```…```
// =============================================================================

// QuoteRawHeredoc encodes s as a raw heredoc “ ```…``` “ (verbatim), choosing a
// fence wider than the longest run of backticks in s so it cannot close early.
func QuoteRawHeredoc(s string) string {
	fence := strings.Repeat("`", fenceWidth(s, '`'))
	return fence + s + fence
}

// UnquoteRawHeredoc decodes a raw heredoc “ ```…``` “ (verbatim), stripping the
// fence and the common leading indentation (multi-line form only).
func UnquoteRawHeredoc(lit string) (string, error) {
	n := fenceOf(lit, '`')
	if n < 3 || !wrappedFence(lit, '`', n) {
		return "", fmt.Errorf("quote: %q is not a ``` heredoc", lit)
	}
	return stripHeredocIndent(heredocBody(lit, n), heredocStripCount(lit, n)), nil
}

// =============================================================================
// Raw (verbatim), best form
// =============================================================================

// QuoteRaw encodes s in the narrowest raw (verbatim) form: a single-backtick
// “ `…` “ when s has no backtick, otherwise a raw heredoc with a wide-enough
// fence. The result always round-trips through UnquoteRaw.
func QuoteRaw(s string) string {
	if !strings.ContainsRune(s, '`') {
		return "`" + s + "`"
	}
	return QuoteRawHeredoc(s)
}

// UnquoteRaw decodes any raw form — “ `…` “ or “ ```…``` “ — dispatching on
// the fence width.
func UnquoteRaw(lit string) (string, error) {
	switch n := fenceOf(lit, '`'); {
	case n == 1:
		return UnquoteRawString(lit)
	case n >= 3:
		return UnquoteRawHeredoc(lit)
	default:
		return "", fmt.Errorf("quote: %q is not a raw string or heredoc", lit)
	}
}

// =============================================================================
// shared helpers
// =============================================================================

// wrapped reports whether lit begins and ends with a single q delimiter.
func wrapped(lit string, q byte) bool {
	return len(lit) >= 2 && lit[0] == q && lit[len(lit)-1] == q
}

// wrappedFence reports whether lit begins and ends with a fence of at least n q
// delimiters (the body between them may itself contain q, as with an empty
// heredoc `""""""`).
func wrappedFence(lit string, q byte, n int) bool {
	if len(lit) < 2*n {
		return false
	}
	for i := 0; i < n; i++ {
		if lit[i] != q || lit[len(lit)-1-i] != q {
			return false
		}
	}
	return true
}

// fenceOf returns the fence width of a delimited literal: the run of q at the
// start, capped at half the length so that a short literal whose whole body is q
// (an empty string `""` or empty heredoc `""""""`) resolves its opening and
// closing fences instead of counting them as one run.
func fenceOf(lit string, q byte) int {
	n := leadingRun(lit, q)
	if half := len(lit) / 2; n > half {
		n = half
	}
	return n
}

// leadingRun returns the length of the run of q at the start of s.
func leadingRun(s string, q byte) int {
	n := 0
	for n < len(s) && s[n] == q {
		n++
	}
	return n
}

// longestRun returns the length of the longest run of q in s.
func longestRun(s string, q byte) int {
	best, cur := 0, 0
	for i := 0; i < len(s); i++ {
		if s[i] == q {
			cur++
			if cur > best {
				best = cur
			}
		} else {
			cur = 0
		}
	}
	return best
}

// fenceWidth returns the narrowest valid heredoc fence width for a body: an odd
// count of at least three that is strictly greater than the longest run of q in
// the body, so the fence cannot appear inside it.
func fenceWidth(body string, q byte) int {
	n := 3
	for n <= longestRun(body, q) {
		n += 2
	}
	return n
}

// heredocBody strips a fence of n delimiters from both ends of lit and, for the
// multi-line form (a newline immediately after the opening fence), drops that
// newline and the final newline before the closing fence.
func heredocBody(lit string, n int) string {
	body := lit[n : len(lit)-n]
	if len(body) > 0 && body[0] == '\n' {
		body = body[1:]
		if i := strings.LastIndexByte(body, '\n'); i >= 0 {
			body = body[:i]
		}
	}
	return body
}

// heredocStripCount returns the common leading indentation (spaces/tabs) removed
// from each body line: the first content line's indentation, and only for the
// multi-line form (a newline right after the opening fence).
func heredocStripCount(lit string, n int) int {
	if n >= len(lit) || lit[n] != '\n' {
		return 0
	}
	c := heredocBody(lit, n)
	i := 0
	for i < len(c) && (c[i] == ' ' || c[i] == '\t') {
		i++
	}
	return i
}

// stripHeredocIndent removes up to n leading spaces/tabs from every line of s.
func stripHeredocIndent(s string, n int) string {
	if n <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		j := 0
		for j < n && j < len(line) && (line[j] == ' ' || line[j] == '\t') {
			j++
		}
		lines[i] = line[j:]
	}
	return strings.Join(lines, "\n")
}

// unescapeHeredoc interprets Go-style escape sequences in a cooked heredoc body,
// keeping an unrecognized `\x` verbatim.
func unescapeHeredoc(s string) string {
	if !strings.ContainsRune(s, '\\') {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		if s[i] != '\\' {
			b.WriteByte(s[i])
			i++
			continue
		}
		value, multibyte, tail, err := strconv.UnquoteChar(s[i:], '"')
		if err != nil {
			b.WriteByte(s[i])
			i++
			continue
		}
		if multibyte {
			b.WriteRune(value)
		} else {
			b.WriteByte(byte(value))
		}
		i = len(s) - len(tail)
	}
	return b.String()
}

// rewriteBraceEscapes rewrites `\{` / `\}` in a double-quoted string literal so
// strconv.Unquote accepts them (it does not know those escapes). When keep is
// false the escape collapses to a literal brace (`\{` → `{`); when true it is
// preserved as a backslash-brace (`\{` → `\\{` → `\{`) for a later interpolation
// pass to strip. Other escapes — including `\\` — pass through unchanged, so a
// literal backslash before a brace (`\\{`) is left intact.
func rewriteBraceEscapes(lit string, keep bool) string {
	if !strings.Contains(lit, `\{`) && !strings.Contains(lit, `\}`) {
		return lit
	}
	var b strings.Builder
	b.Grow(len(lit) + 8)
	for i := 0; i < len(lit); i++ {
		c := lit[i]
		if c == '\\' && i+1 < len(lit) {
			if n := lit[i+1]; n == '{' || n == '}' {
				if keep {
					b.WriteString(`\\`)
					b.WriteByte(n)
				} else {
					b.WriteByte(n)
				}
				i++
				continue
			}
			b.WriteByte(c)
			b.WriteByte(lit[i+1])
			i++
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

// escapeRawNewlines rewrites raw newline / carriage-return bytes in a
// double-quoted string literal to `\n` / `\r`, which strconv.Unquote requires
// (Go interpreted string literals may not contain raw newlines). This is what
// lets a single-quoted `"…"` string span multiple lines: a source line break
// becomes a real newline in the value, verbatim. Scanning guarantees a backslash
// is always followed by a valid escape char, so escape sequences are copied as-is
// and never contain a raw newline.
func escapeRawNewlines(lit string) string {
	if !strings.ContainsAny(lit, "\n\r") {
		return lit
	}
	var b strings.Builder
	b.Grow(len(lit) + 8)
	for i := 0; i < len(lit); i++ {
		switch c := lit[i]; c {
		case '\\':
			b.WriteByte(c)
			if i+1 < len(lit) {
				b.WriteByte(lit[i+1])
				i++
			}
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}
