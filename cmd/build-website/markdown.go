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
	boldRe = regexp.MustCompile(`\*\*(.+?)\*\*`)
	// italicRe matches `_emphasis_` only when the underscores are flanked by a
	// non-word character (or line boundary), per CommonMark's intra-word rule:
	// underscores inside an identifier or filename (01_hello, lang-02_values.html)
	// are literal, never emphasis. The flanking chars are captured ($1,$3) and
	// re-emitted. Without this, sample links were mangled into <em> spans.
	italicRe = regexp.MustCompile(`(^|[^0-9A-Za-z_])_([^_]+?)_([^0-9A-Za-z_]|$)`)
	linkRe   = regexp.MustCompile(`\[(.+?)\]\(([^)]+)\)`)
)

// backtickRun returns the number of leading backticks in s (0 if it does not
// start with one).
func backtickRun(s string) int {
	n := 0
	for n < len(s) && s[n] == '`' {
		n++
	}
	return n
}

// isClosingFence reports whether line closes a fenced code block opened with
// `open` backticks: per CommonMark the closer is a run of at least `open`
// backticks followed only by whitespace (no info string). A shorter run, or a
// run trailed by other text (e.g. an inner ```gad opener), is content.
func isClosingFence(line string, open int) bool {
	t := strings.TrimRight(line, " \t")
	n := backtickRun(t)
	return n >= open && n == len(t)
}

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

		// Fenced code block. CommonMark: the opening fence is a run of >= 3
		// backticks and closes only on a line that is a run of AT LEAST as many
		// backticks (with nothing but whitespace after). Shorter runs inside — a
		// ```gad doctest embedded in the fenced source — are literal content, not
		// a close. Tracking the opening width is what keeps a wider outer fence
		// from being closed early by an inner ``` (see cmd/gad fenceFor).
		if open := backtickRun(strings.TrimRight(line, " ")); open >= 3 {
			lang := strings.TrimSpace(line[open:])
			var code []string
			i++
			for i < len(lines) && !isClosingFence(lines[i], open) {
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

// renderInline renders inline markup. Links are resolved at the OUTERMOST level,
// walking the string in segments, so a link whose text is itself a code span or
// emphasis (e.g. the "Sample source" column's [`samples/NN.gad`](…) links) is
// parsed as one link — its text and the surrounding prose are then rendered for
// code spans + emphasis. Resolving links first also protects link destinations
// (which routinely contain intra-word underscores, lang-01_hello.html) from the
// emphasis pass.
func renderInline(s string) string {
	var b strings.Builder
	last := 0
	for _, loc := range linkRe.FindAllStringSubmatchIndex(s, -1) {
		b.WriteString(inlineSpans(s[last:loc[0]]))
		text, url := s[loc[2]:loc[3]], s[loc[4]:loc[5]]
		b.WriteString(`<a href="` + rewriteLink(url) + `">` + inlineSpans(text) + "</a>")
		last = loc[1]
	}
	b.WriteString(inlineSpans(s[last:]))
	return b.String()
}

// inlineSpans renders code spans and emphasis on a run of text that contains no
// links. Backtick code spans are literal (escaped, no emphasis); the text
// between/around them is HTML-escaped and processed for bold/italic.
func inlineSpans(s string) string {
	var b strings.Builder
	for len(s) > 0 {
		if i := strings.IndexByte(s, '`'); i >= 0 {
			if j := strings.IndexByte(s[i+1:], '`'); j >= 0 {
				b.WriteString(emphasize(htmlEscape(s[:i])))
				b.WriteString("<code>" + htmlEscape(s[i+1:i+1+j]) + "</code>")
				s = s[i+1+j+1:]
				continue
			}
		}
		b.WriteString(emphasize(htmlEscape(s)))
		break
	}
	return b.String()
}

// emphasize applies bold/italic markup to an already-escaped run of text.
func emphasize(s string) string {
	s = boldRe.ReplaceAllString(s, "<strong>$1</strong>")
	s = italicRe.ReplaceAllString(s, "${1}<em>${2}</em>${3}")
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
	// Any link into the samples tree — a rendered `samples/NN.md`, or a raw
	// source `../samples/NN.{gad,gadt,gadx}` from doc/README's "Sample source"
	// column — resolves to the published chapter page lang-<name>.html (whose
	// Example section carries the full runnable source). Raw .gad/.gadt/.gadx
	// files are not published, so without this those links would 404.
	if p := strings.TrimPrefix(url, "../"); strings.HasPrefix(p, "samples/") {
		name := strings.TrimPrefix(p, "samples/")
		for _, ext := range []string{".gadt", ".gadx", ".gad", ".md"} {
			if strings.HasSuffix(name, ext) {
				return "lang-" + strings.TrimSuffix(name, ext) + ".html" + frag
			}
		}
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
	s = italicRe.ReplaceAllString(s, "${1}${2}${3}")
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
