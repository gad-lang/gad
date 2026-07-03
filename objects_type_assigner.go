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

// vmCanAssigner is an optional refinement of TypeAssigner for structural types
// whose assignability check needs the VM (e.g. to resolve a callable's
// signatures). ParamType.Accept prefers CanAssignVM over CanAssign when
// available so the VM is threaded through.
type vmCanAssigner interface {
	CanAssignVM(vm *VM, obj Object) (bool, error)
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

// TypeAssignerName returns a display name for a type assigner (an ObjectType's
// Name, a meti/interface's Name, else its ToString).
func TypeAssignerName(t TypeAssigner) string {
	if n, ok := t.(interface{ Name() string }); ok {
		return n.Name()
	}
	return t.ToString()
}

// TypeAssignerFullName returns a fully-qualified display name for a type
// assigner, falling back to TypeAssignerName.
func TypeAssignerFullName(t TypeAssigner) string {
	if n, ok := t.(interface{ FullName() string }); ok {
		return n.FullName()
	}
	return TypeAssignerName(t)
}

// AssignToType implements the `obj :: to` assign-to-type operator: it returns
// obj when obj is assignable to the type value `to`, otherwise a type error. The
// target may be an ObjectType (plain type assignability) or a structural
// TypeAssigner such as a meti/interface (checked by value, like a parameter
// type). It is the runtime behind OpAssign and chains left-to-right for
// `obj::T1::T2`.
func AssignToType(vm *VM, obj, to Object) (Object, error) {
	if to == TAny {
		return obj, nil
	}
	switch t := to.(type) {
	case vmCanAssigner:
		// structural types (meti/interface) need the VM to resolve signatures.
		if ok, err := t.CanAssignVM(vm, obj); err != nil || ok {
			if err != nil {
				return nil, err
			}
			return obj, nil
		}
	case TypeAssigner:
		// ObjectType (incl. *Class parent-walk) and *Interface.
		if ok, err := t.CanAssign(obj); err != nil || ok {
			if err != nil {
				return nil, err
			}
			return obj, nil
		}
	default:
		return nil, ErrType.NewErrorf("%s is not a type", ReprQuote(to.Type().Name()))
	}
	return nil, ErrIncompatibleAssign.NewErrorf("%s is not assignable to %s",
		ReprQuote(obj.Type().Name()), ReprQuote(TypeAssignerName(to.(TypeAssigner))))
}

// assignerAcceptsType reports whether an arg of type t is accepted by the type
// assigner a. For an ObjectType assigner it is plain type assignability; a
// structural assigner (meti/interface) cannot be decided from a type alone in
// the dispatch tree, so it is permissive here — the value-based check happens in
// ParamType.Accept (TypeAssigner.CanAssign).
func assignerAcceptsType(a TypeAssigner, t ObjectType) bool {
	if aot, ok := a.(ObjectType); ok {
		return IsAssignableTo(t, aot)
	}
	return true
}
