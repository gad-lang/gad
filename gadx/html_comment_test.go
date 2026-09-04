package gadx

import (
	"strings"
	"testing"
)

// An HTML comment is content: it is how a template hands a note to whatever
// reads the rendered page. A `//` comment is the one that stops at the
// template.
func TestHTMLCommentIsRendered(t *testing.T) {
	for name, tpl := range map[string]string{
		"own line":    "@main\n\tdiv\n\t\t<!-- a note -->\n\t\tp x\n",
		"inline html": "@main\n\t<div><!-- a note --><p>x</p></div>\n",
	} {
		out, err := portRun(t, tpl, nil, nil)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !strings.Contains(out, "<!-- a note -->") {
			t.Errorf("%s: comment dropped:\n%s", name, out)
		}
	}
}

// A doctype and a CDATA section carry nothing to render; the doctype has the
// `!!! 5` statement of its own.
func TestHTMLDeclarationsAreDropped(t *testing.T) {
	out, err := portRun(t, "@main\n\t<div><!DOCTYPE html><![CDATA[x]]><p>x</p></div>\n", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out != "<div><p>x</p></div>" {
		t.Errorf("declarations were not dropped: %s", out)
	}
}
