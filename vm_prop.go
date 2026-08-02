package gad

// RawPropContainer is implemented by containers whose IndexGet/IndexSet return
// or store values verbatim — a *Prop held at a key is the value itself, not a
// getter/setter to delegate to. Containers that do NOT implement it (Dict,
// Module, custom IndexGetters, …) delegate index access to a stored *Prop: a
// get runs the prop's getter, a set runs its matching setter.
//
// Implemented by Array (positional storage) and *ClassInstance (which already
// resolves its own properties, so a *Prop it returns is the final value). The
// raw access is still reachable for any container through the `reflect` module
// (reflect.get / reflect.set), which bypasses delegation.
type RawPropContainer interface {
	rawPropContainer()
}

// delegatesProps reports whether an index get/set on target should delegate to a
// stored *Prop. Opt-out containers (Array, *ClassInstance) never delegate.
func delegatesProps(target Object) bool {
	_, raw := target.(RawPropContainer)
	return !raw
}

// indexGetProp resolves value read from container at an index: if the container
// delegates props and value is a *Prop, it runs the prop's getter (via vm.Call,
// a same-VM sub-run) and returns the result; otherwise value is returned as-is.
func (vm *VM) indexGetProp(container, value Object) (Object, error) {
	if p, ok := value.(*Prop); ok && delegatesProps(container) {
		return vm.Call(p, Args{}, nil)
	}
	return value, nil
}

// indexSetProp writes value to container[index]. If the container delegates
// props and already holds a *Prop at index, the prop's setter is invoked with
// value (via vm.Call); otherwise a plain IndexSet is performed. Returns whether
// a setter was invoked (so the caller can skip its own IndexSet).
func (vm *VM) indexSetProp(container IndexSetter, index, value Object) (done bool, err error) {
	if !delegatesProps(container) {
		return false, nil
	}
	ig, ok := container.(IndexGetter)
	if !ok {
		return false, nil
	}
	cur, err := ig.IndexGet(vm, index)
	if err != nil {
		// A missing key (or any read error) means there is no prop to delegate
		// to; fall back to a normal set.
		return false, nil
	}
	if p, ok := cur.(*Prop); ok {
		if _, err = vm.Call(p, Args{Array{value}}, nil); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}
