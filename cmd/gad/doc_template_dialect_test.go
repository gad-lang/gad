// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gad-lang/gad/web/gadbridge"
	"github.com/stretchr/testify/require"
)

// TestRenderDocTemplateDialects verifies a doc template may be written in any of
// the three dialects: .gad and .gadt write Markdown to STDOUT, .gadx renders a
// tag tree (here via a ~~ … ~~ script that also writes to STDOUT). All three
// consume `param (doc dict)`.
func TestRenderDocTemplateDialects(t *testing.T) {
	doc, err := gadbridge.ExtractDoc(docSampleSrc, "gad")
	require.NoError(t, err)
	dict := mustDocDict(t, doc, "greetings.gad", docSampleSrc)

	cases := []struct{ path, src string }{
		{"t.gad", "param (doc dict)\nwrite(\"# \" + doc.name + \"\\n\")\nfor sec in doc.sections { write(\"## \" + sec.title + \"\\n\") }\n"},
		{"t.gadt", "{% param (doc dict) -%}\n# {%= doc.name %}\n{% for sec in doc.sections %}## {%= sec.title %}\n{% end %}"},
		{"t.gadx", "~~\nparam (doc dict)\nwrite(\"# \" + doc.name + \"\\n\")\nfor sec in doc.sections { write(\"## \" + sec.title + \"\\n\") }\n~~"},
	}
	for _, c := range cases {
		t.Run(c.path, func(t *testing.T) {
			out, err := renderDocTemplate([]byte(c.src), c.path, dict)
			require.NoError(t, err)
			require.Contains(t, out, "# greetings")
			require.Contains(t, out, "## Exports")
		})
	}
}

// TestDocTemplatesInSyncWithEmbedded guards the repo-root ./doc-templates copies
// against drifting from the embedded defaults baked into the binary.
func TestDocTemplatesInSyncWithEmbedded(t *testing.T) {
	root := filepath.Join("..", "..", "doc-templates")
	for name, want := range map[string][]byte{
		"md.gadx":         defaultDocTemplateMD,
		"html.gadx":       defaultDocTemplateHTML,
		"md-index.gadx":   defaultDocIndexMD,
		"html-index.gadx": defaultDocIndexHTML,
	} {
		got, err := os.ReadFile(filepath.Join(root, name))
		require.NoError(t, err, name)
		require.Equal(t, strings.TrimRight(string(want), "\n"), strings.TrimRight(string(got), "\n"),
			"doc-templates/%s drifted from the embedded default (cmd/gad/doctemplates/%s)", name, name)
	}
}
