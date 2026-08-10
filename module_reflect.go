package gad

// reflectModuleSpec is the module spec shared by the builtin `reflect` namespace
// members and the importable reflect module.
var reflectModuleSpec = NewModuleSpecFromName("reflect")

// ReflectModule returns the `reflect` builtin namespace. It is also the module
// resolved by import("reflect").
func ReflectModule() StdModuleData { return newReflectModule() }

// newReflectModule builds the `reflect` namespace: raw, delegation-free index
// access — the functional, JS `Reflect.get`/`Reflect.set` analog.
//
// Unlike the `target[key]` / `target[key] = value` operators, which delegate to
// a stored Prop's getter/setter (see vm_prop.go), reflect.get returns the value
// stored at key verbatim (a Prop comes back as a Prop, not its getter's result)
// and reflect.set writes value at key verbatim (overwriting — and thus removing
// — any Prop stored there, without invoking its setter).
func newReflectModule() StdModuleData {
	return StdModuleDataFromDict(Dict{
		// gad:doc
		// # reflect module
		//
		// Raw, delegation-free index access (the functional analog of JavaScript
		// `Reflect.get` / `Reflect.set`). Unlike `t[k]` / `t[k] = v`, these do NOT
		// run a stored property's getter/setter.
		//
		// ## Functions
		// get(target indexGetter, key str|int) -> any
		// Returns the value stored at key verbatim. If a Prop is stored there, the
		// Prop itself is returned (its getter is not run).
		"get": &BuiltinFunction{
			FuncName: "get",
			Module:   reflectModuleSpec,
			Value:    reflectGetFunc,
		},
		// gad:doc
		// set(target indexSetter, key str|int, value any)
		// Writes value at key verbatim, overwriting (and thus removing) any Prop
		// stored there without running its setter.
		"set": &BuiltinFunction{
			FuncName: "set",
			Module:   reflectModuleSpec,
			Value:    reflectSetFunc,
		},
	})
}

// reflectGetFunc implements reflect.get(target, key): the raw value at key, with
// no Prop-getter delegation.
func reflectGetFunc(c Call) (Object, error) {
	var (
		target = &Arg{Name: "target"}
		key    = &Arg{Name: "key", TypeAssertion: TypeAssertionFromTypes(TStr, TInt, TUint)}
	)
	if err := c.Args.Destructure(target, key); err != nil {
		return nil, err
	}
	ig, ok := target.Value.(IndexGetter)
	if !ok {
		return nil, ErrNotIndexable.NewError(target.Value.Type().Name())
	}
	return Val(ig.IndexGet(c.VM, key.Value))
}

// reflectSetFunc implements reflect.set(target, key, value): a raw index set,
// overwriting any Prop stored at key without running its setter.
func reflectSetFunc(c Call) (Object, error) {
	var (
		target = &Arg{Name: "target"}
		key    = &Arg{Name: "key", TypeAssertion: TypeAssertionFromTypes(TStr, TInt, TUint)}
		value  = &Arg{Name: "value"}
	)
	if err := c.Args.Destructure(target, key, value); err != nil {
		return nil, err
	}
	is, ok := target.Value.(IndexSetter)
	if !ok {
		return nil, ErrNotIndexAssignable.NewError(target.Value.Type().Name())
	}
	if err := is.IndexSet(c.VM, key.Value, value.Value); err != nil {
		return nil, err
	}
	return Nil, nil
}
