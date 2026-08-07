// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gad-lang/gad/web/gadbridge"
	cc "github.com/moisespsena-go/command-context"
	"gopkg.in/yaml.v3"
)

// encodeDoc marshals v to JSON (with `<`/`>`/`&` kept literal) or YAML.
func encodeDoc(v any, format string) ([]byte, error) {
	if format == "yaml" {
		return yaml.Marshal(v)
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// docEncoded is the structured documentation of one source file, serialized to
// JSON or YAML by `gad doc --json` / `--yaml`. It is the same shape that feeds
// the templates (`param (doc dict)`): the module name, source file name, fence
// language, prose, sections and the full example source, with `@snippet`
// placeholders and their verified results already expanded.
type docEncoded struct {
	Name     string                 `json:"name" yaml:"name"`
	File     string                 `json:"file" yaml:"file"`
	Lang     string                 `json:"lang" yaml:"lang"`
	Prose    string                 `json:"prose,omitempty" yaml:"prose,omitempty"`
	Sections []gadbridge.DocSection `json:"sections,omitempty" yaml:"sections,omitempty"`
	Source   string                 `json:"source,omitempty" yaml:"source,omitempty"`
}

// renderEncodedOutputs builds the structured documentation for a source file and
// encodes it as JSON and/or YAML (per o.json / o.yaml), each written next to the
// Markdown output with a .json / .yaml extension. Snippets are expanded (and, when
// doctests are enabled, verified) exactly as for the template path.
func (o *docOptions) renderEncodedOutputs(path string, src []byte, res *FileDocResult) ([]docOutput, error) {
	data, err := o.buildDocEncoded(path, src)
	if err != nil {
		return nil, err
	}
	base := res.OutPath[:len(res.OutPath)-len(filepath.Ext(res.OutPath))]
	var outputs []docOutput
	for _, format := range o.encodeFormats() {
		b, err := encodeDoc(data, format)
		if err != nil {
			return nil, fmt.Errorf("encode %s for %s: %w", format, filepath.Base(path), err)
		}
		outputs = append(outputs, docOutput{base + "." + format, string(b)})
	}
	return outputs, nil
}

// buildDocEncoded extracts a source file's documentation, expands (and verifies)
// its snippets, and returns the structure serialized by --json / --yaml.
func (o *docOptions) buildDocEncoded(path string, src []byte) (*docEncoded, error) {
	st := sourceTypeFor(path)
	doc, err := gadbridge.ExtractDoc(string(src), st)
	if err != nil {
		return nil, fmt.Errorf("extract docs from %s: %w", filepath.Base(path), err)
	}
	lang := fenceLangFor(st)
	if err = expandDocSnippets(doc, src, lang, !o.noDoctest); err != nil {
		return nil, err
	}
	return &docEncoded{
		Name:     moduleName(path),
		File:     filepath.Base(path),
		Lang:     lang,
		Prose:    doc.Prose,
		Sections: doc.Sections,
		Source:   exampleSource(src),
	}, nil
}

// docTreeNode is one directory in the whole-tree document encoded to stdout by
// `gad doc --out - --json|--yaml`: the docs directly in it and its subdirectories.
type docTreeNode struct {
	Name string                  `json:"name,omitempty" yaml:"name,omitempty"`
	Docs []*docEncoded           `json:"docs,omitempty" yaml:"docs,omitempty"`
	Dirs map[string]*docTreeNode `json:"dirs,omitempty" yaml:"dirs,omitempty"`
}

// collectEncoded builds the encoded documentation for one source file and inserts
// it into o.docTree at its path relative to base (the stdout `--out -` mode).
func (o *docOptions) collectEncoded(path, base string, src []byte) error {
	data, err := o.buildDocEncoded(path, src)
	if err != nil {
		return err
	}
	if o.docTree == nil {
		o.docTree = &docTreeNode{}
	}
	node := o.docTree
	for _, seg := range relDirSegments(base, path) {
		if node.Dirs == nil {
			node.Dirs = map[string]*docTreeNode{}
		}
		child := node.Dirs[seg]
		if child == nil {
			child = &docTreeNode{Name: seg}
			node.Dirs[seg] = child
		}
		node = child
	}
	node.Docs = append(node.Docs, data)
	return nil
}

// relDirSegments returns the directory path of path relative to base, split into
// segments ("." / an unrelated path yields none).
func relDirSegments(base, path string) []string {
	absBase, e1 := filepath.Abs(base)
	absPath, e2 := filepath.Abs(path)
	if e1 != nil || e2 != nil {
		return nil
	}
	rel, err := filepath.Rel(absBase, absPath)
	if err != nil || rel == "." || len(rel) >= 2 && rel[0] == '.' && rel[1] == '.' {
		return nil
	}
	d := filepath.Dir(rel)
	if d == "." || d == "" {
		return nil
	}
	return strings.Split(filepath.ToSlash(d), "/")
}

// encodeTreeToStdout encodes the collected document tree to ctx.Out in the format
// selected by --json / --yaml (JSON when both or neither is given).
func (o *docOptions) encodeTreeToStdout(ctx *cc.CommandContext) error {
	format := "json"
	if o.yaml && !o.json {
		format = "yaml"
	}
	tree := o.docTree
	if tree == nil {
		tree = &docTreeNode{}
	}
	b, err := encodeDoc(tree, format)
	if err != nil {
		return fmt.Errorf("encode %s tree: %w", format, err)
	}
	_, err = ctx.Out.Write(b)
	return err
}

// encodeFormats returns the encode formats enabled by --json / --yaml.
func (o *docOptions) encodeFormats() []string {
	var f []string
	if o.json {
		f = append(f, "json")
	}
	if o.yaml {
		f = append(f, "yaml")
	}
	return f
}

// indexEntry is one link in an encoded directory index.
type indexEntry struct {
	Name string `json:"name" yaml:"name"`
	File string `json:"file" yaml:"file"`
}

// indexEncoded is the encoded per-directory index: the directory name, the
// documented files in it and the subdirectory indexes — the same structure the
// md-index/html-index templates consume.
type indexEncoded struct {
	Name  string       `json:"name" yaml:"name"`
	Path  string       `json:"path" yaml:"path"`
	Items []indexEntry `json:"items" yaml:"items"`
	Dirs  []indexEntry `json:"dirs" yaml:"dirs"`
}

// writeEncodedIndex writes one directory index encoded as JSON or YAML
// (README.<format>), mirroring the template-rendered README.md / index.html.
func (o *docOptions) writeEncodedIndex(ctx *cc.CommandContext, dir string, node *indexNode, format string) error {
	indexFile := indexFileName(format)
	data := buildIndexData(dir, node, format, indexFile)
	b, err := encodeDoc(data, format)
	if err != nil {
		return fmt.Errorf("encode %s index for %s: %w", format, dir, err)
	}
	out := filepath.Join(dir, indexFile)
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err = os.WriteFile(out, b, 0o644); err != nil {
		return err
	}
	fmt.Fprintln(ctx.Out, out)
	return nil
}
