package gad_test

import (
	"strings"
	"testing"

	. "github.com/gad-lang/gad"
)

// DescribeType discovers what the builtins register of a type: its
// constructors, operators, conversions — each in its group, in order.
func TestDescribeType(t *testing.T) {
	b := NewBuiltins()
	members := DescribeType(b, b.Objects[BuiltinUint].(ObjectType))
	has := map[string]bool{}
	var groups []string
	for _, m := range members {
		has[m.Group+" "+m.Signature] = true
		if len(groups) == 0 || groups[len(groups)-1] != m.Group {
			groups = append(groups, m.Group)
		}
	}
	for _, want := range []string{
		"Constructors uint(v str)", "Constructors uint(v float)",
		"Operators a + b", "Operators a << b", "Operators -a", "Operators a++",
		"Conversions int(v uint)", "Conversions str(v uint)",
	} {
		if !has[want] {
			t.Errorf("no %q", want)
		}
	}
	if strings.Join(groups, ",") != "Constructors,Operators,Conversions" {
		t.Errorf("groups %v", groups)
	}
	// a type without overloads: its default constructor
	arr := DescribeType(b, b.Objects[BuiltinArray].(ObjectType))
	if len(arr) == 0 || arr[0].Group != TypeMemberConstructors {
		t.Fatalf("array: %v", arr)
	}
}

// `-d` of a decimal — what a negative decimal literal is.
func TestDecimalUnary(t *testing.T) {
	testExpectRun(t, `return [-1.5d, +(2d), str([-1.5d])]`, nil,
		Array{MustDecimalFromString("-1.5"), MustDecimalFromString("2"), Str("[-1.5d]")})
}

// keyValueArray.sort orders by key, then by value when the values compare;
// sortReverse is its reverse; the pairs of one key keep their order otherwise.
func TestKeyValueArraySort(t *testing.T) {
	testExpectRun(t, `return [str((;b=2, a=3, a=1, debug).sort(true)), str((;b=2, a=3, a=1).sortReverse())]`, nil,
		Array{Str("(;a=1, a=3, b=2, debug)"), Str("(;b=2, a=3, a=1)")})
}
