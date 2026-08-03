// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/gadconfig"
	"github.com/gad-lang/gad/giom"
	"github.com/gad-lang/gad/web/gadbridge"
)

// docTemplateSet holds the (lazily-read) documentation templates found in the
// workspace. A nil entry means the corresponding template is absent. The
// templates live under WORK_DIR/.gad/doc-templates (html.giom, md.giom); when
// present they override / augment the built-in Markdown rendering: each receives
// the extracted documentation as `param (doc dict)` and renders the output body
// itself (see DocData.GadDict for the dict shape).
type docTemplateSet struct {
	htmlSrc, mdSrc   []byte
	htmlPath, mdPath string
}

// any reports whether at least one documentation template is present.
func (s *docTemplateSet) any() bool { return s.htmlSrc != nil || s.mdSrc != nil }

// resolveDocTemplates reads the optional doc-templates/html.giom and md.giom
// files from the workspace config directory once. Missing templates are left nil.
func (o *docOptions) resolveDocTemplates() *docTemplateSet {
	if o.templates != nil {
		return o.templates
	}
	s := &docTemplateSet{
		htmlPath: gadconfig.DocHTMLTemplate(o.workspace),
		mdPath:   gadconfig.DocMDTemplate(o.workspace),
	}
	if b, err := os.ReadFile(s.htmlPath); err == nil {
		s.htmlSrc = b
	}
	if b, err := os.ReadFile(s.mdPath); err == nil {
		s.mdSrc = b
	}
	o.templates = s
	return s
}

// sourceTypeFor maps a file extension to a gadbridge doc dialect.
func sourceTypeFor(path string) string {
	switch filepath.Ext(path) {
	case ".giom":
		return "giom"
	case ".gadt":
		return "gadTemplate"
	default:
		return "gad"
	}
}

// docOutput is one rendered documentation file: its destination path and body.
type docOutput struct {
	path string
	body string
}

// renderTemplateOutputs builds the structured documentation for a source file and
// renders it through whichever workspace templates are present. doc-templates/
// md.giom replaces the built-in Markdown (same .md destination as res.OutPath);
// doc-templates/html.giom is written next to it as a .html file. When only one
// template exists, only that output is produced.
func (o *docOptions) renderTemplateOutputs(path string, src []byte, res *FileDocResult, tset *docTemplateSet) ([]docOutput, error) {
	doc, err := gadbridge.ExtractDoc(string(src), sourceTypeFor(path))
	if err != nil {
		return nil, fmt.Errorf("extract docs from %s: %w", filepath.Base(path), err)
	}

	var outputs []docOutput
	if tset.mdSrc != nil {
		body, err := renderDocTemplate(tset.mdSrc, tset.mdPath, doc)
		if err != nil {
			return nil, err
		}
		outputs = append(outputs, docOutput{res.OutPath, body})
	}
	if tset.htmlSrc != nil {
		body, err := renderDocTemplate(tset.htmlSrc, tset.htmlPath, doc)
		if err != nil {
			return nil, err
		}
		htmlPath := res.OutPath[:len(res.OutPath)-len(filepath.Ext(res.OutPath))] + ".html"
		outputs = append(outputs, docOutput{htmlPath, body})
	}
	return outputs, nil
}

// renderDocTemplate compiles a giom documentation template and runs it with the
// extracted documentation bound to `param (doc dict)`, returning the rendered
// body (HTML or Markdown, depending on the template). No files are read or
// written here — callers supply the template bytes and persist the result.
func renderDocTemplate(tmplSrc []byte, tmplPath string, doc *gadbridge.DocData) (string, error) {
	builtins := giom.AppendBuiltins(gad.NewBuiltins())
	st := gad.NewSymbolTable(builtins.NameSet)

	opts := gad.CompileOptions{}
	opts.GiomOptions = &gad.GiomOptions{}
	opts.ModuleFile = tmplPath
	cr, err := gad.Compile(st, tmplSrc, opts)
	if err != nil {
		return "", fmt.Errorf("compile %s: %w", filepath.Base(tmplPath), err)
	}

	// `param (doc dict)` is a positional parameter, so the doc dict is passed as
	// the first positional argument.
	var out bytes.Buffer
	vm := gad.NewVM(builtins.Build(), cr.Bytecode)
	ret, err := vm.RunOpts(&gad.RunOpts{Args: gad.Args{gad.Array{doc.GadDict()}}, StdOut: &out})
	if err != nil {
		return "", fmt.Errorf("render %s: %w", filepath.Base(tmplPath), err)
	}
	// A giom `@main` template returns its rendered tree as a giom.Element; a
	// plain `~~ … ~~` script writes to StdOut directly and returns nil.
	if el, ok := ret.(giom.Element); ok {
		if _, err = el.WriteTo(vm, &out); err != nil {
			return "", fmt.Errorf("render %s: %w", filepath.Base(tmplPath), err)
		}
	}
	return out.String(), nil
}
