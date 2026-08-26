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
	"strconv"
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
	// A `gad:samples [flags] <path>` directive links a module to a `.gad` file of
	// usage examples (see api.go). flags is a comma list (e.g. `module,auto`).
	reSamplesDir = regexp.MustCompile(`^\s*gad:samples\s+\[([^\]]*)\]\s+(\S+)`)
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

	// samplesPath is the usage-examples file linked by a `gad:samples [flags]
	// <path>` directive in the module doc (path is repo-relative). samplesAuto is
	// set when the flags include `auto`: missing exported members are scaffolded
	// into that file so every member gets an Example section over time.
	samplesPath string
	samplesAuto bool

	// Structured capture for the `.gad` API emitter (emitAPIGad). Populated in
	// parallel with the Markdown buckets so the public API can also be rendered
	// as documented Gad `export` declarations.
	api        []apiSym
	curFuncIdx int // index into api of the func accumulating desc/overloads; -1 if none
}

// apiSym is one documented public-API symbol captured for the `.gad` emitter.
type apiSym struct {
	kind string // "func" | "const"
	name string // symbol name
	sig  string // const: the value literal
	desc []string
	// overloads holds a function's one-or-more signatures (consecutive gad:doc
	// signature lines sharing the name); each carries its own description.
	overloads []apiOverload
}

// apiOverload is one signature of a (possibly multi-signature) function.
type apiOverload struct {
	sig  string   // `name(params) <ret>`
	desc []string // description lines, trimmed of trailing blanks
}

// captureFuncSig records a signature line for the API emitter, grouping
// consecutive signatures that share the function name as overloads of one entry.
func (dg *docgroup) captureFuncSig(name, sig string) {
	if dg.curFuncIdx >= 0 && dg.api[dg.curFuncIdx].kind == "func" && dg.api[dg.curFuncIdx].name == name {
		dg.api[dg.curFuncIdx].overloads = append(dg.api[dg.curFuncIdx].overloads, apiOverload{sig: sig})
	} else {
		dg.api = append(dg.api, apiSym{kind: "func", name: name, overloads: []apiOverload{{sig: sig}}})
		dg.curFuncIdx = len(dg.api) - 1
	}
}

// appendFuncDesc attaches a description line to the current function's latest
// overload.
func (dg *docgroup) appendFuncDesc(line string) {
	if dg.curFuncIdx < 0 {
		return
	}
	ovs := dg.api[dg.curFuncIdx].overloads
	if n := len(ovs); n > 0 {
		ovs[n-1].desc = append(ovs[n-1].desc, line)
	}
}

func (dg *docgroup) addError(msg string) {
	dg.errs = append(dg.errs, msg)
}

func (dg *docgroup) process(comments []string) {
	dg.curFuncIdx = -1
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
	dg.docs = append(dg.docs, moduleTitle(dg.module))

	end := len(lines)
	if sel+1 < len(hdrs) {
		end = hdrs[sel+1].idx
	}
	dg.processBlocks(lines[hdrs[sel].idx+1 : end])
}

// moduleTitle renders the top `#` heading for a documentation group. Importable
// stdlib modules are titled "`name` module"; the synthetic `types` group is not a
// module — it collects the globally-available built-in object types — so it gets
// a plain title without the "module" suffix. The `# <name> module` source header
// stays a module marker only for boundary detection (see reModuleHeader).
func moduleTitle(module string) string {
	if module == "types" {
		return "# Built-in Types"
	}
	return fmt.Sprintf("# `%s` module", module)
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
		// A `gad:samples [flags] <path>` directive links the module to a usage
		// examples file; capture it and drop it from the rendered doc.
		if m := reSamplesDir.FindStringSubmatch(line); m != nil {
			dg.samplesPath = strings.TrimSpace(m[2])
			for _, fl := range strings.Split(m[1], ",") {
				if strings.TrimSpace(fl) == "auto" {
					dg.samplesAuto = true
				}
			}
			continue
		}
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
	name := strings.TrimSpace(line)
	// Capture the const for the `.gad` emitter only when it is a real module
	// member (a bare name that resolves to a value); prose lines are skipped.
	if lit := valueLiteral(dg.module, name); lit != "" {
		dg.api = append(dg.api, apiSym{kind: "const", name: name, sig: lit})
		dg.curFuncIdx = -1 // a const ends any function's description accumulation
	}
	line = fmt.Sprintf("- `%s`: %s", name, getModuleItem(dg.module, line))
	dg.consts = append(dg.consts, line)
}

// valueLiteral returns the Gad source literal of a module const (a quoted str,
// or the numeric/bool literal), or "" when the name is not a documentable
// scalar value of the module.
func valueLiteral(module, name string) string {
	v := moduleData(module)[name]
	if v == nil {
		return ""
	}
	switch v := v.(type) {
	case gad.Str:
		return strconv.Quote(string(v))
	case gad.Int, gad.Uint, gad.Float, gad.Bool, gad.Char:
		return v.ToString()
	}
	return ""
}

func (dg *docgroup) processFuncBlock(line string) {
	if !reFuncAnnot.MatchString(line) {
		// description line: skip it when the doc comes from the function Usage
		if !dg.skipDesc {
			dg.funcs = append(dg.funcs, line)
		}
		// Capture the description for the `.gad` emitter regardless of skipDesc:
		// the API file documents each export with its own doc, attached to the
		// current function's latest overload.
		dg.appendFuncDesc(line)
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

	// Capture the signature for the `.gad` emitter (groups overloads by name).
	dg.captureFuncSig(name, sig)

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

	dg.funcs = append(dg.funcs, fmt.Sprintf("```gad\n%s\n```\n", sig))

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
	// The `types` doc module is a documentation-only grouping of global object
	// types (see builtin_types_doc.go). It has no importable members to resolve —
	// its `## Type …` blocks are pure prose — so expose an empty data dict.
	if module == "types" {
		d := gad.Dict{}
		moduleDataCache[module] = d
		return d
	}
	// base64 is a const-only builtin namespace (Go's encoding/base64); expose its
	// encodings so `# base64 module` gad:doc resolves them.
	if module == "base64" {
		d := gad.Base64Module().ToDict()
		moduleDataCache[module] = d
		return d
	}
	// gad is the language's own reflective/meta builtin namespace; expose its
	// members so `# gad module` gad:doc resolves them.
	if module == "gad" {
		d := gad.GadModule()
		moduleDataCache[module] = d
		return d
	}
	// The root builtins are not an importable module; expose them as a flat dict
	// (name -> object) so `# builtins module` gad:doc resolves every member.
	if module == "builtins" {
		b := gad.NewBuiltins()
		d := gad.Dict{}
		for name, bt := range b.NameSet {
			if obj := b.Objects[bt]; obj != nil {
				d[name] = obj
			}
		}
		moduleDataCache[module] = d
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
	// `gaddoc api <source dir> <output file.gad> <module>` emits the module's
	// public API as a documented Gad source file (see api.go) instead of Markdown.
	if len(os.Args) >= 2 && os.Args[1] == "api" {
		if len(os.Args) < 5 {
			fmt.Printf("usage: %s api <source dir> <output file.gad> <module>\n"+
				"single \"-\" can be used as output to write to stdout\n", os.Args[0])
			os.Exit(1)
		}
		checkerr(runAPIMode(os.Args[2], os.Args[3], os.Args[4]))
		return
	}

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
	if dir := filepath.Dir(outFile); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create output dir '%s' error: %w", dir, err)
		}
	}
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
