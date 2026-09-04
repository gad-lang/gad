package parser

import (
	"strings"

	gadxnode "github.com/gad-lang/gad/gadx/node"
)

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

// htmlRawTextElements hold text, not markup: everything up to their close tag
// is content, so a `<` in a script or a `{` in a stylesheet is not markup and
// must not be read as one.
var htmlRawTextElements = map[string]bool{
	"script": true, "style": true,
}

// skipMarkupDeclaration advances past a `<!…>` that is not an element: a
// comment, a doctype, or a CDATA section. These carry no nesting, so a region
// containing one is no less complete for it. It returns the index just after
// the declaration and whether the terminator was found — a declaration ending
// exactly at the end of s is complete, which an index alone cannot say.
func skipMarkupDeclaration(s string, i int) (end int, ok bool) {
	if strings.HasPrefix(s[i:], "<!--") {
		if e := strings.Index(s[i+4:], "-->"); e >= 0 {
			return i + 4 + e + 3, true
		}
		return len(s), false
	}
	if strings.HasPrefix(s[i:], "<![CDATA[") {
		if e := strings.Index(s[i+9:], "]]>"); e >= 0 {
			return i + 9 + e + 3, true
		}
		return len(s), false
	}
	if e := strings.IndexByte(s[i:], '>'); e >= 0 {
		return i + e + 1, true
	}
	return len(s), false
}

// skipRawTextElement locates the close tag of a raw-text element whose opening
// tag ends at i. It returns where the content stops (the `<` of the close tag)
// and where the element stops (just after its `>`). Both are -1 when the close
// tag is not in s, so the caller knows the region is still incomplete.
func skipRawTextElement(s string, i int, name string) (contentEnd, end int) {
	k := indexFoldASCII(s[i:], "</"+name)
	if k < 0 {
		return -1, -1
	}
	k += i
	gt := strings.IndexByte(s[k:], '>')
	if gt < 0 {
		return -1, -1
	}
	return k, k + gt + 1
}

// indexFoldASCII is strings.Index with ASCII case folding, so `</SCRIPT>`
// closes a `<script>`.
func indexFoldASCII(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	for i := 0; i+len(substr) <= len(s); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			a, b := s[i+j], substr[j]
			if 'A' <= a && a <= 'Z' {
				a += 'a' - 'A'
			}
			if 'A' <= b && b <= 'Z' {
				b += 'a' - 'A'
			}
			if a != b {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// htmlRegionEnd finds the end of a self-contained HTML region beginning at
// s[start] (`<`). The region spans from the opening tag (or `<>` fragment) to
// its matching close tag, tracking nesting depth while ignoring `<`/`>` inside
// quoted attribute values and `{ … }` interpolations, the content of raw-text
// elements, and markup declarations such as comments. It returns the index just
// after the region and whether a complete region was found.
func htmlRegionEnd(s string, start int) (end int, ok bool) {
	depth := 0
	i := start
	for i < len(s) {
		if s[i] != '<' {
			i++
			continue
		}
		if i+1 < len(s) && s[i+1] == '!' {
			// A comment, doctype or CDATA section: content, not nesting.
			next, done := skipMarkupDeclaration(s, i)
			if !done {
				return 0, false
			}
			i = next
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
		lower := strings.ToLower(name)
		if !selfClose && htmlRawTextElements[lower] {
			// The element's content is text: skipping to its close tag keeps a
			// `<` in a script, or a `{` in a stylesheet, from reading as markup.
			_, after := skipRawTextElement(s, i, lower)
			if after < 0 {
				return 0, false
			}
			i = after
			if depth == 0 {
				return i, true
			}
			continue
		}
		if !selfClose && !gadxnode.IsSelfClosing(lower) {
			depth++
		}
		if depth == 0 {
			// a single self-closing / void element at the top level
			return i, true
		}
	}
	return 0, false
}
