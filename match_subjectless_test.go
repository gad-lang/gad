package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// TestMatchSubjectless verifies `match { … }` (no subject) is sugar for
// `match true { … }`: the first arm whose condition is truthy wins.
func TestMatchSubjectless(t *testing.T) {
	src := `
		classify := func(n) => match {
			n < 0:          "negative"
			n == 0:         "zero"
			n == 1, n == 2: "one or two"
			n < 10:         "small"
			else:           "large"
		}
		return [classify(-3), classify(0), classify(1), classify(2), classify(4), classify(42)]`
	testExpectRun(t, src, nil,
		Array{Str("negative"), Str("zero"), Str("one or two"), Str("one or two"), Str("small"), Str("large")})
}

// TestMatchSubjectlessEqualsTrue verifies `match { … }` and `match true { … }`
// compile to the same behaviour.
func TestMatchSubjectlessEqualsTrue(t *testing.T) {
	testExpectRun(t, `
		f := func(n) => match      { n > 0: "p", else: "np" }
		g := func(n) => match true { n > 0: "p", else: "np" }
		return [f(5), g(5), f(-5), g(-5)]`,
		nil, Array{Str("p"), Str("p"), Str("np"), Str("np")})
}

// TestMatchSubjectlessNoElseNil verifies a subject-less match with no matching
// arm and no else yields nil (like the subject form).
func TestMatchSubjectlessNoElseNil(t *testing.T) {
	testExpectRun(t, `return match { false: "x", 1 > 2: "y" }`, nil, Nil)
}

// TestMatchSubjectlessStmtForm verifies the statement (block) arm form works
// without a subject.
func TestMatchSubjectlessStmtForm(t *testing.T) {
	testExpectRun(t, `
		out := ""
		n := 3
		match {
			n < 0 { out = "neg" }
			n < 10 { out = "small" }
			else { out = "big" }
		}
		return out`,
		nil, Str("small"))
}

// TestMatchSubjectlessDictSubject verifies a dict-literal subject still works when
// parenthesized — the `match {` sugar only triggers on a brace immediately after
// `match`.
func TestMatchSubjectlessDictSubject(t *testing.T) {
	testExpectRun(t, `
		d := {a: 1}
		return match (d) { {a: 1}: "hit", else: "miss" }`,
		nil, Str("hit"))
}
