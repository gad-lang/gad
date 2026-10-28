package gadx

import (
	"strings"
	"testing"
)

// The arguments of `+comp(…)` are gad's: after `;` a name alone is a flag
// (on: true), and the named arguments of a call over several lines are separated
// by the new lines — each of them taken, none dropped after the first.
func TestCompCallArgsAsGad(t *testing.T) {
	const decl = "@comp lay(; title=\"\", seo=\"\", flag=false)\n\tp {= title }|{= seo }|{= flag ? \"yes\" : \"no\" }\n\n"

	for name, c := range map[string]struct{ call, want string }{
		"a flag":              {"+lay(; flag)\n", "<p>||yes</p>"},
		"a flag after others": {"+lay(; title=\"t\", flag)\n", "<p>t||yes</p>"},
		"lines":               {"+lay(\n\t; title=\"t\"\n\tseo=\"s\"\n)\n", "<p>t|s|no</p>"},
		"lines with a flag":   {"+lay(\n\t; title=\"t\"\n\tseo=\"s\"\n\tflag\n)\n", "<p>t|s|yes</p>"},
		"a call in a line":    {"+lay(\n\t; title=str(1 + 1)\n\tseo=\"s\"\n)\n", "<p>2|s|no</p>"},
	} {
		out, err := portRun(t, decl+c.call, nil, nil)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if !strings.Contains(out, c.want) {
			t.Errorf("%s:\n got %s\nwant %s", name, out, c.want)
		}
	}
}
