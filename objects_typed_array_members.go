package gad

// Class-like members of a typed array type (`type T []int { … }`): constructor
// overloads (`new`), fields, properties and methods. They reuse the Class
// machinery — the body members live in an internal *Class (TypedArrayType.members)
// and the constructors in a FuncSpec — with the TypedArray as the receiver.

// typedArrayConstructor dispatches a typed array type's `new(…)` overloads (each
// takes the TypedArrayInitiator `new` as its first parameter), like
// ClassConstructor.
type typedArrayConstructor struct {
	t *TypedArrayType
	f *FuncSpec
}

var _ FuncWrapper = (*typedArrayConstructor)(nil)

func (c *typedArrayConstructor) Type() ObjectType        { return TClassConstructor }
func (c *typedArrayConstructor) IsFalsy() bool           { return c.f.IsFalsy() }
func (c *typedArrayConstructor) Equal(right Object) bool { return right == Object(c) }
func (c *typedArrayConstructor) Name() string            { return "new" }
func (c *typedArrayConstructor) FullName() string        { return c.t.FullName() + "#new" }
func (c *typedArrayConstructor) ToString() string        { return c.String() }
func (c *typedArrayConstructor) String() string          { return string(MustToStr(nil, c)) }
func (c *typedArrayConstructor) FuncSpecName() string {
	return "typed array constructor of " + ReprQuote(c.t.FullName())
}
func (c *typedArrayConstructor) Print(state *PrinterState) error {
	return c.f.PrintFuncWrapper(state, c)
}

// addConstructors registers the `new=` constructor overloads (a func-with-methods
// or an array of them).
func (t *TypedArrayType) addConstructors(vm *VM, v Object) error {
	if t.ctor == nil {
		t.ctor = &typedArrayConstructor{t: t}
		t.ctor.f = NewFuncSpec(t.ctor)
	}
	handlers := Array{v}
	if arr, ok := v.(Array); ok {
		handlers = arr
	}
	for _, h := range handlers {
		if err := SplitCaller(vm, h, func(co CallerObject, types ParamsTypes) error {
			return t.ctor.f.Methods.Add(types, NewCallerMethod(t.ctor, co), false, nil)
		}); err != nil {
			return err
		}
	}
	return nil
}

// TypedArrayInitiator is the `new` first parameter of a typed array constructor
// overload. Calling it builds the instance: `new(*items; field=v, …)` with the
// positional items and named field values. A call whose argument types match
// another constructor overload dispatches to it; one that re-enters a running
// overload (or matches none) builds directly, so construction terminates.
type TypedArrayInitiator struct {
	t         *TypedArrayType
	instance  *TypedArray
	callStack []CallerObject
}

func (in *TypedArrayInitiator) Type() ObjectType        { return TClassInitiator }
func (in *TypedArrayInitiator) IsFalsy() bool           { return false }
func (in *TypedArrayInitiator) Equal(right Object) bool { return Object(in) == right }
func (in *TypedArrayInitiator) Name() string            { return "new" }
func (in *TypedArrayInitiator) ToString() string {
	return "typedArrayInitiator of " + in.t.FullName()
}

// Call runs the matching constructor overload (or the default build) and
// returns the instance.
func (in *TypedArrayInitiator) Call(c Call) (Object, error) {
	if in.t.ctor != nil {
		args := append(Args{Array{in}}, c.Args...)
		if caller, validate := in.t.ctor.f.CallerMethodWithValidationCheckOfArgs(args); caller != nil && !in.running(caller) {
			c.Args = args
			c.SkipValidation = !validate
			in.callStack = append(in.callStack, caller)
			ret, err := DoCall(caller, c)
			in.callStack = in.callStack[:len(in.callStack)-1]
			if err != nil {
				return nil, err
			}
			if in.instance == nil {
				// The overload returned an instance itself instead of calling new(…).
				if ta, ok := ret.(*TypedArray); ok && ta.ArrayType == in.t {
					in.instance = ta
				} else {
					return nil, ErrType.NewErrorf("constructor of %s must build the instance with new(…)", in.t.TypeName)
				}
			}
			return in.instance, nil
		}
	}
	ta, err := in.t.build(c.VM, c.Args.Values(), c.NamedArgs.Dict())
	if err != nil {
		return nil, err
	}
	in.instance = ta
	return ta, nil
}

func (in *TypedArrayInitiator) running(caller CallerObject) bool {
	for _, co := range in.callStack {
		if co == caller {
			return true
		}
	}
	return false
}

// build makes a TypedArray from items (each checked against the element type)
// and field values (validated against the declared fields; the rest take their
// defaults).
func (t *TypedArrayType) build(vm *VM, items Array, fields Dict) (*TypedArray, error) {
	out := make(Array, len(items))
	for i, v := range items {
		iv, err := t.acceptItem(vm, v, false)
		if err != nil {
			return nil, ErrType.NewErrorf("%s item #%d: %v", t.TypeName, i, err)
		}
		out[i] = iv
	}
	ta := &TypedArray{Items: out, ArrayType: t}
	if err := ta.initFields(vm, fields); err != nil {
		return nil, err
	}
	return ta, nil
}

// initFields sets the instance fields: the non-literal defaults (initFields),
// the given values (checked), then the literal/computed defaults.
func (o *TypedArray) initFields(vm *VM, given Dict) (err error) {
	m := o.ArrayType.members
	if m == nil || len(m.fieldsMap) == 0 {
		if len(given) > 0 {
			for name := range given {
				return ErrInvalidIndex.NewErrorf("%s has no field %q", o.ArrayType.TypeName, name)
			}
		}
		return nil
	}
	o.Fields = make(Dict, len(m.fieldsMap))
	if err = applyInitFields(vm, m.initFields, o.Fields); err != nil {
		return
	}
	for name, v := range given {
		fd, ok := m.fieldsMap[name]
		if !ok {
			return ErrInvalidIndex.NewErrorf("%s has no field %q", o.ArrayType.TypeName, name)
		}
		if v, err = acceptFieldValue(vm, fd, v); err != nil {
			return
		}
		o.Fields[name] = v
	}
	for name, fd := range m.fieldsMap {
		if _, ok := o.Fields[name]; ok {
			continue
		}
		switch dv := fd.Value.(type) {
		case nil:
			o.Fields[name] = Nil
		case *ComputedValue:
			if o.Fields[name], err = DoCall(dv.CallerObject, Call{VM: vm}); err != nil {
				return
			}
		default:
			o.Fields[name] = dv
		}
	}
	return nil
}

// member resolves a body member of the type by name: a property, a field or a
// method (for reflection: `T.name.@meta`).
func (t *TypedArrayType) member(name string) Object {
	if t.members == nil {
		return nil
	}
	if p := t.members.propertiesMap[name]; p != nil {
		return p
	}
	if f := t.members.fieldsMap[name]; f != nil {
		return f
	}
	if m := t.members.methodsMap[name]; m != nil {
		return m
	}
	return nil
}

// getMember reads a named member of the instance: a property (its getter called
// with the instance as `this`), a field, or a method bound to the instance.
func (o *TypedArray) getMember(vm *VM, name string) (Object, error) {
	if m := o.ArrayType.members; m != nil {
		if p := m.propertiesMap[name]; p != nil {
			return p.f.Call(Call{VM: vm, Args: Args{Array{o}}})
		}
		if v, ok := o.Fields[name]; ok {
			return v, nil
		}
		if meth := m.methodsMap[name]; meth != nil {
			return &Function{
				FuncName: name,
				ToStringFunc: func() string {
					return ReprQuote("typedArrayMethod of " + o.ArrayType.FullName() + "#" + name)
				},
				Value: func(c Call) (Object, error) {
					c.Args = append([]Array{{o}}, c.Args...)
					return YieldCall(meth, &c), nil
				},
			}, nil
		}
	}
	return nil, ErrInvalidIndex.NewError(name)
}

// setMember writes a named member: a property setter (called with `this` and the
// value) or a declared field (the value checked against its types).
func (o *TypedArray) setMember(vm *VM, name string, value Object) error {
	if m := o.ArrayType.members; m != nil {
		if p := m.propertiesMap[name]; p != nil {
			_, err := p.f.Call(Call{VM: vm, Args: Args{Array{o, value}}})
			return err
		}
		if fd := m.fieldsMap[name]; fd != nil {
			v, err := acceptFieldValue(vm, fd, value)
			if err != nil {
				return err
			}
			if o.Fields == nil {
				o.Fields = Dict{}
			}
			o.Fields[name] = v
			return nil
		}
	}
	return ErrInvalidIndex.NewErrorf("%s has no field or property %q", o.ArrayType.TypeName, name)
}
