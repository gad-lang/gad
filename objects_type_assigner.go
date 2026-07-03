package gad

// TypeAssigner is a value that can decide whether another value is assignable to
// it — the abstraction behind parameter/field type checking. An ObjectType
// assigns by type (assignability), a *MethodInterface by a structural
// `implements` check, and a *Interface by structural satisfaction.
type TypeAssigner interface {
	Object
	// AssignTo returns obj when obj (of the receiver's kind) is assignable to
	// `to`, otherwise an error (ErrIncompatibleCast). It returns obj unchanged on
	// success (the value already satisfies the target).
	AssignTo(vm *VM, obj Object, to TypeAssigner) (Object, error)

	// CanAssign returns if obj can assign to this
	CanAssign(obj Object) (bool, error)
}

// TypeAssignerArray is a list of type assigners (e.g. the allowed types of a
// parameter): ObjectTypes and/or structural types (meti/interface). Named to
// avoid a clash with the existing TypeAssigners walker function.
type TypeAssignerArray []TypeAssigner

// assignByTypeChain implements the ObjectType flavour of AssignTo: obj is
// assignable to `to` when `to` is an ObjectType in the receiver type's ancestry
// chain (the classic IsTypeAssignableTo walk).
func assignByTypeChain(t ObjectType, obj Object, to TypeAssigner) (Object, error) {
	tot, ok := to.(ObjectType)
	if !ok {
		return nil, ErrIncompatibleAssign
	}
	for a := t; a != nil; a = a.Type() {
		if a.Equal(tot) {
			return obj, nil
		}
	}
	return nil, ErrIncompatibleAssign
}

// canAssignByType is the default CanAssign for an ObjectType: obj is assignable
// to the type t when obj's type is assignable to t.
func canAssignByType(t ObjectType, obj Object) (bool, error) {
	return IsTypeAssignableTo(obj.Type(), t), nil
}
