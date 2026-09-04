package gadx

import (
	"strings"
	"testing"
)

// A `<pre>` lays its content out as written, so the whitespace in it is
// content: collapsing it is what turns preformatted text into a paragraph.
func TestPreKeepsWhitespace(t *testing.T) {
	for name, c := range map[string]struct{ tpl, want string }{
		"pre": {
			"@main\n\t<div><pre>  a   b\n  c    d\n</pre></div>\n",
			"<pre>  a   b\n  c    d\n</pre>",
		},
		"textarea": {
			"@main\n\t<textarea>\n  keep\n</textarea>\n",
			"<textarea>\n  keep\n</textarea>",
		},
		// A `<pre>` may carry tags, unlike a script or a stylesheet.
		"nested tag": {
			"@main\n\t<pre><code>fn x()\n  return 1\n</code></pre>\n",
			"<pre><code>fn x()\n  return 1\n</code></pre>",
		},
		// Outside one, the whitespace is the source's indentation and collapses.
		"outside": {
			"@main\n\t<div>  a   b\n  c    d\n</div>\n",
			"<div> a b c d </div>",
		},
	} {
		out, err := portRun(t, c.tpl, nil, nil)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if !strings.Contains(out, c.want) {
			t.Errorf("%s:\n got %q\nwant %q", name, out, c.want)
		}
	}
}
