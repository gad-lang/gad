package gad

func Callable(o Object) (ok bool) {
	if _, ok = o.(CallerObject); ok {
		if cc, _ := o.(CanCallerObject); cc != nil {
			ok = cc.CanCall()
		}
	}
	return
}

func Writeable(o Object) (ok bool) {
	_, ok = o.(Writer)
	return
}

func Readable(o Object) (ok bool) {
	_, ok = o.(Reader)
	return
}

func IsIterator(obj Object) bool {
	switch obj.(type) {
	case Iterator:
		return true
	}
	return false
}

func Iterable(vm *VM, obj Object) bool {
	ret, err := Val(vm.Builtins.Call(BuiltinIsIterable, Call{VM: vm, Args: Args{Array{obj}}}))
	if err != nil {
		return false
	}
	return ret == True
}

func Filterable(obj Object) bool {
	if it, _ := obj.(Filterabler); it != nil {
		if cit, _ := obj.(CanFilterabler); cit != nil {
			return cit.CanFilter()
		}
		return true
	}
	return false
}

func Mapable(obj Object) bool {
	if it, _ := obj.(Mapabler); it != nil {
		if cit, _ := obj.(CanMapeabler); cit != nil {
			return cit.CanMap()
		}
		return true
	}
	return false
}

func Reducable(obj Object) bool {
	if it, _ := obj.(Reducer); it != nil {
		if cit, _ := obj.(CanReducer); cit != nil {
			return cit.CanReduce()
		}
		return true
	}
	return false
}

func IsType(obj Object) (ok bool) {
	_, ok = obj.(ObjectType)
	return
}

func IsObjector(obj Object) (ok bool) {
	_, ok = obj.(Objector)
	return
}

func IsIndexDeleter(obj Object) (ok bool) {
	_, ok = obj.(IndexDeleter)
	return
}

func IsIndexSetter(obj Object) (ok bool) {
	_, ok = obj.(IndexSetter)
	return
}

func IsIndexGetter(obj Object) (ok bool) {
	_, ok = obj.(IndexGetter)
	return
}

func IsTypeAssignableTo(a, b ObjectType) bool {
	for a != nil {
		if a == b {
			return true
		}
		a = a.Type()
	}
	return false
}

func ToIterationDoner(obj any) IterationDoner {
	if ite, _ := obj.(IterationDoner); ite != nil {
		if cite, _ := obj.(CanIterationDoner); cite != nil {
			if !cite.CanIterationDone() {
				return nil
			}
		}
		return ite
	}
	return nil
}
