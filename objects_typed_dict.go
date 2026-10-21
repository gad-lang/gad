package gad

import (
	"sort"
)

// A typed dict type (issue #19): `type NAME dict[K, V]` (declared),
// `type dict[K, V]` (a value, anonymous) or `type NAME dict[K, V]` written where
// a value goes (a value, named). A new object type, nominal, a child of the
// generic instance DictType[K, V] (and so of `dict`): its instances — TypedDict
// — are bound to it, every write checked; its constructors and overloads are
// its own (`met NAME(x T) => …`).

// TTypedDictType is the type of the typed dict types.
var TTypedDictType = NewType("typedDictType", TBase)

// TypedDictType is a typed dict type.
type TypedDictType struct {
	*FuncSpec
	TypeName string
	Module   *ModuleSpec
	Meta     KeyValueArray
	// Key and Value are the types of its generic instance (str and any for
	// `type NAME dict`).
	Key, Value Object
	// generics are its generic methods, `met ~NAME[_, V](…)`
	generics genericMethods
}

// addGenericMethod takes `met NAME[K, V](…)`.
func (t *TypedDictType) addGenericMethod(cf *CompiledFunction) bool {
	if len(cf.TypeParams) != 2 {
		return false
	}
	t.generics = t.generics.with(cf)
	return true
}

var (
	_ ObjectType  = (*TypedDictType)(nil)
	_ IndexGetter = (*TypedDictType)(nil)
)

// NewTypedDictTypeFunc is the `TypedDictType(name, generic; meta=…)` builtin a
// typed dict type lowers to: name a str (nil: anonymous), generic `dict` or a
// DictType[K, V].
func NewTypedDictTypeFunc(c Call) (Object, error) {
	if err := c.Args.CheckLen(2); err != nil {
		return nil, err
	}
	t := &TypedDictType{Key: TStr, Value: TAny}
	if n := c.Args.Get(0); n != Nil {
		t.TypeName = n.ToString()
	}
	switch g := c.Args.Get(1).(type) {
	case *DictOfType:
		t.Key, t.Value = g.Key, g.Value
	default:
		if !sameType(g, TDict) {
			return nil, ErrType.NewErrorf("TypedDictType: a dict type, not %s", g.ToString())
		}
	}
	if m, ok := c.NamedArgs.GetValueOrNil("meta").(KeyValueArray); ok {
		t.Meta = m
	}
	t.FuncSpec = NewFuncSpec(t)
	t.FuncSpec.defaul = &Function{
		FuncName: t.Name(),
		Value:    t.construct,
	}
	return t, nil
}

func (t *TypedDictType) GadObjectType()          {}
func (t *TypedDictType) GetModule() *ModuleSpec  { return t.Module }
func (t *TypedDictType) SetModule(m *ModuleSpec) { t.Module = m }
func (t *TypedDictType) IsFalsy() bool           { return false }

// Type is `dict`: its instances are dicts (`s :: dict`, a param `x dict`), the
// chain of the types a value's type is walked through.
func (t *TypedDictType) Type() ObjectType { return TDict }

func (t *TypedDictType) generic() string {
	if sameType(t.Key, TStr) && sameType(t.Value, TAny) {
		return "dict"
	}
	return "dict[" + typeName(t.Key) + ", " + typeName(t.Value) + "]"
}

func (t *TypedDictType) Name() string {
	if t.TypeName == "" {
		return t.generic()
	}
	return t.TypeName
}

func (t *TypedDictType) FullName() string {
	if t.Module != nil && t.TypeName != "" {
		return t.Module.Name + "." + t.TypeName
	}
	return t.Name()
}

func (t *TypedDictType) FuncSpecName() string { return "typed dict type " + ReprQuote(t.FullName()) }

func (t *TypedDictType) ToString() string {
	if t.TypeName == "" {
		return ReprQuoteTyped("typedDictType", t.generic())
	}
	return ReprQuoteTyped("typedDictType", t.FullName()+" "+t.generic())
}
func (t *TypedDictType) String() string { return t.ToString() }

func (t *TypedDictType) Print(state *PrinterState) error {
	return state.WriteString(t.ToString())
}

func (t *TypedDictType) Equal(right Object) bool { return right == Object(t) }

// IndexGet reflects the type: `@name`, `@key`, `@value`, `@meta`, `@generic`.
func (t *TypedDictType) IndexGet(vm *VM, index Object) (Object, error) {
	switch index.ToString() {
	case "@name":
		return Str(t.TypeName), nil
	case "@key":
		return t.Key, nil
	case "@value":
		return t.Value, nil
	case "@meta":
		return metaObject(t.Meta), nil
	case "@generic":
		return newDictOfType(vm, t.Key, t.Value)
	}
	if v, ok, err := t.FuncSpec.reflectIndex(vm, t.Name(), index); ok || err != nil {
		return v, err
	}
	return nil, ErrInvalidIndex.NewError(index.ToString())
}

// CanAssign is nominal: an instance of this type.
func (t *TypedDictType) CanAssign(obj Object) (bool, error) {
	d, ok := obj.(*TypedDict)
	return ok && d.DictType == t, nil
}

func (t *TypedDictType) AssignTo(_ *VM, obj Object, to TypeAssigner) (Object, error) {
	if ok, _ := t.CanAssign(obj); ok && to == TypeAssigner(t) {
		return obj, nil
	}
	return nil, ErrIncompatibleAssign.NewErrorf("%s is not assignable to %s", ReprQuote(obj.Type().Name()), ReprQuote(t.Name()))
}

// check checks a key and a value against the type.
func (t *TypedDictType) check(vm *VM, k string, v Object) error {
	if !keyOK(t.Key, k) {
		return ErrType.NewErrorf("%s: key %q is not of %s", t.Name(), k, ReprQuote(typeName(t.Key)))
	}
	if t.Value != TAny {
		if _, err := AssignToType(vm, v, t.Value); err != nil {
			return ErrType.NewErrorf("%s: value %q: %s is not of %s", t.Name(), k, ReprQuote(v.Type().Name()), ReprQuote(typeName(t.Value)))
		}
	}
	return nil
}

// construct makes an instance: the entries of the call (DictOfType.Call's
// forms), each checked.
func (t *TypedDictType) construct(c Call) (Object, error) {
	c, err := callGenerics(c, t.generics, BuiltinDictType, Array{t.Key, t.Value})
	if err != nil {
		return nil, err
	}
	d, err := buildDictEntries(c, t.Key, func(k string, v Object) error { return t.check(c.VM, k, v) })
	if err != nil {
		return nil, err
	}
	return &TypedDict{Value: d, DictType: t}, nil
}

// transform is `obj ::> T`: obj converted (as into its generic) into an
// instance.
func (t *TypedDictType) transform(vm *VM, obj Object) (Object, error) {
	g, err := newDictOfType(vm, t.Key, t.Value)
	if err != nil {
		return nil, err
	}
	var d Dict
	if dt, ok := g.(*DictOfType); ok {
		out, err := dt.transform(vm, obj)
		if err != nil {
			return nil, err
		}
		d = out.(Dict)
	} else if items, ok := dictItems(obj); ok {
		d = items.Copy().(Dict)
	} else {
		return nil, ErrIncompatibleAssign.NewErrorf("%s is not assignable to %s", ReprQuote(obj.Type().Name()), ReprQuote(t.Name()))
	}
	return &TypedDict{Value: d, DictType: t}, nil
}

// TypedDict is an instance of a typed dict type: a dict bound to it, every
// write checked.
type TypedDict struct {
	Value    Dict
	DictType *TypedDictType
}

var (
	_ Object          = (*TypedDict)(nil)
	_ IndexGetSetter  = (*TypedDict)(nil)
	_ LengthGetter    = (*TypedDict)(nil)
	_ ItemsGetter     = (*TypedDict)(nil)
	_ ToDictConverter = (*TypedDict)(nil)
)

func (o *TypedDict) Type() ObjectType { return o.DictType }
func (o *TypedDict) IsFalsy() bool    { return len(o.Value) == 0 }
func (o *TypedDict) Length() int      { return len(o.Value) }
func (o *TypedDict) ToDict() Dict     { return o.Value.Copy().(Dict) }

func (o *TypedDict) ToString() string { return o.DictType.Name() + o.Value.ToString() }

// Print writes the typed dict: its text `sm{a: 1}`; its literal, the
// constructor's call `sm({a: 1})` (gad code); its repr `‹sm: {a: ‹int: 1›}›`.
func (o *TypedDict) Print(state *PrinterState) error {
	if state.IsRepr {
		return o.Value.PrintObject(state, o)
	}
	if state.IsLiteral() {
		name := o.DictType.TypeName
		if name == "" {
			name = "(type " + o.DictType.Name() + ")"
		}
		state.WriteString(name + "(")
		if err := o.Value.Print(state); err != nil {
			return err
		}
		return state.WriteString(")")
	}
	if err := state.WriteString(o.DictType.Name()); err != nil {
		return err
	}
	return o.Value.Print(state)
}

func (o *TypedDict) Equal(right Object) bool {
	if r, ok := right.(*TypedDict); ok {
		return r.DictType == o.DictType && o.Value.Equal(r.Value)
	}
	return false
}

func (o *TypedDict) Copy() Object {
	return &TypedDict{Value: o.Value.Copy().(Dict), DictType: o.DictType}
}

func (o *TypedDict) IndexGet(vm *VM, index Object) (Object, error) {
	return o.Value.IndexGet(vm, index)
}

// IndexSet sets an entry, checked against the type.
func (o *TypedDict) IndexSet(vm *VM, index, value Object) error {
	k, ok := keyOf(o.DictType.Key, index, false)
	if !ok {
		return ErrType.NewErrorf("%s: a key %s is not of %s", o.DictType.Name(), ReprQuote(index.ToString()), ReprQuote(typeName(o.DictType.Key)))
	}
	if err := o.DictType.check(vm, k, value); err != nil {
		return err
	}
	o.Value[k] = value
	return nil
}

func (o *TypedDict) IndexDelete(vm *VM, index Object) error { return o.Value.IndexDelete(vm, index) }

func (o *TypedDict) Items(vm *VM, cb ItemsGetterCallback) error { return o.Value.Items(vm, cb) }

func (o *TypedDict) Keys() Array {
	keys := make([]string, 0, len(o.Value))
	for k := range o.Value {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make(Array, len(keys))
	for i, k := range keys {
		out[i] = Str(k)
	}
	return out
}

func (o *TypedDict) Values() Array { return o.Value.Values() }

// BinOpIn is `key in d`.
func (o *TypedDict) BinOpIn(vm *VM, left Object) (Object, error) { return o.Value.BinOpIn(vm, left) }

// Iterate iterates the entries, as a dict does.
func (o *TypedDict) Iterate(vm *VM, na *NamedArgs) Iterator { return o.Value.Iterate(vm, na) }

// DeepCopy copies the entries, deeply.
func (o *TypedDict) DeepCopy(vm *VM) (Object, error) {
	c, err := o.Value.DeepCopy(vm)
	if err != nil {
		return nil, err
	}
	return &TypedDict{Value: c.(Dict), DictType: o.DictType}, nil
}
