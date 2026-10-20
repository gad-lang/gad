package gad

// The methods of a generic type (issue #23): `met DictType[K, V](arr []V) =>
// …` is a method of every instance of DictType (`DictType[str, int]`) and of
// every child of one (a typed dict type), its type parameters bound to the
// instance's types; `met ~scores[_, V](arr []V) => …` one of the child scores
// alone, tried before the generic's. A method is tried when the call fits it
// (its positional arguments, their types); what it gives is built into the
// instance as any argument of the constructor is.

// genericMethods are the generic methods of a type, the last added tried
// first.
type genericMethods []*CompiledFunction

// genericMethodAdder is a type that takes generic methods: arity its number of
// type parameters (DictType's two).
type genericMethodAdder interface {
	addGenericMethod(cf *CompiledFunction) bool
}

// with is the list with cf: a new one (a builtin type's copies share theirs).
func (g genericMethods) with(cf *CompiledFunction) genericMethods {
	out := make(genericMethods, len(g)+1)
	copy(out, g)
	out[len(g)] = cf
	return out
}

// call calls the last method added that fits the call, its type parameters
// bound to types; it reports false when none fits.
func (g genericMethods) call(c Call, types Array) (Object, bool, error) {
	if len(g) == 0 || c.VM == nil || len(c.NamedArgs.Join()) > 0 {
		return nil, false, nil
	}
	n := c.Args.Length()
	for i := len(g) - 1; i >= 0; i-- {
		cf := g[i]
		if len(cf.TypeParams) > len(types) {
			continue
		}
		if cf.Params.Variadic() {
			if n < cf.Params.PosLen() {
				continue
			}
		} else if n != cf.Params.Len() {
			continue
		}
		b, err := cf.BindTypes(c.VM, types[:len(cf.TypeParams)])
		if err != nil {
			continue
		}
		bf := b.(*CompiledFunction)
		if bf.ValidateParamTypes(c.VM, c.Args) != nil {
			continue
		}
		v, err := c.VM.Call(bf, c.Args, nil)
		return v, true, err
	}
	return nil, false, nil
}

// genericCallee is the generic function o is (a compiled function, or a
// function with one method), nil when it is not one.
func genericCallee(o Object) *CompiledFunction {
	if f, ok := o.(*Func); ok {
		return f.genericFunc()
	}
	return genericOf(o)
}

// genericsOf are the generic methods of the builtin generic type bt, in the
// VM of the call.
func genericsOf(vm *VM, bt BuiltinType) genericMethods {
	if vm == nil {
		return nil
	}
	if t, ok := vm.Builtins.Get(bt).(*BuiltinObjType); ok {
		return t.generics
	}
	return nil
}

// callGenerics calls the generic methods of own, then of the generic bt, with
// types: the arguments of the constructor (what fits gives the instance's
// entries), or c's, when none fits.
func callGenerics(c Call, own genericMethods, bt BuiltinType, types Array) (Call, error) {
	for _, g := range [2]genericMethods{own, genericsOf(c.VM, bt)} {
		v, ok, err := g.call(c, types)
		if err != nil {
			return c, err
		}
		if ok {
			return Call{VM: c.VM, Args: Args{Array{v}}}, nil
		}
	}
	return c, nil
}
