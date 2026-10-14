package parser_test

import (
	"testing"

	"github.com/gad-lang/gad/parser/node"
	"github.com/gad-lang/gad/parser/test"
)

// A class body keeps the order of its fields, parents (`*P`), mixins (`use
// M`) and spreads (`**E`) as written — several of each, in any order —:
// TypeLitExpr.Order.
func TestParseClassOrder(t *testing.T) {
	cls := parseClass(t, `class C {
	z str
	*P1
	a int
	use MA, MB
	**s1
	b int
	*P2
	use MC
	**s2
	c int
	**s3
}`)
	kinds := map[node.ClassBodyKind]string{node.ClassBodyField: "f", node.ClassBodyParent: "p",
		node.ClassBodyUse: "u", node.ClassBodySpread: "s"}
	var got string
	for _, it := range cls.Order {
		got += kinds[it.Kind] + string(rune('0'+it.Index)) + " "
	}
	if want := "f0 p0 f1 u0 u1 s0 f2 p1 u2 s1 f3 s2 "; got != want {
		t.Errorf("the order: %q, want %q", got, want)
	}
}

// The body written as declared — fields, parents, mixins (a `use` of
// several, one clause) and spreads where they are —, `[ordered]` or not: a
// class is always ordered.
func TestFormatClassOrder(t *testing.T) {
	test.New(t, "[ordered] class C { z str; *P1; a int; use MA, MB; **s1; b int; *P2; use MC; **s2; c int; **s3 }").
		FormattedCode("[ordered]\nclass C {\n\tz str\n\t*P1\n\ta int\n\tuse MA, MB\n\t**s1\n\tb int\n\t*P2\n\tuse MC\n\t**s2\n\tc int\n\t**s3\n}")
	test.New(t, "class C { z str; *P1; a int; use MA; **s1; b int }").
		FormattedCode("class C {\n\tz str\n\t*P1\n\ta int\n\tuse MA\n\t**s1\n\tb int\n}")
}
