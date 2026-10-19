package parser_test

import (
	"testing"

	"github.com/gad-lang/gad/parser/test"
)

// The casts `:>` (transform) and `::>` (convert) scan as one token each and
// format with spaces around them, like the other binary operators.
func TestFormatTransformCasts(t *testing.T) {
	test.New(t, "a :> B").Code("a :> B").FormattedCode("a :> B")
	test.New(t, "a ::> B").Code("a ::> B").FormattedCode("a ::> B")
	test.New(t, "a :> B :> C").Code("a :> B :> C").FormattedCode("a :> B :> C")
	// written tight, they still scan as the cast, not as `:` and `>`
	test.New(t, "a:>B").Code("a :> B").FormattedCode("a :> B")
	test.New(t, "a::>B").Code("a ::> B").FormattedCode("a ::> B")
	// as the base of a selector, an index or a call they keep their parens
	test.New(t, "(a :> B).c").Code("(a :> B).c").FormattedCode("(a :> B).c")
	test.New(t, "(a ::> B)[0]").Code("(a ::> B)[0]").FormattedCode("(a ::> B)[0]")
	// `::` keeps its tight form beside them
	test.New(t, "a :: B").Code("a::B").FormattedCode("a::B")
}
