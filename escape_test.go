package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
	"github.com/gad-lang/gad/quote"
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

// TestRawInterpolatedStringNoEscape verifies that raw interpolated forms
// (“ #`…` “ and “ #```…``` “) are verbatim: a backslash stays literal and
// `{` always interpolates, so there is no `\{` escape — `\{name}` is a literal
// backslash followed by the interpolation of name.
func TestRawInterpolatedStringNoEscape(t *testing.T) {
	testExpectRun(t, "name := \"Gad\"; return #`path C:\\{name}`", nil, RawStr(`path C:\Gad`))
	testExpectRun(t, "return #`raw \\t {1 + 1}`", nil, RawStr(`raw \t 2`))
	testExpectRun(t, "name := \"Gad\"; return #```C:\\{name}```", nil, RawStr(`C:\Gad`))
}

// TestGadQuoteBuiltins verifies the gad.quote / gad.unquote builtins: the str and
// rawstr overloads pick the cooked vs raw literal form, maxCols switches to a
// multiline heredoc, and quote/unquote round-trip the value.
func TestGadQuoteBuiltins(t *testing.T) {
	testExpectRun(t, `return gad.quote("a\tb")`, nil, Str(`"a\tb"`))
	testExpectRun(t, "return gad.quote(`a\\tb`)", nil, Str("`a\\tb`"))
	testExpectRun(t, `return gad.quote("x\ny\nz"; maxCols=5)`, nil, Str("\"\"\"x\ny\nz\"\"\""))
	testExpectRun(t, `return gad.unquote("\"a\\tb\"")`, nil, Str("a\tb"))
	testExpectRun(t, `x := "a\tb\nc"; return gad.unquote(gad.quote(x)) == x`, nil, True)
}

// TestQuoteCompilerRoundTrip verifies quote.Quote output always parses in the Gad
// compiler back to the original value, across cooked/raw forms, widths and fence
// widths — including edge content (empty, boundary quotes/backticks, indentation,
// delimiter runs).
func TestQuoteCompilerRoundTrip(t *testing.T) {
	run := func(src string) (Object, error) {
		cr, err := Compile(NewSymbolTable(NewBuiltins().NameSet), []byte(src), CompileOptions{})
		if err != nil {
			return nil, err
		}
		return NewVM(NewBuiltins().Build(), cr.BC()).Run()
	}
	values := []string{
		"", "hi", "a\tb", "a\nb", "  indent\nb", "has \" q", "ends \"",
		"\"starts", "l1\nl2\nl3\nl4", "back`tick", "runs```here", "`edge", "edge`",
	}
	opts := []quote.Options{{}, {Raw: true}, {MaxLineWidth: 3}, {MaxLineWidth: 3, Fence: 5}, {Raw: true, MaxLineWidth: 3}}
	for _, s := range values {
		for _, o := range opts {
			lit := quote.Quote(s, o)
			got, err := run("return " + lit)
			if err != nil || got.ToString() != s {
				t.Errorf("Quote(%q, %+v)=%q -> compiler err=%v ret=%q", s, o, lit, err, got)
			}
		}
	}
}
