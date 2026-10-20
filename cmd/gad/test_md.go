// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	gad "github.com/gad-lang/gad"
	"github.com/gad-lang/gad/gadx"
	"github.com/gad-lang/gad/parser"
	cc "github.com/moisespsena-go/command-context"
)

// `gad test FILE.md` runs the code fences of a Markdown file (issue #21): each
// ```gad / ```gadt / ```gadx fence (exactly three backticks) runs on its own,
// in its dialect, with its inline checks verified (`= EXPR`, `<<< text`); one
// with `ignore` or `no-run` in its info string is skipped, as any other
// language is. A fence is a test: it passes when it runs and its checks hold.

// mdFence is one runnable code fence of a Markdown file.
type mdFence struct {
	lang string // gad, gadt or gadx
	code string
	line int // the line of its opening fence (1-based)
}

// markdownFences are the runnable gad/gadt/gadx fences of md.
func markdownFences(md string) []mdFence {
	var (
		out  []mdFence
		cur  *mdFence
		body []string
	)
	for i, ln := range strings.Split(md, "\n") {
		t := strings.TrimSpace(ln)
		if cur != nil {
			if t == "```" {
				cur.code = strings.Join(body, "\n")
				out = append(out, *cur)
				cur = nil
				continue
			}
			body = append(body, ln)
			continue
		}
		info, ok := strings.CutPrefix(t, "```")
		if !ok || strings.HasPrefix(info, "`") {
			continue
		}
		fields := strings.Fields(info)
		if len(fields) == 0 {
			continue
		}
		lang := strings.ToLower(fields[0])
		if lang != "gad" && lang != "gadt" && lang != "gadx" {
			// another language's fence: skip to its end
			continue
		}
		skip := false
		for _, f := range fields[1:] {
			if f == "ignore" || f == "no-run" {
				skip = true
			}
		}
		if skip {
			continue
		}
		cur, body = &mdFence{lang: lang, line: i + 1}, nil
	}
	return out
}

// runMarkdown runs the fences of the Markdown file path, each a test.
func (o *testOptions) runMarkdown(ctx *cc.CommandContext, path string) (pass, fail, skip int) {
	src, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(ctx.Err, "FAIL %s: %s\n", path, err)
		return 0, 1, 0
	}
	for _, f := range markdownFences(string(src)) {
		name := fmt.Sprintf("%s:%d", path, f.line)
		if o.runRe != nil && !o.runRe.MatchString(name) {
			continue
		}
		if err := o.runFence(filepath.Dir(path), f); err != nil {
			fail++
			fmt.Fprintf(ctx.Err, "FAIL %s (%s): %s\n", name, f.lang, err)
			continue
		}
		pass++
		if o.verbose {
			fmt.Fprintf(ctx.Out, "PASS %s (%s)\n", name, f.lang)
		}
	}
	return
}

// runFence runs one fence in its dialect, its checks on, relative imports and
// embeds resolved against dir (the Markdown file's).
func (o *testOptions) runFence(dir string, f mdFence) error {
	base := gad.NewBuiltins()
	co := gad.CompilerOptions{
		ModuleMap:   DefaultModuleMap(dir, &sourcePath),
		EmbededdMap: DefaultEmbedMap(dir),
		Checks:      true,
	}
	opts := gad.CompileOptions{}
	switch f.lang {
	case "gadx":
		base = gadx.AppendBuiltins(base)
		co.ModuleFile = filepath.Join(dir, "fence.gadx")
	case "gadt":
		opts.ParserOptions.Mode |= parser.ParseMixed
		opts.ScannerOptions = parser.ScannerOptions{Mode: parser.ScanMixed | parser.ScanConfigDisabled}
	}
	opts.CompilerOptions = co
	builtins := base.Build()
	eval := gad.NewEval(builtins, defaultSymbolTable(builtins.Builtins().NameSet), opts,
		&gad.RunOpts{StdOut: io.Discard, StdErr: io.Discard})
	eval.VM.Builtins = builtins
	runCtx := context.Background()
	if o.timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(runCtx, o.timeout)
		defer cancel()
	}
	_, _, err := eval.RunScript(runCtx, []byte(f.code))
	return err
}
