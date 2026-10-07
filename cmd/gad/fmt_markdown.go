package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// fenceOpen is the line that opens a fenced code block of Gad in Markdown:
// its indentation, its fence (``` or ~~~, three or more) and its language —
// gad, gadt, gadx —, what follows the language kept.
var fenceOpen = regexp.MustCompile("^([ \\t]*)(`{3,}|~{3,})[ \\t]*(gad|gadt|gadx)\\b(.*)$")

// formatMarkdown formats the fenced code blocks of Gad in a Markdown file —
// ```gad, ```gadt, ```gadx —, each as a file of its dialect; the rest is
// left as it is. A block indented (in a list) keeps its indentation. A block
// that does not format (a fragment, an example of code refused on purpose)
// is left as it is, said on the standard error; the file is still formatted.
func (o *fmtOptions) formatMarkdown(name string, src []byte) (string, error) {
	lines := strings.SplitAfter(string(src), "\n")
	var out strings.Builder
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		m := fenceOpen.FindStringSubmatch(strings.TrimRight(line, "\r\n"))
		if m == nil {
			out.WriteString(line)
			continue
		}
		indent, fence, lang := m[1], m[2], m[3]
		// the block's lines, up to its closing fence (the same character,
		// as long or longer, alone on its line)
		end := -1
		for j := i + 1; j < len(lines); j++ {
			t := strings.TrimSpace(lines[j])
			if strings.HasPrefix(t, fence) && strings.Trim(t, fence[:1]) == "" {
				end = j
				break
			}
		}
		if end < 0 {
			// not closed: the rest is the block's, left as it is
			for ; i < len(lines); i++ {
				out.WriteString(lines[i])
			}
			break
		}
		var code strings.Builder
		for _, l := range lines[i+1 : end] {
			code.WriteString(strings.TrimPrefix(l, indent))
		}
		out.WriteString(line)
		formatted, err := o.formatSource(fmt.Sprintf("%s:%d.%s", name, i+1, lang), []byte(code.String()), false)
		if err != nil || strings.TrimSpace(code.String()) == "" {
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s:%d: a ```%s block left as it is: %v\n", name, i+1, lang, err)
			}
			for _, l := range lines[i+1 : end] {
				out.WriteString(l)
			}
		} else {
			for _, l := range strings.SplitAfter(strings.TrimRight(formatted, "\n")+"\n", "\n") {
				if l == "" {
					continue
				}
				if strings.TrimSpace(l) != "" {
					out.WriteString(indent)
				}
				out.WriteString(l)
			}
		}
		out.WriteString(lines[end])
		i = end
	}
	return out.String(), nil
}
