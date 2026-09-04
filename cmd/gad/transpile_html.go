// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

// htmlInlineElements render in the line box, so whitespace beside them is part
// of the text: the space in `<b>a</b> <i>b</i>` is what separates the two words.
// Between anything else — a block, a table cell, a list item — that whitespace
// only exists because the source was indented, and a browser lays the boxes out
// the same without it.
var htmlInlineElements = map[string]bool{
	"a": true, "abbr": true, "b": true, "bdi": true, "bdo": true, "br": true,
	"cite": true, "code": true, "data": true, "dfn": true, "em": true,
	"i": true, "img": true, "kbd": true, "label": true, "mark": true,
	"q": true, "rp": true, "rt": true, "ruby": true, "s": true, "samp": true,
	"small": true, "span": true, "strong": true, "sub": true, "sup": true,
	"time": true, "u": true, "var": true, "wbr": true,
}

var (
	// RE2 has no backreference, so the two raw-text elements are spelled out.
	rgxRawRegion = regexp.MustCompile(
		`(?is)<script\b[^>]*>.*?</script\s*>|<style\b[^>]*>.*?</style\s*>`)
	rgxTagBefore  = regexp.MustCompile(`(?s)<(/?)([a-zA-Z][-\w]*)[^<>]*>\s*$`)
	rgxTagAfter   = regexp.MustCompile(`^<(/?)([a-zA-Z][-\w]*)`)
	rgxTagGapOnly = regexp.MustCompile(`(?s)>[ \t\r\n]+<`)
	rgxPreRegion  = regexp.MustCompile(
		`(?is)<pre\b[^>]*>.*?</pre\s*>|<textarea\b[^>]*>.*?</textarea\s*>`)
	rgxOpenEdge  = regexp.MustCompile(`(?s)<([a-zA-Z][-\w]*)([^<>]*)>[ \t\r\n]+`)
	rgxCloseEdge = regexp.MustCompile(`(?s)[ \t\r\n]+</([a-zA-Z][-\w]*)[ \t]*>`)
)

// htmlPreserveElements lay their content out as written, so no whitespace in
// them may be touched.
var htmlPreserveElements = map[string]bool{"pre": true, "textarea": true}

// htmlVoidElements have no content of their own, so they have no inner edges to
// trim — whatever follows one is its sibling, not something inside it.
var htmlVoidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true,
	"hr": true, "img": true, "input": true, "link": true, "meta": true,
	"param": true, "source": true, "track": true, "wbr": true,
}

// blockEdged reports whether an element's inner edges may be trimmed: a block
// with content of its own, laid out normally.
func blockEdged(name string) bool {
	name = strings.ToLower(name)
	return !htmlInlineElements[name] && !htmlPreserveElements[name] &&
		!htmlVoidElements[name]
}

// dropLayoutWhitespace removes the whitespace between two tags when neither is
// an inline element.
//
// That whitespace is a text node, so once it reaches the template Gadx has to
// keep it — it is what separates two words in `<b>a</b> <i>b</i>`. Between
// blocks it exists only because the source was indented, and carrying it would
// put a `{= " " }` line between every pair of tags in the file.
//
// Script and stylesheet content is code, so it is held aside under a stand-in
// while the rule runs and put back after: that way the gaps *around* those
// elements are considered like any other, without the rule ever looking inside
// them.
func dropLayoutWhitespace(src string) string {
	var raws []string
	mask := func(trim bool) func(string) string {
		return func(m string) string {
			if trim {
				m = trimRawEdges(m)
			}
			raws = append(raws, m)
			return fmt.Sprintf("<gad-raw-%d></gad-raw-%d>", len(raws)-1, len(raws)-1)
		}
	}
	masked := rgxRawRegion.ReplaceAllStringFunc(src, mask(true))
	// A `<pre>` is held aside untouched: the gaps inside it are content, and
	// so is the whitespace at its edges.
	masked = rgxPreRegion.ReplaceAllStringFunc(masked, mask(false))

	masked = dropEdges(dropGaps(masked))

	for i, raw := range raws {
		masked = strings.Replace(masked,
			fmt.Sprintf("<gad-raw-%d></gad-raw-%d>", i, i), raw, 1)
	}
	return masked
}

// trimRawEdges strips a `<script>` or `<style>` body of the indentation the
// page put around it: the line break that follows the open tag, the whitespace
// before the closing one, and the indentation common to every line between.
//
// None of it is part of the language inside — it sits outside every token, and
// is there because the element itself was indented. It has to go for the body
// to be written under its tag: a body opening with a blank line has no
// indentation to read, and one carrying the page's own would gain the tag's on
// top of it. What the lines have *beyond* the common prefix is the code's own
// nesting, and that is kept.
func trimRawEdges(region string) string {
	open := strings.IndexByte(region, '>')
	close := strings.LastIndexByte(region, '<')
	if open < 0 || close <= open {
		return region
	}
	body := strings.TrimLeft(strings.TrimRight(region[open+1:close], " \t\r\n"), "\r\n")
	return region[:open+1] + dedentLines(body) + region[close:]
}

// dedentLines removes the indentation common to every non-blank line.
func dedentLines(s string) string {
	lines := strings.Split(s, "\n")
	prefix, first := "", true
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		lead := l[:len(l)-len(strings.TrimLeft(l, " \t"))]
		if first {
			prefix, first = lead, false
			continue
		}
		n := 0
		for n < len(prefix) && n < len(lead) && prefix[n] == lead[n] {
			n++
		}
		prefix = prefix[:n]
	}
	for i, l := range lines {
		lines[i] = strings.TrimPrefix(l, prefix)
	}
	return strings.Join(lines, "\n")
}

// dropEdges removes the whitespace just inside a block element — right after
// its open tag and right before its close tag.
//
// A browser drops it too: whitespace at the edges of a block box is trimmed
// when the lines are laid out, so `<p>\n    Hello\n</p>` and `<p>Hello</p>`
// render identically. Kept, it would have to be carried through the template
// as `{= " Hello " }`, since a text line strips its own edges — a whole
// paragraph written as a quoted literal for whitespace that never shows.
//
// Inline elements keep theirs, because there the space is what separates two
// words. So do `<pre>` and `<textarea>`, whose content is laid out as written,
// and void elements, which have no inside.
func dropEdges(src string) string {
	src = rgxOpenEdge.ReplaceAllStringFunc(src, func(m string) string {
		g := rgxOpenEdge.FindStringSubmatch(m)
		if !blockEdged(g[1]) {
			return m
		}
		return "<" + g[1] + g[2] + ">"
	})
	return rgxCloseEdge.ReplaceAllStringFunc(src, func(m string) string {
		g := rgxCloseEdge.FindStringSubmatch(m)
		if !blockEdged(g[1]) {
			return m
		}
		return "</" + g[1] + ">"
	})
}

// dropGaps removes every gap whose two sides are non-inline elements.
func dropGaps(src string) string {
	var b strings.Builder
	last := 0
	for _, m := range rgxTagGapOnly.FindAllStringIndex(src, -1) {
		gapStart, gapEnd := m[0]+1, m[1]-1 // between the `>` and the `<`
		if inlineBeside(src[:gapStart], src[gapEnd:]) {
			continue
		}
		b.WriteString(src[last:gapStart])
		last = gapEnd
	}
	b.WriteString(src[last:])
	return b.String()
}

// inlineBeside reports whether either side of a gap is an inline element, in
// which case the whitespace between them is content.
func inlineBeside(before, after string) bool {
	if m := rgxTagBefore.FindStringSubmatch(before); m != nil && htmlInlineElements[strings.ToLower(m[2])] {
		return true
	}
	if m := rgxTagAfter.FindStringSubmatch(after); m != nil && htmlInlineElements[strings.ToLower(m[2])] {
		return true
	}
	return false
}

// isHTMLFile reports whether name is an HTML document that `transpile` lifts
// into a Gadx template.
func isHTMLFile(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".html") || strings.HasSuffix(lower, ".htm")
}

// doctypeMark stands in for the doctype between the lift's two passes.
const doctypeMark = "\x00gad-doctype\x00"

var (
	rgxHTML5Doctype = regexp.MustCompile(`(?im)^\s*<!DOCTYPE\s+html\s*>\s*$`)
	rgxRawTextOpen  = regexp.MustCompile(`(?i)<(script|style)\b[^>]*>`)
	rgxRawTextClose = regexp.MustCompile(`(?i)</(script|style)\s*>`)
)

// htmlToGadx lifts an HTML document into a Gadx template: the markup becomes the
// body of a `@main` component, indented under it.
//
// Two things are rewritten on the way, and nothing else — the point is that the
// result renders the page it came from:
//
//   - The HTML5 doctype becomes the `!!! 5` statement. Written as markup it is a
//     declaration, and Gadx drops declarations, so the page would lose it.
//   - A `{` outside a script or a stylesheet is escaped. In Gadx it opens an
//     interpolation; in HTML text it is just a brace. Script and stylesheet
//     content is raw, so braces there are left as they are.
//   - The whitespace between two tags that are not inline elements is dropped.
//     It is a text node, so the template would have to keep it, and a page
//     indented over hundreds of lines would carry one for every pair of tags.
//     Beside an inline element it separates words, and is left alone.
//   - A character entity becomes the character it names. Gadx escapes what it
//     writes, so an entity left as it is would go out with its `&` escaped and
//     the reader would see "&copy;" instead of ©. `&lt;` and `&gt;` are the
//     exception: decoded, they would turn into markup.
//
// Indentation is preserved and shifted by one level: inside an HTML region it
// carries no meaning to Gadx, and keeping it keeps the file readable. The
// content of a script or a stylesheet is left where it is, since that text goes
// out verbatim and shifting it would shift the rendered page.
func htmlToGadx(src string) string {
	var (
		b      strings.Builder
		inRaw  bool
		inPre  int
		indent = "    "
	)
	b.WriteString("@main\n")

	// The doctype is lifted before the gaps are closed: closing them first
	// would glue it to the next tag and it would no longer be a line of its own.
	src = rgxHTML5Doctype.ReplaceAllString(strings.ReplaceAll(src, "\r\n", "\n"), doctypeMark)
	src = dropLayoutWhitespace(src)
	for _, line := range strings.Split(src, "\n") {
		// Inside a `<pre>` or a `<textarea>` the line is laid out as written, so
		// it goes out at its own indentation: shifting it would shift the page.
		// Its content is still HTML — entities decode, braces are escaped —
		// which is what separates it from a script's or a stylesheet's.
		if inPre > 0 && !inRaw {
			out, endsInRaw := convertLine(line, false)
			b.WriteString(out + "\n")
			inRaw = endsInRaw
			inPre += preDepthDelta(line)
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			b.WriteString("\n")
			continue
		}
		if !inRaw && trimmed == doctypeMark {
			b.WriteString(indent + "!!! 5\n")
			continue
		}

		out, endsInRaw := convertLine(line, inRaw)
		if inRaw {
			// Already inside a script or a stylesheet: its content is written
			// out verbatim, so shifting it would shift the rendered page too.
			b.WriteString(out + "\n")
		} else {
			b.WriteString(indent + out + "\n")
		}
		inRaw = endsInRaw
		if !inRaw {
			inPre += preDepthDelta(line)
		}
	}
	return b.String()
}

var rgxPreTag = regexp.MustCompile(`(?i)<(/?)(pre|textarea)\b`)

// preDepthDelta reports how many preserve-whitespace elements a line opens
// minus how many it closes, so the lift knows when it is inside one.
func preDepthDelta(line string) int {
	d := 0
	for _, m := range rgxPreTag.FindAllStringSubmatch(line, -1) {
		if m[1] == "/" {
			d--
		} else {
			d++
		}
	}
	return d
}

// rgxEntity matches a character entity. `&lt;` and `&gt;` are left out on
// purpose: decoding them would produce markup where the document meant text.
var rgxEntity = regexp.MustCompile(`&(?:#[0-9]+|#[xX][0-9a-fA-F]+|[a-zA-Z][a-zA-Z0-9]*);`)

// decodeEntities replaces the character entities of s with the characters they
// name, leaving `&lt;` and `&gt;` alone.
func decodeEntities(s string) string {
	return rgxEntity.ReplaceAllStringFunc(s, func(e string) string {
		switch strings.ToLower(e) {
		case "&lt;", "&gt;":
			return e
		}
		if d := html.UnescapeString(e); d != e {
			return d
		}
		return e
	})
}

// convertLine rewrites one line for Gadx: entities become characters and the
// braces Gadx would read as an interpolation are escaped. It reports whether
// the line ends inside a raw-text element; inRaw says whether it began inside
// one, where both rewrites are off because the content is code.
func convertLine(line string, inRaw bool) (out string, endsInRaw bool) {
	var b strings.Builder
	for i := 0; i < len(line); {
		if inRaw {
			// Inside a script or a stylesheet: braces are content. Only the close
			// tag matters, and only to leave raw mode.
			if m := rgxRawTextClose.FindStringIndex(line[i:]); m != nil && m[0] == 0 {
				b.WriteString(line[i : i+m[1]])
				i += m[1]
				inRaw = false
				continue
			}
			b.WriteByte(line[i])
			i++
			continue
		}
		if m := rgxRawTextOpen.FindStringIndex(line[i:]); m != nil && m[0] == 0 {
			b.WriteString(line[i : i+m[1]])
			i += m[1]
			inRaw = true
			continue
		}
		if c := line[i]; c == '{' || c == '}' {
			b.WriteByte('\\')
			b.WriteByte(c)
			i++
			continue
		}
		if m := rgxEntity.FindStringIndex(line[i:]); m != nil && m[0] == 0 {
			b.WriteString(decodeEntities(line[i : i+m[1]]))
			i += m[1]
			continue
		}
		b.WriteByte(line[i])
		i++
	}
	return b.String(), inRaw
}
