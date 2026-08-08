// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/gadconfig"
	"github.com/gad-lang/gad/gadx"
	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/web/gadbridge"
)

// The default documentation templates baked into the binary. They render the
// official documentation layout (title, prose, documented symbols and the full
// runnable example) and are used whenever a workspace provides no template of
// its own — so `gad doc` produces template-driven output out of the box. The
// repo-root ./doc-templates/{md,html}.gadx are byte-for-byte copies (guarded by
// TestDocTemplatesInSyncWithEmbedded).
//
//go:embed doctemplates/md.gadx
var defaultDocTemplateMD []byte

//go:embed doctemplates/html.gadx
var defaultDocTemplateHTML []byte

//go:embed doctemplates/md-index.gadx
var defaultDocIndexMD []byte

//go:embed doctemplates/html-index.gadx
var defaultDocIndexHTML []byte

// docTemplateSet holds the (lazily-read) documentation templates found in the
// workspace. A nil entry means the corresponding template is absent. By default
// the templates live under WORK_DIR/.gad/doc-templates (html.gadx, md.gadx);
// --doc-template-md / --doc-template-html override those paths and may point at a
// .gad, .gadt or .gadx file. When present a template overrides / augments the
// built-in Markdown rendering: it receives the extracted documentation as
// `param (doc dict)` and renders the output body itself (see DocData.GadDict for
// the dict shape). The dialect is chosen from the template extension by
// renderDocTemplate.
type docTemplateSet struct {
	htmlSrc, mdSrc   []byte
	htmlPath, mdPath string
	// Per-directory index templates (README.md / index.html). indexMdSrc is
	// always set (workspace file or embedded default); indexHtmlSrc only when
	// HTML output is enabled.
	indexMdSrc, indexHtmlSrc   []byte
	indexMdPath, indexHtmlPath string
}

// any reports whether at least one documentation template is present.
func (s *docTemplateSet) any() bool { return s.htmlSrc != nil || s.mdSrc != nil }

// resolveDocTemplates reads the documentation templates once. Per template the
// precedence is: the --doc-template-md/--doc-template-html path, then the
// workspace config template (.gad/doc-templates/{md,html}.gadx), then the
// embedded default (defaultDocTemplate*). So `gad doc` renders template-driven
// Markdown out of the box; --no-template opts out entirely (leaving both nil so
// the built-in Go Markdown renderer is used).
func (o *docOptions) resolveDocTemplates() *docTemplateSet {
	if o.templates != nil {
		return o.templates
	}
	s := &docTemplateSet{}
	if o.noTemplate {
		o.templates = s
		return s
	}

	s.mdPath = gadconfig.DocMDTemplate(o.workspace)
	s.htmlPath = gadconfig.DocHTMLTemplate(o.workspace)
	if o.docTemplateMD != "" {
		s.mdPath = o.absFrom(o.workspace, o.docTemplateMD)
	}
	if o.docTemplateHTML != "" {
		s.htmlPath = o.absFrom(o.workspace, o.docTemplateHTML)
	}

	// Markdown: flag/workspace path, else the embedded default.
	if b, err := os.ReadFile(s.mdPath); err == nil {
		s.mdSrc = b
	} else if o.docTemplateMD == "" {
		s.mdSrc = defaultDocTemplateMD
		s.mdPath = "doctemplates/md.gadx" // synthetic; drives the .gadx dialect
	}
	// HTML output is opt-in: a workspace html.gadx, --doc-template-html PATH, or
	// --html enables the extra .html file next to each .md. The template is the
	// flag/workspace file when present, otherwise (under --html) the embedded
	// default. A plain `gad doc` still emits only Markdown.
	if b, err := os.ReadFile(s.htmlPath); err == nil {
		s.htmlSrc = b
	} else if o.html && o.docTemplateHTML == "" {
		s.htmlSrc = defaultDocTemplateHTML
		s.htmlPath = "doctemplates/html.gadx" // synthetic; drives the .gadx dialect
	}

	// Per-directory index templates: workspace override else embedded default.
	// The Markdown index is always available; the HTML index rides along with
	// the HTML output.
	s.indexMdPath = gadconfig.DocMDIndexTemplate(o.workspace)
	if b, err := os.ReadFile(s.indexMdPath); err == nil {
		s.indexMdSrc = b
	} else {
		s.indexMdSrc = defaultDocIndexMD
		s.indexMdPath = "doctemplates/md-index.gadx"
	}
	if s.htmlSrc != nil {
		s.indexHtmlPath = gadconfig.DocHTMLIndexTemplate(o.workspace)
		if b, err := os.ReadFile(s.indexHtmlPath); err == nil {
			s.indexHtmlSrc = b
		} else {
			s.indexHtmlSrc = defaultDocIndexHTML
			s.indexHtmlPath = "doctemplates/html-index.gadx"
		}
	}

	o.templates = s
	return s
}

// sourceTypeFor maps a file extension to a gadbridge doc dialect.
func sourceTypeFor(path string) string {
	switch filepath.Ext(path) {
	case ".gadx":
		return "gadx"
	case ".gadt":
		return "gadTemplate"
	default:
		return "gad"
	}
}

// fenceLangFor maps a doc source type to the Markdown/Prism fence language.
func fenceLangFor(sourceType string) string {
	switch sourceType {
	case "gadx":
		return "gadx"
	case "gadTemplate", "template":
		return "gadt"
	default:
		return "gad"
	}
}

// buildDocDict builds the `param (doc dict)` value a documentation template
// consumes. Besides the extracted prose/sections it exposes the module name, the
// source file name, its fence language and the full (marker-stripped) source, and
// it expands `@snippet NAME` placeholders — in the prose and in each symbol's doc
// — into fenced code blocks taken from the source's `//snippet NAME … //endsnippet`
// regions (see extractSnippets), so documented code is never copied by hand. When
// run is true a snippet's declared result (`/**= … **/` / `/**< … **/`) is
// executed and verified; a mismatch or run error aborts generation.
func buildDocDict(doc *gadbridge.DocData, path string, src []byte, sourceType string, run bool) (gad.Dict, error) {
	lang := fenceLangFor(sourceType)
	if err := expandDocSnippets(doc, src, lang, run); err != nil {
		return nil, err
	}

	d := doc.GadDict()
	d["name"] = gad.Str(moduleName(path))
	d["file"] = gad.Str(filepath.Base(path))
	d["lang"] = gad.Str(lang)
	// The file's snippets — with their `uses` references and verified
	// result/output — are exposed so a template can render them itself.
	infos, err := collectSnippets(src, run)
	if err != nil {
		return nil, err
	}
	d["snippets"] = snippetsGadArray(infos)
	example := exampleSource(src)
	d["source"] = gad.Str(example)
	// The Markdown fence for the example must be wider than any backticks inside
	// the source (e.g. a doctest ``` fence in a doc comment).
	d["fence"] = gad.Str(fenceFor(example))
	// A template emits its own `# name` title only when the prose does not
	// already start with a Markdown heading (migrated samples lead with `# Title`).
	d["proseHasTitle"] = gad.Bool(strings.HasPrefix(strings.TrimSpace(doc.Prose), "#"))
	return d, nil
}

// expandDocSnippets expands the `@snippet NAME` placeholders in a DocData's prose
// and in each symbol's doc, in place, from the source's `//snippet` regions (see
// buildDocDict). When run is true a snippet's declared result is executed and
// verified.
func expandDocSnippets(doc *gadbridge.DocData, src []byte, lang string, run bool) error {
	snips := extractSnippets(src)
	var err error
	if doc.Prose, err = expandSnippets(doc.Prose, snips, lang, run); err != nil {
		return err
	}
	for si := range doc.Sections {
		for yi := range doc.Sections[si].Symbols {
			if doc.Sections[si].Symbols[yi].Doc, err = expandSnippets(doc.Sections[si].Symbols[yi].Doc, snips, lang, run); err != nil {
				return err
			}
		}
	}
	return nil
}

// docOutput is one rendered documentation file: its destination path and body.
type docOutput struct {
	path string
	body string
}

// renderTemplateOutputs builds the structured documentation for a source file and
// renders it through whichever workspace templates are present. doc-templates/
// md.gadx replaces the built-in Markdown (same .md destination as res.OutPath);
// doc-templates/html.gadx is written next to it as a .html file. When only one
// template exists, only that output is produced.
func (o *docOptions) renderTemplateOutputs(path string, src []byte, res *FileDocResult, tset *docTemplateSet) ([]docOutput, error) {
	st := sourceTypeFor(path)
	doc, err := gadbridge.ExtractDoc(string(src), st)
	if err != nil {
		return nil, fmt.Errorf("extract docs from %s: %w", filepath.Base(path), err)
	}

	docDict, err := buildDocDict(doc, path, src, st, !o.noDoctest)
	if err != nil {
		return nil, err
	}

	var outputs []docOutput
	if tset.mdSrc != nil {
		body, err := renderDocTemplate(tset.mdSrc, tset.mdPath, docDict)
		if err != nil {
			return nil, err
		}
		outputs = append(outputs, docOutput{res.OutPath, body})
	}
	if tset.htmlSrc != nil {
		body, err := renderDocTemplate(tset.htmlSrc, tset.htmlPath, docDict)
		if err != nil {
			return nil, err
		}
		htmlPath := res.OutPath[:len(res.OutPath)-len(filepath.Ext(res.OutPath))] + ".html"
		outputs = append(outputs, docOutput{htmlPath, body})
	}
	return outputs, nil
}

// renderDocTemplate compiles a documentation template and runs it with the
// documentation bound to `param (doc dict)`, returning the rendered body (HTML or
// Markdown, depending on the template). The dialect is chosen from the template
// extension: a `.gadx` template renders a tag tree (and may `return` a
// gadx.Element); a `.gadt` template is parsed in mixed/template mode; a plain
// `.gad` template is a script. All three write their body to STDOUT, which is
// captured here. No files are read or written — callers supply the template
// bytes and persist the result.
func renderDocTemplate(tmplSrc []byte, tmplPath string, docDict gad.Dict) (string, error) {
	builtins := gad.NewBuiltins()
	opts := gad.CompileOptions{}
	opts.ModuleFile = tmplPath
	switch sourceTypeFor(tmplPath) {
	case "gadx":
		// opts.ModuleFile (tmplPath) ends in .gadx, selecting the Gadx front-end.
		builtins = gadx.AppendBuiltins(builtins)
	case "gadTemplate":
		opts.ParserOptions.Mode |= parser.ParseMixed
		opts.ScannerOptions.Mode |= parser.ScanMixed | parser.ScanConfigDisabled
		opts.ScannerOptions.MixedDelimiter = parser.DefaultMixedDelimiter
	}

	st := gad.NewSymbolTable(builtins.NameSet)
	cr, err := gad.Compile(st, tmplSrc, opts)
	if err != nil {
		return "", fmt.Errorf("compile %s: %w", filepath.Base(tmplPath), err)
	}

	// `param (doc dict)` is a positional parameter, so the doc dict is passed as
	// the first positional argument.
	var out bytes.Buffer
	vm := gad.NewVM(builtins.Build(), cr.Bytecode)
	ret, err := vm.RunOpts(&gad.RunOpts{Args: gad.Args{gad.Array{docDict}}, StdOut: &out})
	if err != nil {
		return "", fmt.Errorf("render %s: %w", filepath.Base(tmplPath), err)
	}
	// A gadx `@main` template returns its rendered tree as a gadx.Element; a
	// `.gadt`/`.gad` script (or a gadx `~~ … ~~` block) writes to STDOUT and
	// returns nil.
	if el, ok := ret.(gadx.Element); ok {
		if _, err = el.WriteTo(vm, &out); err != nil {
			return "", fmt.Errorf("render %s: %w", filepath.Base(tmplPath), err)
		}
	}
	return out.String(), nil
}
