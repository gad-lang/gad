// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.
//
// gaddoc reads a go package, which must be a gad stdlib module, extracts and
// groups package comments to create the gad module documentation.
//
// usage: ./gaddoc <source dir> <output file>
//
// Examples:
//
// go run ./cmd/gaddoc ./stdlib/time ./doc/stdlib-time.md
//
// go run ./cmd/gaddoc ./stdlib/json -
package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/gad-lang/gad"
	gadfmt "github.com/gad-lang/gad/stdlib/fmt"
	gadjson "github.com/gad-lang/gad/stdlib/json"
	gadstrings "github.com/gad-lang/gad/stdlib/strings"
	gadtime "github.com/gad-lang/gad/stdlib/time"
)

const gadDocPrefix = "gad:doc"

var (
	reModuleHeader  = regexp.MustCompile(`^\s*#\s+(\w+)\s+module`)
	reTypeHeader    = regexp.MustCompile(`^\s*##\s+Types`)
	reConstHeader   = regexp.MustCompile(`^\s*##\s+Constants`)
	reFuncHeader    = regexp.MustCompile(`^\s*##\s+Functions`)
	reConvHeader    = regexp.MustCompile(`^\s*##\s+Converters`)
	reMethHeader    = regexp.MustCompile(`^\s*##\s+Method Overrides`)
	reClassHeader   = regexp.MustCompile(`^\s*##\s+Classes`)
	reEnumHeader    = regexp.MustCompile(`^\s*##\s+Enums`)
	reGadMethHeader = regexp.MustCompile(`^\s*##\s+Methods`)
	rePropHeader    = regexp.MustCompile(`^\s*##\s+Properties`)
	// Function header annotation: `Name(params) <ret>` (new syntax) or the
	// legacy `Name(params) -> ret`. The params may include named params (`;`).
	reFuncAnnot    = regexp.MustCompile(`^\s*(\w+)\(.*\)\s*(?:<[^>]*>|->\s+\S.*)\s*$`)
	reLevel2header = regexp.MustCompile(`^\s*##\s`)
	reWordStart    = regexp.MustCompile(`^\s*\w+`)
)

type docgroup struct {
	module    string
	docs      []string
	types     []string
	consts    []string
	funcs     []string
	convs     []string
	methods   []string // Go-level method overrides
	classes   []string
	enums     []string
	gadMeths  []string // gad-level methods
	props     []string
	errs      []string
	funcHLine bool
	// skipDesc skips the gad:doc comment description lines of the current
	// function because the description is taken from the function's Usage.
	skipDesc bool
}

func (dg *docgroup) addError(msg string) {
	dg.errs = append(dg.errs, msg)
}

func (dg *docgroup) process(comments []string) {
	dg.types = append(dg.types, "## Types\n")
	dg.consts = append(dg.consts, "## Constants\n")
	dg.funcs = append(dg.funcs, "## Functions\n")
	dg.convs = append(dg.convs, "## Converters\n")
	dg.methods = append(dg.methods, "## Method Overrides\n")
	dg.classes = append(dg.classes, "## Classes\n")
	dg.enums = append(dg.enums, "## Enums\n")
	dg.gadMeths = append(dg.gadMeths, "## Methods\n")
	dg.props = append(dg.props, "## Properties\n")
	var lines []string
	for _, p := range comments {
		lines = append(lines, strings.Split(p, "\n")...)
	}

	// Collect every `# NAME module` header position. A source directory may now
	// hold several modules (the builtin module namespaces live together in the
	// root package), so moduleFilter selects which one to emit; the module's
	// blocks run from its header up to the next module header (or EOF).
	type hdr struct {
		idx  int
		name string
	}
	var hdrs []hdr
	for i, p := range lines {
		if m := reModuleHeader.FindStringSubmatch(p); len(m) > 1 {
			hdrs = append(hdrs, hdr{i, m[len(m)-1]})
		}
	}
	if len(hdrs) == 0 {
		dg.addError("no module header found")
		return
	}

	sel := 0
	if moduleFilter != "" {
		sel = -1
		for k, h := range hdrs {
			if h.name == moduleFilter {
				sel = k
				break
			}
		}
		if sel < 0 {
			dg.addError("module header not found: " + moduleFilter)
			return
		}
	}

	// Preamble before the first module header (intro text); later modules'
	// ranges are bounded by the preceding module, so they have none.
	if sel == 0 {
		dg.docs = append(dg.docs, lines[:hdrs[0].idx]...)
	}
	dg.module = hdrs[sel].name
	dg.docs = append(dg.docs, fmt.Sprintf("# `%s` module", dg.module))

	end := len(lines)
	if sel+1 < len(hdrs) {
		end = hdrs[sel+1].idx
	}
	dg.processBlocks(lines[hdrs[sel].idx+1 : end])
}

// moduleFilter, when non-empty, selects which module's gad:doc to emit from a
// source directory that defines more than one (set from the optional 3rd CLI
// argument).
var moduleFilter string

func (dg *docgroup) processBlocks(lines []string) {
	const (
		unknown = iota
		typeBlock
		constBlock
		funcBlock
		convBlock
		methBlock
		classBlock
		enumBlock
		gadMethBlock
		propBlock
	)
	block := unknown
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		line = strings.ReplaceAll(line, "\r", "")
		line = strings.ReplaceAll(line, "\t", "    ")
		switch block {
		case unknown:
			if reTypeHeader.MatchString(line) {
				block = typeBlock
			} else if reConstHeader.MatchString(line) {
				block = constBlock
			} else if reFuncHeader.MatchString(line) {
				block = funcBlock
			} else if reConvHeader.MatchString(line) {
				block = convBlock
			} else if reMethHeader.MatchString(line) {
				block = methBlock
			} else if reClassHeader.MatchString(line) {
				block = classBlock
			} else if reEnumHeader.MatchString(line) {
				block = enumBlock
			} else if reGadMethHeader.MatchString(line) {
				block = gadMethBlock
			} else if rePropHeader.MatchString(line) {
				block = propBlock
			} else {
				dg.docs = append(dg.docs, line)
			}
		case typeBlock,
			constBlock,
			funcBlock,
			convBlock,
			methBlock,
			classBlock,
			enumBlock,
			gadMethBlock,
			propBlock:
			if reLevel2header.MatchString(line) {
				if i > 0 {
					i--
				}
				block = unknown
				continue
			}
			switch block {
			case typeBlock:
				dg.processTypeBlock(line)
			case constBlock:
				dg.processConstBlock(line)
			case funcBlock:
				dg.processFuncBlock(line)
			case convBlock:
				dg.convs = append(dg.convs, line)
			case methBlock:
				dg.methods = append(dg.methods, line)
			case classBlock:
				dg.classes = append(dg.classes, line)
			case enumBlock:
				dg.enums = append(dg.enums, line)
			case gadMethBlock:
				dg.gadMeths = append(dg.gadMeths, line)
			case propBlock:
				dg.props = append(dg.props, line)
			}
		}
	}
}

func (dg *docgroup) processTypeBlock(line string) {
	dg.types = append(dg.types, line)
}

func (dg *docgroup) processConstBlock(line string) {
	matched := reWordStart.MatchString(line)
	if !matched {
		dg.consts = append(dg.consts, line)
		return
	}
	line = fmt.Sprintf("- `%s`: %s", strings.TrimSpace(line), getModuleItem(dg.module, line))
	dg.consts = append(dg.consts, line)
}

func (dg *docgroup) processFuncBlock(line string) {
	if !reFuncAnnot.MatchString(line) {
		// description line: skip it when the doc comes from the function Usage
		if !dg.skipDesc {
			dg.funcs = append(dg.funcs, line)
		}
		return
	}

	dg.skipDesc = false
	line = strings.TrimSpace(line)
	parts := reFuncAnnot.FindStringSubmatch(line)

	var name string
	if len(parts) >= 2 {
		name = parts[len(parts)-1]
	}

	// Prefer the live function definition: the signature is generated from the
	// function Header (set via WithHeader / FunctionWithParams /
	// FunctionWithNamedParams) and the description from its Usage. Fall back to
	// the gad:doc comment when the metadata is absent.
	sig := line
	var usage string
	if fm, ok := getModuleFunc(dg.module, name); ok {
		if fm.header != nil {
			sig = fm.name + fm.header.String()
		}
		usage = strings.TrimSpace(fm.usage)
	}

	if dg.funcHLine {
		dg.funcs = append(dg.funcs, "---\n")
	} else {
		dg.funcHLine = true
	}

	if name == "" {
		dg.addError(fmt.Sprintf("invalid function name at %s", line))
	} else if getModuleItem(dg.module, name) == "" {
		dg.addError(fmt.Sprintf("function not exist in module:%s", line))
	}

	dg.funcs = append(dg.funcs, fmt.Sprintf("`%s`\n", sig))

	if usage != "" {
		dg.funcs = append(dg.funcs, "", usage, "")
		dg.skipDesc = true
	}
}

var moduleDataCache = map[string]gad.Dict{}

// moduleData returns (and caches) the runtime data dict of a stdlib module.
func moduleData(module string) gad.Dict {
	if d, ok := moduleDataCache[module]; ok {
		return d
	}
	var initFn gad.ModuleInitFunc
	switch module {
	case "time":
		initFn = gadtime.ModuleInit
	case "strings":
		initFn = gadstrings.ModuleInit
	case "fmt":
		initFn = gadfmt.ModuleInit
	case "json":
		initFn = gadjson.ModuleInit
	default:
		panic(fmt.Errorf("unknown module:%s", module))
	}
	// the module init requires a real *Module (it reads module.Spec), so build
	// one from the module name instead of passing nil
	d := initFn.MustGetData(
		gad.NewModule(gad.NewModuleSpecFromName(module))).ToDict()
	moduleDataCache[module] = d
	return d
}

// funcMeta is the doc-relevant metadata shared by *gad.Function and
// *gad.BuiltinFunction.
type funcMeta struct {
	name   string
	header *gad.FunctionHeader
	usage  string
}

// getModuleFunc returns the documentable metadata for the named module item
// when it is a plain function (*gad.Function or *gad.BuiltinFunction). The
// signature is built from FuncName + Header and the description from Usage
// (e.g. set via FunctionWithUsage).
func getModuleFunc(module, name string) (m funcMeta, ok bool) {
	switch fn := moduleData(module)[name].(type) {
	case *gad.Function:
		return funcMeta{fn.FuncName, fn.Header, fn.Usage}, true
	case *gad.BuiltinFunction:
		return funcMeta{fn.FuncName, fn.Header, fn.Usage}, true
	}
	return funcMeta{}, false
}

func getModuleItem(module, key string) string {
	v := moduleData(module)[key]
	if v == nil {
		return ""
	}
	t := v.Type().Name()
	format := "%s(%q)"
	if t != "string" {
		format = "%s(%s)"
	}
	return fmt.Sprintf(format, v.Type().Name(), v.ToString())
}

// headingSlug converts a markdown heading text to a GitHub-style anchor slug.
func headingSlug(heading string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(heading) {
		switch {
		case r == ' ':
			b.WriteByte('-')
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-':
			b.WriteRune(r)
		}
	}
	return b.String()
}

// generateTOC scans lines for ## headings and returns a TOC block.
func generateTOC(lines []string) []string {
	var entries []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			heading := strings.TrimPrefix(trimmed, "## ")
			entries = append(entries, fmt.Sprintf("- [%s](#%s)", heading, headingSlug(heading)))
		}
	}
	if len(entries) == 0 {
		return nil
	}
	result := []string{"## Contents", ""}
	result = append(result, entries...)
	result = append(result, "")
	return result
}

func formatComments(comments []string) ([]string, error) {
	d := docgroup{}
	d.process(comments)
	if len(d.errs) > 0 {
		return nil, errors.New(strings.Join(d.errs, "\n"))
	}

	for len(d.funcs) > 0 {
		s := strings.Trim(d.funcs[len(d.funcs)-1], "\n")
		if s == "" {
			d.funcs = d.funcs[:len(d.funcs)-1]
		} else {
			break
		}
	}

	// Build the section blocks in canonical order (classes+enums before consts;
	// gad methods+props after consts; Go-level converters+overrides last).
	var sections []string
	if len(d.classes) > 1 {
		sections = append(sections, d.classes...)
	}
	if len(d.enums) > 1 {
		sections = append(sections, d.enums...)
	}
	if len(d.types) > 1 {
		sections = append(sections, d.types...)
	}
	if len(d.consts) > 1 {
		sections = append(sections, d.consts...)
	}
	if len(d.props) > 1 {
		sections = append(sections, d.props...)
	}
	if len(d.gadMeths) > 1 {
		sections = append(sections, d.gadMeths...)
	}
	if len(d.funcs) > 1 {
		sections = append(sections, d.funcs...)
	}
	if len(d.convs) > 1 {
		sections = append(sections, d.convs...)
	}
	if len(d.methods) > 1 {
		sections = append(sections, d.methods...)
	}

	toc := generateTOC(sections)

	var out []string
	// Title is the first element of d.docs; insert the TOC right after it.
	if len(d.docs) > 0 {
		out = append(out, d.docs[0])
		if len(toc) > 0 {
			out = append(out, "")
			out = append(out, toc...)
		}
		out = append(out, d.docs[1:]...)
	}
	out = append(out, sections...)
	return out, nil
}

type file struct {
	file *ast.File
	name string
}

func sortedFiles(pkg *ast.Package) []file {
	files := make([]file, 0, len(pkg.Files))

	for name, f := range pkg.Files {
		files = append(files, file{file: f, name: filepath.Base(name)})
	}

	// Sort files passed in according to these rules:
	// 1. file with name "doc.go"
	// 2. file with name "module.go"
	// 3. alphabetical order
	sort.Slice(files, func(i, j int) bool {
		ni, nj := files[i].name, files[j].name

		switch ni {
		case "doc.go":
			return true
		case "module.go":
			switch nj {
			case "doc.go":
				return false
			default:
				return true
			}
		default:
			switch nj {
			case "doc.go", "module.go":
				return false
			default:
				return ni < nj
			}
		}
	})
	return files
}

func extractComment(cgrp *ast.CommentGroup) (string, bool) {
	s := cgrp.Text()
	parts := strings.SplitN(s, "\n", 2)
	p0 := strings.TrimSpace(parts[0])
	if strings.HasPrefix(p0, gadDocPrefix) {
		return parts[1], true
	}
	return "", false
}

func extractPackageComments(pkg *ast.Package) ([]string, error) {
	files := sortedFiles(pkg)

	var comments []string
	for _, f := range files {
		for _, c := range f.file.Comments {
			s, ok := extractComment(c)
			if ok {
				comments = append(comments, s)
			}
		}
	}
	return formatComments(comments)
}

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("usage: %s <source dir> <output file>\n"+
			"single \"-\" can be used to write to stdout", os.Args[0])
		return
	}

	srcDir := os.Args[1]
	outFile := os.Args[2]
	if len(os.Args) > 3 {
		moduleFilter = os.Args[3]
	}

	fset := token.NewFileSet()

	pkgs, err := parser.ParseDir(fset, srcDir, nil, parser.ParseComments)
	if err != nil {
		err = fmt.Errorf("failed to parse in '%s' error: %w", srcDir, err)
		checkerr(err)
	}

	if outFile == "-" {
		err = writeTo(pkgs, os.Stdout)
	} else {
		err = writeToFile(pkgs, outFile)
	}
	checkerr(err)
}

func writeToFile(pkgs map[string]*ast.Package, outFile string) error {
	f, err := os.Create(outFile)
	if err != nil {
		return fmt.Errorf("failed to create output file '%s' error: %w", outFile, err)
	}
	_, err = fmt.Fprintf(f, "\n[//]: <> (Generated by gaddoc. DO NOT EDIT.)\n\n")
	if err != nil {
		_ = f.Close()
		return fmt.Errorf("failed to write header to output '%s' error: %w", outFile, err)
	}

	err = writeTo(pkgs, f)
	errClose := f.Close()
	if err != nil {
		return fmt.Errorf("failed to write to output '%s' error: %w", outFile, err)
	}
	if errClose != nil {
		err = fmt.Errorf("failed to close output '%s' error: %w", outFile, errClose)
	}
	return err
}

func writeTo(pkgs map[string]*ast.Package, dst io.Writer) error {
	for _, pkg := range pkgs {
		if strings.HasSuffix(pkg.Name, "_test") {
			continue
		}
		comments, err := extractPackageComments(pkg)
		if err != nil {
			return err
		}
		for _, c := range comments {
			fmt.Fprintln(dst, c)
		}
	}
	return nil
}

func checkerr(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
