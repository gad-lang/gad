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

	// class/mixin/interface are CONTEXTUAL keywords: the parser reserves them
	// only in declaration position (`class [NAME] {`, `mixin [NAME] {`,
	// `interface {`/`[`/`NAME {`) and treats them as ordinary identifiers
	// everywhere else (parameter names, variables, keys). Pull them out of the
	// general keyword rule and highlight them as keywords only when a
	// declaration body follows (mirroring isDeclBodyStart / isInterfaceDeclStart),
	// so `fn(class)`, `x.mixin`, `{interface: 1}` stay un-keyworded.
	contextualDecl := map[string]bool{"class": true, "mixin": true, "interface": true}
	plainKeywords := make([]string, 0, len(lang.Keywords))
	for _, k := range lang.Keywords {
		if !contextualDecl[k] {
			plainKeywords = append(plainKeywords, k)
		}
	}

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
			// The block body does NOT embed `text.html.markdown`. Its paragraph is a
			// `begin`/`while` block anchored with `\G`, and IntelliJ's TextMate engine
			// mishandles a `while`-based child nested inside a `begin`/`end` parent —
			// on some doc bodies (e.g. a line after a `[link](x)`) the closing `**/`
			// and the lines before it lost the doc-comment scope and rendered as plain
			// code (vscode-textmate tokenizes it fine, but the editors shipping this
			// grammar do not). Instead `#docmarkup` re-adds a self-contained subset of
			// inline Markdown as single-line `match` rules (no `begin`/`while`), which
			// every TextMate engine handles: `**bold**`, `*italic*`, `` `code` ``,
			// headings and `[links]`. Plain text keeps the doc-comment scope.
			{
				Name:     "comment.documentation.block.gad",
				Begin:    `/\*\*\*`,
				End:      `\*\*\*/\s*$`,
				Patterns: []tmRule{{Include: "#docmarkup"}},
			},
			{
				Name:     "comment.documentation.block.gad",
				Begin:    `/\*\*`,
				End:      `\*\*/\s*$`,
				Patterns: []tmRule{{Include: "#docmarkup"}},
			},
			// A `///` line doc is a single line; block docs close at a fence that
			// finishes a line. Neither embeds the Markdown grammar (see #docmarkup).
			{Name: "comment.documentation.line.gad", Match: `///(?!/).*$`},
			{Name: "comment.line.double-slash.gad", Match: `//.*$`},
			{Name: "comment.block.gad", Begin: `/\*`, End: `\*/`},
		}},
		// A self-contained subset of inline Markdown for doc-comment bodies, as
		// single-line `match` rules only (no `begin`/`while` — those break IntelliJ's
		// TextMate engine when nested in the doc block; see the "comments" note).
		// Order matters: `code` first so `*`/`_` inside it are literal, then bold
		// before italic so `**x**` is bold, not two italics. Emphasis is
		// word-boundary guarded so prose like `a*b` or `some_id` is not styled, and
		// the fence `**/` can never match (it needs a closing `**`/`_` with a
		// non-space before it). Scopes mirror `text.html.markdown` so editors color
		// them as they do Markdown elsewhere.
		"docmarkup": {Patterns: []tmRule{
			{Name: "markup.inline.raw.string.gad", Match: "`[^`\\n]+`"},
			{Name: "markup.bold.gad", Match: `\*\*(?=\S)[^*\n]+?(?<=\S)\*\*`},
			{Name: "markup.bold.gad", Match: `(?<![\w])__(?=\S)[^_\n]+?(?<=\S)__(?![\w])`},
			{Name: "markup.italic.gad", Match: `(?:^|(?<=[\s(]))\*(?=\S)[^*\n]+?(?<=\S)\*(?=[\s).,;:!?]|$)`},
			{Name: "markup.italic.gad", Match: `(?<![\w])_(?=\S)[^_\n]+?(?<=\S)_(?![\w])`},
			{Name: "markup.heading.gad", Match: `^\s*#{1,6}\s+.*$`},
			{Match: `(\[)([^\]\n]+)(\])(\()([^)\n]+)(\))`, Captures: map[string]tmCap{
				"2": {Name: "string.other.link.title.gad"},
				"5": {Name: "markup.underline.link.gad"},
			}},
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
			{Name: "keyword.control.gad", Match: wordRegex(plainKeywords)},
			// Contextual declaration keywords: `class`/`mixin` before `{` or a
			// `NAME {` body, `interface` also before `[` (the `interface[]` form).
			{Name: "keyword.control.gad", Match: `\b(?:class|mixin)\b(?=\s*\{|\s+[A-Za-z_]\w*\s*\{)`},
			{Name: "keyword.control.gad", Match: `\binterface\b(?=\s*[\{\[]|\s+[A-Za-z_]\w*\s*\{)`},
			{Name: "constant.language.gad", Match: wordRegex(lang.Atoms)},
			{Name: "constant.language.gad", Match: wordRegex(lang.Constants)},
			{Name: "support.function.gad", Match: wordRegex(lang.Builtins)},
		}},
		"specials": {Patterns: []tmRule{
			// @-prefixed specials (@args, @module, @main, …).
			{Name: "variable.language.gad", Match: `@[A-Za-z_$][\w$]*`},
		}},
		"operators": {Patterns: []tmRule{
			// `:::` (transforming cast) before `::` (checked cast) so the longer
			// operator wins; `**` covers both the power operator and an interface
			// `**rest` member.
			{Name: "keyword.operator.gad", Match: `:::|::|\?\?=?|\.\.|=>|:=|\|\||&&|\*\*=?|<<<?=?|>>>?=?|&\^=?|%%=?|===?|!==?|[-+*/%&|^!<>=]=?|[~?:]`},
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
