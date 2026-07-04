// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/source"
)

// DocGenerator renders Gad source documentation to Markdown. It is the
// filesystem-free core behind the `doc` subcommand: FromContent and FromFile
// neither read nor write files — callers supply the source bytes and persist the
// returned Markdown themselves (see (*docOptions).processFile).
type DocGenerator struct {
	// OnError, when non-nil, receives a diagnostic message for each embedded
	// example that fails during FromFile.
	OnError func(message string)
	// MustExported documents only exported symbols (omitting the Internal
	// section).
	MustExported bool
	// NoTest skips running the ```gad examples embedded in doc comments.
	NoTest bool
}

// FromContent renders the godoc-style Markdown for source content: the module
// heading, any ROOT_BLOCK (`/***`) prose, then the documented symbols. The source
// is parsed with comments so doc comments are attached to their nodes. path is
// used only for the module name; no file is read or written.
//
// When MustExported is true only the exported symbols are documented, in
// top-level Constants and Types sections. When it is false the documented
// internal (non-exported) declarations are included too, and the output is split
// into two root sections, "Exported" and "Internal", each with its own Constants
// and Types subsections.
func (d *DocGenerator) FromContent(path string, src []byte) (string, error) {
	fs := source.NewFileSet()
	f := fs.AddFileData(path, -1, src)
	file, err := parser.NewParserWithOptions(
		f, &parser.ParserOptions{Mode: parser.ParseComments}, nil).ParseFile()
	if err != nil {
		return "", err
	}

	var exported []docEntry
	for _, stmt := range file.Stmts {
		if es, ok := stmt.(*node.ExportStmt); ok {
			exported = append(exported, exportEntries(es)...)
		}
	}
	exp := bucketize(exported)

	var b strings.Builder
	b.WriteString("# " + moduleName(path) + "\n")

	for _, root := range rootBlocks(file.Comments) {
		b.WriteString("\n" + root + "\n")
	}

	// `test`/`bench` statements are documented in their own sections, in both
	// layouts, regardless of the exported/internal split.
	tests, benches := testEntries(file)

	if d.MustExported {
		writeTOC(&b, exp, tests, benches)
		writeSection(&b, 2, "Constants", exp.consts)
		writeSection(&b, 2, "Variables", exp.vars)
		writeTypesSection(&b, 2, exp.types)
		writeSection(&b, 2, "Tests", tests)
		writeSection(&b, 2, "Benchs", benches)
		return b.String(), nil
	}

	// Two-root-section mode: gather the documented internal declarations and
	// render Exported + Internal groups.
	internal := bucketize(internalEntries(file, f))
	writeGroupedTOC(&b, exp, internal, tests, benches)
	writeRootGroup(&b, "Exported", exp)
	writeRootGroup(&b, "Internal", internal)
	writeSection(&b, 2, "Tests", tests)
	writeSection(&b, 2, "Benchs", benches)
	return b.String(), nil
}

// FileDocResult is the outcome of rendering one source file's documentation.
type FileDocResult struct {
	// Markdown is the rendered documentation.
	Markdown string
	// OutPath is the .md output path: the source path mirrored under dst.
	OutPath string
	// ExamplesFailed counts the embedded examples that failed (always 0 when
	// NoTest is set).
	ExamplesFailed int
}

// FromFile renders the documentation Markdown for an already-read source file and
// computes its output path, without touching the filesystem. data is the source
// bytes; path is the source path; dst is the output root and base is the tree
// root the source path is mirrored against. Unless NoTest is set, the embedded
// examples are run and each failure is reported via OnError; the failure count is
// returned in the result.
func (d *DocGenerator) FromFile(data []byte, path, dst, base string) (*FileDocResult, error) {
	md, err := d.FromContent(path, data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	res := &FileDocResult{Markdown: md, OutPath: docOutPath(dst, base, path)}
	if !d.NoTest {
		for _, r := range checkFileExamples(path, data) {
			if r.err != nil {
				res.ExamplesFailed++
				if d.OnError != nil {
					d.OnError(fmt.Sprintf("doc: %s:%d: example failed: %s", path, r.line, r.err))
				}
			}
		}
	}
	return res, nil
}

// docOutPath returns the .md output path for the source path mirrored under dst
// relative to base. base and path may not share the same abs/rel form (e.g. base
// is the config-derived absolute workspace while path is cwd-relative from a
// recursive "." scan), so both are normalized to absolute before computing the
// relative path; otherwise filepath.Rel fails and the tree is flattened to base
// names.
func docOutPath(dst, base, path string) string {
	absBase, baseErr := filepath.Abs(base)
	absPath, pathErr := filepath.Abs(path)
	rel := filepath.Base(path)
	if baseErr == nil && pathErr == nil {
		if r, err := filepath.Rel(absBase, absPath); err == nil && !strings.HasPrefix(r, "..") {
			rel = r
		}
	}
	return filepath.Join(dst, strings.TrimSuffix(rel, filepath.Ext(rel))+".md")
}
