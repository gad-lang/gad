package gad

// StdModuleData is the standard ModuleData implementation. It keeps mutable
// variables and read-only constants apart: reads see a merged view of both,
// writes land in Vars, and assigning to a name that is a constant is rejected.
//
// Populate it with Set (a variable) and SetConst (a constant); the compiler
// routes `export const NAME` through SetConst. A constant's binding cannot be
// reassigned through the module (`m.NAME = x` errors), though the value it refers
// to may still be mutable.
type StdModuleData struct {
	// Vars holds the module's mutable members.
	Vars Dict
	// Consts holds the module's read-only members. A key never appears in both.
	Consts Dict

	Funcs Dict
}

// StdModuleData satisfies ModuleData by value (its methods have value
// receivers), so a Go module may assign either `StdModuleData{…}` or a pointer.
var (
	_ ModuleData = StdModuleData{}
	_ ModuleData = (*StdModuleData)(nil)
)

// NewStdModuleData returns an empty StdModuleData ready to populate.
func NewStdModuleData() *StdModuleData {
	return &StdModuleData{Vars: Dict{}, Consts: Dict{}, Funcs: Dict{}}
}

// merged returns a combined view of the variables, functions and constants. The
// three buckets hold disjoint keys, so the merge order is irrelevant.
func (o StdModuleData) merged() Dict {
	d := make(Dict, len(o.Vars)+len(o.Funcs)+len(o.Consts))
	for k, v := range o.Vars {
		d[k] = v
	}
	for k, v := range o.Funcs {
		d[k] = v
	}
	for k, v := range o.Consts {
		d[k] = v
	}
	return d
}

// isConst reports whether key names a constant.
func (o StdModuleData) isConst(key string) bool {
	_, ok := o.Consts[key]
	return ok
}

// isModuleFunc reports whether a value is a function (routed to Funcs). Types,
// classes and other callables are not — they are ordinary members.
func isModuleFunc(v Object) bool {
	switch v.(type) {
	case *CompiledFunction, *Function, *BuiltinFunction:
		return true
	}
	return false
}

// Type implements Object.
func (o StdModuleData) Type() ObjectType { return o.merged().Type() }

// ToString implements Object.
func (o StdModuleData) ToString() string { return o.merged().ToString() }

// Equal implements Object.
func (o StdModuleData) Equal(right Object) bool { return o.merged().Equal(right) }

// IsFalsy implements Object: the data is falsy when it holds nothing.
func (o StdModuleData) IsFalsy() bool {
	return len(o.Vars) == 0 && len(o.Funcs) == 0 && len(o.Consts) == 0
}

// Length implements LengthGetter.
func (o StdModuleData) Length() int { return len(o.Vars) + len(o.Funcs) + len(o.Consts) }

// IndexGet implements IndexGetter: constants shadow variables (they never
// coexist), then variables are consulted.
func (o StdModuleData) IndexGet(vm *VM, index Object) (Object, error) {
	key := index.ToString()
	switch key {
	case "@vars":
		return o.Vars, nil
	case "@consts":
		return o.Consts, nil
	case "@funcs":
		return o.Funcs, nil
	default:
		if v, ok := o.Consts[key]; ok {
			return v, nil
		}
		if v, ok := o.Funcs[key]; ok {
			return v, nil
		}
		return o.Vars.IndexGet(vm, index)
	}
}

// isFunc reports whether key names a function.
func (o StdModuleData) isFunc(key string) bool {
	_, ok := o.Funcs[key]
	return ok
}

// IndexSet implements IndexSetter: `module.name = x` writes only to a variable.
// A constant or a function cannot be reassigned this way (mutate them through
// `module.@consts` / `module.@funcs`, which expose the live dicts).
func (o StdModuleData) IndexSet(vm *VM, index, value Object) error {
	key := index.ToString()
	switch {
	case o.isConst(key):
		return ErrNotIndexAssignable.NewError("cannot assign to constant " + key + " (use .@consts)")
	case o.isFunc(key):
		return ErrNotIndexAssignable.NewError("cannot assign to function " + key + " (use .@funcs)")
	}
	if o.Vars == nil {
		o.Vars = Dict{}
	}
	return o.Vars.IndexSet(vm, index, value)
}

// Set implements StringIndexSetter: it declares (or updates) a member, routing
// functions to Funcs and everything else to Vars (the three buckets stay
// disjoint). Use SetConst to declare a constant.
func (o StdModuleData) Set(key string, value Object) {
	delete(o.Consts, key)
	if isModuleFunc(value) {
		if o.Funcs == nil {
			o.Funcs = Dict{}
		}
		delete(o.Vars, key)
		o.Funcs[key] = value
		return
	}
	if o.Vars == nil {
		o.Vars = Dict{}
	}
	delete(o.Funcs, key)
	o.Vars[key] = value
}

// SetConst declares (or updates) a read-only constant, removing any variable or
// function of the same name so the three buckets stay disjoint.
func (o StdModuleData) SetConst(key string, value Object) {
	if o.Consts == nil {
		o.Consts = Dict{}
	}
	delete(o.Vars, key)
	delete(o.Funcs, key)
	o.Consts[key] = value
}

// ToDict implements ToDictConverter: the merged view of all members.
func (o StdModuleData) ToDict() Dict { return o.merged() }

// Keys implements KeysGetter.
func (o StdModuleData) Keys() Array { return o.merged().Keys() }

// Values implements ValuesGetter.
func (o StdModuleData) Values() Array { return o.merged().Values() }

// Items implements ItemsGetter.
func (o StdModuleData) Items(vm *VM, cb ItemsGetterCallback) error {
	return o.merged().Items(vm, cb)
}

// Iterate implements Iterabler.
func (o StdModuleData) Iterate(vm *VM, na *NamedArgs) Iterator {
	return o.merged().Iterate(vm, na)
}
