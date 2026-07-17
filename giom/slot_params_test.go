package giom

import (
	"strings"
	"testing"

	"github.com/gad-lang/gad"
)

// This file recreates the upstream giom v1 `Test_RunPrintLines`
// (run_test.go / compiler_test.go at d1edc53) in the current syntax, exercising
// scoped slots (slots with parameters), per-index dynamic slot overrides and the
// auto-injected `super`.
//
// `compPrintLines` renders each row through a scoped `@slot line(i, line)`. Its
// default looks up a per-index override `slots["line[i]"]`; when present it
// invokes it, passing the default renderer `print_line` as the override's
// `super` (its auto-injected first parameter), otherwise it renders the default
// line directly.
const compPrintLines = "@comp print_lines(rows)\n" +
	"\t@func print_line(i, line)\n" +
	"\t\t| {=i}: {=line}{=\"\\n\"}\n" +
	"\n" +
	"\t@for i, line in rows\n" +
	"\t\t@slot line(i, line)\n" +
	"\t\t\t~~\n" +
	"\t\t\tconst custom = slots[\"line[\"+i+\"]\"]\n" +
	"\t\t\ttag += (custom ? (*args) => custom(print_line, *args) : print_line)(i, line)\n" +
	"\t\t\t~~\n"

// printLinesExpect renders tpl and asserts the trimmed output equals want.
func printLinesExpect(t *testing.T, name, tpl, want string) {
	t.Helper()
	got := strings.TrimSpace(renderGiom(t, tpl, gad.Dict{}))
	if got != want {
		t.Errorf("[%s] render mismatch\n got: %q\nwant: %q", name, got, want)
	}
}

func TestSlotParams_PrintLines(t *testing.T) {
	// A component definition alone (no @main) renders nothing.
	printLinesExpect(t, "comp only",
		compPrintLines+"\n@enum Levels (primary, secondary)\n",
		"")

	// A plain @main renders independently of the component.
	printLinesExpect(t, "plain main",
		compPrintLines+"@main\n\tdiv\n",
		"<div></div>")

	// No override: every row uses the default line renderer.
	printLinesExpect(t, "defaults",
		compPrintLines+"@main\n\t+print_lines([\"a\", \"b\", \"c\", \"d\"])\n",
		"0: a\n1: b\n2: c\n3: d")

	// Overriding the whole `line` slot replaces every row. The override receives
	// `i` and `line` as scope; `super` is auto-injected but unused here.
	printLinesExpect(t, "override all",
		compPrintLines+
			"@main\n"+
			"\t+print_lines([\"a\", \"b\", \"c\", \"d\"])\n"+
			"\t\t@slot #line(i, line)\n"+
			"\t\t\t| line {=str(i, \"\\n\")}\n",
		"line 0\nline 1\nline 2\nline 3")

	// Per-index overrides: `line[1]` replaces row 1; `line[3]` replaces row 3 and
	// renders the default afterwards via `super(i, line)`. `super` is the
	// override's auto-injected first parameter.
	printLinesExpect(t, "per-index super",
		compPrintLines+
			"@main\n"+
			"\t+print_lines([\"a\", \"b\", \"c\", \"d\"])\n"+
			"\t\t~ const index=3\n"+
			"\t\t@slot #line[1](super, i, line)\n"+
			"\t\t\t| line 1 {=\"\\n\"}\n"+
			"\t\t@slot #(line[{index}])(super, i, line)\n"+
			"\t\t\t| line 3 @ \n"+
			"\t\t\t~ tag += super(i, line)\n",
		"0: a\nline 1 \n2: c\nline 3 @3: d")

	// A per-index override that ignores `super` just declares its scope
	// parameters; `super` is still auto-injected (first) and simply not called.
	printLinesExpect(t, "per-index ignore super",
		compPrintLines+
			"@main\n"+
			"\t+print_lines([\"a\", \"b\", \"c\", \"d\"])\n"+
			"\t\t@slot #line[1](i, line)\n"+
			"\t\t\t| line 1 {=\"\\n\"}\n",
		"0: a\nline 1 \n2: c\n3: d")
}
