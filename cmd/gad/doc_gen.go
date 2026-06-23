// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package main

import (
	"path/filepath"
	"strings"

	"github.com/gad-lang/gad/parser"
	"github.com/gad-lang/gad/parser/source"
)

// moduleName derives the documentation module name from a source path: its base
// name without the extension.
func moduleName(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// generateDoc renders the godoc-style Markdown for a Gad source file. The file
// is parsed with comments so doc comments are attached to their nodes.
//
// This is the scaffold stub: it validates the source parses and emits only the
// module heading. The constants/types sections (ROOT_BLOCKs and exported-ident
// docs) are produced by a later slice.
func generateDoc(path string, src []byte) (string, error) {
	fs := source.NewFileSet()
	f := fs.AddFileData(path, -1, src)
	if _, err := parser.NewParserWithOptions(
		f, &parser.ParserOptions{Mode: parser.ParseComments}, nil).ParseFile(); err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString("# " + moduleName(path) + "\n")
	return b.String(), nil
}
