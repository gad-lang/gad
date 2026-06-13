package main

import (
	"regexp"
	"strings"
)

// A small, dependency-free Markdown renderer covering the subset used by the
// Gad docs: ATX headings, fenced code blocks, tables, ordered/unordered lists,
// blockquotes, horizontal rules, paragraphs, and inline code/bold/italic/links.
// Link targets ending in .md are rewritten to .html (README.md -> index.html).

// Heading is a rendered heading, used to build the page TOC and search index.
type Heading struct {
	Level int
	Text  string
	ID    string
}

var (
	boldRe   = regexp.MustCompile(`\*\*(.+?)\*\*`)
	italicRe = regexp.MustCompile(`_([^_]+)_`)
	linkRe   = regexp.MustCompile(`\[(.+?)\]\(([^)]+)\)`)
)

// renderMarkdown converts src to an HTML body and returns the headings found.
func renderMarkdown(src string) (string, []Heading) {
	lines := strings.Split(src, "\n")
	var (
		b        strings.Builder
		headings []Heading
		seen     = map[string]int{}
	)

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Fenced code block.
		if fence := strings.TrimRight(line, " "); strings.HasPrefix(fence, "```") {
			lang := strings.TrimSpace(strings.TrimPrefix(fence, "```"))
			var code []string
			i++
			for i < len(lines) && !strings.HasPrefix(strings.TrimRight(lines[i], " "), "```") {
				code = append(code, lines[i])
				i++
			}
			cls := ""
			if lang != "" {
				cls = ` class="language-` + htmlEscape(lang) + `"`
			}
			b.WriteString("<pre><code" + cls + ">")
			b.WriteString(htmlEscape(strings.Join(code, "\n")))
			b.WriteString("</code></pre>\n")
			continue
		}

		// Heading.
		if m := headingLevel(line); m > 0 {
			text := strings.TrimSpace(line[m:])
			id := slug(text, seen)
			headings = append(headings, Heading{Level: m, Text: stripInline(text), ID: id})
			b.WriteString("<h" + itoa(m) + ` id="` + id + `">` + renderInline(text) +
				"</h" + itoa(m) + ">\n")
			continue
		}

		// Table.
		if isTableRow(line) && i+1 < len(lines) && isTableSep(lines[i+1]) {
			i = renderTable(&b, lines, i) // returns index of last consumed line
			continue
		}

		// Horizontal rule.
		if t := strings.TrimSpace(line); t == "---" || t == "***" || t == "___" {
			b.WriteString("<hr>\n")
			continue
		}

		// Blockquote.
		if strings.HasPrefix(line, ">") {
			var quote []string
			for i < len(lines) && strings.HasPrefix(lines[i], ">") {
				quote = append(quote, strings.TrimPrefix(strings.TrimPrefix(lines[i], ">"), " "))
				i++
			}
			i--
			b.WriteString("<blockquote>" + renderInline(strings.Join(quote, " ")) + "</blockquote>\n")
			continue
		}

		// List (unordered or ordered).
		if isListItem(line) {
			i = renderList(&b, lines, i)
			continue
		}

		// Blank line.
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Paragraph: gather until a blank line or a block start.
		var para []string
		for i < len(lines) && strings.TrimSpace(lines[i]) != "" &&
			!strings.HasPrefix(lines[i], "```") && headingLevel(lines[i]) == 0 &&
			!isListItem(lines[i]) && !strings.HasPrefix(lines[i], ">") {
			para = append(para, lines[i])
			i++
		}
		i--
		b.WriteString("<p>" + renderInline(strings.Join(para, " ")) + "</p>\n")
	}

	return b.String(), headings
}

func headingLevel(line string) int {
	n := 0
	for n < len(line) && line[n] == '#' {
		n++
	}
	if n >= 1 && n <= 6 && n < len(line) && line[n] == ' ' {
		return n
	}
	return 0
}

func isListItem(line string) bool {
	t := strings.TrimLeft(line, " ")
	if strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* ") {
		return true
	}
	return orderedItemRe.MatchString(t)
}

var orderedItemRe = regexp.MustCompile(`^\d+\. `)

// renderList renders a (possibly nested) list starting at lines[start] and
// returns the index of the last consumed line.
func renderList(b *strings.Builder, lines []string, start int) int {
	ordered := orderedItemRe.MatchString(strings.TrimLeft(lines[start], " "))
	baseIndent := indentOf(lines[start])
	if ordered {
		b.WriteString("<ol>\n")
	} else {
		b.WriteString("<ul>\n")
	}
	i := start
	for i < len(lines) && isListItem(lines[i]) && indentOf(lines[i]) >= baseIndent {
		if indentOf(lines[i]) > baseIndent {
			// Nested list belongs to the previous item; render inline.
			// renderList returns the last consumed index, so advance past it.
			i = renderList(b, lines, i) + 1
			continue
		}
		content := stripBullet(strings.TrimLeft(lines[i], " "))
		b.WriteString("<li>" + renderInline(content) + "</li>\n")
		i++
	}
	if ordered {
		b.WriteString("</ol>\n")
	} else {
		b.WriteString("</ul>\n")
	}
	return i - 1
}

func indentOf(line string) int {
	n := 0
	for n < len(line) && line[n] == ' ' {
		n++
	}
	return n
}

func stripBullet(t string) string {
	if strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* ") {
		return t[2:]
	}
	return orderedItemRe.ReplaceAllString(t, "")
}

func isTableRow(line string) bool {
	t := strings.TrimSpace(line)
	return strings.HasPrefix(t, "|") && strings.HasSuffix(t, "|")
}

func isTableSep(line string) bool {
	t := strings.TrimSpace(line)
	if !isTableRow(line) {
		return false
	}
	return strings.Trim(t, "|-: ") == ""
}

func renderTable(b *strings.Builder, lines []string, start int) int {
	header := splitRow(lines[start])
	i := start + 2 // skip header + separator
	b.WriteString("<table>\n<thead><tr>")
	for _, c := range header {
		b.WriteString("<th>" + renderInline(c) + "</th>")
	}
	b.WriteString("</tr></thead>\n<tbody>\n")
	for i < len(lines) && isTableRow(lines[i]) {
		b.WriteString("<tr>")
		for _, c := range splitRow(lines[i]) {
			b.WriteString("<td>" + renderInline(c) + "</td>")
		}
		b.WriteString("</tr>\n")
		i++
	}
	b.WriteString("</tbody>\n</table>\n")
	return i - 1
}

func splitRow(line string) []string {
	t := strings.TrimSpace(line)
	t = strings.TrimPrefix(t, "|")
	t = strings.TrimSuffix(t, "|")
	parts := strings.Split(t, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// renderInline renders inline markup, treating backtick code spans literally.
func renderInline(s string) string {
	var b strings.Builder
	for len(s) > 0 {
		if i := strings.IndexByte(s, '`'); i >= 0 {
			if j := strings.IndexByte(s[i+1:], '`'); j >= 0 {
				b.WriteString(renderInlineNoCode(s[:i]))
				b.WriteString("<code>" + htmlEscape(s[i+1:i+1+j]) + "</code>")
				s = s[i+1+j+1:]
				continue
			}
		}
		b.WriteString(renderInlineNoCode(s))
		break
	}
	return b.String()
}

func renderInlineNoCode(s string) string {
	s = htmlEscape(s)
	s = boldRe.ReplaceAllString(s, "<strong>$1</strong>")
	s = italicRe.ReplaceAllString(s, "<em>$1</em>")
	s = linkRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := linkRe.FindStringSubmatch(m)
		return `<a href="` + rewriteLink(sub[2]) + `">` + sub[1] + "</a>"
	})
	return s
}

// rewriteLink maps doc-relative .md links to .html for the generated site.
func rewriteLink(url string) string {
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") ||
		strings.HasPrefix(url, "#") {
		return url
	}
	frag := ""
	if i := strings.IndexByte(url, '#'); i >= 0 {
		frag = url[i:]
		url = url[:i]
	}
	switch {
	case strings.EqualFold(url, "README.md"):
		url = "index.html"
	case strings.HasSuffix(url, ".md"):
		url = strings.TrimSuffix(url, ".md") + ".html"
	}
	return url + frag
}

// stripInline removes markup to produce plain text (for the search index/TOC).
func stripInline(s string) string {
	s = strings.ReplaceAll(s, "`", "")
	s = boldRe.ReplaceAllString(s, "$1")
	s = italicRe.ReplaceAllString(s, "$1")
	s = linkRe.ReplaceAllString(s, "$1")
	return s
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func slug(text string, seen map[string]int) string {
	t := strings.ToLower(stripInline(text))
	var b strings.Builder
	for _, r := range t {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			b.WriteByte('-')
		}
	}
	base := strings.Trim(b.String(), "-")
	if base == "" {
		base = "section"
	}
	if n, ok := seen[base]; ok {
		seen[base] = n + 1
		return base + "-" + itoa(n+1)
	}
	seen[base] = 0
	return base
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	neg := i < 0
	if neg {
		i = -i
	}
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
