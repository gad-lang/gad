package gadx

import (
	"strings"
	"testing"
)

// `(N)` writes the tag N times, with everything under it.
func TestTagRepeat(t *testing.T) {
	for name, c := range map[string]struct{ tpl, want string }{
		"literal":  {"@main\n\thr(3)\n", "<hr /><hr /><hr />"},
		"subtree":  {"@main\n\tdiv(2)\n\t\tp x\n", "<div><p>x</p></div><div><p>x</p></div>"},
		"computed": {"@param (; n=2)\n\n@main\n\ti(n)\n", "<i></i><i></i>"},
		"zero":     {"@main\n\tdiv\n\t\thr(0)\n", "<div></div>"},
		// The space before the parenthesis is what keeps it text.
		"text": {"@main\n\tp (781) 496-8660\n", "<p>(781) 496-8660</p>"},
	} {
		out, err := portRun(t, c.tpl, nil, nil)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if !strings.Contains(out, c.want) {
			t.Errorf("%s: got %s, want %s", name, out, c.want)
		}
	}
}

// A repeat past what the lowering writes out as copies becomes a loop, and has
// to render the same.
func TestTagRepeatBeyondUnroll(t *testing.T) {
	out, err := portRun(t, "@main\n\thr(40)\n", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(out, "<hr />"); got != 40 {
		t.Errorf("got %d copies, want 40", got)
	}
}
