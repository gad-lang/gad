package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// gad fmt formats the fenced code blocks of Gad in Markdown — ```gad,
// ```gadt, ```gadx — each as its dialect; an indented block (in a list)
// keeps its indentation; a block that does not read is left as it is; the
// text, and blocks of other languages, are left as they are.
func TestFormatMarkdown(t *testing.T) {
	src := "# Title\n\nText with `[x] y str` inline.\n\n" +
		"```gad\n[label=\"Name\"] class F { [label=\"A\"] a str }\n```\n\n" +
		"- an item:\n\n  ```gad\n  x := [1,2]\n  ```\n\n" +
		"```gadx\n@main\n    div[data=(a?b:c)]\n        p hi\n```\n\n" +
		"~~~gadt\nHello {%= name %}\n~~~\n\n" +
		"```gad\nthis is ( not gad\n```\n\n" +
		"```go\nx :=   1\n```\n"
	want := "# Title\n\nText with `[x] y str` inline.\n\n" +
		"```gad\n[label=\"Name\"]\nclass F {\n\t[label=\"A\"]\n\ta str\n}\n```\n\n" +
		"- an item:\n\n  ```gad\n  x := [1, 2]\n  ```\n\n" +
		"```gadx\n@main\n\tdiv[data=(a ? b : c)]\n\t\tp hi\n```\n\n" +
		"~~~gadt\nHello {%= name %}\n~~~\n\n" +
		"```gad\nthis is ( not gad\n```\n\n" +
		"```go\nx :=   1\n```\n"
	o := &fmtOptions{codeFlags: fmtFormatFlag()}
	out, err := o.formatSource("README.md", []byte(src), false)
	require.NoError(t, err)
	require.Equal(t, want, out)
	again, err := o.formatSource("README.md", []byte(out), false)
	require.NoError(t, err)
	require.Equal(t, out, again)
	require.True(t, isFmtSource("doc/README.md"), "gad fmt takes the Markdown files of a directory")
	require.False(t, isGadSource("doc/README.md"), "gad doc, test… do not")
}
