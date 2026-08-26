// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"io"
	"os"

	cc "github.com/moisespsena-go/command-context"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	ghtml "github.com/yuin/goldmark/renderer/html"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/web/gadbridge"
)

// renderStdin documents a single source read from stdin and streams the result
// to stdout, writing no files. The dialect and module name come from --name
// (default "stdin.gad"). With --html the rendered Markdown is converted to an
// HTML fragment (goldmark, GFM). Embedded examples are never run here — a live
// preview must stay fast and side-effect free.
func (o *docOptions) renderStdin(ctx *cc.CommandContext) error {
	src, err := io.ReadAll(os.Stdin)
	if err != nil {
		return err
	}
	name := o.stdinName
	if name == "" {
		name = "stdin.gad"
	}
	gen := &DocGenerator{MustExported: o.mustExported, NoTest: true}
	md, err := gen.FromContent(name, src)
	if err != nil {
		return err
	}
	if !o.html {
		fmt.Fprint(ctx.Out, md)
		return nil
	}
	// Render through the HTML doc template (workspace html.gadx, --doc-template-html
	// PATH, or the embedded default), so a live preview matches the generated docs
	// and can be a full page (sidebar/search/theme, --full-page). Fall back to the
	// bare goldmark fragment when no HTML template is resolved.
	if tset := o.resolveDocTemplates(); tset != nil && len(tset.htmlSrc) > 0 {
		if html, herr := o.renderStdinTemplate(name, src, tset); herr == nil {
			fmt.Fprint(ctx.Out, html)
			return nil
		}
		// A template failure should not blank the panel: fall through to goldmark.
	}
	html, err := docMarkdownToHTML(md)
	if err != nil {
		return err
	}
	fmt.Fprint(ctx.Out, html)
	return nil
}

// renderStdinTemplate builds the doc dict for a stdin source and renders it
// through the resolved HTML template, passing the `fullPage` named arg.
func (o *docOptions) renderStdinTemplate(name string, src []byte, tset *docTemplateSet) (string, error) {
	st := sourceTypeFor(name)
	doc, err := gadbridge.ExtractDoc(string(src), st)
	if err != nil {
		return "", err
	}
	dict, err := buildDocDict(doc, name, src, st, false)
	if err != nil {
		return "", err
	}
	na := gad.NewNamedArgs(gad.KeyValueArray{
		&gad.KeyValue{K: gad.Str("fullPage"), V: gad.Bool(o.fullPage)},
	})
	return renderDocTemplate(tset.htmlSrc, tset.htmlPath, dict, na)
}

// docMarkdownToHTML converts documentation Markdown to an HTML fragment with the
// GFM extension set (tables, strikethrough, task lists, autolinks). Raw HTML in
// the Markdown is preserved. The caller supplies any surrounding page/CSS.
func docMarkdownToHTML(md string) (string, error) {
	gm := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(ghtml.WithUnsafe()),
	)
	var buf bytes.Buffer
	if err := gm.Convert([]byte(md), &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}
