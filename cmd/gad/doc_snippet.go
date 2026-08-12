// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	gad "github.com/gad-lang/gad"
)

// Snippets are a "go doc"-style mechanism to incorporate real, verified source
// code into the generated documentation without copying it by hand. A region of
// the source is delimited by line comments:
//
//	//snippet greet
//	greet := func(name) => "hi " + name
//	greet("Gad")
//	/**= "hi Gad" **/
//	//endsnippet
//
// and referenced from a doc comment (the module `/*** … ***/` prose or a `///`
// symbol comment) by a placeholder line:
//
//	@snippet greet
//
// During documentation generation the placeholder is replaced by a fenced code
// block holding the region's code, fenced with the source dialect (gad, gadt or
// gadx). The marker lines never appear in the rendered code.
//
// A region may declare an expected result, verified at generation time (unless
// --no-doctest), then shown in the rendered block:
//
//   - `/**= EXPR **/` — the region's value must equal the value of EXPR. The
//     rendered block gains a `// => <value>` line.
//   - `/**< TEXT **/` — the region's STDOUT must equal TEXT. The rendered block
//     is followed by an `Output:` block with the captured output.
//
// The `/**= … **/` / `/**< … **/` markers may be single- or multi-line. A terse
// single-line form is also accepted, where the whole line is the marker and the
// rest of it is the expected text:
//
//   - `//= EXPR`  — value assertion (like `/**= EXPR **/`).
//   - `//< TEXT`  — output assertion (like `/**< TEXT **/`).
//
// A snippet runs standalone, so it must be self-contained. To reuse another
// snippet's definitions without repeating them, list it as an execution context
// on the open line:
//
//	//snippet usage uses define other
//
// The `uses` contexts' code is prepended when the snippet runs (transitively, in
// order, each once), so the snippet can reference their definitions and its
// result still verifies — but the context is not rendered and its own result
// marker is ignored; only the snippet's own code appears in the block.

// snippetResultKind classifies a snippet's expected-result marker.
type snippetResultKind int

const (
	snippetNoResult snippetResultKind = iota
	snippetValue                      // /**= EXPR **/
	snippetOutput                     // /**< TEXT **/
)

// snippetCheck is one assertion inside a snippet: the region code up to lineEnd
// (whose last line is the statement under test) must produce expected. A snippet
// may hold several, so successive statements can each be verified — e.g.
// `1\n//= 1\n\n"x"\n//= "x"` checks two statements.
type snippetCheck struct {
	lineEnd  int // the check tests code lines[:lineEnd]
	kind     snippetResultKind
	expected string
}

// snippet is one `//snippet NAME … //endsnippet` region: its code (dedented, with
// result markers removed), the lines that code splits into, and its ordered
// result checks.
type snippet struct {
	name   string
	code   string
	lines  []string // code split into lines (indices match snippetCheck.lineEnd)
	checks []snippetCheck
	// kind/expected mirror the last check, for the single-result consumers
	// (templates/JSON); snippetNoResult when there is no check.
	kind     snippetResultKind
	expected string
	// uses names other snippets whose code is prepended as an execution context
	// (so this snippet can reuse their definitions), declared on the open line as
	// `//snippet NAME uses A B C`. The context code runs but is NOT rendered and
	// its own result marker is ignored — only this snippet's code and result show.
	uses []string
}

// extractSnippets scans src for `//snippet NAME … //endsnippet` regions and
// returns them by name. An unterminated region is ignored.
func extractSnippets(src []byte) map[string]*snippet {
	out := map[string]*snippet{}
	var name string
	var uses []string
	var body []string
	for _, ln := range strings.Split(string(src), "\n") {
		t := strings.TrimSpace(ln)
		switch {
		case name == "":
			if n, u, ok := snippetOpen(t); ok {
				name, uses, body = n, u, nil
			}
		case snippetClose(t):
			lines, checks := parseSnippetChecks(body)
			lines, checks = trimSnippetBlankEdges(lines, checks)
			lines = removeCommonIndent(lines)
			s := &snippet{name: name, code: strings.Join(lines, "\n"), lines: lines, checks: checks, uses: uses}
			if n := len(checks); n > 0 {
				s.kind, s.expected = checks[n-1].kind, checks[n-1].expected
			}
			out[name] = s
			name = ""
		default:
			body = append(body, ln)
		}
	}
	return out
}

// parseSnippetChecks separates a region's code lines from its result markers,
// returning the code (markers removed) and the ordered checks. A marker asserts
// the value/output of the code up to that point (its last line being the
// statement under test), so several markers verify successive statements. Both
// the terse single-line forms (`//= EXPR`, `//< TEXT`) and the block forms
// (`/**= … **/`, `/**< … **/`, single- or multi-line) are recognized.
func parseSnippetChecks(body []string) (code []string, checks []snippetCheck) {
	for i := 0; i < len(body); i++ {
		t := strings.TrimSpace(body[i])
		if rest, ok := strings.CutPrefix(t, "//="); ok {
			checks = append(checks, snippetCheck{len(code), snippetValue, strings.TrimSpace(rest)})
			continue
		}
		if rest, ok := strings.CutPrefix(t, "//<"); ok {
			checks = append(checks, snippetCheck{len(code), snippetOutput, strings.TrimSpace(rest)})
			continue
		}
		var kind snippetResultKind
		var sigil string
		switch {
		case strings.HasPrefix(t, "/**="):
			kind, sigil = snippetValue, "/**="
		case strings.HasPrefix(t, "/**<"):
			kind, sigil = snippetOutput, "/**<"
		default:
			code = append(code, body[i])
			continue
		}
		// Collect the block marker body across lines until the closing `**/`.
		var parts []string
		cur := strings.TrimPrefix(t, sigil)
		for {
			if idx := strings.LastIndex(cur, "**/"); idx >= 0 {
				parts = append(parts, cur[:idx])
				break
			}
			parts = append(parts, cur)
			if i++; i >= len(body) {
				break
			}
			cur = body[i]
		}
		checks = append(checks, snippetCheck{len(code), kind, strings.TrimSpace(strings.Join(parts, "\n"))})
	}
	return code, checks
}

// trimSnippetBlankEdges drops leading and trailing blank code lines, shifting the
// checks' line indices so they keep pointing at the same statements.
func trimSnippetBlankEdges(lines []string, checks []snippetCheck) ([]string, []snippetCheck) {
	lead := 0
	for lead < len(lines) && strings.TrimSpace(lines[lead]) == "" {
		lead++
	}
	lines = lines[lead:]
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	for i := range checks {
		if checks[i].lineEnd -= lead; checks[i].lineEnd < 0 {
			checks[i].lineEnd = 0
		}
		if checks[i].lineEnd > len(lines) {
			checks[i].lineEnd = len(lines)
		}
	}
	return lines, checks
}

// removeCommonIndent strips the common leading indentation shared by the non-blank
// lines, without changing the line count (so check indices stay valid).
func removeCommonIndent(lines []string) []string {
	indent := -1
	for _, ln := range lines {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		n := len(ln) - len(strings.TrimLeft(ln, " \t"))
		if indent < 0 || n < indent {
			indent = n
		}
	}
	if indent <= 0 {
		return lines
	}
	for i, ln := range lines {
		if len(ln) >= indent {
			lines[i] = ln[indent:]
		}
	}
	return lines
}

// snippetOpen reports whether the trimmed line opens a snippet region and returns
// its name and its execution-context references. It matches `//snippet NAME` and
// the optional `//snippet NAME uses A B C` form (the names after `uses` are other
// snippets whose code is prepended when this one runs).
func snippetOpen(trimmed string) (name string, uses []string, ok bool) {
	fields, isComment := markerFields(trimmed)
	if !isComment || len(fields) < 2 || fields[0] != "snippet" {
		return "", nil, false
	}
	name = fields[1]
	rest := fields[2:]
	if len(rest) > 0 {
		if rest[0] != "uses" || len(rest) < 2 {
			return "", nil, false // malformed trailer: not a snippet open
		}
		uses = rest[1:]
	}
	return name, uses, true
}

// snippetClose reports whether the trimmed line closes a snippet region
// (`//endsnippet`).
func snippetClose(trimmed string) bool {
	fields, ok := markerFields(trimmed)
	return ok && len(fields) == 1 && fields[0] == "endsnippet"
}

// markerFields splits a `//…` comment line into its space-separated words after
// the `//`. It reports false when the line is not a line comment.
func markerFields(trimmed string) ([]string, bool) {
	if !strings.HasPrefix(trimmed, "//") {
		return nil, false
	}
	return strings.Fields(strings.TrimPrefix(trimmed, "//")), true
}

// expandSnippets replaces every `@snippet NAME` placeholder line in text with a
// fenced code block (fenced with lang) holding the named snippet, plus its
// verified result. When run is true, a snippet with a result marker is executed
// and its actual result checked against the marker; a mismatch or run error is
// returned. An unknown snippet name leaves the placeholder untouched.
func expandSnippets(text string, snippets map[string]*snippet, lang string, run bool) (string, error) {
	if text == "" || !strings.Contains(text, "@snippet") {
		return text, nil
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
		block, err := renderSnippet(snip, snippetPrelude(snip, snippets), lang, run)
		if err != nil {
			return "", err
		}
		out = append(out, block...)
	}
	return strings.Join(out, "\n"), nil
}

// snippetInfo is the serializable view of a snippet exposed on the doc object
// (and in the JSON/YAML encoding): its name, `uses` context references, code,
// result kind, the declared expected text and — when doctests run — the actual
// verified value/output.
type snippetInfo struct {
	Name     string   `json:"name" yaml:"name"`
	Uses     []string `json:"uses,omitempty" yaml:"uses,omitempty"`
	Code     string   `json:"code" yaml:"code"`
	Kind     string   `json:"kind,omitempty" yaml:"kind,omitempty"` // "value" | "output" | ""
	Expected string   `json:"expected,omitempty" yaml:"expected,omitempty"`
	Result   string   `json:"result,omitempty" yaml:"result,omitempty"` // actual value (value kind)
	Output   string   `json:"output,omitempty" yaml:"output,omitempty"` // actual stdout (output kind)
}

// snippetOrder returns the snippet names in source (declaration) order.
func snippetOrder(src []byte) []string {
	var names []string
	for _, ln := range strings.Split(string(src), "\n") {
		if n, _, ok := snippetOpen(strings.TrimSpace(ln)); ok {
			names = append(names, n)
		}
	}
	return names
}

// collectSnippets returns every `//snippet` region in src as a snippetInfo, in
// declaration order. When run is true a snippet with a result marker is executed
// (with its `uses` context prepended) and its actual value/output is filled in;
// a run error aborts. It is the source of the snippets exposed to templates and
// encoded by --json/--yaml.
func collectSnippets(src []byte, run bool) ([]snippetInfo, error) {
	snips := extractSnippets(src)
	names := snippetOrder(src)
	infos := make([]snippetInfo, 0, len(names))
	for _, name := range names {
		s := snips[name]
		if s == nil {
			continue
		}
		info := snippetInfo{Name: s.name, Uses: s.uses, Code: s.code, Expected: s.expected}
		prelude := snippetPrelude(s, snips)
		if len(s.checks) > 0 {
			switch s.kind {
			case snippetValue:
				info.Kind = "value"
			case snippetOutput:
				info.Kind = "output"
			}
			if run {
				// Verify every check; report the last one's result/output (the
				// single-result view exposed to templates and --json/--yaml).
				values, outputs, err := runSnippetChecks(s, prelude)
				if err != nil {
					return nil, err
				}
				last := s.checks[len(s.checks)-1]
				if last.kind == snippetValue {
					info.Result = values[last.lineEnd]
				} else {
					info.Output = outputs[last.lineEnd]
				}
			}
		}
		infos = append(infos, info)
	}
	return infos, nil
}

// snippetsGadArray converts snippet infos to the gad.Array of gad.Dict exposed as
// `doc.snippets` to documentation templates.
func snippetsGadArray(infos []snippetInfo) gad.Array {
	arr := make(gad.Array, len(infos))
	for i, s := range infos {
		d := gad.Dict{"name": gad.Str(s.Name), "code": gad.Str(s.Code)}
		if len(s.Uses) > 0 {
			u := make(gad.Array, len(s.Uses))
			for j, x := range s.Uses {
				u[j] = gad.Str(x)
			}
			d["uses"] = u
		}
		if s.Kind != "" {
			d["kind"] = gad.Str(s.Kind)
		}
		if s.Expected != "" {
			d["expected"] = gad.Str(s.Expected)
		}
		if s.Result != "" {
			d["result"] = gad.Str(s.Result)
		}
		if s.Output != "" {
			d["output"] = gad.Str(s.Output)
		}
		arr[i] = d
	}
	return arr
}

// snippetPrelude returns the concatenated code of a snippet's `uses` execution
// contexts (transitively, in declaration order, each included once), to be
// prepended when the snippet runs so it can reference their definitions. A
// context's own result marker is ignored — only its code is reused. An unknown
// or self reference is skipped; cycles are broken by the seen set.
func snippetPrelude(snip *snippet, snippets map[string]*snippet) string {
	var (
		parts []string
		seen  = map[string]bool{snip.name: true}
		add   func(names []string)
	)
	add = func(names []string) {
		for _, n := range names {
			if seen[n] {
				continue
			}
			seen[n] = true
			ctx, ok := snippets[n]
			if !ok {
				continue
			}
			add(ctx.uses) // context's own contexts come first
			parts = append(parts, ctx.code)
		}
	}
	add(snip.uses)
	return strings.Join(parts, "\n")
}

// withPrelude prepends the execution-context prelude to code (nothing when the
// prelude is empty), so a snippet runs with its `uses` definitions in scope.
func withPrelude(prelude, code string) string {
	if prelude == "" {
		return code
	}
	return prelude + "\n" + code
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

// renderSnippet renders a snippet as Markdown lines: a fenced code block and,
// when the snippet declares a result, its verified value/output shown
// Python-docs style — the value inline as a `// => …` line inside the same code
// block, and STDOUT in an `Output:` text block below it. Fence widths adapt to
// any backticks inside the code/result so an embedded ``` never closes them.
// runSnippetChecks verifies every check of the snippet against the code up to it
// (with the `uses` prelude prepended) and returns, keyed by the check's lineEnd,
// the shown value (value checks) and captured output (output checks). A mismatch
// or run error aborts.
func runSnippetChecks(snip *snippet, prelude string) (values, outputs map[int]string, err error) {
	values, outputs = map[int]string{}, map[int]string{}
	for i, c := range snip.checks {
		code := strings.Join(snip.lines[:c.lineEnd], "\n")
		switch c.kind {
		case snippetValue:
			got, err := evalGadExample(withPrelude(prelude, code))
			if err != nil {
				return nil, nil, fmt.Errorf("snippet %q: %w", snip.name, err)
			}
			exp, err := evalGadExample(withPrelude(prelude, "return "+c.expected))
			if err != nil {
				return nil, nil, fmt.Errorf("snippet %q: evaluating expected %q: %w", snip.name, c.expected, err)
			}
			if !objectsEqual(got, exp) {
				return nil, nil, fmt.Errorf("snippet %q: result mismatch at check %d: got %s, want %s",
					snip.name, i+1, objectStr(got), objectStr(exp))
			}
			values[c.lineEnd] = objectStr(got)
		case snippetOutput:
			_, stdout, err := evalGadExampleCapture(withPrelude(prelude, code))
			if err != nil {
				return nil, nil, fmt.Errorf("snippet %q: %w", snip.name, err)
			}
			if strings.TrimRight(stdout, "\n") != strings.TrimRight(c.expected, "\n") {
				return nil, nil, fmt.Errorf("snippet %q: output mismatch at check %d:\n got: %q\nwant: %q",
					snip.name, i+1, stdout, c.expected)
			}
			outputs[c.lineEnd] = strings.TrimRight(stdout, "\n")
		}
	}
	return values, outputs, nil
}

func renderSnippet(snip *snippet, prelude, lang string, run bool) ([]string, error) {
	if len(snip.checks) == 0 {
		fence := fenceFor(snip.code)
		return []string{fence + lang, snip.code, fence}, nil
	}

	// Resolve each check's shown value/output — verified when run, else the
	// declared expected text.
	values, outputs := map[int]string{}, map[int]string{}
	if run {
		var err error
		if values, outputs, err = runSnippetChecks(snip, prelude); err != nil {
			return nil, err
		}
	} else {
		for _, c := range snip.checks {
			if c.kind == snippetValue {
				values[c.lineEnd] = c.expected
			} else {
				outputs[c.lineEnd] = c.expected
			}
		}
	}

	// Value results render inline as `// => …` right after their statement;
	// output results become `Output:` blocks after the code fence, in order.
	var codeOut, outBlocks []string
	for i, ln := range snip.lines {
		codeOut = append(codeOut, ln)
		if v, ok := values[i+1]; ok {
			codeOut = append(codeOut, "// => "+v)
		}
		if o, ok := outputs[i+1]; ok {
			outBlocks = append(outBlocks, o)
		}
	}
	codeStr := strings.Join(codeOut, "\n")
	fence := fenceFor(codeStr)
	res := []string{fence + lang, codeStr, fence}
	for _, o := range outBlocks {
		tf := fenceFor(o)
		res = append(res, "", "Output:", "", tf+"text", o, tf)
	}
	return res, nil
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

// exampleSource returns the source rendered in the generated "Example" block: the
// snippet markers removed (stripSnippetMarkers) and the leading `/*** … ***/`
// module doc block dropped (it is already rendered as the page prose).
func exampleSource(src []byte) string {
	s := dropLeadingModuleDoc(stripSnippetMarkers(string(src)))
	return strings.TrimLeft(s, "\n")
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

// stripSnippetMarkers removes the `//snippet …` / `//endsnippet` region markers
// and any `/**= … **/` / `/**< … **/` result markers from src so the rendered
// full-example source shows only code.
func stripSnippetMarkers(src string) string {
	var out []string
	inResult := false
	for _, ln := range strings.Split(src, "\n") {
		t := strings.TrimSpace(ln)
		if inResult {
			if strings.Contains(t, "**/") {
				inResult = false
			}
			continue
		}
		if _, _, ok := snippetOpen(t); ok {
			continue
		}
		if snippetClose(t) {
			continue
		}
		if strings.HasPrefix(t, "/**=") || strings.HasPrefix(t, "/**<") {
			if !strings.Contains(t, "**/") {
				inResult = true
			}
			continue
		}
		out = append(out, ln)
	}
	return strings.Join(out, "\n")
}

// dedent removes the longest common leading whitespace from the non-blank lines
// and trims leading/trailing blank lines.
// evalGadExampleCapture runs src with the default builtins and module map,
// capturing (and returning) its standard output alongside the last value.
func evalGadExampleCapture(src string) (gad.Object, string, error) {
	builtins := gad.NewBuiltins().Build()
	opts := gad.CompileOptions{CompilerOptions: gad.CompilerOptions{
		ModuleMap: DefaultModuleMap(".", &sourcePath),
	}}
	var out bytes.Buffer
	eval := gad.NewEval(builtins, defaultSymbolTable(builtins.Builtins().NameSet), opts,
		&gad.RunOpts{StdOut: &out, StdErr: io.Discard})
	eval.VM.Builtins = builtins
	ret, _, err := eval.RunScript(context.Background(), []byte(src))
	return ret, out.String(), err
}
