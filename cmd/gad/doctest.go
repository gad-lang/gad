// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	gad "github.com/gad-lang/gad"
	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/source"
	cc "github.com/moisespsena-go/command-context"
)

// doctestCommand is the `gad doctest [PATH...]` subcommand: it runs the ```gad
// examples found in doc comments and reports their pass/fail status.
func doctestCommand() *cc.Command {
	return &cc.Command{
		Name:  "doctest",
		Usage: "[PATH...]",
		Description: "Run the ```gad examples embedded in doc comments and report their status.\n" +
			"\nPATH may be a file or a directory; write DIR/... to recurse. Exits non-zero\n" +
			"if any example fails.",
		Run: func(ctx *cc.CommandContext) error {
			if len(ctx.Args) == 0 {
				return fmt.Errorf("no input: provide a PATH argument")
			}
			return runDoctest(ctx)
		},
	}
}

// runDoctest walks the args, runs every example and reports the results.
func runDoctest(ctx *cc.CommandContext) error {
	var passed, failed int
	for _, arg := range ctx.Args {
		files, err := gadFiles(arg)
		if err != nil {
			return err
		}
		for _, path := range files {
			src, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for _, r := range checkFileExamples(path, src) {
				loc := fmt.Sprintf("%s:%d", path, r.line)
				if r.err != nil {
					failed++
					fmt.Fprintf(ctx.Err, "FAIL %s: %s\n", loc, r.err)
				} else {
					passed++
					fmt.Fprintf(ctx.Out, "ok   %s\n", loc)
				}
			}
		}
	}
	fmt.Fprintf(ctx.Out, "doctest: %d passed, %d failed\n", passed, failed)
	if failed > 0 {
		return fmt.Errorf("doctest: %d example(s) failed", failed)
	}
	return nil
}

// exampleResult is the outcome of running one example.
type exampleResult struct {
	line int
	err  error // nil on success
}

// checkFileExamples extracts and runs every example in src, returning a result
// per example. A parse error yields a single failing result.
func checkFileExamples(path string, src []byte) []exampleResult {
	fs := source.NewFileSet()
	f := fs.AddFileData(path, -1, src)
	file, err := parser.NewParserWithOptions(
		f, &parser.ParserOptions{Mode: parser.ParseComments}, nil).ParseFile()
	if err != nil {
		return []exampleResult{{line: 1, err: err}}
	}
	var out []exampleResult
	for _, ex := range extractExamples(f, file) {
		out = append(out, exampleResult{line: ex.line, err: runExample(ex.code)})
	}
	return out
}

// gadFiles resolves a path arg (file, directory or DIR/...) to .gad files.
func gadFiles(arg string) ([]string, error) {
	recursive, path := splitRecursive(arg)
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{path}, nil
	}
	return scanDir(path, recursive, &fileFilter{})
}

// docExample is a ```gad code block found inside a doc comment.
type docExample struct {
	code string
	line int // 1-based source line of the doc comment
}

// extractExamples returns the ```gad code blocks contained in the file's doc
// comments (any BLOCK/ROOT_BLOCK form), in source order.
func extractExamples(srcFile *source.File, file *parser.File) []docExample {
	var out []docExample
	for _, g := range file.Comments {
		content := docContent(g)
		if content == "" {
			continue
		}
		line := source.MustFileLine(srcFile, g.Pos())
		for _, code := range fencedGadBlocks(content) {
			out = append(out, docExample{code: code, line: line})
		}
	}
	return out
}

// fencedGadBlocks returns the bodies of ```gad … ``` fenced code blocks in md.
func fencedGadBlocks(md string) []string {
	var (
		blocks  []string
		cur     []string
		inBlock bool
	)
	for _, ln := range strings.Split(md, "\n") {
		t := strings.TrimSpace(ln)
		switch {
		case !inBlock && (t == "```gad" || t == "```Gad"):
			inBlock, cur = true, nil
		case inBlock && t == "```":
			inBlock = false
			blocks = append(blocks, strings.Join(cur, "\n"))
		case inBlock:
			cur = append(cur, ln)
		}
	}
	return blocks
}

// runExample runs an example's code. A line beginning with `>>> ` asserts that
// the value of the program up to that point equals the value of the expression
// after `>>>` (doctest style); other lines are ordinary code. It returns nil on
// success or a descriptive error on a run failure or a doctest mismatch.
func runExample(code string) error {
	var prog []string
	for _, ln := range strings.Split(code, "\n") {
		t := strings.TrimSpace(ln)
		if t == ">>>" || strings.HasPrefix(t, ">>> ") {
			want := stripLineComment(strings.TrimSpace(strings.TrimPrefix(t, ">>>")))
			if want == "" {
				continue
			}
			got, err := evalGadExample(strings.Join(prog, "\n"))
			if err != nil {
				return fmt.Errorf("running example: %w", err)
			}
			exp, err := evalGadExample("return " + want)
			if err != nil {
				return fmt.Errorf("evaluating expected %q: %w", want, err)
			}
			if !objectsEqual(got, exp) {
				return fmt.Errorf("doctest mismatch: got %s, want %s",
					objectStr(got), objectStr(exp))
			}
			continue
		}
		prog = append(prog, ln)
	}
	if _, err := evalGadExample(strings.Join(prog, "\n")); err != nil {
		return fmt.Errorf("running example: %w", err)
	}
	return nil
}

// evalGadExample compiles and runs src with the default builtins and module map,
// discarding its standard output, and returns the last value on the stack.
func evalGadExample(src string) (gad.Object, error) {
	builtins := gad.NewBuiltins().Build()
	opts := gad.CompileOptions{CompilerOptions: gad.CompilerOptions{
		ModuleMap: DefaultModuleMap(".", &sourcePath),
	}}
	eval := gad.NewEval(builtins, defaultSymbolTable(builtins.Builtins().NameSet), opts,
		&gad.RunOpts{StdOut: io.Discard, StdErr: io.Discard})
	eval.VM.Builtins = builtins
	ret, _, err := eval.RunScript(context.Background(), []byte(src))
	return ret, err
}

// objectsEqual reports whether two result objects are equal, treating nil as a
// distinct value.
func objectsEqual(a, b gad.Object) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(b)
}

func objectStr(o gad.Object) string {
	if o == nil {
		return "nil"
	}
	return o.ToString()
}

// stripLineComment removes a trailing `// …` comment from a doctest expected
// value, respecting string literals.
func stripLineComment(s string) string {
	inStr := byte(0)
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case inStr != 0:
			if c == inStr {
				inStr = 0
			}
		case c == '"' || c == '\'' || c == '`':
			inStr = c
		case c == '/' && i+1 < len(s) && s[i+1] == '/':
			return strings.TrimSpace(s[:i])
		}
	}
	return strings.TrimSpace(s)
}
