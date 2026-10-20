package gadbridge

import (
	"encoding/json"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// DocEncoded is the structure serialized by DocEncode (JSON/YAML): the extracted
// prose and sections plus the source's snippets — its named examples.
type DocEncoded struct {
	Prose    string       `json:"prose,omitempty" yaml:"prose,omitempty"`
	Sections []DocSection `json:"sections,omitempty" yaml:"sections,omitempty"`
	Snippets []DocSnippet `json:"snippets,omitempty" yaml:"snippets,omitempty"`
}

// DocSnippet is a named `example NAME { … }` (issue #22) exposed in the encoded
// documentation: its body, dedented, and the output checks right after its `}`.
type DocSnippet struct {
	Name string `json:"name" yaml:"name"`
	Code string `json:"code" yaml:"code"`
}

// DocEncode extracts the documentation from src and returns it encoded as JSON or
// YAML (format "json" | "yaml"), including the source's snippets.
func DocEncode(src, sourceType, format string) (string, error) {
	doc, err := ExtractDoc(src, sourceType)
	if err != nil {
		return "", err
	}
	enc := &DocEncoded{
		Prose:    doc.Prose,
		Sections: doc.Sections,
		Snippets: Snippets([]byte(src)),
	}
	switch strings.ToLower(format) {
	case "yaml", "yml":
		b, err := yaml.Marshal(enc)
		return string(b), err
	default:
		b, err := json.MarshalIndent(enc, "", "  ")
		return string(b), err
	}
}

// reExampleOpen matches the line opening a named example: its indentation and
// its name.
var reExampleOpen = regexp.MustCompile(`^(\s*)example\s+(\w+)\s*\{\s*$`)

// reGadxExampleOpen matches gadx's `@example NAME`: its indentation and name.
var reGadxExampleOpen = regexp.MustCompile(`^(\s*)@example\s+(\w+)\s*$`)

// Snippets are the named examples of src, in source order: each one's body
// (dedented, without its braces) and the output checks written right after
// its closing brace, at its level (`<<< text`, `<<<` … `>>>`) — or, in gadx,
// `@example NAME`'s indented body and the `<<<` lines after it.
func Snippets(src []byte) []DocSnippet {
	var out []DocSnippet
	lines := strings.Split(string(src), "\n")
	for i := 0; i < len(lines); i++ {
		if m := reGadxExampleOpen.FindStringSubmatch(lines[i]); m != nil {
			var s DocSnippet
			s, i = gadxSnippet(lines, i, m[1], m[2])
			out = append(out, s)
			continue
		}
		m := reExampleOpen.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		indent, name := m[1], m[2]
		closer := indent + "}"
		j := i + 1
		for j < len(lines) && strings.TrimRight(lines[j], " \t") != closer {
			j++
		}
		if j >= len(lines) {
			continue
		}
		body := dedent(trimBlank(append([]string(nil), lines[i+1:j]...)))
		k := j + 1
		var after []string
		for k < len(lines) && strings.HasPrefix(lines[k], indent+"<<<") {
			block := strings.TrimSpace(lines[k]) == "<<<"
			after = append(after, strings.TrimPrefix(lines[k], indent))
			k++
			for block && k < len(lines) {
				after = append(after, strings.TrimPrefix(lines[k], indent))
				k++
				if strings.TrimSpace(lines[k-1]) == ">>>" {
					break
				}
			}
		}
		code := strings.Join(body, "\n")
		if len(after) > 0 {
			code += "\n" + strings.Join(after, "\n")
		}
		out = append(out, DocSnippet{Name: name, Code: code})
		i = k - 1
	}
	return out
}

func trimBlank(lines []string) []string {
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// dedent strips the indentation common to the non-blank lines.
func dedent(lines []string) []string {
	indent := -1
	for _, ln := range lines {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		if n := len(ln) - len(strings.TrimLeft(ln, " \t")); indent < 0 || n < indent {
			indent = n
		}
	}
	for i, ln := range lines {
		if indent > 0 && len(ln) >= indent {
			lines[i] = ln[indent:]
		} else if indent > 0 {
			lines[i] = strings.TrimLeft(ln, " \t")
		}
	}
	return lines
}

// gadxSnippet reads the gadx example opened at lines[i]: the lines indented
// deeper than indent (blank ones within it kept), then the output checks at
// indent — `<<< text`, or `<<<` and the lines indented under it. It returns
// the snippet and the index of its last line.
func gadxSnippet(lines []string, i int, indent, name string) (DocSnippet, int) {
	deeper := func(l string) bool {
		return strings.TrimSpace(l) == "" || (strings.HasPrefix(l, indent) && len(l) > len(indent) &&
			(l[len(indent)] == ' ' || l[len(indent)] == '\t'))
	}
	j := i + 1
	for j < len(lines) && deeper(lines[j]) {
		j++
	}
	body := dedent(trimBlank(append([]string(nil), lines[i+1:j]...)))
	last := j - 1
	k := j
	for k < len(lines) && strings.TrimSpace(lines[k]) == "" {
		k++
	}
	var after []string
	for k < len(lines) && strings.HasPrefix(lines[k], indent+"<<<") {
		after = append(after, strings.TrimPrefix(lines[k], indent))
		block := strings.TrimSpace(lines[k]) == "<<<"
		last = k
		k++
		for block && k < len(lines) && strings.TrimSpace(lines[k]) != "" && deeper(lines[k]) {
			after = append(after, strings.TrimPrefix(lines[k], indent))
			last = k
			k++
		}
		for k < len(lines) && strings.TrimSpace(lines[k]) == "" {
			k++
		}
	}
	code := strings.Join(body, "\n")
	if len(after) > 0 {
		code += "\n\n" + strings.Join(after, "\n")
	}
	return DocSnippet{Name: name, Code: code}, last
}
