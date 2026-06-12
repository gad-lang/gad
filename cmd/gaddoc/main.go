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
// go run ./cmd/gaddoc ./stdlib/time ./docs/stdlib-time.md
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
	reModuleHeader = regexp.MustCompile(`^\s*#\s+(\w+)\s+module`)
	reTypeHeader   = regexp.MustCompile(`^\s*##\s+Types`)
	reConstHeader  = regexp.MustCompile(`^\s*##\s+Constants`)
	reFuncHeader   = regexp.MustCompile(`^\s*##\s+Functions`)
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
	var lines []string
	for _, p := range comments {
		lines = append(lines, strings.Split(p, "\n")...)
	}
	for i, p := range lines {
		if reModuleHeader.MatchString(p) {
			parts := reModuleHeader.FindStringSubmatch(p)
			if len(parts) > 1 {
				dg.module = parts[len(parts)-1]
				dg.docs = append(dg.docs,
					fmt.Sprintf("# `%s` module", dg.module))
			} else {
				dg.addError("module header is invalid")
			}
			dg.processBlocks(lines[i+1:])
			return
		}
		dg.docs = append(dg.docs, p)
	}
}

func (dg *docgroup) processBlocks(lines []string) {
	const (
		unknown = iota
		typeBlock
		constBlock
		funcBlock
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
			} else {
				dg.docs = append(dg.docs, line)
			}
		case typeBlock,
			constBlock,
			funcBlock:
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

	var out []string
	out = append(out, d.docs...)
	if len(d.types) > 1 {
		out = append(out, d.types...)
	}
	if len(d.consts) > 1 {
		out = append(out, d.consts...)
	}
	if len(d.funcs) > 1 {
		out = append(out, d.funcs...)
	}
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
