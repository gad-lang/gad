package gad

import "strings"

// RangeOfType is `Range[T]`: the type of a range whose bounds are each of
// T — `Range[int]`, `Range[time.CalendarDate]`, `Range[int|float]` —, a
// type as a field's or a parameter's is. It is STRUCTURAL, as an array type
// is: a value is of it when it is a Range whose From and To are of T.
type RangeOfType struct {
	// Elem is the type of the bounds: a type, or a union of them.
	Elem Object
}

var (
	_ IndexGetter   = (*RangeOfType)(nil)
	_ Object        = (*RangeOfType)(nil)
	_ TypeAssigner  = (*RangeOfType)(nil)
	_ vmCanAssigner = (*RangeOfType)(nil)
)

func init() {
	RangeType.WithIndexType(func(_ *VM, index Object) (Object, error) {
		return &RangeOfType{Elem: index}, nil
	})
}

func (t *RangeOfType) Type() ObjectType { return RangeType }

func (t *RangeOfType) Name() string {
	name := t.Elem.ToString()
	if n, ok := t.Elem.(interface{ Name() string }); ok {
		name = n.Name()
	}
	return "Range[" + strings.TrimSpace(name) + "]"
}

func (t *RangeOfType) ToString() string { return t.Name() }
func (t *RangeOfType) String() string   { return t.Name() }
func (t *RangeOfType) IsFalsy() bool    { return false }

func (t *RangeOfType) Equal(right Object) bool {
	r, ok := right.(*RangeOfType)
	return ok && (r == t || r.Elem.Equal(t.Elem))
}

// AssignTo is the check of a value against the type (the receiver is `to`,
// as of a structural type).
func (t *RangeOfType) AssignTo(vm *VM, obj Object, to TypeAssigner) (Object, error) {
	if to != TypeAssigner(t) {
		return nil, ErrIncompatibleAssign
	}
	ok, err := t.CanAssignVM(vm, obj)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrIncompatibleAssign
	}
	return obj, nil
}

// CanAssign, with no VM, sees a Range only.
func (t *RangeOfType) CanAssign(obj Object) (bool, error) {
	_, ok := obj.(*Range)
	return ok, nil
}

// CanAssignVM reports whether obj is a Range whose bounds are of Elem.
func (t *RangeOfType) CanAssignVM(vm *VM, obj Object) (bool, error) {
	r, ok := obj.(*Range)
	if !ok {
		return false, nil
	}
	for _, bound := range []Object{r.From, r.To} {
		if _, err := AssignToType(vm, bound, t.Elem); err != nil {
			return false, nil
		}
	}
	return true, nil
}

// IndexGet reflects the type: `@elem` — the type of the bounds.
func (t *RangeOfType) IndexGet(_ *VM, index Object) (Object, error) {
	if index.ToString() == "@elem" {
		return t.Elem, nil
	}
	return nil, ErrInvalidIndex.NewError(index.ToString())
}
