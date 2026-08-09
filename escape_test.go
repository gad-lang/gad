package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// TestInterpolatedStringBraceEscape verifies `\{` / `\}` escape the interpolation
// delimiters to literal braces in `#"…"` interpolated strings, while unescaped
// braces still interpolate and standard escapes (\t, \\) keep working.
func TestInterpolatedStringBraceEscape(t *testing.T) {
	testExpectRun(t, `return #"a \{ b } c"`, nil, Str("a { b } c"))
	testExpectRun(t, `x := 1; return #"v=\{ {x} }"`, nil, Str("v={ 1 }"))
	testExpectRun(t, `return #"close \}"`, nil, Str("close }"))
	testExpectRun(t, `return #"plain { 1 + 1 }"`, nil, Str("plain 2"))
	testExpectRun(t, `return #"back \\ slash {1}"`, nil, Str(`back \ slash 1`))
	testExpectRun(t, `return #"tab\tafter {1}"`, nil, Str("tab\tafter 1"))
	// heredoc interpolated form escapes the same way
	testExpectRun(t, "return #\"\"\"a \\{ b } c\"\"\"", nil, Str("a { b } c"))
}

// TestRegularStringBraceEscape verifies `\{` / `\}` collapse to literal braces in
// ordinary (non-interpolated) string literals too, keeping escapes consistent
// across all string forms.
func TestRegularStringBraceEscape(t *testing.T) {
	testExpectRun(t, `return "a \{ b \} c"`, nil, Str("a { b } c"))
	testExpectRun(t, `return "back \\ slash"`, nil, Str(`back \ slash`))
}
