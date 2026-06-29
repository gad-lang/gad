package pluginsync

import (
	"encoding/json"
	"strings"
)

// tmRule is a TextMate grammar rule (a subset: match or begin/end with a name).
type tmRule struct {
	Name     string           `json:"name,omitempty"`
	Match    string           `json:"match,omitempty"`
	Begin    string           `json:"begin,omitempty"`
	End      string           `json:"end,omitempty"`
	Include  string           `json:"include,omitempty"`
	Patterns []tmRule         `json:"patterns,omitempty"`
	Captures map[string]tmCap `json:"captures,omitempty"`
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
			// `/*`/`//` comments. A block doc ends only at a line that is exactly the
			// fence (`**/` / `***/`), so inline `**bold**` / `***hr***` Markdown in
			// the doc text does not close it early. `///` is a single-line doc.
			{Name: "comment.block.documentation.gad", Begin: `/\*\*\*`, End: `^\s*\*\*\*/\s*$`},
			{Name: "comment.block.documentation.gad", Begin: `/\*\*`, End: `^\s*\*\*/\s*$`},
			{Name: "comment.line.documentation.gad", Match: `///(?!/).*$`},
			{Name: "comment.line.double-slash.gad", Match: `//.*$`},
			{Name: "comment.block.gad", Begin: `/\*`, End: `\*/`},
		}},
		"strings": {Patterns: []tmRule{
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
			{Name: "keyword.operator.gad", Match: `\?\?=?|\.\.|=>|:=|\|\||&&|\*\*=?|<<<?=?|>>>?=?|&\^=?|%%=?|===?|!==?|[-+*/%&|^!<>=]=?|[~?:]`},
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
