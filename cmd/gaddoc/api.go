// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

// api.go implements gaddoc's `.gad` API-file emitter: it renders a module's
// public API as documented Gad `export` declarations (typed functions and
// constants). The generated file is itself valid Gad source, so
// `gad doc <module>_api.gad` produces the module documentation from a single
// source of truth. See emitAPIGad.

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"strings"

	gadparser "github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/source"
)

// Signature-normalization patterns. The gad:doc signatures use a readable
// documentation dialect that is slightly richer than the function-header
// grammar; these rewrite it into valid Gad (see normalizeSig).
var (
	// reLegacyRet matches the legacy `) -> ret` return annotation.
	reLegacyRet = regexp.MustCompile(`\)\s*->\s*(.+?)\s*$`)
	// reRet captures a trailing ` <ret>` return annotation.
	reRet = regexp.MustCompile(`\s*<([^>]*)>\s*$`)
	// reArrayType matches a `[elem]` array type (return grammar accepts only the
	// bare `array`, not the element form).
	reArrayType = regexp.MustCompile(`\[[^\]]*\]`)
	// reFuncParamType matches a function-typed parameter (`cb func(...) ret`),
	// unsupported in a parameter type position; the type is dropped.
	reFuncParamType = regexp.MustCompile(`(\b\w+)\s+func\([^)]*\)(\s*\w+)?`)
	// rePipeSpaces collapses the spaces around a union `|`.
	rePipeSpaces = regexp.MustCompile(`\s*\|\s*`)
	// reOnOff maps the `on`/`off` named-default toggles to boolean literals.
	reOnOff = regexp.MustCompile(`=(on|off)\b`)
)

// collectRawComments returns the raw gad:doc comment blocks of the non-test
// packages, in file order (the same ordering formatComments consumes).
func collectRawComments(pkgs map[string]*ast.Package) []string {
	var comments []string
	for _, pkg := range pkgs {
		if strings.HasSuffix(pkg.Name, "_test") {
			continue
		}
		for _, f := range sortedFiles(pkg) {
			for _, c := range f.file.Comments {
				if s, ok := extractComment(c); ok {
					comments = append(comments, s)
				}
			}
		}
	}
	return comments
}

// apiDocgroup parses a source directory and returns the docgroup for the
// selected module (moduleFilter), with its structured api slice populated.
func apiDocgroup(srcDir string) (*docgroup, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, srcDir, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse in %q: %w", srcDir, err)
	}
	dg := &docgroup{}
	dg.process(collectRawComments(pkgs))
	if len(dg.errs) > 0 {
		return nil, fmt.Errorf("%s", strings.Join(dg.errs, "\n"))
	}
	return dg, nil
}

// normalizeSig rewrites a captured gad:doc signature into valid current Gad. The
// documentation dialect is slightly richer than the function-header grammar, so:
//   - legacy `) -> ret` becomes `) <ret>`;
//   - a `<nil>` return (no value) is dropped;
//   - an array return `<[elem]>` becomes `<array>` (the grammar has no element
//     form in return position);
//   - a union return `<a|b>` gains a `_` result name (`<_ a|b>`), which the
//     grammar requires;
//   - a function-typed parameter (`cb func(...) ret`) drops its type;
//   - the `on`/`off` named-default toggles become `true`/`false`.
//
// Union *parameter* types (`x int|uint`) and named result unions (`<r a|b>`) are
// already valid and pass through unchanged.
func normalizeSig(sig string) string {
	sig = strings.TrimSpace(sig)
	if reLegacyRet.MatchString(sig) {
		sig = reLegacyRet.ReplaceAllString(sig, ") <$1>")
	}
	// Parameter-list fixes.
	sig = reFuncParamType.ReplaceAllString(sig, "$1")
	sig = reOnOff.ReplaceAllStringFunc(sig, func(m string) string {
		if m == "=on" {
			return "=true"
		}
		return "=false"
	})
	// Return-type fixes: operate on the trailing ` <ret>` only.
	if m := reRet.FindStringSubmatch(sig); m != nil {
		inner := strings.TrimSpace(m[1])
		head := strings.TrimSuffix(sig, m[0])
		switch {
		case inner == "" || inner == "nil":
			sig = head // drop a valueless / nil return
		default:
			inner = reArrayType.ReplaceAllString(inner, "array")
			inner = rePipeSpaces.ReplaceAllString(inner, "|")
			// A bare union (no result name) needs one: `a|b` -> `_ a|b`.
			if strings.Contains(inner, "|") && !strings.Contains(inner, " ") {
				inner = "_ " + inner
			}
			sig = head + " <" + inner + ">"
		}
	}
	return sig
}

// neutralizeFences retags runnable ```gad fences as ```gad ignore so the prose
// and descriptions copied into an API stub file are never executed as doctests
// (they reference the live module, absent in the stub).
func neutralizeFences(lines []string) []string {
	out := make([]string, len(lines))
	copy(out, lines)
	for i := 0; i < len(out); i++ {
		t := strings.TrimSpace(out[i])
		if t != "```gad" && t != "```Gad" {
			continue
		}
		// A fenced block that contains a `>>>` doctest assertion is a real,
		// runnable example: leave it executable so `gad doc` verifies it. Purely
		// illustrative fences (no assertion) are neutralized to `gad ignore`.
		hasDoctest := false
		j := i + 1
		for ; j < len(out); j++ {
			ct := strings.TrimSpace(out[j])
			if ct == "```" {
				break
			}
			if strings.HasPrefix(ct, ">>>") {
				hasDoctest = true
			}
		}
		if !hasDoctest {
			out[i] = strings.Replace(out[i], t, "```gad ignore", 1)
		}
		i = j
	}
	return out
}

// parsesRaw reports whether src is syntactically valid Gad. The parser may panic
// (rather than return an error) on some malformed input, so the panic is
// recovered and reported as "does not parse".
func parsesRaw(src string) (ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	fs := source.NewFileSet()
	f := fs.AddFileData("api", -1, []byte(src+"\n"))
	_, err := gadparser.NewParser(f, nil).ParseFile()
	return err == nil
}

// exportParses reports whether `export <decl>` is valid Gad.
func exportParses(decl string) bool { return parsesRaw("export " + decl) }

// exportParsesRaw reports whether a full export statement source is valid Gad.
func exportParsesRaw(src string) bool { return parsesRaw(src) }

// emitAPIGad renders the module's public API as a documented Gad source file and
// returns it together with any warnings (signatures that did not parse and were
// emitted with a fallback, so the source gad:doc can be fixed).
// --- gad:samples usage examples ------------------------------------------

// reSnippetOpen matches the `//snippet NAME …` opening of a doctest region (the
// same regions `gad doc` validates via `/**= … **/`), keyed here by member name.
var reSnippetOpen = regexp.MustCompile(`^\s*//snippet\s+(\w+)`)

// sampleFile holds the usage snippets parsed from a `gad:samples` file, keyed by
// the exported member name. Each member's example lives in a
// `//snippet NAME … //endsnippet` region — the standard doctest snippet form, so
// the file is validated by `gad doc` (its `/**= … **/` results are checked) and
// no new format is introduced.
type sampleFile struct {
	path     string
	snippets map[string][]string
}

// parseSampleFile reads path into per-member snippets. A missing file yields an
// empty set (it is created on demand when auto-scaffolding).
func parseSampleFile(path string) (*sampleFile, error) {
	sf := &sampleFile{path: path, snippets: map[string][]string{}}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return sf, nil
		}
		return nil, err
	}
	cur := ""
	var buf []string
	for _, ln := range strings.Split(string(data), "\n") {
		t := strings.TrimSpace(ln)
		if cur == "" {
			if m := reSnippetOpen.FindStringSubmatch(t); m != nil {
				cur, buf = m[1], nil
			}
			continue
		}
		if t == "//endsnippet" {
			sf.snippets[cur] = trimBlankEdges(buf)
			cur = ""
			continue
		}
		buf = append(buf, ln)
	}
	return sf, nil
}

// exampleDoc returns the `## Example` doc lines (a non-running fenced `gad`
// block) for a member's snippet, or nil when the file has no snippet for it.
func exampleDoc(sf *sampleFile, name string) []string {
	if sf == nil {
		return nil
	}
	snip := sf.snippets[name]
	if len(snip) == 0 {
		return nil
	}
	// Emit a runnable ```gad fence so `gad doc` executes the example when it
	// renders the final documentation. The snippet's `//= EXPR` value markers are
	// translated to the fence doctest form `>>> EXPR` so the value is verified
	// there too; `//< TEXT` (fences have no output assertion) is kept as a comment.
	out := []string{"", "## Example", "", "```gad"}
	for _, ln := range snip {
		t := strings.TrimSpace(ln)
		if rest, ok := strings.CutPrefix(t, "//="); ok {
			out = append(out, ">>> "+strings.TrimSpace(rest))
		} else if rest, ok := strings.CutPrefix(t, "//<"); ok {
			out = append(out, "// output: "+strings.TrimSpace(rest))
		} else {
			out = append(out, ln)
		}
	}
	return append(out, "```")
}

// scaffoldMissingSamples appends a `/** ### Name **/` stub for every exported
// member that has no section yet, so `auto` files grow to cover the whole API.
// Existing snippets are left untouched. It creates the file with a header when
// absent.
func scaffoldMissingSamples(module string, dg *docgroup, sf *sampleFile) error {
	var missing []string
	seen := map[string]bool{}
	for _, s := range dg.api {
		if seen[s.name] {
			continue
		}
		seen[s.name] = true
		if _, ok := sf.snippets[s.name]; !ok {
			missing = append(missing, s.name)
		}
	}

	existing, err := os.ReadFile(sf.path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if len(missing) == 0 && err == nil {
		return nil
	}

	var b strings.Builder
	if os.IsNotExist(err) {
		b.WriteString("// Samples of module " + module + ": one snippet region per exported\n// member. These are standard doctest snippets, run and checked by `gad doc`,\n// and merged into the module API as `## Example` sections by gaddoc (the\n// `gad:samples` directive).\n")
	} else {
		b.Write(existing)
		if !strings.HasSuffix(string(existing), "\n") {
			b.WriteByte('\n')
		}
	}
	for _, name := range missing {
		b.WriteString("\n//snippet " + name + "\n//endsnippet\n")
	}
	return os.WriteFile(sf.path, []byte(b.String()), 0o644)
}

func emitAPIGad(dg *docgroup, sf *sampleFile) (string, []string) {
	var (
		b        strings.Builder
		warnings []string
	)

	// Module block: the `# <module> module` title + prose, as a leading `/** **/`
	// section doc. A blank line after it detaches it as the module/section doc.
	prose := trimBlankEdges(neutralizeFences(dg.docs))
	b.WriteString("/**\n")
	for _, ln := range prose {
		b.WriteString(ln)
		b.WriteByte('\n')
	}
	b.WriteString("**/\n\n")

	for _, s := range dg.api {
		switch s.kind {
		case "func":
			if len(s.overloads) > 1 {
				emitOverloadedFunc(&b, s.name, s.overloads, sf, &warnings)
			} else if len(s.overloads) == 1 {
				emitSingleFunc(&b, s.name, s.overloads[0], sf, &warnings)
			}
		case "const":
			desc := trimBlankEdges(neutralizeFences(s.desc))
			desc = append(desc, exampleDoc(sf, s.name)...)
			writeDocBlock(&b, desc)
			b.WriteString("export const " + s.name + " = " + s.sig + "\n\n")
		}
	}

	return strings.TrimRight(b.String(), "\n") + "\n", warnings
}

// emitSingleFunc writes a single-signature function export: the typed
// `export name(params) <ret> => nil` when it parses, an untyped variadic stub
// (with the real signature kept in the doc) when it does not, or nothing when
// the name is reserved.
func emitSingleFunc(b *strings.Builder, name string, ov apiOverload, sf *sampleFile, warnings *[]string) {
	sig := normalizeSig(ov.sig)
	decl := sig + " => nil"
	desc := trimBlankEdges(neutralizeFences(ov.desc))
	desc = append(desc, exampleDoc(sf, name)...)
	switch {
	case exportParses(decl):
		writeDocBlock(b, desc)
		b.WriteString("export " + decl + "\n\n")
	case exportParses(name + "(*args) => nil"):
		*warnings = append(*warnings, fmt.Sprintf("%s: signature does not parse: %s", name, sig))
		writeDocBlock(b, append([]string{"`" + sig + "`", ""}, desc...))
		b.WriteString("export " + name + "(*args) => nil\n\n")
	default:
		*warnings = append(*warnings, fmt.Sprintf("%s: reserved name, skipped", name))
	}
}

// emitOverloadedFunc writes a multi-signature function as
// `export func NAME { (params) <ret> => nil … }`, one method per overload with
// its own doc. If the whole declaration does not parse it degrades to a single
// untyped variadic stub keeping every signature in the doc.
func emitOverloadedFunc(b *strings.Builder, name string, overloads []apiOverload, sf *sampleFile, warnings *[]string) {
	headers := make([]string, len(overloads))
	for i, ov := range overloads {
		headers[i] = stripSigName(normalizeSig(ov.sig))
	}

	// The usage example is per-member, so it leads the whole func-with-methods.
	if ex := exampleDoc(sf, name); len(ex) > 0 {
		writeDocBlock(b, trimBlankEdges(ex))
	}

	var body strings.Builder
	body.WriteString("export func " + name + " {\n")
	for i, ov := range overloads {
		for _, d := range trimBlankEdges(neutralizeFences(ov.desc)) {
			body.WriteString(docLine(d))
		}
		body.WriteString(headers[i] + " => nil\n")
	}
	body.WriteString("}")

	if exportParsesRaw(body.String()) {
		b.WriteString(body.String())
		b.WriteString("\n\n")
		return
	}

	// Fallback: an untyped stub documenting each real signature.
	*warnings = append(*warnings, fmt.Sprintf("%s: overloads do not parse", name))
	doc := make([]string, 0, len(overloads)*2)
	for _, ov := range overloads {
		doc = append(doc, "`"+name+stripSigName(normalizeSig(ov.sig))+"`")
	}
	writeDocBlock(b, doc)
	b.WriteString("export " + name + "(*args) => nil\n\n")
}

// stripSigName removes the leading function name from a signature, leaving the
// parenthesized parameter list and return: `f(a int) <int>` -> `(a int) <int>`.
func stripSigName(sig string) string {
	if i := strings.IndexByte(sig, '('); i >= 0 {
		return sig[i:]
	}
	return sig
}

// docLine renders one description line as a `///` doc-comment line (used for a
// method's doc inside a func-with-methods body).
func docLine(s string) string {
	if s == "" {
		return "///\n"
	}
	return "/// " + s + "\n"
}

// writeDocBlock writes a `/** … **/` block doc comment for the given lines,
// directly above the declaration that follows (no blank line between, so it
// attaches as a lead doc). Nothing is written when there is no description.
func writeDocBlock(b *strings.Builder, lines []string) {
	if len(lines) == 0 {
		return
	}
	b.WriteString("/**\n")
	for _, ln := range lines {
		b.WriteString(ln)
		b.WriteByte('\n')
	}
	b.WriteString("**/\n")
}

// trimBlankEdges drops leading and trailing blank lines.
func trimBlankEdges(lines []string) []string {
	i, j := 0, len(lines)
	for i < j && strings.TrimSpace(lines[i]) == "" {
		i++
	}
	for j > i && strings.TrimSpace(lines[j-1]) == "" {
		j--
	}
	return lines[i:j]
}

// runAPIMode implements `gaddoc api <srcDir> <outFile> <module>`.
func runAPIMode(srcDir, outFile, module string) error {
	moduleFilter = module
	dg, err := apiDocgroup(srcDir)
	if err != nil {
		return err
	}

	// A `gad:samples [flags] <path>` directive links the module to a file of
	// per-member usage examples that are merged into each export's doc as an
	// `## Example` section. With the `auto` flag, members without a section yet
	// are scaffolded into the file so coverage grows over time.
	var sf *sampleFile
	if dg.samplesPath != "" {
		if dg.samplesAuto {
			pre, err := parseSampleFile(dg.samplesPath)
			if err != nil {
				return fmt.Errorf("read samples %q: %w", dg.samplesPath, err)
			}
			if err := scaffoldMissingSamples(module, dg, pre); err != nil {
				return fmt.Errorf("scaffold samples %q: %w", dg.samplesPath, err)
			}
		}
		if sf, err = parseSampleFile(dg.samplesPath); err != nil {
			return fmt.Errorf("read samples %q: %w", dg.samplesPath, err)
		}
	}

	src, warnings := emitAPIGad(dg, sf)
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "gaddoc api: %s: %s\n", module, w)
	}
	if outFile == "-" {
		fmt.Print(src)
		return nil
	}
	if err := os.WriteFile(outFile, []byte(src), 0o644); err != nil {
		return fmt.Errorf("failed to write %q: %w", outFile, err)
	}
	return nil
}
