package gad

// FuncType is a function header written where a type goes: `<(x int) <ret any>>`.
// It accepts a CALLABLE whose signature matches the header — the same structural
// match a method member of an interface uses (see headerMatchesAny).
//
// A header is the one single type that must be enveloped as a slice element
// (`[]<(x int)>`), because its own syntax already is the `<…>` envelope.
type FuncType struct {
	Header *FuncHeaderObject
}

var (
	_ Object        = (*FuncType)(nil)
	_ TypeAssigner  = (*FuncType)(nil)
	_ vmCanAssigner = (*FuncType)(nil)
)

// Like SliceType, a func type is STRUCTURAL, not an ObjectType: it answers about
// the shape of a value, and there is no type chain to walk to it.
func (t *FuncType) Type() ObjectType { return TFunctionHeader }

func (t *FuncType) Name() string     { return t.Header.String() }
func (t *FuncType) ToString() string { return t.Name() }
func (t *FuncType) String() string   { return t.Name() }
func (t *FuncType) IsFalsy() bool    { return false }

func (t *FuncType) Equal(right Object) bool {
	r, ok := right.(*FuncType)
	return ok && r == t
}

// AssignTo is how a structural type is asked "does this value satisfy you": the
// caller passes the receiver itself as `to` (see ifaceFieldTypeOK).
func (t *FuncType) AssignTo(vm *VM, obj Object, to TypeAssigner) (Object, error) {
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

// CanAssign without a VM cannot split a caller into its signatures, so it only
// answers the part it can see: obj must be callable. CanAssignVM is the real
// check, and ParamType.Accept prefers it.
func (t *FuncType) CanAssign(obj Object) (bool, error) {
	_, ok := obj.(CallerObject)
	return ok, nil
}

// CanAssignVM reports whether obj is a callable with a signature the header
// matches.
func (t *FuncType) CanAssignVM(vm *VM, obj Object) (bool, error) {
	if _, ok := obj.(CallerObject); !ok {
		return false, nil
	}
	var sigs []ParamsTypes
	if err := SplitCaller(vm, obj,
		func(_ CallerObject, types ParamsTypes) error { sigs = append(sigs, types); return nil },
		func(_ CallerObject) error { sigs = append(sigs, nil); return nil },
	); err != nil {
		return false, nil
	}
	return headerMatchesAny(t.Header, sigs), nil
}
