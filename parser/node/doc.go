// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package node

import (
	"strings"

	"github.com/gad-lang/gad/parser/ast"
)

// docKind classifies a doc comment by its marker.
type docKind int

const (
	docSingle docKind = iota // `/? text`
	docBlock                 // `/??` … `??`
	docRoot                  // `/???` … `???`
)

// docComment is a parsed doc comment: its kind and Markdown content (markers
// stripped, inner lines joined by '\n').
type docComment struct {
	kind    docKind
	content string
}

// parseDocComment extracts the kind and content of a doc comment group. It
// returns ok=false when g is nil/empty or not a doc comment.
func parseDocComment(g *ast.CommentGroup) (d docComment, ok bool) {
	if g == nil || len(g.List) == 0 {
		return docComment{}, false
	}
	first := g.List[0].Text
	switch {
	case strings.HasPrefix(first, "/???"):
		return docComment{docRoot, blockDocContent(first, "???")}, true
	case strings.HasPrefix(first, "/??"):
		return docComment{docBlock, blockDocContent(first, "??")}, true
	case strings.HasPrefix(first, "/?"):
		lines := make([]string, len(g.List))
		for i, c := range g.List {
			lines[i] = strings.TrimPrefix(strings.TrimPrefix(c.Text, "/?"), " ")
		}
		return docComment{docSingle, strings.Join(lines, "\n")}, true
	}
	return docComment{}, false
}

// blockDocContent returns the inner text of a fenced block doc, dropping the
// opening `/<fence>` line and the closing `<fence>` line.
func blockDocContent(text, fence string) string {
	body := strings.TrimPrefix(text, "/")
	body = strings.TrimPrefix(body, fence)
	body = strings.TrimSuffix(body, fence)
	return strings.Trim(body, "\n")
}

// renderDocLines renders d as the formatted doc lines (without the leading
// prefix on the first line; callers indent continuation lines). width is the
// available column budget at the doc's indentation. A SINGLE/BLOCK doc is
// rendered as `/? …` when its content reflows to a single line that fits, else
// as a `/??` … `??` block; a ROOT_BLOCK always stays a `/???` … `???` block.
func renderDocLines(d docComment, width int) []string {
	if width < 8 {
		width = 8
	}
	switch d.kind {
	case docRoot:
		return wrapDocBlock("???", reflowMarkdown(d.content, width))
	default: // docSingle, docBlock
		body := reflowMarkdown(d.content, width-len("/? "))
		if len(body) == 1 && len("/? ")+len(body[0]) <= width {
			return []string{"/? " + body[0]}
		}
		return wrapDocBlock("??", reflowMarkdown(d.content, width))
	}
}

// wrapDocBlock wraps body lines in `/<fence>` … `<fence>` fence lines.
func wrapDocBlock(fence string, body []string) []string {
	out := make([]string, 0, len(body)+2)
	out = append(out, "/"+fence)
	out = append(out, body...)
	out = append(out, fence)
	return out
}

// reflowMarkdown reflows plain Markdown paragraphs to width: soft-wrapped lines
// of a paragraph are joined and greedily re-wrapped. Fenced code blocks
// (```), list items, ATX headings, blockquotes and table rows pass through
// line-for-line so their structure is preserved.
func reflowMarkdown(content string, width int) []string {
	var (
		out     []string
		para    []string
		inFence bool
	)
	flush := func() {
		if len(para) == 0 {
			return
		}
		out = append(out, wrapWords(strings.Fields(strings.Join(para, " ")), width)...)
		para = nil
	}
	for _, ln := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(ln)
		switch {
		case strings.HasPrefix(trimmed, "```"):
			flush()
			inFence = !inFence
			out = append(out, ln)
		case inFence:
			out = append(out, ln)
		case trimmed == "":
			flush()
			out = append(out, "")
		case isMarkdownStructural(trimmed):
			flush()
			out = append(out, ln)
		default:
			para = append(para, trimmed)
		}
	}
	flush()
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	if len(out) == 0 {
		out = append(out, "")
	}
	return out
}

// isMarkdownStructural reports whether a trimmed line is a structural Markdown
// element (list item, heading, blockquote or table row) that must not be joined
// into a paragraph.
func isMarkdownStructural(t string) bool {
	switch {
	case strings.HasPrefix(t, "- "), strings.HasPrefix(t, "* "), strings.HasPrefix(t, "+ "),
		strings.HasPrefix(t, "> "), strings.HasPrefix(t, "#"), strings.HasPrefix(t, "|"):
		return true
	}
	// ordered list: "<digits>. "
	i := 0
	for i < len(t) && t[i] >= '0' && t[i] <= '9' {
		i++
	}
	return i > 0 && i < len(t) && t[i] == '.' &&
		i+1 < len(t) && t[i+1] == ' '
}

// wrapWords greedily packs words into lines no wider than width (each word kept
// whole even if it alone exceeds width).
func wrapWords(words []string, width int) []string {
	if width < 1 {
		width = 1
	}
	var (
		lines []string
		cur   string
	)
	for _, w := range words {
		switch {
		case cur == "":
			cur = w
		case len(cur)+1+len(w) <= width:
			cur += " " + w
		default:
			lines = append(lines, cur)
			cur = w
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	if len(lines) == 0 {
		lines = append(lines, "")
	}
	return lines
}
