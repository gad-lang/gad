package gad

import (
	"strconv"
	"strings"
)

// TArraySize is the type of the size of a level of an array type (@size).
var TArraySize = NewType("arraySize")

// ArraySize is the size of a level of an array type, as `@size` gives it:
// an array of three, [eq, min, max] — `[2]` is [2, nil, nil], `[2..3]` is
// [nil, 2, 3], `[..3]` [nil, nil, 3], `[2..]` [nil, 2, nil], `[]` [nil, nil,
// nil] —, its items by index or by name (`.eq`, `.min`, `.max`), nil where the
// level says none.
type ArraySize struct {
	Eq, Min, Max Object
}

var (
	_ IndexGetter  = (*ArraySize)(nil)
	_ LengthGetter = (*ArraySize)(nil)
	_ Iterabler    = (*ArraySize)(nil)
)

// newArraySize is the ArraySize of a resolved level.
func newArraySize(b arrayBound) *ArraySize {
	s := &ArraySize{Eq: Nil, Min: Nil, Max: Nil}
	if b.hasEq {
		s.Eq = Int(b.eq)
	}
	if b.hasMin {
		s.Min = Int(b.min)
	}
	if b.hasMax {
		s.Max = Int(b.max)
	}
	return s
}

// Array is the size as an array: [eq, min, max].
func (s *ArraySize) Array() Array { return Array{s.Eq, s.Min, s.Max} }

func (s *ArraySize) Type() ObjectType { return TArraySize }
func (s *ArraySize) IsFalsy() bool    { return false }
func (s *ArraySize) Length() int      { return 3 }
func (s *ArraySize) ToString() string { return s.Array().ToString() }
func (s *ArraySize) String() string   { return s.ToString() }

// Equal is an ArraySize, or an array, of the same items.
func (s *ArraySize) Equal(right Object) bool {
	switch r := right.(type) {
	case *ArraySize:
		return s.Array().Equal(r.Array())
	case Array:
		return s.Array().Equal(r)
	}
	return false
}

func (s *ArraySize) IndexGet(vm *VM, index Object) (Object, error) {
	switch i := index.(type) {
	case Int:
		if i < 0 || i > 2 {
			return nil, ErrIndexOutOfBounds.NewError(strconv.Itoa(int(i)))
		}
		return s.Array()[i], nil
	case Uint:
		if i > 2 {
			return nil, ErrIndexOutOfBounds.NewError(strconv.Itoa(int(i)))
		}
		return s.Array()[i], nil
	}
	switch index.ToString() {
	case "eq":
		return s.Eq, nil
	case "min":
		return s.Min, nil
	case "max":
		return s.Max, nil
	}
	return nil, ErrInvalidIndex.NewError(index.ToString())
}

func (s *ArraySize) Iterate(vm *VM, na *NamedArgs) Iterator {
	return s.Array().Iterate(vm, na)
}

// ArraySizeRef is a bound of a level as written (Set): a number (N), or a
// variable (Sym, the index of its symbol in ArrayType.SizeSyms; -1 for a
// number).
type ArraySizeRef struct {
	Set bool
	N   int64
	Sym int
}

// ArrayLevel is the size of a level of an array type as written: exactly Eq,
// or at least Min and at most Max — each a number, or a variable read when a
// value is checked.
type ArrayLevel struct {
	Eq, Min, Max ArraySizeRef
}

// arrayBound is a level's size resolved: the numbers it says.
type arrayBound struct {
	eq, min, max          int64
	hasEq, hasMin, hasMax bool
}

// accepts reports whether n items are of the size.
func (b arrayBound) accepts(n int) bool {
	switch {
	case b.hasEq:
		return int64(n) == b.eq
	case b.hasMin && int64(n) < b.min:
		return false
	case b.hasMax && int64(n) > b.max:
		return false
	}
	return true
}

// String is the level with its numbers: `[2]`, `[2..5]`, `[..3]`, `[2..]`, `[]`.
func (b arrayBound) String() string {
	var sb strings.Builder
	sb.WriteByte('[')
	switch {
	case b.hasEq:
		sb.WriteString(strconv.FormatInt(b.eq, 10))
	case b.hasMin || b.hasMax:
		if b.hasMin {
			sb.WriteString(strconv.FormatInt(b.min, 10))
		}
		sb.WriteString("..")
		if b.hasMax {
			sb.WriteString(strconv.FormatInt(b.max, 10))
		}
	}
	sb.WriteByte(']')
	return sb.String()
}

// want says what a level of the size wants: "exactly 2 items", "2 to 5
// items", "at least 2 items", "at most 3 items".
func (b arrayBound) want() string {
	items := func(n int64) string {
		if n == 1 {
			return "1 item"
		}
		return strconv.FormatInt(n, 10) + " items"
	}
	switch {
	case b.hasEq:
		return "exactly " + items(b.eq)
	case b.hasMin && b.hasMax:
		return strconv.FormatInt(b.min, 10) + " to " + items(b.max)
	case b.hasMin:
		return "at least " + items(b.min)
	case b.hasMax:
		return "at most " + items(b.max)
	}
	return "any number of items"
}
