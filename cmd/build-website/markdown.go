package main

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// Markdown rendering for the docs site is delegated to goldmark (a CommonMark
// compliant, pure-Go library), so the full inline/block grammar — nested
// emphasis and links, GFM tables, fenced code with info strings — is handled
// correctly instead of by a hand-rolled subset. Two site-specific behaviors are
// layered on: link destinations are rewritten for the generated site
// (rewriteLink, via an AST transformer) and headings are collected for the
// page TOC/search (with the auto-generated ids goldmark also renders).

// Heading is a rendered heading, used to build the page TOC and search index.
type Heading struct {
	Level int
	Text  string
	ID    string
}

// mdParser is the shared goldmark instance. A rich extension set is enabled —
// GFM (tables, strikethrough, linkify, task lists), typographer (smart quotes/
// dashes), definition lists and footnotes — so the docs can use the full
// Markdown vocabulary. Auto heading ids give stable anchors; linkRewriter maps
// doc-relative links to their published .html targets.
var mdParser = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		extension.Typographer,
		extension.DefinitionList,
		extension.Footnote,
	),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
		parser.WithASTTransformers(util.Prioritized(linkRewriter{}, 100)),
	),
)

// renderMarkdown converts src to an HTML body and returns the headings found.
func renderMarkdown(src string) (string, []Heading) {
	source := []byte(src)
	reader := text.NewReader(source)
	doc := mdParser.Parser().Parse(reader)

	headings := collectHeadings(doc, source)

	var buf bytes.Buffer
	if err := mdParser.Renderer().Render(&buf, source, doc); err != nil {
		return "", headings
	}
	return buf.String(), headings
}

// collectHeadings walks the parsed document for heading nodes, returning their
// level, plain text and the id goldmark assigned (WithAutoHeadingID).
func collectHeadings(doc ast.Node, source []byte) []Heading {
	var headings []Heading
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		h, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}
		id := ""
		if v, found := h.AttributeString("id"); found {
			if b, ok := v.([]byte); ok {
				id = string(b)
			}
		}
		headings = append(headings, Heading{Level: h.Level, Text: nodeText(n, source), ID: id})
		return ast.WalkSkipChildren, nil
	})
	return headings
}

// nodeText extracts the concatenated plain text of a node's inline descendants
// (heading text for the TOC/search index).
func nodeText(n ast.Node, source []byte) string {
	var b strings.Builder
	_ = ast.Walk(n, func(c ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch t := c.(type) {
		case *ast.Text:
			b.Write(t.Segment.Value(source))
		case *ast.String:
			b.Write(t.Value)
		case *ast.CodeSpan:
			b.WriteString(string(t.Text(source))) //nolint:staticcheck // fine for leaf code spans
		}
		return ast.WalkContinue, nil
	})
	return b.String()
}

// linkRewriter is a goldmark AST transformer that rewrites every link and image
// destination through rewriteLink, so doc-relative `.md` targets resolve to the
// generated `.html` pages (and sample sources to their chapter pages).
type linkRewriter struct{}

func (linkRewriter) Transform(doc *ast.Document, _ text.Reader, _ parser.Context) {
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch l := n.(type) {
		case *ast.Link:
			l.Destination = []byte(rewriteLink(string(l.Destination)))
		case *ast.Image:
			l.Destination = []byte(rewriteLink(string(l.Destination)))
		}
		return ast.WalkContinue, nil
	})
}

// rewriteLink maps doc-relative .md links to .html for the generated site.
func rewriteLink(url string) string {
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") ||
		strings.HasPrefix(url, "#") {
		return url
	}
	frag := ""
	if i := strings.IndexByte(url, '#'); i >= 0 {
		frag = url[i:]
		url = url[:i]
	}
	// Any link into the samples tree — a rendered `samples/NN.md`, or a raw
	// source `../samples/NN.{gad,gadt,gadx}` from doc/README's "Sample source"
	// column — resolves to the published chapter page lang-<name>.html (whose
	// Example section carries the full runnable source). Raw .gad/.gadt/.gadx
	// files are not published, so without this those links would 404.
	if p := strings.TrimPrefix(url, "../"); strings.HasPrefix(p, "samples/") {
		name := strings.TrimPrefix(p, "samples/")
		for _, ext := range []string{".gadt", ".gadx", ".gad", ".md"} {
			if strings.HasSuffix(name, ext) {
				return "lang-" + strings.TrimSuffix(name, ext) + ".html" + frag
			}
		}
	}
	switch {
	case strings.EqualFold(url, "README.md"):
		url = "index.html"
	case strings.HasSuffix(url, ".md"):
		url = strings.TrimSuffix(url, ".md") + ".html"
	}
	return url + frag
}

// Inline-markup strippers for the search index's plain text (see plainText).
// These are deliberately lightweight — the search snippet does not need perfect
// CommonMark, just readable text with the markup removed.
var (
	boldRe   = regexp.MustCompile(`\*\*(.+?)\*\*`)
	italicRe = regexp.MustCompile(`(^|[^0-9A-Za-z_])_([^_]+?)_([^0-9A-Za-z_]|$)`)
	starRe   = regexp.MustCompile(`(^|[^0-9A-Za-z*])\*([^\s*](?:[^*]*[^\s*])?)\*([^0-9A-Za-z*]|$)`)
	linkRe   = regexp.MustCompile(`\[(.+?)\]\(([^)]+)\)`)
)

// stripInline removes inline markup to produce plain text (for the search index).
func stripInline(s string) string {
	s = strings.ReplaceAll(s, "`", "")
	s = boldRe.ReplaceAllString(s, "$1")
	s = starRe.ReplaceAllString(s, "${1}${2}${3}")
	s = italicRe.ReplaceAllString(s, "${1}${2}${3}")
	s = linkRe.ReplaceAllString(s, "$1")
	return s
}
