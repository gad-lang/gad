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
	for i, ln := range lines {
		t := strings.TrimSpace(ln)
		if t == "```gad" || t == "```Gad" {
			out[i] = strings.Replace(ln, t, "```gad ignore", 1)
		} else {
			out[i] = ln
		}
	}
	return out
}

// exportParses reports whether `export <decl>` is syntactically valid Gad. The
// parser may panic (rather than return an error) on some malformed input, so the
// panic is recovered and reported as "does not parse".
func exportParses(decl string) (ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	src := "export " + decl + "\n"
	fs := source.NewFileSet()
	f := fs.AddFileData("api", -1, []byte(src))
	_, err := gadparser.NewParser(f, nil).ParseFile()
	return err == nil
}

// emitAPIGad renders the module's public API as a documented Gad source file and
// returns it together with any warnings (signatures that did not parse and were
// emitted with a fallback, so the source gad:doc can be fixed).
func emitAPIGad(dg *docgroup) (string, []string) {
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
			sig := normalizeSig(s.sig)
			decl := sig + " => nil"
			desc := trimBlankEdges(neutralizeFences(s.desc))
			switch {
			case exportParses(decl):
				writeDocBlock(&b, desc)
				b.WriteString("export " + decl + "\n\n")
			case exportParses(s.name + "(*args) => nil"):
				// The typed signature does not parse (a form the grammar cannot
				// express); fall back to an untyped variadic stub and keep the
				// real (prose) signature in the doc for the reader.
				warnings = append(warnings, fmt.Sprintf("%s: signature does not parse: %s", s.name, sig))
				writeDocBlock(&b, append([]string{"`" + sig + "`", ""}, desc...))
				b.WriteString("export " + s.name + "(*args) => nil\n\n")
			default:
				// Even the bare name does not parse as an export (a reserved
				// keyword such as `in`); skip it so the file stays valid.
				warnings = append(warnings, fmt.Sprintf("%s: reserved name, skipped", s.name))
			}
		case "const":
			writeDocBlock(&b, trimBlankEdges(neutralizeFences(s.desc)))
			b.WriteString("export const " + s.name + " = " + s.sig + "\n\n")
		}
	}

	return strings.TrimRight(b.String(), "\n") + "\n", warnings
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
	src, warnings := emitAPIGad(dg)
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
