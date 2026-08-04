package parser

import (
	"strings"

	gnode "github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
)

// sentinel is the single byte a gadx block is collapsed to inside the rewritten
// HTML region. It is one byte so the rewrite is length-preserving (the rest of
// the block's span is overwritten with spaces), which keeps every following
// byte at its original offset — so HTML interpolation positions stay exact.
const sentinel = '\x00'

// htmlSubError carries an error from a recursively parsed inline gadx block back
// to the enclosing parser, so it can be reported at the right source position.
type htmlSubError struct {
	pos source.Pos
	msg string
}

// srcLine is one physical line of the raw HTML region with its byte offset
// (relative to the region start) and its verbatim text (without the newline).
type srcLine struct {
	start int
	text  string
}

// splitLines splits raw into physical lines, preserving each line's start offset.
func splitLines(raw string) []srcLine {
	var lines []srcLine
	start := 0
	for i := 0; i < len(raw); i++ {
		if raw[i] == '\n' {
			lines = append(lines, srcLine{start: start, text: raw[start:i]})
			start = i + 1
		}
	}
	lines = append(lines, srcLine{start: start, text: raw[start:]})
	return lines
}

// leadWidth returns the number of leading space/tab bytes in t.
func leadWidth(t string) int {
	i := 0
	for i < len(t) && (t[i] == ' ' || t[i] == '\t') {
		i++
	}
	return i
}

// isGadxMarker reports whether a line whose leading whitespace has been stripped
// (rest) begins a block-level gadx statement that may be interleaved inside an
// HTML region: a keyword (`@if`/`@for`/…), inline code (`~`), or a component
// call (`+name`). Text/id/class shorthands are intentionally excluded — inside
// HTML they read as literal content.
func isGadxMarker(rest string) bool {
	if rest == "" {
		return false
	}
	switch rest[0] {
	case '@', '~':
		return true
	case '+':
		return len(rest) > 1 && isTagNameChar(rest[1])
	}
	return false
}

// isGadxContinuation reports whether a same-indent line continues the preceding
// gadx block rather than starting a new one — i.e. an `@else` / `@else if`
// clause of an enclosing `@if`/`@for`.
func isGadxContinuation(rest string) bool {
	return strings.HasPrefix(rest, "@else")
}

// rewriteGadxBlocks finds block-level gadx statements interleaved in a raw HTML
// region (a directive line plus its more-indented body) and replaces each with a
// one-byte sentinel (padded with spaces to preserve length and thus offsets).
// Every extracted block is parsed with the real gadx parser — so it may itself
// contain `@if`/`@for`/nested HTML — and the resulting statement lists are
// returned in source order, to be spliced back in as siblings by parseNodes.
func rewriteGadxBlocks(raw string, base source.Pos) (string, []gnode.Stmts, []htmlSubError) {
	lines := splitLines(raw)
	buf := []byte(raw)
	var (
		blocks []gnode.Stmts
		errs   []htmlSubError
	)
	for li := 0; li < len(lines); li++ {
		rest := strings.TrimLeft(lines[li].text, " \t")
		if !isGadxMarker(rest) {
			continue
		}
		gIndent := leadWidth(lines[li].text)

		// Gather the block: following lines indented deeper than the directive
		// (its body), plus `@else`/`@else if` continuations at the same indent
		// (blank lines are included tentatively, then trimmed off the tail).
		last := li
		for j := li + 1; j < len(lines); j++ {
			if strings.TrimSpace(lines[j].text) == "" {
				continue
			}
			w := leadWidth(lines[j].text)
			if w > gIndent {
				last = j
				continue
			}
			if w == gIndent && isGadxContinuation(strings.TrimLeft(lines[j].text, " \t")) {
				last = j
				continue
			}
			break
		}

		// Dedent the block by the directive's indentation and parse it.
		var sb strings.Builder
		for j := li; j <= last; j++ {
			t := lines[j].text
			k := 0
			for k < len(t) && k < gIndent && (t[k] == ' ' || t[k] == '\t') {
				k++
			}
			if j > li {
				sb.WriteByte('\n')
			}
			sb.WriteString(t[k:])
		}
		subBase := base + source.Pos(lines[li].start+gIndent)
		stmts, err := subParseGadx(sb.String(), subBase)
		if err != nil {
			errs = append(errs, htmlSubError{pos: subBase, msg: err.Error()})
		}

		// Collapse the block's byte span to a single sentinel + spaces, keeping
		// the region length unchanged so later offsets stay aligned.
		blockStart := lines[li].start
		blockEnd := lines[last].start + len(lines[last].text)
		for x := blockStart; x < blockEnd; x++ {
			buf[x] = ' '
		}
		buf[blockStart] = sentinel

		blocks = append(blocks, stmts)
		li = last
	}
	return string(buf), blocks, errs
}

// subParseGadx parses an inline gadx block (already dedented) with the gadx
// parser, aligning the fragment file's base to base so node positions resolve
// back onto the original .gadx source (line-accurate; columns land at the start
// of the dedented line).
func subParseGadx(text string, base source.Pos) (gnode.Stmts, error) {
	fs := source.NewFileSet()
	fbase := -1
	if base != noBase && int(base) >= fs.Base {
		fbase = int(base)
	}
	f := fs.AddFileData("(gadx-inline)", fbase, []byte(text))
	file, err := NewParser(f).ParseFile()
	if err != nil {
		return nil, err
	}
	return file.Stmts, nil
}
