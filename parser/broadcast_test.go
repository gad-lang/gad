package parser_test

import (
	"testing"

	"github.com/gad-lang/gad/parser/test"
)

// `a .op b` (Julia's broadcast): a `.` right before an operator; the call
// form `a.op(b; …)` — a `(` right after the operator — is postfix and takes
// the options.
func TestParseBroadcast(t *testing.T) {
	test.ExpectParseString(t, `a .+ b`, `(a .+ b)`)
	test.ExpectParseString(t, `a.+b`, `(a .+ b)`)
	test.ExpectParseString(t, `a .+ b * c`, `(a .+ (b * c))`)
	test.ExpectParseString(t, `a .* b + c`, `((a .* b) + c)`)
	test.ExpectParseString(t, `a .+ 2 == b`, `((a .+ 2) == b)`)
	test.ExpectParseString(t, `a.b .< c.d`, `(a.b .< c.d)`)
	test.ExpectParseString(t, `a .<= b .== c`, `((a .<= b) .== c)`)
	test.ExpectParseString(t, `a .!= b`, `(a .!= b)`)
	test.ExpectParseString(t, `a .** 2`, `(a .** 2)`)
	test.ExpectParseString(t, `a .<< 1`, `(a .<< 1)`)
	test.ExpectParseString(t, `a .=== b`, `(a .=== b)`)
	test.ExpectParseString(t, `a .- -1`, `(a .- (-1))`)
	// spaced, the parentheses are an operand's
	test.ExpectParseString(t, `a .+ (1) * 2`, `(a .+ ((1) * 2))`)
	// glued, the call form: postfix
	test.ExpectParseString(t, `a.+(1) * 2`, `(a.+(1) * 2)`)
	test.ExpectParseString(t, `a.+(1; nested)`, `a.+(1; nested=yes)`)
	test.ExpectParseString(t, `a.+(f(1, 2); nested=yes) * 2`, `(a.+(f(1, 2); nested=yes) * 2)`)
	test.ExpectParseString(t, `a.<(b; nested=x > 1, **o)`, `a.<(b; nested=(x > 1), **o)`)
	test.ExpectParseString(t, `a.b.+(c; nested).d`, `a.b.+(c; nested=yes).d`)
	test.ExpectParseString(t, `x * a.+(1; nested)[0]`, `(x * a.+(1; nested=yes)[0])`)
	// a `.` spaced from the operator is not one
	test.ExpectParseError(t, `a. + b`)
	test.ExpectParseError(t, `a.+(1, 2; nested)`)
	test.ExpectParseError(t, `a.+(; nested)`)
}
