// Package prism is the PrismJS bundle with the Gad family's grammars (gad,
// gadt, gadx — @gad-lang/prism-gad) and the languages most documentation
// shows: markup (html, xml, svg), css, javascript, typescript, jsx, tsx, go,
// json, bash (sh, shell), yaml, markdown, ini, toml, sql, diff, python,
// docker and makefile.
//
// Put in a page, it highlights every `<code class="language-…">` — what a
// Markdown fence renders to —, those the page gets later too: a page drawn
// without a reload. It brings no theme: the colors are the page's, by
// Prism's token classes (`.token.keyword`, …). It is built from bundle.mjs
// by `make doc-prism`.
package prism

import _ "embed"

//go:embed prism.js
var js []byte

// JS is the bundle, a script that runs on its own.
func JS() []byte { return js }
