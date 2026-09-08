package gadx

import (
	"strings"
	"testing"
)

// `+(expr)` renders whatever the expression evaluates to. The bare `+name`
// form reads a name, and a name cannot hold an operator — so a fallback or a
// branch had no way to say which component to render, and the line was left as
// text, silently.
func TestCompCallByExpression(t *testing.T) {
	const decls = "@comp a(t)\n\tp A{= t }\n\n@comp b(t)\n\tp B{= t }\n\n"

	for name, c := range map[string]struct{ tpl, want string }{
		"a branch": {
			decls + "@main\n\t+(1 < 2 ? a : b)(\"x\")\n",
			"<p>Ax</p>",
		},
		"a fallback, taken": {
			decls + "@main\n\t~ cfg := {}\n\t+(cfg.comp ?? b)(\"x\")\n",
			"<p>Bx</p>",
		},
		"a fallback, not taken": {
			decls + "@main\n\t~ cfg := {comp: a}\n\t+(cfg.comp ?? b)(\"x\")\n",
			"<p>Ax</p>",
		},
		"parentheses around a plain name": {
			decls + "@main\n\t+(a)(\"x\")\n",
			"<p>Ax</p>",
		},
		"no arguments": {
			"@comp a()\n\tp A\n\n@main\n\t+(a)\n",
			"<p>A</p>",
		},
	} {
		out, err := portRun(t, c.tpl, nil, nil)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if !strings.Contains(out, c.want) {
			t.Errorf("%s:\n got %s\nwant %s", name, out, c.want)
		}
	}
}

// The head only decides which component runs; a call block still applies.
func TestCompCallByExpressionTakesSlots(t *testing.T) {
	tpl := "@comp box()\n\tdiv\n\t\t@slot main\n\t\t\tp default\n\n" +
		"@main\n\t+(box)\n\t\t@slot #main\n\t\t\tem filled\n"

	out, err := portRun(t, tpl, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "<div><em>filled</em></div>") {
		t.Errorf("got %s", out)
	}
}

// An empty head is not a call: it stays text, as `+` alone always has.
func TestCompCallEmptyExpressionIsText(t *testing.T) {
	out, err := portRun(t, "@main\n\t+()\n", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "+()") {
		t.Errorf("got %s, want the line as text", out)
	}
}
