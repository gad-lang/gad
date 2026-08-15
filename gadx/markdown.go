package gadx

import (
	"bytes"

	gadxnode "github.com/gad-lang/gad/gadx/node"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// Install the transpile-time Markdown renderer hook used by gadx/node to
// pre-render fully static `@md` blocks (see node.MarkdownRenderer). It reads the
// package-level Markdown each call, so an embedder that replaces Markdown before
// transpiling still gets its renderer.
func init() {
	gadxnode.MarkdownRenderer = func(src []byte) ([]byte, error) {
		return renderMarkdown(Markdown, src)
	}
}

// Markdown is the goldmark instance used by the `@md` block to render Markdown
// content to HTML. It enables the full extension set (GFM — tables,
// strikethrough, linkify, task lists —, Typographer, definition lists and
// footnotes) plus auto heading ids and raw-HTML passthrough, so inline HTML
// produced by nested `@` directives inside a `@md` block survives conversion.
//
// It is a package-level variable so embedders can customize or replace the
// renderer (for example to add their own extensions or tighten the HTML
// sanitizer); assign a new goldmark.Markdown before rendering templates.
var Markdown = NewMarkdown()

// NewMarkdown builds the default `@md` goldmark renderer with every bundled
// extension enabled. Use it as a base when customizing Markdown, for example:
//
//	gadx.Markdown = gadx.NewMarkdown() // then wrap/extend as needed
func NewMarkdown() goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Typographer,
			extension.DefinitionList,
			extension.Footnote,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			// Nested `@` directives inside `@md` render to HTML; keep it in the
			// output instead of stripping it.
			html.WithUnsafe(),
		),
	)
}

// renderMarkdown converts Markdown source to an HTML fragment using md when
// provided, else the package-level Markdown renderer.
func renderMarkdown(md goldmark.Markdown, src []byte) ([]byte, error) {
	if md == nil {
		md = Markdown
	}
	var buf bytes.Buffer
	if err := md.Convert(src, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
