// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"regexp"
	"strings"

	gad "github.com/gad-lang/gad"
	"github.com/gad-lang/gad/parser/ast"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/web/gadbridge"
)

// Snippets are a "go doc"-style mechanism to incorporate real, verified source
// code into the generated documentation without copying it by hand. A snippet
// is an `example NAME { … }` block of the language (issue #22):
//
//	greet := func(name) => "hi " + name
//
//	example greet {
//		println(greet("Gad"))
//		greet("Ana")
//		= "hi Ana"
//	}
//	<<< hi Gad
//
// and is referenced from a doc comment (the module `/** … **/` prose or a
// symbol's doc) by a placeholder line:
//
//	@snippet greet
//
// During documentation generation the placeholder is replaced by a fenced code
// block holding the example's body (dedented, without its braces) and the
// output checks written right after its `}`, fenced with the source dialect.
//
// The checks are the language's: `= EXPR` (the value of the statement before
// it, as str), `<<< text` / `<<<`⏎ … ⏎`>>>` (the output since the previous
// output check; after the `}`, all the example printed). They are verified by
// running the file once with checks on (unless --no-doctest), so a documented
// result can never drift from what the code does.
//
// An example's code is of the scope around it, as the file's code is: a later
// example uses the definitions of an earlier one — its context, which is not
// rendered with it.

// snippet is one named example: its body (dedented) and the output checks
// after it, as rendered (gadbridge.Snippets — the same reading as the web's).
type snippet struct {
	name string
	code string
}

// extractSnippets returns the named examples of src by name.
func extractSnippets(src []byte) map[string]*snippet {
	out := map[string]*snippet{}
	for _, s := range scanSnippets(src) {
		out[s.name] = s
	}
	return out
}

// scanSnippets is extractSnippets in source order.
func scanSnippets(src []byte) []*snippet {
	var out []*snippet
	for _, s := range gadbridge.Snippets(src) {
		out = append(out, &snippet{name: s.Name, code: s.Code})
	}
	return out
}

// expandSnippets replaces every `@snippet NAME` placeholder line in text with a
// fenced code block (fenced with lang) holding the named example. An unknown
// name leaves the placeholder untouched.
func expandSnippets(text string, snippets map[string]*snippet, lang string) string {
	if text == "" || !strings.Contains(text, "@snippet") {
		return text
	}
	var out []string
	for _, ln := range strings.Split(text, "\n") {
		name, ok := snippetRef(ln)
		if !ok {
			out = append(out, ln)
			continue
		}
		snip, found := snippets[name]
		if !found {
			out = append(out, ln)
			continue
		}
		fence := fenceFor(snip.code)
		out = append(out, fence+lang, snip.code, fence)
	}
	return strings.Join(out, "\n")
}

// snippetInfo is the serializable view of a snippet exposed on the doc object
// (and in the JSON/YAML encoding): its name and its code.
type snippetInfo struct {
	Name string `json:"name" yaml:"name"`
	Code string `json:"code" yaml:"code"`
}

// collectSnippets returns every named example of src as a snippetInfo, in
// declaration order: the source of the snippets exposed to templates and
// encoded by --json/--yaml.
func collectSnippets(src []byte) []snippetInfo {
	snips := scanSnippets(src)
	infos := make([]snippetInfo, 0, len(snips))
	for _, s := range snips {
		infos = append(infos, snippetInfo{Name: s.name, Code: s.code})
	}
	return infos
}

// snippetsGadArray converts snippet infos to the gad.Array of gad.Dict exposed as
// `doc.snippets` to documentation templates.
func snippetsGadArray(infos []snippetInfo) gad.Array {
	arr := make(gad.Array, len(infos))
	for i, s := range infos {
		arr[i] = gad.Dict{"name": gad.Str(s.Name), "code": gad.Str(s.Code)}
	}
	return arr
}

// verifyExamples runs a gad file once with its checks on — its examples', and
// any other — when it holds an example or a check: what its documentation
// shows is what it does. Other dialects and files without checks are not run.
func verifyExamples(path string, src []byte, sourceType string) error {
	if sourceType != "" && sourceType != "gad" {
		return nil
	}
	if !hasChecks(src) {
		return nil
	}
	if _, err := evalGadExample(string(src)); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

var reCheckLine = regexp.MustCompile(`(?m)^\s*(?:example(?:\s+\w+)*\s*\{|=\s|<<<)`)

// hasChecks reports whether src holds an example or an inline check — in its
// code, not in a comment (a doc comment's fences are checked by the doctest).
func hasChecks(src []byte) bool {
	if !reCheckLine.Match(src) {
		return false
	}
	file, err := astParse("check.gad", src)
	if err != nil {
		return true // the run says what is wrong
	}
	found := false
	for _, st := range file.Stmts {
		node.Walk(st, func(n ast.Node) bool {
			switch n.(type) {
			case *node.ExampleStmt, *node.ValueCheckStmt, *node.OutputCheckStmt:
				found = true
			}
			return !found
		})
		if found {
			break
		}
	}
	return found
}

// fenceFor returns the Markdown code-fence (a run of backticks) that safely wraps
// content: at least three backticks, and always one more than the longest run of
// backticks inside content, so an embedded ```` ``` ```` (e.g. a doctest fence in
// a doc comment) never closes the outer fence early.
func fenceFor(content ...string) string {
	longest := 0
	for _, s := range content {
		run := 0
		for _, r := range s {
			if r == '`' {
				run++
				if run > longest {
					longest = run
				}
			} else {
				run = 0
			}
		}
	}
	n := longest + 1
	if n < 3 {
		n = 3
	}
	return strings.Repeat("`", n)
}

// snippetRef reports whether a line is a `@snippet NAME` placeholder (ignoring
// surrounding whitespace) and returns the name.
func snippetRef(line string) (string, bool) {
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) == 2 && fields[0] == "@snippet" {
		return fields[1], true
	}
	return "", false
}

// exampleSource returns the source rendered in the generated "Example" block:
// the leading `/** … **/` module doc block dropped (it is already rendered as
// the page prose).
func exampleSource(src []byte) string {
	return strings.TrimLeft(dropLeadingModuleDoc(dropGeneratedHeader(string(src))), "\n")
}

// reGeneratedHeader matches the line that marks a generated file (Go's
// convention, written by gaddoc.GeneratedHeader).
var reGeneratedHeader = regexp.MustCompile(`^// Code generated .* DO NOT EDIT\.$`)

// dropGeneratedHeader removes a leading generated-file marker line (and the
// blank lines after it): a note for whoever edits the file, not documentation.
func dropGeneratedHeader(s string) string {
	first, rest, _ := strings.Cut(s, "\n")
	if !reGeneratedHeader.MatchString(strings.TrimSpace(first)) {
		return s
	}
	return strings.TrimLeft(rest, "\n")
}

// dropLeadingModuleDoc removes a leading `/*** … ***/` block (and the blank lines
// after it) from s. Other comments are left untouched.
func dropLeadingModuleDoc(s string) string {
	lines := strings.Split(s, "\n")
	i := 0
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}
	if i >= len(lines) {
		return s
	}
	// The module doc is a leading `/** … **/` block (three-star `/*** … ***/`
	// still accepted), with the opening and closing fences alone on their line —
	// so a `**/` appearing mid-line inside the prose never closes it early.
	var closer string
	switch strings.TrimSpace(lines[i]) {
	case "/***":
		closer = "***/"
	case "/**":
		closer = "**/"
	default:
		return s
	}
	for j := i + 1; j < len(lines); j++ {
		if strings.TrimSpace(lines[j]) == closer {
			rest := lines[j+1:]
			for len(rest) > 0 && strings.TrimSpace(rest[0]) == "" {
				rest = rest[1:]
			}
			return strings.Join(rest, "\n")
		}
	}
	return s
}
