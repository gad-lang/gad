package gadbridge

import (
	"encoding/json"
	"strings"

	"gopkg.in/yaml.v3"
)

// DocEncoded is the structure serialized by DocEncode (JSON/YAML): the extracted
// prose and sections plus the source's snippets, each with its `uses` references
// and — when run — the verified result/output.
type DocEncoded struct {
	Prose    string       `json:"prose,omitempty" yaml:"prose,omitempty"`
	Sections []DocSection `json:"sections,omitempty" yaml:"sections,omitempty"`
	Snippets []DocSnippet `json:"snippets,omitempty" yaml:"snippets,omitempty"`
}

// DocSnippet is a `//snippet` region exposed in the encoded documentation.
type DocSnippet struct {
	Name     string   `json:"name" yaml:"name"`
	Uses     []string `json:"uses,omitempty" yaml:"uses,omitempty"`
	Code     string   `json:"code" yaml:"code"`
	Kind     string   `json:"kind,omitempty" yaml:"kind,omitempty"` // "value" | "output" | ""
	Expected string   `json:"expected,omitempty" yaml:"expected,omitempty"`
	Result   string   `json:"result,omitempty" yaml:"result,omitempty"`
	Output   string   `json:"output,omitempty" yaml:"output,omitempty"`
}

// DocEncode extracts the documentation from src and returns it encoded as JSON or
// YAML (format "json" | "yaml"), including the source's snippets with their
// references and (when run) verified results.
func DocEncode(src, sourceType, format string) (string, error) {
	doc, err := ExtractDoc(src, sourceType)
	if err != nil {
		return "", err
	}
	enc := &DocEncoded{
		Prose:    doc.Prose,
		Sections: doc.Sections,
		Snippets: collectDocSnippets([]byte(src), sourceType),
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

// --- snippet extraction (mirrors the CLI `//snippet` grammar) ---

type bridgeSnippet struct {
	name, code, kind, expected string
	uses                       []string
}

// collectDocSnippets extracts every `//snippet NAME [uses …]` region of src (in
// source order), runs each (with its `uses` contexts prepended) to fill in the
// verified result/output, and returns them for encoding.
func collectDocSnippets(src []byte, sourceType string) []DocSnippet {
	snips, order := extractBridgeSnippets(src)
	out := make([]DocSnippet, 0, len(order))
	for _, name := range order {
		s := snips[name]
		info := DocSnippet{Name: s.name, Uses: s.uses, Code: s.code, Kind: s.kind, Expected: s.expected}
		if s.kind != "" {
			ctx := snippetContext(s, snips)
			switch s.kind {
			case "value":
				// A value snippet ends in a bare expression; return it so the run
				// reports it as the result.
				if r := RunSource(ctx+returnLast(s.code), sourceType); r.OK {
					info.Result = r.Result
				}
			case "output":
				if r := RunSource(ctx+s.code, sourceType); r.OK {
					info.Output = strings.TrimRight(r.Stdout, "\n")
				}
			}
		}
		out = append(out, info)
	}
	return out
}

// returnLast prefixes the last non-blank line of code with `return ` (unless it
// already returns), so a value snippet's trailing expression is reported as the
// run's result. Indentation is preserved.
func returnLast(code string) string {
	lines := strings.Split(code, "\n")
	i := len(lines) - 1
	for i >= 0 && strings.TrimSpace(lines[i]) == "" {
		i--
	}
	if i >= 0 {
		t := strings.TrimSpace(lines[i])
		if t != "return" && !strings.HasPrefix(t, "return ") {
			indent := lines[i][:len(lines[i])-len(strings.TrimLeft(lines[i], " \t"))]
			lines[i] = indent + "return " + t
		}
	}
	return strings.Join(lines, "\n")
}

// snippetContext returns the concatenated code of a snippet's `uses` contexts
// (transitively, each once, in order), each terminated by a newline, to be
// prepended when the snippet runs.
func snippetContext(s *bridgeSnippet, snips map[string]*bridgeSnippet) string {
	var (
		b    strings.Builder
		seen = map[string]bool{s.name: true}
		add  func(names []string)
	)
	add = func(names []string) {
		for _, n := range names {
			if seen[n] {
				continue
			}
			seen[n] = true
			ctx, ok := snips[n]
			if !ok {
				continue
			}
			add(ctx.uses)
			b.WriteString(ctx.code)
			b.WriteString("\n")
		}
	}
	add(s.uses)
	return b.String()
}

// extractBridgeSnippets scans src for `//snippet NAME [uses A B] … //endsnippet`
// regions, returning them by name plus the names in source order.
func extractBridgeSnippets(src []byte) (map[string]*bridgeSnippet, []string) {
	out := map[string]*bridgeSnippet{}
	var order []string
	var cur *bridgeSnippet
	var body []string
	for _, ln := range strings.Split(string(src), "\n") {
		t := strings.TrimSpace(ln)
		switch {
		case cur == nil:
			if name, uses, ok := parseSnippetOpen(t); ok {
				cur = &bridgeSnippet{name: name, uses: uses}
				body = nil
			}
		case t == "//endsnippet":
			cur.code, cur.kind, cur.expected = splitBridgeResult(body)
			out[cur.name] = cur
			order = append(order, cur.name)
			cur = nil
		default:
			body = append(body, ln)
		}
	}
	return out, order
}

// parseSnippetOpen matches `//snippet NAME` and `//snippet NAME uses A B …`.
func parseSnippetOpen(trimmed string) (name string, uses []string, ok bool) {
	if !strings.HasPrefix(trimmed, "//") {
		return "", nil, false
	}
	f := strings.Fields(strings.TrimPrefix(trimmed, "//"))
	if len(f) < 2 || f[0] != "snippet" {
		return "", nil, false
	}
	rest := f[2:]
	if len(rest) > 0 {
		if rest[0] != "uses" || len(rest) < 2 {
			return "", nil, false
		}
		uses = rest[1:]
	}
	return f[1], uses, true
}

// splitBridgeResult separates the snippet code from a `/**= EXPR **/` (value) or
// `/**< TEXT **/` (output) result marker (single- or multi-line).
func splitBridgeResult(body []string) (code, kind, expected string) {
	var codeLines, marker []string
	inMarker := false
	var sigil string
	for _, ln := range body {
		t := strings.TrimSpace(ln)
		if inMarker {
			if i := strings.LastIndex(ln, "**/"); i >= 0 {
				marker = append(marker, strings.TrimSpace(ln[:i]))
				inMarker = false
			} else {
				marker = append(marker, ln)
			}
			continue
		}
		switch {
		case strings.HasPrefix(t, "/**="):
			kind, sigil = "value", "/**="
		case strings.HasPrefix(t, "/**<"):
			kind, sigil = "output", "/**<"
		default:
			codeLines = append(codeLines, ln)
			continue
		}
		rest := strings.TrimPrefix(t, sigil)
		if i := strings.LastIndex(rest, "**/"); i >= 0 {
			marker = append(marker, strings.TrimSpace(rest[:i]))
		} else {
			marker = append(marker, strings.TrimSpace(rest))
			inMarker = true
		}
	}
	return strings.Trim(strings.Join(codeLines, "\n"), "\n"), kind, strings.TrimSpace(strings.Join(marker, "\n"))
}
