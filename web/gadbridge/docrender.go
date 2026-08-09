package gadbridge

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

// docMarkdown is the shared goldmark instance used to render generated
// documentation Markdown to HTML. It enables the same rich extension set as the
// docs website (GFM, Typographer, definition lists, footnotes) plus auto heading
// ids, so the browser-rendered doc matches the published site.
var docMarkdown = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		extension.Typographer,
		extension.DefinitionList,
		extension.Footnote,
	),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
		// `### name {data-src-pos="L,C"}` heading attributes carry source positions
		// for click-to-navigate in the Doc panel.
		parser.WithHeadingAttribute(),
	),
)

// RenderMarkdownToHTML converts documentation Markdown to an HTML fragment using
// the shared goldmark instance (the same conversion the website build uses).
func RenderMarkdownToHTML(md string) (string, error) {
	var buf bytes.Buffer
	if err := docMarkdown.Convert([]byte(md), &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderHTML renders a DocData as an HTML fragment: its Markdown (RenderMarkdown)
// converted through goldmark.
func RenderHTML(d *DocData) (string, error) {
	return RenderMarkdownToHTML(RenderMarkdown(d))
}

// DocHTML extracts the documentation from src and renders it as an HTML fragment.
// sourceType selects the dialect ("gad" | "gadTemplate" | "gadx").
func DocHTML(src, sourceType string) (string, error) {
	d, err := ExtractDoc(src, sourceType)
	if err != nil {
		return "", err
	}
	return RenderHTML(d)
}
