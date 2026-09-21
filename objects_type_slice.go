package gad

import "strings"

// SliceType is a slice written where a type goes: `[]int`, `[][]str`,
// `[]<int|str>`. It accepts an ARRAY nested to Depth whose leaf elements are
// each accepted by one of Elem — the same rule a parameter's declared types
// follow, because Elem is exactly that: the element's allowed types, as the
// symbols the compiler resolved them to.
//
// `[]int` is the short form of `[]<int>`: one type needs no envelope.
type SliceType struct {
	// Depth is how many `[]` were written: 1 for `[]int`, 3 for `[][][]int`.
	Depth int
	// Elem are the types an element may be, as symbols resolved per VM — so an
	// element type may be anything a parameter type may be, a class or an
	// interface included.
	Elem ParamType
	// ElemNames are those types as written, for the type's own name.
	ElemNames []string
}

var (
	_ Object        = (*SliceType)(nil)
	_ TypeAssigner  = (*SliceType)(nil)
	_ vmCanAssigner = (*SliceType)(nil)
)

// A slice type is STRUCTURAL, not an ObjectType: it answers about the shape of
// a value (an array nested so deep, of such elements), and there is no type
// chain to walk to it.
func (t *SliceType) Type() ObjectType { return TArray }

func (t *SliceType) Name() string {
	var b strings.Builder
	for i := 0; i < t.Depth; i++ {
		b.WriteString("[]")
	}
	if len(t.ElemNames) == 1 {
		b.WriteString(t.ElemNames[0])
	} else {
		b.WriteString("<" + strings.Join(t.ElemNames, "|") + ">")
	}
	return b.String()
}

func (t *SliceType) ToString() string { return t.Name() }
func (t *SliceType) String() string   { return t.Name() }
func (t *SliceType) IsFalsy() bool    { return false }

func (t *SliceType) Equal(right Object) bool {
	r, ok := right.(*SliceType)
	return ok && r == t
}

// AssignTo is how a structural type is asked "does this value satisfy you":
// the caller passes the receiver itself as `to` (see ifaceFieldTypeOK), so the
// answer is the check, not an identity test.
func (t *SliceType) AssignTo(vm *VM, obj Object, to TypeAssigner) (Object, error) {
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

// CanAssign without a VM cannot resolve the element types, so it only answers
// the part it can see: obj must be an array nested to Depth. CanAssignVM is the
// real check, and ParamType.Accept prefers it.
func (t *SliceType) CanAssign(obj Object) (bool, error) {
	return t.nested(obj, t.Depth, nil)
}

// CanAssignVM reports whether obj is an array nested to Depth whose leaves are
// each accepted by one of the element types.
func (t *SliceType) CanAssignVM(vm *VM, obj Object) (bool, error) {
	return t.nested(obj, t.Depth, vm)
}

// nested walks depth levels of array, checking the leaves when a VM is at hand
// to resolve the element types.
func (t *SliceType) nested(obj Object, depth int, vm *VM) (bool, error) {
	if depth == 0 {
		if vm == nil || len(t.Elem) == 0 {
			return true, nil
		}
		return t.Elem.Accept(vm, obj)
	}

	arr, ok := obj.(Array)
	if !ok {
		return false, nil
	}
	for _, item := range arr {
		ok, err := t.nested(item, depth-1, vm)
		if err != nil || !ok {
			return ok, err
		}
	}
	return true, nil
}
