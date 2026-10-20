package gad

// A typed key-value array type (issue #20): `type NAME keyValueArray[K, V]`
// (declared), `type keyValueArray[K, V]` (a value, anonymous), `type NAME
// keyValueArray[…]` where a value goes (a value, named). Nominal, a child of the
// generic KeyValueArrayType[K, V] (and so of `keyValueArray`): its instances —
// TypedKeyValueArray — are bound to it, every pair added checked; its
// constructors and overloads are its own.

// TTypedKeyValueArrayType is the type of the typed key-value array types.
var TTypedKeyValueArrayType = NewType("typedKeyValueArrayType", TBase)

// TypedKeyValueArrayType is a typed key-value array type.
type TypedKeyValueArrayType struct {
	*FuncSpec
	TypeName   string
	Module     *ModuleSpec
	Meta       KeyValueArray
	Key, Value Object
}

var (
	_ ObjectType  = (*TypedKeyValueArrayType)(nil)
	_ IndexGetter = (*TypedKeyValueArrayType)(nil)
)

// NewTypedKVATypeFunc is the `TypedKeyValueArrayType(name, generic; meta=…)`
// builtin a typed key-value array type lowers to.
func NewTypedKVATypeFunc(c Call) (Object, error) {
	if err := c.Args.CheckLen(2); err != nil {
		return nil, err
	}
	t := &TypedKeyValueArrayType{Key: TAny, Value: TAny}
	if n := c.Args.Get(0); n != Nil {
		t.TypeName = n.ToString()
	}
	switch g := c.Args.Get(1).(type) {
	case *KVAOfType:
		t.Key, t.Value = g.Key, g.Value
	default:
		if !sameType(g, TKeyValueArray) {
			return nil, ErrType.NewErrorf("TypedKeyValueArrayType: a key-value array type, not %s", g.ToString())
		}
	}
	if m, ok := c.NamedArgs.GetValueOrNil("meta").(KeyValueArray); ok {
		t.Meta = m
	}
	t.FuncSpec = NewFuncSpec(t)
	t.FuncSpec.defaul = &Function{FuncName: t.Name(), Value: t.construct}
	return t, nil
}

func (t *TypedKeyValueArrayType) GadObjectType()          {}
func (t *TypedKeyValueArrayType) GetModule() *ModuleSpec  { return t.Module }
func (t *TypedKeyValueArrayType) SetModule(m *ModuleSpec) { t.Module = m }
func (t *TypedKeyValueArrayType) IsFalsy() bool           { return false }

// Type is `keyValueArray`: its instances are key-value arrays.
func (t *TypedKeyValueArrayType) Type() ObjectType { return TKeyValueArray }

func (t *TypedKeyValueArrayType) generic() string {
	if sameType(t.Key, TAny) && sameType(t.Value, TAny) {
		return "keyValueArray"
	}
	return "keyValueArray[" + typeName(t.Key) + ", " + typeName(t.Value) + "]"
}

func (t *TypedKeyValueArrayType) Name() string {
	if t.TypeName == "" {
		return t.generic()
	}
	return t.TypeName
}

func (t *TypedKeyValueArrayType) FullName() string {
	if t.Module != nil && t.TypeName != "" {
		return t.Module.Name + "." + t.TypeName
	}
	return t.Name()
}

func (t *TypedKeyValueArrayType) FuncSpecName() string {
	return "typed key-value array type " + ReprQuote(t.FullName())
}

func (t *TypedKeyValueArrayType) ToString() string {
	if t.TypeName == "" {
		return ReprQuoteTyped("typedKeyValueArrayType", t.generic())
	}
	return ReprQuoteTyped("typedKeyValueArrayType", t.FullName()+" "+t.generic())
}
func (t *TypedKeyValueArrayType) String() string { return t.ToString() }
func (t *TypedKeyValueArrayType) Print(state *PrinterState) error {
	return state.WriteString(t.ToString())
}
func (t *TypedKeyValueArrayType) Equal(right Object) bool { return right == Object(t) }

func (t *TypedKeyValueArrayType) IndexGet(vm *VM, index Object) (Object, error) {
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
		return newKVAOfType(vm, t.Key, t.Value)
	}
	if v, ok, err := t.FuncSpec.reflectIndex(vm, t.Name(), index); ok || err != nil {
		return v, err
	}
	return nil, ErrInvalidIndex.NewError(index.ToString())
}

func (t *TypedKeyValueArrayType) CanAssign(obj Object) (bool, error) {
	a, ok := obj.(*TypedKeyValueArray)
	return ok && a.ArrayType == t, nil
}

func (t *TypedKeyValueArrayType) AssignTo(_ *VM, obj Object, to TypeAssigner) (Object, error) {
	if ok, _ := t.CanAssign(obj); ok && to == TypeAssigner(t) {
		return obj, nil
	}
	return nil, ErrIncompatibleAssign.NewErrorf("%s is not assignable to %s", ReprQuote(obj.Type().Name()), ReprQuote(t.Name()))
}

func (t *TypedKeyValueArrayType) construct(c Call) (Object, error) {
	pairs, err := buildKVAPairs(c, t.Name(), t.Key, t.Value)
	if err != nil {
		return nil, err
	}
	return &TypedKeyValueArray{Value: pairs, ArrayType: t}, nil
}

// checked is pairs as an instance: each checked.
func (t *TypedKeyValueArrayType) checked(vm *VM, pairs KeyValueArray) (*TypedKeyValueArray, error) {
	for i, kv := range pairs {
		if why := pairMismatch(vm, t.Key, t.Value, kv, i); why != "" {
			return nil, ErrType.NewErrorf("%s: %s", t.Name(), why)
		}
	}
	return &TypedKeyValueArray{Value: pairs, ArrayType: t}, nil
}

func (t *TypedKeyValueArrayType) transform(vm *VM, obj Object) (Object, error) {
	g, err := newKVAOfType(vm, t.Key, t.Value)
	if err != nil {
		return nil, err
	}
	var pairs KeyValueArray
	if gt, ok := g.(*KVAOfType); ok {
		out, err := gt.transform(vm, obj)
		if err != nil {
			return nil, err
		}
		pairs = out.(KeyValueArray)
	} else if p, ok := kvaPairs(obj); ok {
		pairs = p.Copy().(KeyValueArray)
	} else {
		return nil, ErrIncompatibleAssign.NewErrorf("%s is not assignable to %s", ReprQuote(obj.Type().Name()), ReprQuote(t.Name()))
	}
	return &TypedKeyValueArray{Value: pairs, ArrayType: t}, nil
}

// TypedKeyValueArray is an instance of a typed key-value array type: the pairs
// bound to it, each pair added (`+`, `++`) checked.
type TypedKeyValueArray struct {
	Value     KeyValueArray
	ArrayType *TypedKeyValueArrayType
}

var (
	_ Object          = (*TypedKeyValueArray)(nil)
	_ LengthGetter    = (*TypedKeyValueArray)(nil)
	_ ItemsGetter     = (*TypedKeyValueArray)(nil)
	_ ToDictConverter = (*TypedKeyValueArray)(nil)
)

func (o *TypedKeyValueArray) Type() ObjectType { return o.ArrayType }
func (o *TypedKeyValueArray) IsFalsy() bool    { return len(o.Value) == 0 }
func (o *TypedKeyValueArray) Length() int      { return len(o.Value) }
func (o *TypedKeyValueArray) ToDict() Dict     { return o.Value.ToDict() }
func (o *TypedKeyValueArray) ToString() string { return o.ArrayType.Name() + o.Value.ToString() }

func (o *TypedKeyValueArray) Print(state *PrinterState) error {
	if err := state.WriteString(o.ArrayType.Name()); err != nil {
		return err
	}
	return o.Value.Print(state)
}

func (o *TypedKeyValueArray) Equal(right Object) bool {
	r, ok := right.(*TypedKeyValueArray)
	return ok && r.ArrayType == o.ArrayType && o.Value.Equal(r.Value)
}

func (o *TypedKeyValueArray) Copy() Object {
	return &TypedKeyValueArray{Value: o.Value.Copy().(KeyValueArray), ArrayType: o.ArrayType}
}

func (o *TypedKeyValueArray) DeepCopy(vm *VM) (Object, error) {
	c, err := o.Value.DeepCopy(vm)
	if err != nil {
		return nil, err
	}
	return &TypedKeyValueArray{Value: c.(KeyValueArray), ArrayType: o.ArrayType}, nil
}

func (o *TypedKeyValueArray) IndexGet(vm *VM, index Object) (Object, error) {
	return o.Value.IndexGet(vm, index)
}
func (o *TypedKeyValueArray) Iterate(vm *VM, na *NamedArgs) Iterator { return o.Value.Iterate(vm, na) }
func (o *TypedKeyValueArray) Items(vm *VM, cb ItemsGetterCallback) error {
	return o.Value.Items(vm, cb)
}
func (o *TypedKeyValueArray) Keys() Array   { return o.Value.Keys() }
func (o *TypedKeyValueArray) Values() Array { return o.Value.Values() }
func (o *TypedKeyValueArray) BinOpIn(vm *VM, v Object) (Object, error) {
	return o.Value.BinOpIn(vm, v)
}

// BinOpAdd is `o + pair`: a new instance, the pairs added checked.
func (o *TypedKeyValueArray) BinOpAdd(vm *VM, right Object) (Object, error) {
	r, err := o.Value.BinOpAdd(vm, right)
	if err != nil {
		return nil, err
	}
	return o.ArrayType.checked(vm, r.(KeyValueArray))
}

// BinOpInc is `o ++ pairs`: a new instance, the pairs added checked.
func (o *TypedKeyValueArray) BinOpInc(vm *VM, right Object) (Object, error) {
	r, err := o.Value.BinOpInc(vm, right)
	if err != nil {
		return nil, err
	}
	return o.ArrayType.checked(vm, r.(KeyValueArray))
}

// isTypedCollection reports whether obj is an instance of a typed dict or a
// typed key-value array type: a dict / a key-value array too (a param of those).
func isTypedCollection(obj Object) bool {
	switch obj.(type) {
	case *TypedDict, *TypedKeyValueArray:
		return true
	}
	return false
}

// registered here, not in BuiltinObjects' literal: its constructor reaches the
// casts, which reach the builtins (an initialization cycle)
func init() {
	BuiltinObjects[BuiltinNewTypedKVAType] = &BuiltinFunction{
		FuncName:              "TypedKeyValueArrayType",
		Value:                 NewTypedKVATypeFunc,
		AcceptMethodsDisabled: true,
	}
}
