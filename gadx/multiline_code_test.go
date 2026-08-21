package gadx

import (
	"strings"
	"testing"

	"github.com/gad-lang/gad"
)

// TestMultilineCodeLine checks that a single `~` code line spans continuation
// lines until its brackets balance (a call / func literal across several lines).
func TestMultilineCodeLine(t *testing.T) {
	src := "@main\n" +
		"    ~ items := [\n" +
		"        \"a\",\n" +
		"        \"b\",\n" +
		"    ]\n" +
		"    @for it in items\n" +
		"        li {= it }\n"
	out := renderGadx(t, src, gad.Dict{})
	if want := "<li>a</li><li>b</li>"; !strings.Contains(out, want) {
		t.Fatalf("multi-line ~ code did not work: want %q in %q", want, out)
	}
}

// TestMultilineCompCall checks that a `+comp(…)` call spans continuation lines
// until the parentheses balance (one named argument per line).
func TestMultilineCompCall(t *testing.T) {
	src := "@comp box(; title = \"\")\n" +
		"    div {= title }\n" +
		"\n" +
		"@main\n" +
		"    +box(;\n" +
		"        title = \"Hello\",\n" +
		"    )\n"
	out := renderGadx(t, src, gad.Dict{})
	if want := "<div>Hello</div>"; !strings.Contains(out, want) {
		t.Fatalf("multi-line +comp call did not work: want %q in %q", want, out)
	}
}
