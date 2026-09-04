// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/web/gadbridge"
	cc "github.com/moisespsena-go/command-context"
)

func init() { registerCommand("transpile", transpileCommand) }

// transpileCommand transpiles Gad templates to plain Gad source: a `.gadt`
// mixed-mode template and a `.gadx` indentation template are each lowered to a
// `.gad` file of the same name. A directory is processed recursively.
func transpileCommand() *cc.Command {
	return &cc.Command{
		Name:  "transpile",
		Usage: "PATH [PATH...]",
		Description: "Transpile Gad templates to plain Gad source. " +
			"A .gadt or .gadx file is written as a .gad file of the same name; " +
			"an .html file is lifted into a .gadx template whose @main renders it; " +
			"a directory is transpiled recursively.",
		ParseArgs: func(ctx *cc.CommandContext) error { return ctx.Args.Min(1) },
		Run: func(ctx *cc.CommandContext) error {
			for _, arg := range ctx.Args {
				if err := transpilePath(ctx, arg); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

// isTemplateFile reports whether name is a transpilable Gad template — a `.gadt`
// mixed template or a `.gadx` indentation template.
func isTemplateFile(name string) bool {
	return strings.HasSuffix(name, ".gadt") || strings.HasSuffix(name, ".gadx")
}

// transpilePath transpiles a single file or, recursively, every template file in
// a directory.
func transpilePath(ctx *cc.CommandContext, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		if isHTMLFile(path) {
			return transpileHTMLFile(ctx, path)
		}
		if !isTemplateFile(path) {
			return fmt.Errorf("not a Gad template (.gadt/.gadx) or an HTML document: %s", path)
		}
		return transpileFile(ctx, path)
	}
	return filepath.WalkDir(path, func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			if p != path && isHidden(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if isHidden(d.Name()) {
			return nil
		}
		if isHTMLFile(d.Name()) {
			return transpileHTMLFile(ctx, p)
		}
		if !isTemplateFile(d.Name()) {
			return nil
		}
		return transpileFile(ctx, p)
	})
}

// transpileFile transpiles one template file to a `.gad` file of the same name.
func transpileFile(ctx *cc.CommandContext, path string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	dest := strings.TrimSuffix(path, filepath.Ext(path)) + ".gad"

	var content string
	if isGadxFile(path) {
		// A `.gadx` template is lowered to Gad by the Gadx front-end, then run
		// through the Gad formatter so the emitted `.gad` is canonically
		// formatted (no raw blank-line runs), just like the `.gadt` path below.
		lowered, err := gad.TranspileGadxSource(path, src)
		if err != nil {
			return fmt.Errorf("transpile %s: %w", path, err)
		}
		o := &fmtOptions{codeFlags: fmtFormatFlag()}
		content, err = o.formatSource(dest, lowered, false)
		if err != nil {
			return fmt.Errorf("transpile %s: %w", path, err)
		}
	} else {
		// A `.gadt` mixed template reuses the formatter's transpile pipeline.
		// fmtFormatFlag() makes the emitted Gad canonically formatted (the
		// transpile flag alone lowers the template but doesn't reflow the code).
		o := &fmtOptions{transpileOn: true, codeFlags: fmtFormatFlag()}
		o.finalizeTranspile()
		var err error
		content, err = o.formatSource(path, src, true)
		if err != nil {
			return fmt.Errorf("transpile %s: %w", path, err)
		}
	}
	if err := os.WriteFile(dest, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", dest, err)
	}

	fmt.Fprintf(ctx.Out, "%s -> %s\n", path, dest)
	return nil
}

// htmlGadxMaxColumns is the line budget for the templates lifted out of HTML,
// wider than the one `gad fmt` keeps for hand-written code.
//
// A page's markup carries attributes nobody typed — a utility-class list runs
// past 80 columns on its own — and the narrower budget spends that overflow by
// pushing a tag's text onto a line of its own, which is where the tag and the
// words it holds stop reading as one thing. The wider budget keeps them
// together and lets only the genuinely long attribute lists wrap.
const htmlGadxMaxColumns = 200

// htmlToGadxFormatted lifts an HTML page into Gadx and rewrites it in the tag
// syntax the language is written in — `div.card` in place of `<div class="card">`.
//
// The rewrite is the pass `gad fmt` runs, so the result is what formatting the
// lifted file would produce anyway, and its guards apply: a rewrite that would
// not settle, or that would change what the page renders, is refused and the
// inline form is kept.
func htmlToGadxFormatted(src string) string {
	opts := gadbridge.GadxFormatOptions{
		Indent:     "\t",
		MaxColumns: htmlGadxMaxColumns,
		EmbedFlags: fmtFormatFlag(),
	}
	content := htmlToGadx(src)
	res := gadbridge.FormatGadx(content, opts)
	if !res.OK {
		return content
	}
	again := gadbridge.FormatGadx(res.Source, opts)
	if !again.OK || again.Source != res.Source {
		return content
	}
	before, ok1 := gadbridge.GadxLowered(content)
	after, ok2 := gadbridge.GadxLowered(res.Source)
	if !ok1 || !ok2 || before != after {
		return content
	}
	return res.Source
}

// transpileHTMLFile lifts one HTML document into a `.gadx` template of the same
// name, whose `@main` renders the page.
func transpileHTMLFile(ctx *cc.CommandContext, path string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	dest := strings.TrimSuffix(path, filepath.Ext(path)) + ".gadx"

	content := htmlToGadxFormatted(string(src))

	if err := os.WriteFile(dest, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", dest, err)
	}

	fmt.Fprintf(ctx.Out, "%s -> %s\n", path, dest)
	return nil
}
