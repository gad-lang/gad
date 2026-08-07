// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/gad-lang/gad/web/gadbridge"
	"gopkg.in/yaml.v3"
)

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
	st := sourceTypeFor(path)
	doc, err := gadbridge.ExtractDoc(string(src), st)
	if err != nil {
		return nil, fmt.Errorf("extract docs from %s: %w", filepath.Base(path), err)
	}
	lang := fenceLangFor(st)
	if err = expandDocSnippets(doc, src, lang, !o.noDoctest); err != nil {
		return nil, err
	}

	data := &docEncoded{
		Name:     moduleName(path),
		File:     filepath.Base(path),
		Lang:     lang,
		Prose:    doc.Prose,
		Sections: doc.Sections,
		Source:   exampleSource(src),
	}

	base := res.OutPath[:len(res.OutPath)-len(filepath.Ext(res.OutPath))]
	var outputs []docOutput
	if o.json {
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false) // keep `<`, `>`, `&` literal in the doc prose
		enc.SetIndent("", "  ")
		if err := enc.Encode(data); err != nil {
			return nil, fmt.Errorf("encode JSON for %s: %w", filepath.Base(path), err)
		}
		outputs = append(outputs, docOutput{base + ".json", buf.String()})
	}
	if o.yaml {
		b, err := yaml.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("encode YAML for %s: %w", filepath.Base(path), err)
		}
		outputs = append(outputs, docOutput{base + ".yaml", string(b)})
	}
	return outputs, nil
}
