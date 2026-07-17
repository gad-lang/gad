package parser

import "strings"

// htmlVoidElements are HTML elements that never have a closing tag, so they do
// not increase the tag nesting depth when scanning an HTML region.
var htmlVoidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true,
	"hr": true, "img": true, "input": true, "link": true, "meta": true,
	"param": true, "source": true, "track": true, "wbr": true,
}

func isTagNameChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '-' || c == '_' || c == ':' || c == '.'
}

// skipString advances past a quoted string beginning at s[i] (the opening
// quote), returning the index just after the closing quote (or len(s) at EOF).
func skipString(s string, i int) int {
	q := s[i]
	i++
	for i < len(s) {
		if s[i] == '\\' {
			i += 2
			continue
		}
		if s[i] == q {
			return i + 1
		}
		i++
	}
	return i
}

// skipBraces advances past a `{ … }` interpolation beginning at s[i] (the `{`),
// balancing nested braces and skipping string contents, returning the index just
// after the matching `}` (or len(s) if unbalanced).
func skipBraces(s string, i int) int {
	depth := 0
	for i < len(s) {
		switch s[i] {
		case '"', '\'', '`':
			i = skipString(s, i)
			continue
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
		i++
	}
	return i
}

// scanOpenTagEnd scans an opening (or self-closing) tag starting at s[i] (`<`),
// skipping quoted attribute values and `{ … }` interpolations. It returns the
// index just after the closing `>`, whether the tag is self-closing (`… />`),
// and the tag name (empty for a `<>` fragment). end is -1 when no `>` is found.
func scanOpenTagEnd(s string, i int) (end int, selfClose bool, name string) {
	j := i + 1
	nameStart := j
	for j < len(s) && isTagNameChar(s[j]) {
		j++
	}
	name = s[nameStart:j]
	for j < len(s) {
		switch s[j] {
		case '"', '\'':
			j = skipString(s, j)
		case '{':
			j = skipBraces(s, j)
		case '>':
			k := j - 1
			for k > i && (s[k] == ' ' || s[k] == '\t' || s[k] == '\n' || s[k] == '\r') {
				k--
			}
			return j + 1, s[k] == '/', name
		default:
			j++
		}
	}
	return -1, false, name
}

// htmlRegionEnd finds the end of a self-contained HTML region beginning at
// s[start] (`<`). The region spans from the opening tag (or `<>` fragment) to
// its matching close tag, tracking nesting depth while ignoring `<`/`>` inside
// quoted attribute values and `{ … }` interpolations. It returns the index just
// after the region and whether a complete region was found.
func htmlRegionEnd(s string, start int) (end int, ok bool) {
	depth := 0
	i := start
	for i < len(s) {
		if s[i] != '<' {
			i++
			continue
		}
		if i+1 < len(s) && s[i+1] == '/' {
			// close tag `</name>` or fragment close `</>`
			gt := strings.IndexByte(s[i:], '>')
			if gt < 0 {
				return 0, false
			}
			i += gt + 1
			depth--
			if depth <= 0 {
				return i, true
			}
			continue
		}
		if i+1 < len(s) && s[i+1] == '>' {
			// fragment open `<>`
			depth++
			i += 2
			continue
		}
		tagEnd, selfClose, name := scanOpenTagEnd(s, i)
		if tagEnd < 0 {
			return 0, false
		}
		i = tagEnd
		if !selfClose && !htmlVoidElements[strings.ToLower(name)] {
			depth++
		}
		if depth == 0 {
			// a single self-closing / void element at the top level
			return i, true
		}
	}
	return 0, false
}
