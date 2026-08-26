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

// AssignToTypeTransform implements the `obj ::: to` transforming cast. For a dict
// cast to an interface it coerces: fields typed by a class/interface are built
// from their nested dicts and a `**name` rest member gathers the interface's
// unnamed keys (see Interface.coerceDict). For every other combination it behaves
// exactly like AssignToType (a checked cast that returns obj unchanged).
func AssignToTypeTransform(vm *VM, obj, to Object) (Object, error) {
	// `expr ::: bool` converts to a boolean by truthiness — the same result as
	// bool(expr), but without the builtin function call (a direct IsFalsy check).
	// Compared by TypeKey (not pointer identity): the `bool` referenced in code and
	// BuiltinObjects[BuiltinBool] can be distinct *BuiltinObjType instances.
	if bt, ok := to.(*BuiltinObjType); ok && bt.TypeKey() == TBool {
		return Bool(!obj.IsFalsy()), nil
	}
	switch t := to.(type) {
	case *Interface:
		if t.ArrayDepth > 0 {
			return t.coerceArray(vm, obj, t.ArrayDepth)
		}
		if d, ok := asTransformDict(vm, obj); ok {
			return t.coerceDict(vm, d)
		}
	case *Class:
		// `src ::: Class` builds an instance of the target class from the source's
		// members, keeping only the fields the class declares — a conversion between
		// class shapes. An instance already of the class is returned unchanged.
		if _, err := AssignToType(vm, obj, t); err == nil {
			return obj, nil
		}
		if d, ok := asTransformDict(vm, obj); ok {
			return t.coerceFrom(vm, d)
		}
	case *BuiltinObjType:
		// `src ::: T` for a builtin type T (other than bool, handled above) converts
		// by calling T's constructor, so the typed overloads registered with
		// AddMethod apply — `5 ::: str` -> str(5), `"5" ::: int` -> int("5"),
		// `65 ::: char` -> char(65).
		return vm.CallBuiltin(t.BuiltinType(), nil, obj)
	case CallerObject:
		// `src ::: fn` applies a transformer function: it calls fn(src) and returns
		// the result, so any expression can post-process the value inline, e.g.
		// `5 ::: ((v) => v * 10)` -> 50. The transformer always receives src as its
		// single argument, so it is written `(v) => …`. Only a plain callable is a
		// transformer: a type target that also happens to be callable (`any`, a
		// *Class/*BuiltinObjType/*Interface — the latter matched above) is not called
		// with src, so it falls through to the checked-cast behaviour below.
		if _, isType := t.(ObjectType); !isType {
			return vm.Call(t, Args{Array{obj}}, nil)
		}
	}
	return AssignToType(vm, obj, to)
}

// transformCallee reports whether `obj ::: to` is a plain "call to(obj)" transform
// and returns the callable to invoke. Two targets are call-transforms: a builtin
// type other than bool (its constructor converts, honouring AddMethod overloads —
// `5 ::: str` -> str(5)) and a transformer function (`5 ::: ((v) => v*10)`). bool
// (call-free truthiness), any (identity) and the structural coercions (*Interface,
// *Class) are handled in AssignToTypeTransform and are not call-targets. The VM's
// OpAssignTransform handler uses this to invoke the callee through the stack-based
// call path (no argument array allocated), matching a direct call's cost.
func transformCallee(to Object) (CallerObject, bool) {
	switch t := to.(type) {
	case *BuiltinObjType:
		if t.TypeKey() == TBool {
			return nil, false
		}
		return t, true
	case *Interface, *Class:
		return nil, false
	case ObjectType:
		// any (a *Type) and any other type object: identity/checked cast.
		return nil, false
	case CallerObject:
		return t, true
	}
	return nil, false
}

// asTransformDict materialises obj as a Dict for the `:::` transform. Besides a
// plain Dict it accepts any key/value source that can enumerate its members — a
// KeyValueArray (and any other ToDictConverter) or a class instance — so the
// transform works for "any item getter", not only dict literals. Reports false
// when obj has no enumerable members.
func asTransformDict(vm *VM, obj Object) (Dict, bool) {
	switch v := obj.(type) {
	case Dict:
		return v, true
	case *ClassInstance:
		return v.Fields(), true
	case ToDictConverter:
		return v.ToDict(), true
	}
	return nil, false
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
