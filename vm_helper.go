package gad

func ToStr(vm *VM, o Object) (_ Str, err error) {
	var v Object
	if v, err = Val(vm.Builtins.Call(BuiltinStr, Call{VM: vm, Args: Args{Array{o}}})); err != nil {
		return
	}
	return v.(Str), nil
}

func ToRawStr(vm *VM, o Object) (_ RawStr, err error) {
	var v Object
	if v, err = Val(vm.Builtins.Call(BuiltinRawStr, Call{VM: vm, Args: Args{Array{o}}})); err != nil {
		return
	}
	return v.(RawStr), nil
}

func ToRepr(vm *VM, o Object) (_ Str, err error) {
	var v Object
	if v, err = Val(vm.Builtins.Call(BuiltinRepr, Call{VM: vm, Args: Args{Array{o}}})); err != nil {
		return
	}
	return v.(Str), nil
}

func DeepCopy(vm *VM, o Object) (Object, error) {
	return Val(vm.Builtins.Call(BuiltinDeepCopy, Call{VM: vm, Args: Args{Array{o}}}))
}

func Copy(o Object) Object {
	if cp, _ := o.(Copier); cp != nil {
		return cp.Copy()
	}
	return o
}

func KeysOf(vm *VM, o Iterabler) (keys Array, err error) {
	if kg, _ := o.(KeysGetter); kg != nil {
		return kg.Keys(), nil
	}
	it := o.Iterate(vm)
	if itl, _ := it.(LengthIterator); itl != nil {
		l := itl.Length()
		keys = make(Array, l)
		for i := 0; i < l && itl.Next(); i++ {
			keys[i] = itl.Key()
		}
	} else {
		for it.Next() {
			keys = append(keys, it.Key())
		}
	}
	return
}

func ValuesOf(vm *VM, o Iterabler) (values Array, err error) {
	var ok bool
	if values, _ = o.(Array); ok {
		return values, nil
	}

	if kg, _ := o.(ValuesGetter); kg != nil {
		return kg.Values(), nil
	}
	it := o.Iterate(vm)
	if itl, _ := it.(LengthIterator); itl != nil {
		l := itl.Length()
		values = make(Array, l)
		for i := 0; i < l && itl.Next(); i++ {
			if values[i], err = itl.Value(); err != nil {
				return
			}
		}
	} else {
		var v Object
		for it.Next() {
			if v, err = it.Value(); err != nil {
				return
			}
			values = append(values, v)
		}
	}
	return
}

func DoCall(co CallerObject, c Call) (ret Object, err error) {
	var yc *yieldCall

	for {
		if ret, err = co.Call(c); err == nil {
			if yc, _ = ret.(*yieldCall); yc != nil {
				co, c = yc.CallerObject, *yc.c
				continue
			}
		}
		return
	}
}

func Val(v Object, e error) (ret Object, err error) {
	if e != nil {
		return nil, e
	}

	ret = v

	var yc *yieldCall

	for {
		if yc, _ = ret.(*yieldCall); yc != nil {
			if ret, err = yc.CallerObject.Call(*yc.c); err == nil {
				continue
			}
		}
		return
	}
}
