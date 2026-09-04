// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package main

import (
	"html"
	"regexp"
	"strings"
)

// isHTMLFile reports whether name is an HTML document that `transpile` lifts
// into a Gadx template.
func isHTMLFile(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".html") || strings.HasSuffix(lower, ".htm")
}

var (
	rgxHTML5Doctype = regexp.MustCompile(`(?i)^<!DOCTYPE\s+html\s*>$`)
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
		indent = "    "
	)
	b.WriteString("@main\n")

	src = strings.ReplaceAll(src, "\r\n", "\n")
	for _, line := range strings.Split(src, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			b.WriteString("\n")
			continue
		}
		if !inRaw && rgxHTML5Doctype.MatchString(trimmed) {
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
	}
	return b.String()
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
