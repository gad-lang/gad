package pluginsync

import (
	"encoding/json"
	"strings"
)

// tmRule is a TextMate grammar rule (a subset: match or begin/end with a name).
type tmRule struct {
	Name          string           `json:"name,omitempty"`
	ContentName   string           `json:"contentName,omitempty"`
	Match         string           `json:"match,omitempty"`
	Begin         string           `json:"begin,omitempty"`
	End           string           `json:"end,omitempty"`
	Include       string           `json:"include,omitempty"`
	Patterns      []tmRule         `json:"patterns,omitempty"`
	Captures      map[string]tmCap `json:"captures,omitempty"`
	BeginCaptures map[string]tmCap `json:"beginCaptures,omitempty"`
	EndCaptures   map[string]tmCap `json:"endCaptures,omitempty"`
}

type tmCap struct {
	Name string `json:"name"`
}

// tmGrammar is the top-level TextMate grammar document.
type tmGrammar struct {
	Schema     string            `json:"$schema,omitempty"`
	Name       string            `json:"name"`
	ScopeName  string            `json:"scopeName"`
	Patterns   []tmRule          `json:"patterns"`
	Repository map[string]tmRule `json:"repository"`
}

// wordRegex builds a `\b(?:a|b|c)\b` alternation, longest-first so multi-char
// keywords (e.g. defer_ok) win over their prefixes.
func wordRegex(words []string) string {
	sorted := append([]string(nil), words...)
	// longest first
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if len(sorted[j]) > len(sorted[i]) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	return `\b(?:` + strings.Join(sorted, "|") + `)\b`
}

// TextMateGrammar generates the Gad TextMate grammar (source.gad) from the
// current language vocabulary, for the VS Code extension's syntax highlighting.
func TextMateGrammar() ([]byte, error) {
	lang := Extract()

	repo := map[string]tmRule{
		"comments": {Patterns: []tmRule{
			// Doc comments first so `/**`/`/***`/`///` are not read as ordinary
			// `/*`/`//` comments. A block doc ends at a fence (`**/` / `***/`) that
			// finishes a line (`\s*$`), so a single-line `/** foo **/` closes on its
			// own line while inline `**bold**` / `***hr***` and an embedded `***/` in
			// the middle of the doc text (e.g. mentioning the `/*** … ***/` form) do
			// NOT close it early. `///` is a single-line doc.
			//
			// The scope name starts with `comment.documentation` on purpose: the
			// IntelliJ TextMate integration maps that prefix to the shared DOC_COMMENT
			// color key (`comment.line`/`comment.block` map to the ordinary comment
			// colors), so a user can color Gad doc comments by setting the IDE's "Doc
			// Comment" color. VS Code themes still see the `comment` root; the Gad
			// extension's configurationDefaults tint these scopes.
			//
			// The block body embeds the Markdown grammar (`text.html.markdown`): its
			// plain text carries the doc-comment scope (→ orange), while `markup.bold`
			// / `markup.italic` / fenced code keep their own Markdown colors. The
			// content scope also starts with `comment.documentation` so the body text
			// is DOC_COMMENT-colored in IntelliJ too.
			{
				Name:        "comment.documentation.block.gad",
				Begin:       `/\*\*\*`,
				End:         `\*\*\*/\s*$`,
				ContentName: "comment.documentation.block.markdown.gad",
				Patterns:    []tmRule{{Include: "text.html.markdown"}},
			},
			{
				Name:        "comment.documentation.block.gad",
				Begin:       `/\*\*`,
				End:         `\*\*/\s*$`,
				ContentName: "comment.documentation.block.markdown.gad",
				Patterns:    []tmRule{{Include: "text.html.markdown"}},
			},
			// A `///` line doc does NOT embed Markdown: `text.html.markdown` continues
			// a paragraph across lines (a `while`-based block), so on a single-line
			// comment it would bleed into the following code line until a blank line.
			// Only the fenced block docs, which have a hard closing fence, embed it.
			{Name: "comment.documentation.line.gad", Match: `///(?!/).*$`},
			{Name: "comment.line.double-slash.gad", Match: `//.*$`},
			{Name: "comment.block.gad", Begin: `/\*`, End: `\*/`},
		}},
		"strings": {Patterns: []tmRule{
			// Interpolated strings (`#`-prefixed): a `{ … }` island embeds a Gad
			// expression, highlighted via `#interpolation` (which includes $self).
			// These must precede the plain forms so the leading `#` is consumed.
			// Cooked forms (`#"…"`, `#"""…"""`) honor escapes, including the `\{` /
			// `\}` delimiter escapes, so `\{` is an escape, not an island start.
			{Name: "string.quoted.triple.interpolated.gad", Begin: `#"""`, End: `"""`, Patterns: []tmRule{
				{Name: "constant.character.escape.gad", Match: `\\.`},
				{Include: "#interpolation"},
			}},
			{Name: "string.quoted.double.interpolated.gad", Begin: `#"`, End: `"`, Patterns: []tmRule{
				{Name: "constant.character.escape.gad", Match: `\\.`},
				{Include: "#interpolation"},
			}},
			// Raw interpolated form (`#`…``) is verbatim: no escapes, so every
			// unescaped `{` opens an island (there is no `\{` escape here).
			{Name: "string.quoted.raw.interpolated.gad", Begin: "#`", End: "`", Patterns: []tmRule{
				{Include: "#interpolation"},
			}},
			// Plain (non-interpolated) strings.
			{Name: "string.quoted.triple.gad", Begin: `"""`, End: `"""`},
			{Name: "string.quoted.raw.gad", Begin: "```", End: "```"},
			{Name: "string.quoted.double.gad", Begin: `[bh]?"`, End: `"`, Patterns: []tmRule{
				{Name: "constant.character.escape.gad", Match: `\\.`},
			}},
			{Name: "string.quoted.raw.gad", Begin: "[bh]?`", End: "`"},
			{Name: "string.quoted.single.gad", Begin: `'`, End: `'`, Patterns: []tmRule{
				{Name: "constant.character.escape.gad", Match: `\\.`},
			}},
		}},
		// A `{ … }` interpolation island inside a `#`-string. The braces are
		// embedded-section punctuation; the body is marked `meta.embedded` (so
		// TextMate hosts — VS Code and the IntelliJ TextMate engine — re-highlight
		// it as code rather than string) and matched as full Gad via $self. It
		// includes itself first so nested `{ … }` (e.g. a map/block literal in the
		// expression) balances instead of closing the island at the first `}`.
		"interpolation": {
			Name:          "meta.interpolation.gad",
			Begin:         `\{`,
			End:           `\}`,
			BeginCaptures: map[string]tmCap{"0": {Name: "punctuation.section.embedded.begin.gad"}},
			EndCaptures:   map[string]tmCap{"0": {Name: "punctuation.section.embedded.end.gad"}},
			ContentName:   "meta.embedded.gad",
			Patterns: []tmRule{
				{Include: "#interpolation"},
				{Include: "$self"},
			},
		},
		"numbers": {Patterns: []tmRule{
			{Name: "constant.numeric.gad", Match: `\b0[xX][0-9a-fA-F]+\b|\b\d+(?:\.\d+)?(?:[eE][-+]?\d+)?[uUdD]?\b`},
		}},
		"keywords": {Patterns: []tmRule{
			{Name: "keyword.control.gad", Match: wordRegex(lang.Keywords)},
			{Name: "constant.language.gad", Match: wordRegex(lang.Atoms)},
			{Name: "constant.language.gad", Match: wordRegex(lang.Constants)},
			{Name: "support.function.gad", Match: wordRegex(lang.Builtins)},
		}},
		"specials": {Patterns: []tmRule{
			// @-prefixed specials (@args, @module, @main, …).
			{Name: "variable.language.gad", Match: `@[A-Za-z_$][\w$]*`},
		}},
		"operators": {Patterns: []tmRule{
			{Name: "keyword.operator.gad", Match: `::|\?\?=?|\.\.|=>|:=|\|\||&&|\*\*=?|<<<?=?|>>>?=?|&\^=?|%%=?|===?|!==?|[-+*/%&|^!<>=]=?|[~?:]`},
		}},
	}

	g := tmGrammar{
		Schema:    "https://raw.githubusercontent.com/martinring/tmlanguage/master/tmlanguage.json",
		Name:      "Gad",
		ScopeName: "source.gad",
		Patterns: []tmRule{
			{Include: "#comments"},
			{Include: "#strings"},
			{Include: "#numbers"},
			{Include: "#keywords"},
			{Include: "#specials"},
			{Include: "#operators"},
		},
		Repository: repo,
	}
	return json.MarshalIndent(g, "", "  ")
}
