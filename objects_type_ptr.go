package gad

import "strings"

// PtrType is a pointer type written where a type goes: `*int`, `*array`,
// `*<int|str>`. It accepts a Pointer (a `ptr`, see the `&` operator) whose
// CURRENT value is accepted by one of Elem — the check runs when the value is
// assigned (a call's argument, a field), not on later writes through `.v`.
// Elem are the pointed-to types as the symbols the compiler resolved them to,
// so they may be anything a parameter type may be.
type PtrType struct {
	Elem ParamType
	// ElemNames are those types as written, for the type's own name.
	ElemNames []string
}

var (
	_ IndexGetter   = (*PtrType)(nil)
	_ Object        = (*PtrType)(nil)
	_ TypeAssigner  = (*PtrType)(nil)
	_ vmCanAssigner = (*PtrType)(nil)
)

// A pointer type is STRUCTURAL: it answers about a value's shape (a pointer to
// such a value).
func (t *PtrType) Type() ObjectType { return TPtr }

func (t *PtrType) Name() string {
	if len(t.ElemNames) == 1 {
		return "*" + t.ElemNames[0]
	}
	return "*<" + strings.Join(t.ElemNames, "|") + ">"
}

func (t *PtrType) ToString() string { return t.Name() }
func (t *PtrType) String() string   { return t.Name() }
func (t *PtrType) IsFalsy() bool    { return false }

func (t *PtrType) Equal(right Object) bool {
	r, ok := right.(*PtrType)
	return ok && r == t
}

// AssignTo answers "does obj satisfy this pointer type" (see ArrayType.AssignTo).
func (t *PtrType) AssignTo(vm *VM, obj Object, to TypeAssigner) (Object, error) {
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

// CanAssign without a VM cannot resolve the pointed-to types: obj must only be a
// Pointer. CanAssignVM is the real check.
func (t *PtrType) CanAssign(obj Object) (bool, error) {
	_, ok := obj.(Pointer)
	return ok, nil
}

// CanAssignVM reports whether obj is a Pointer whose current value is accepted
// by one of the pointed-to types.
func (t *PtrType) CanAssignVM(vm *VM, obj Object) (bool, error) {
	p, ok := obj.(Pointer)
	if !ok {
		return false, nil
	}
	if vm == nil || len(t.Elem) == 0 {
		return true, nil
	}
	v, err := p.PtrGet(vm)
	if err != nil {
		return false, err
	}
	return t.Elem.Accept(vm, v)
}

// IndexGet reflects the pointer type: `@elem` — the pointed-to types, resolved
// in vm.
func (t *PtrType) IndexGet(vm *VM, index Object) (Object, error) {
	if index.ToString() == "@elem" {
		if vm == nil {
			return nil, ErrInvalidIndex.NewError("@elem (no VM to resolve the pointed-to types)")
		}
		out := make(Array, 0, len(t.Elem))
		for _, s := range t.Elem {
			v, err := vm.GetSymbolValue(s)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	}
	return nil, ErrInvalidIndex.NewError(index.ToString())
}
