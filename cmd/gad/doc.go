// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"path/filepath"
	"strings"
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

// FromContent renders the documentation Markdown for source content. path is used
// only for the module name and example diagnostics; no file is read or written.
func (d *DocGenerator) FromContent(path string, src []byte) (string, error) {
	return generateDoc(path, src, d.MustExported)
}

// FromFile renders the documentation Markdown for an already-read source file and
// computes its output path, without touching the filesystem. data is the source
// bytes; path is the source path; dst is the output root and base is the tree
// root the source path is mirrored against. Unless NoTest is set, the embedded
// examples are run and each failure is reported via OnError; the number of failed
// examples is returned.
func (d *DocGenerator) FromFile(
	data []byte, path, dst, base string,
) (md, outPath string, examplesFailed int, err error) {
	md, err = d.FromContent(path, data)
	if err != nil {
		return "", "", 0, fmt.Errorf("%s: %w", path, err)
	}
	if !d.NoTest {
		for _, r := range checkFileExamples(path, data) {
			if r.err != nil {
				examplesFailed++
				if d.OnError != nil {
					d.OnError(fmt.Sprintf("doc: %s:%d: example failed: %s", path, r.line, r.err))
				}
			}
		}
	}
	return md, docOutPath(dst, base, path), examplesFailed, nil
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
