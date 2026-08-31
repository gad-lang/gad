package gad

// TMetaType is the base type of every `type<X>` meta-type value.
var TMetaType = NewType("metaType", TBase)

// MetaType is the runtime type of a `type<X>` parameter: it matches a TYPE VALUE
// (an ObjectType such as a class) rather than an instance of it. As a dispatch key
// it stays DISTINCT from X-used-as-an-instance-constraint, so
//
//	func d {
//	    (t X)        => …   // an instance of X
//	    (t type<X>)  => …   // the type value X itself
//	}
//
// dispatch an instance of X to the first overload and the type value X to the
// second. Passing a type value as a call argument keys it as a MetaType (see
// Args.Types), which is why a bare class value no longer matches a plain `(t X)`
// parameter (Option A: a type value is not an instance).
//
// MetaType is a comparable value type — Target is an interface over a pointer — so
// two MetaType{X} with the same target are equal as Go map keys, letting the
// argument side (Args.Types) and the parameter side (a `type<X>` parameter) agree
// on the dispatch key.
//
// The compiled constant for a `type<X>` parameter carries TargetSym (X resolved
// from the symbol per VM in resolve); a resolved MetaType — the one used as a
// dispatch key or for validation — carries Target.
type MetaType struct {
	// Target is the resolved target type X (nil means the bare `type`: any type
	// value — reserved, not yet produced by the parser).
	Target ObjectType
	// TargetSym is set on the compiled constant; resolve turns it into Target.
	TargetSym *SymbolInfo
}

var (
	_ Object       = MetaType{}
	_ ObjectType   = MetaType{}
	_ TypeAssigner = MetaType{}
)

func (MetaType) GadObjectType() {}

func (m MetaType) Type() ObjectType { return TMetaType }

func (m MetaType) Name() string {
	switch {
	case m.Target != nil:
		return "type<" + m.Target.Name() + ">"
	case m.TargetSym != nil:
		return "type<" + m.TargetSym.Name + ">"
	default:
		return "type"
	}
}

func (m MetaType) FullName() string {
	if m.Target != nil {
		return "type<" + m.Target.FullName() + ">"
	}
	return m.Name()
}

func (m MetaType) ToString() string { return m.Name() }
func (m MetaType) String() string   { return m.Name() }
func (m MetaType) IsFalsy() bool    { return false }

func (m MetaType) Equal(right Object) bool {
	r, ok := right.(MetaType)
	if !ok {
		return false
	}
	if m.Target != nil && r.Target != nil {
		return m.Target.Equal(r.Target)
	}
	return m.Target == r.Target && m.TargetSym == r.TargetSym
}

// CanAssign reports whether obj is a TYPE VALUE assignable to the target: obj must
// itself be an ObjectType, equal to (or a subtype of) Target. A nil Target (the
// bare `type`) accepts any type value.
func (m MetaType) CanAssign(obj Object) (bool, error) {
	ot, ok := obj.(ObjectType)
	if !ok {
		return false, nil
	}
	if m.Target == nil {
		return true, nil
	}
	return IsTypeAssignableTo(ot, m.Target), nil
}

func (m MetaType) AssignTo(_ *VM, obj Object, to TypeAssigner) (Object, error) {
	return assignByTypeChain(m, obj, to)
}

// Call — a meta type is a constraint, not a constructor: to build or use the
// underlying type, use the type value itself.
func (m MetaType) Call(Call) (Object, error) {
	return nil, ErrNotCallable.NewError(m.Name())
}

// resolve returns the dispatch-key / validation form of the meta type: a MetaType
// whose Target is resolved from TargetSym against vm. The compiled constant
// carries only the symbol; a MetaType that already has a Target is returned as is.
func (m MetaType) resolve(vm *VM) (MetaType, error) {
	if m.Target != nil || m.TargetSym == nil {
		return m, nil
	}
	v, err := vm.GetSymbolValue(m.TargetSym)
	if err != nil {
		return m, err
	}
	ot, ok := v.(ObjectType)
	if !ok {
		return m, ErrType.NewError("type<" + m.TargetSym.Name + ">: not a type")
	}
	return MetaType{Target: ot}, nil
}
