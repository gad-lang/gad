package gad

import (
	"strings"
)

// TTypedArrayType is the type of a TypedArrayType value (what `typeName` of a
// `type NAME []…` declaration reports).
var TTypedArrayType = NewType("typedArrayType", TBase)

// TypedArrayType is a named array type declared by `type NAME []…`:
//
//	type numerics []<int|uint|float|decimal>   // elements of a type union
//	type users []{ name; id }                  // elements satisfying an interface
//	type grid [][]int                          // arrays of int (depth 2)
//
// It is a first-class type: calling it (`numerics(1, 2.5)`) builds a TypedArray
// whose items are checked against the element type; it is a cast target
// (`x :: numerics` checks it is a numerics; `[1, 2] ::: numerics` converts an
// array to one, validating each item) and a parameter/field type.
type TypedArrayType struct {
	// TypeName is the declared name (`numerics`).
	TypeName string
	// Module is the module the type was declared in (qualifies FullName).
	Module *ModuleSpec
	// Meta is the `[k=v, …]` metadata of the declaration (or nil); `T.@meta`.
	Meta KeyValueArray
	// Elem is the leaf element type: a builtin/class type, a type union or an
	// interface — anything AssignToType accepts as a target.
	Elem Object
	// Depth is how many `[]` were written: an item of a depth-N typed array is an
	// array nested N-1 deep whose leaves match Elem.
	Depth int

	// members holds the class-like members of the declaration body
	// (`type T []int { fields; props {…}; methods {…} }`): its fieldsMap,
	// propertiesMap and methodsMap (and initFields). nil without a body.
	members *Class
	// ctor dispatches the `new(…)` constructor overloads; nil without any.
	ctor *typedArrayConstructor
}

var (
	_ ObjectType    = (*TypedArrayType)(nil)
	_ IndexGetter   = (*TypedArrayType)(nil)
	_ vmCanAssigner = (*TypedArrayType)(nil)
)

// NewTypedArrayType returns a typed array type named name whose leaves match
// elem, nested depth deep (depth >= 1).
func NewTypedArrayType(name string, module *ModuleSpec, depth int, elem Object) *TypedArrayType {
	if depth < 1 {
		depth = 1
	}
	return &TypedArrayType{TypeName: name, Module: module, Depth: depth, Elem: elem}
}

// NewTypedArrayTypeFunc is the `TypedArrayType(name, depth, elem[, define];
// meta=…)` builtin a `type NAME []…` declaration lowers to. The optional define
// handler `(T, define) => define(; fields=…, props=…, methods=…, new=…)`
// registers the body members (see Define), like a class.
func NewTypedArrayTypeFunc(c Call) (_ Object, err error) {
	var (
		name  = &Arg{Name: "name", TypeAssertion: TypeAssertionFromTypes(TStr)}
		depth = &Arg{Name: "depth", TypeAssertion: TypeAssertionFromTypes(TInt)}
		elem  = &Arg{Name: "elem"}
		meta  = &NamedArgVar{Name: "meta", TypeAssertion: TypeAssertionFromTypes(TKeyValueArray)}
		rest  Array
	)
	if rest, err = c.Args.DestructureRangeVar(1, name, depth, elem); err != nil {
		return
	}
	if err = c.NamedArgs.Get(meta); err != nil {
		return
	}
	t := NewTypedArrayType(string(name.Value.(Str)), c.VM.CurrentModuleSpec(), int(depth.Value.(Int)), elem.Value)
	t.Meta, _ = meta.Value.(KeyValueArray)
	if len(rest) > 0 {
		handler, ok := rest[0].(CallerObject)
		if !ok {
			return nil, NewArgumentTypeError("4th (define)", "callable", rest[0].Type().Name())
		}
		_, err = handler.Call(Call{
			Context: c.Context,
			VM:      c.VM,
			Args: Args{{
				t,
				NewFunction("define", func(c Call) (Object, error) {
					return nil, t.Define(c)
				}),
			}},
		})
	}
	return t, err
}

// Define registers the declaration body's members from named args, like
// Class.Define: `fields` (a key-value array of name=default, a field with
// metadata as the spec `(; meta=…, default=…)`), `initFields` (the non-literal
// defaults), `props`, `methods` and `new` (constructor overloads).
func (t *TypedArrayType) Define(c Call) error {
	m := t.memberClass()
	var (
		fields = &NamedArgVar{Name: "fields", TypeAssertion: TypeAssertionFromTypes(TKeyValueArray),
			Do: func(v Object) error { return m.CallAddFields(Call{VM: c.VM, Args: Args{Array{v}}}) }}
		initFields = &NamedArgVar{Name: "initFields", TypeAssertion: NewTypeAssertion(TypeAssertions(WithCallable())),
			Do: func(v Object) error { m.initFields, _ = v.(CallerObject); return nil }}
		props = &NamedArgVar{Name: "props", TypeAssertion: TypeAssertionFromTypes(TDict),
			Do: func(v Object) error { return m.CallAddProperties(Call{VM: c.VM, Args: Args{Array{v}}}) }}
		methods = &NamedArgVar{Name: "methods", TypeAssertion: TypeAssertionFromTypes(TDict, TKeyValueArray, TArray),
			Do: func(v Object) error { return m.CallAddMethods(Call{VM: c.VM, Args: Args{Array{v}}}) }}
		ctor = &NamedArgVar{Name: "new", TypeAssertion: NewTypeAssertion(TypeAssertions(WithCallable(), WithArray())),
			Do: func(v Object) error { return t.addConstructors(c.VM, v) }}
	)
	return c.NamedArgs.GetDo(fields, initFields, props, methods, ctor)
}

// memberClass returns (creating it on first use) the internal class that holds
// the body members.
func (t *TypedArrayType) memberClass() *Class {
	if t.members == nil {
		t.members = NewClass(t.TypeName, t.Module)
	}
	return t.members
}

func (t *TypedArrayType) GadObjectType()          {}
func (t *TypedArrayType) Type() ObjectType        { return TTypedArrayType }
func (t *TypedArrayType) Name() string            { return t.TypeName }
func (t *TypedArrayType) GetModule() *ModuleSpec  { return t.Module }
func (t *TypedArrayType) SetModule(m *ModuleSpec) { t.Module = m }
func (t *TypedArrayType) IsFalsy() bool           { return false }

func (t *TypedArrayType) FullName() string {
	if t.Module != nil && t.Module.Name != "" {
		return t.Module.Name + "." + t.TypeName
	}
	return t.TypeName
}

// Signature renders the array shape: `[]<int|uint>`, `[][]int`, `[]interface {…}`.
func (t *TypedArrayType) Signature() string {
	var b strings.Builder
	for i := 0; i < t.Depth; i++ {
		b.WriteString("[]")
	}
	if t.Elem != nil {
		b.WriteString(typedArrayElemName(t.Elem))
	}
	return b.String()
}

func typedArrayElemName(elem Object) string {
	switch e := elem.(type) {
	case *TypeUnion:
		return "<" + e.ToString() + ">"
	case ObjectType:
		return e.Name()
	}
	return elem.ToString()
}

func (t *TypedArrayType) ToString() string {
	return ReprQuoteTyped("typedArrayType", t.FullName()+" "+t.Signature())
}
func (t *TypedArrayType) String() string { return t.ToString() }

func (t *TypedArrayType) Equal(right Object) bool {
	r, ok := right.(*TypedArrayType)
	return ok && r == t
}

// IndexGet implements the reflection keys:
//
//	T.@name    — the declared name (str)
//	T.@module  — the declaring module
//	T.@meta    — the `[k=v, …]` metadata (a key-value array, empty when none)
//	T.@elem    — the leaf element type
//	T.@depth   — the number of `[]`
//	T.@fields / T.@props / T.@methods — the body members (dicts by name)
//	T.@new     — the constructor overloads (nil when none)
//	T.NAME     — a body member (field, property or method), for `T.NAME.@meta`
func (t *TypedArrayType) IndexGet(vm *VM, index Object) (Object, error) {
	switch index.ToString() {
	case "@name":
		return Str(t.TypeName), nil
	case "@module":
		if t.Module == nil || vm == nil {
			return Nil, nil
		}
		return vm.ModuleFromIndex(t.Module.Index), nil
	case "@meta":
		return metaObject(t.Meta), nil
	case "@elem":
		if t.Elem == nil {
			return Nil, nil
		}
		return t.Elem, nil
	case "@depth":
		return Int(t.Depth), nil
	case "@fields", "@props", "@methods":
		// Member registration is optional: a type without a body has none.
		if t.members == nil {
			return Dict{}, nil
		}
		switch index.ToString() {
		case "@fields":
			return t.members.Fields(), nil
		case "@props":
			return t.members.Properties(), nil
		}
		return t.members.Methods(), nil
	case "@new":
		if t.ctor == nil {
			return Nil, nil
		}
		return t.ctor, nil
	}
	// A bare name reflects a body member (`T.total.@meta`).
	if m := t.member(index.ToString()); m != nil {
		return m, nil
	}
	return nil, ErrInvalidIndex.NewError(index.ToString())
}

// Call builds a TypedArray of this type: `numerics(1, 2.5)` — each positional
// argument an item, checked against the element type (spread an array with
// `numerics(*arr)`); named arguments set the declared fields
// (`numerics(1; label="x")`). With `new(…)` constructor overloads in the body the
// matching overload runs first (see TypedArrayInitiator).
func (t *TypedArrayType) Call(c Call) (Object, error) {
	return (&TypedArrayInitiator{t: t}).Call(c)
}

// acceptItem checks (or, when transform, converts) one ITEM: an array nested
// Depth-1 deep whose leaves match Elem. It returns the (possibly converted) item.
// With RunFlagSkipTypedArrayItemCheck set on the VM the item is trusted and
// returned as is.
func (t *TypedArrayType) acceptItem(vm *VM, v Object, transform bool) (Object, error) {
	if skipTypedArrayItemCheck(vm) {
		return v, nil
	}
	return t.acceptNested(vm, v, t.Depth-1, transform)
}

// skipTypedArrayItemCheck reports whether the run disabled the typed-array
// element check (RunFlagSkipTypedArrayItemCheck).
func skipTypedArrayItemCheck(vm *VM) bool {
	return vm != nil && vm.runFlags.Has(RunFlagSkipTypedArrayItemCheck)
}

func (t *TypedArrayType) acceptNested(vm *VM, v Object, depth int, transform bool) (Object, error) {
	if depth == 0 {
		if t.Elem == nil {
			return v, nil
		}
		if transform {
			return AssignToTypeTransform(vm, v, t.Elem)
		}
		if _, err := AssignToType(vm, v, t.Elem); err != nil {
			return nil, err
		}
		return v, nil
	}
	arr, ok := v.(Array)
	if !ok {
		if ta, isTA := v.(*TypedArray); isTA {
			arr = ta.Items
		} else {
			return nil, ErrType.NewErrorf("expected an array, got %s", v.Type().Name())
		}
	}
	out := arr
	if transform {
		out = make(Array, len(arr))
	}
	for i, el := range arr {
		c, err := t.acceptNested(vm, el, depth-1, transform)
		if err != nil {
			return nil, err
		}
		if transform {
			out[i] = c
		}
	}
	return out, nil
}

// items returns obj's items when it is an array (or a typed array), for a cast.
func typedArraySource(obj Object) (Array, bool) {
	switch v := obj.(type) {
	case Array:
		return v, true
	case *TypedArray:
		return v.Items, true
	}
	return nil, false
}

// CanAssign reports whether obj is a TypedArray of this type. Like a class, a
// typed array type is nominal: `v :: numerics` and a `numerics` parameter accept
// only a numerics value; convert a plain array (or another typed array, or an
// interface array's value) with the transforming cast `arr ::: numerics`.
func (t *TypedArrayType) CanAssign(obj Object) (bool, error) {
	ta, ok := obj.(*TypedArray)
	return ok && ta.ArrayType == t, nil
}

// CanAssignVM is CanAssign (no VM-dependent check: the items were validated
// when the typed array was built or written).
func (t *TypedArrayType) CanAssignVM(_ *VM, obj Object) (bool, error) {
	return t.CanAssign(obj)
}

// AssignTo answers "does obj satisfy this type" (`obj :: T`): the caller passes
// the receiver as `to`. obj is returned unchanged on success.
func (t *TypedArrayType) AssignTo(vm *VM, obj Object, to TypeAssigner) (Object, error) {
	if to != TypeAssigner(t) {
		return nil, ErrIncompatibleAssign
	}
	ok, err := t.CanAssignVM(vm, obj)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrIncompatibleAssign.NewErrorf("%s is not assignable to %s",
			ReprQuote(obj.Type().Name()), ReprQuote(t.TypeName))
	}
	return obj, nil
}

// From converts an array (or another typed array) into a TypedArray of this
// type, converting each leaf with the transforming cast (`arr ::: T`).
func (t *TypedArrayType) From(vm *VM, obj Object) (*TypedArray, error) {
	if ta, ok := obj.(*TypedArray); ok && ta.ArrayType == t {
		return ta, nil
	}
	arr, ok := typedArraySource(obj)
	if !ok {
		return nil, ErrType.NewErrorf("%s: expected an array, got %s", t.TypeName, obj.Type().Name())
	}
	out := make(Array, len(arr))
	for i, v := range arr {
		c, err := t.acceptItem(vm, v, true)
		if err != nil {
			return nil, ErrType.NewErrorf("%s item #%d: %v", t.TypeName, i, err)
		}
		out[i] = c
	}
	ta := &TypedArray{Items: out, ArrayType: t}
	if err := ta.initFields(vm, nil); err != nil {
		return nil, err
	}
	return ta, nil
}

// TypedArray is an array bound to a TypedArrayType: it behaves as an Array
// (index get/set/delete, slicing, iteration, len, sort, `+`/`++`, `in`, append,
// copy) but every item written to it is checked against its type's element
// type. Its Type() is its TypedArrayType, so `typeName(x)` is the declared name
// and `x :: numerics` holds.
//
// An int (or uint) index reads/writes/deletes an item (`ta[0]`); any other key
// reaches the body members: a property (getter/setter with the instance as
// `this`), a field, or a method bound to the instance (`ta.sum()`).
//
// (The type field is named ArrayType, not Type: Type() is the Object method.)
type TypedArray struct {
	Items     Array
	ArrayType *TypedArrayType
	// Fields are the values of the fields declared in the type's body (nil when
	// it declares none).
	Fields Dict
}

var (
	_ Object                          = (*TypedArray)(nil)
	_ IndexGetter                     = (*TypedArray)(nil)
	_ IndexSetter                     = (*TypedArray)(nil)
	_ IndexDeleter                    = (*TypedArray)(nil)
	_ LengthGetter                    = (*TypedArray)(nil)
	_ Slicer                          = (*TypedArray)(nil)
	_ Iterabler                       = (*TypedArray)(nil)
	_ Copier                          = (*TypedArray)(nil)
	_ DeepCopier                      = (*TypedArray)(nil)
	_ Sorter                          = (*TypedArray)(nil)
	_ ReverseSorter                   = (*TypedArray)(nil)
	_ KeysGetter                      = (*TypedArray)(nil)
	_ ToArrayAppenderObject           = (*TypedArray)(nil)
	_ ObjectWithAddSelfAssignOperator = (*TypedArray)(nil)
	_ ObjectWithIncSelfAssignOperator = (*TypedArray)(nil)
	_ Printabler                      = (*TypedArray)(nil)
)

func (o *TypedArray) Type() ObjectType { return o.ArrayType }
func (o *TypedArray) IsFalsy() bool    { return len(o.Items) == 0 }
func (o *TypedArray) Length() int      { return len(o.Items) }

func (o *TypedArray) ToString() string {
	if len(o.Fields) > 0 {
		return o.ArrayType.TypeName + o.Items.ToString() + o.Fields.ToString()
	}
	return o.ArrayType.TypeName + o.Items.ToString()
}

func (o *TypedArray) Print(state *PrinterState) error {
	if err := state.WriteString(o.ArrayType.TypeName); err != nil {
		return err
	}
	if err := o.Items.Print(state); err != nil {
		return err
	}
	if len(o.Fields) > 0 {
		return o.Fields.Print(state)
	}
	return nil
}

func (o *TypedArray) ToInterface(vm *VM) any { return o.Items.ToInterface(vm) }

// Equal reports whether right is a TypedArray of the same type with equal items.
func (o *TypedArray) Equal(right Object) bool {
	r, ok := right.(*TypedArray)
	return ok && r.ArrayType == o.ArrayType && o.Items.Equal(r.Items) &&
		(len(o.Fields) == 0 && len(r.Fields) == 0 || o.Fields.Equal(r.Fields))
}

// with wraps items as a TypedArray of the same type (no check: items come from
// this array).
func (o *TypedArray) with(items Array) *TypedArray {
	var fields Dict
	if o.Fields != nil {
		fields = o.Fields.Copy().(Dict)
	}
	return &TypedArray{Items: items, ArrayType: o.ArrayType, Fields: fields}
}

// check validates items against the element type (a no-op when the run set
// RunFlagSkipTypedArrayItemCheck).
func (o *TypedArray) check(vm *VM, items ...Object) error {
	if skipTypedArrayItemCheck(vm) {
		return nil
	}
	for _, v := range items {
		if _, err := o.ArrayType.acceptItem(vm, v, false); err != nil {
			return ErrType.NewErrorf("%s: %v", o.ArrayType.TypeName, err)
		}
	}
	return nil
}

// IndexGet reads an item for an int/uint index, else a body member.
func (o *TypedArray) IndexGet(vm *VM, index Object) (Object, error) {
	switch index.(type) {
	case Int, Uint:
		return o.Items.IndexGet(vm, index)
	}
	return o.getMember(vm, index.ToString())
}

// IndexSet writes an item for an int/uint index (checked against the element
// type), else a property or field.
func (o *TypedArray) IndexSet(vm *VM, index, value Object) error {
	switch index.(type) {
	case Int, Uint:
		if err := o.check(vm, value); err != nil {
			return err
		}
		return o.Items.IndexSet(vm, index, value)
	}
	return o.setMember(vm, index.ToString(), value)
}

// IndexDelete removes the item at index (negative counts from the end).
func (o *TypedArray) IndexDelete(_ *VM, index Object) error {
	var idx int
	switch v := index.(type) {
	case Int:
		idx = int(v)
	case Uint:
		idx = int(v)
	default:
		return NewIndexTypeError("int|uint", index.Type().Name())
	}
	if idx < 0 {
		idx += len(o.Items)
	}
	if idx < 0 || idx >= len(o.Items) {
		return ErrIndexOutOfBounds
	}
	o.Items = append(o.Items[:idx:idx], o.Items[idx+1:]...)
	return nil
}

// Slice returns a TypedArray of the same type over items[low:high].
func (o *TypedArray) Slice(low, high int) Object { return o.with(o.Items[low:high]) }

func (o *TypedArray) Iterate(vm *VM, na *NamedArgs) Iterator { return o.Items.Iterate(vm, na) }

func (o *TypedArray) Copy() Object { return o.with(o.Items.Copy().(Array)) }

func (o *TypedArray) DeepCopy(vm *VM) (Object, error) {
	c, err := o.Items.DeepCopy(vm)
	if err != nil {
		return nil, err
	}
	return o.with(c.(Array)), nil
}

func (o *TypedArray) Sort(vm *VM, less CallerObject) (Object, error) {
	if _, err := o.Items.Sort(vm, less); err != nil {
		return nil, err
	}
	return o, nil
}

func (o *TypedArray) SortReverse(vm *VM) (Object, error) {
	if _, err := o.Items.SortReverse(vm); err != nil {
		return nil, err
	}
	return o, nil
}

func (o *TypedArray) Keys() Array                                { return o.Items.Keys() }
func (o *TypedArray) AppendToArray(arr Array) Array              { return append(arr, o.Items...) }
func (o *TypedArray) BinOpIn(vm *VM, v Object) (Object, error)   { return o.Items.BinOpIn(vm, v) }
func (o *TypedArray) BinOpLess(vm *VM, r Object) (Object, error) { return o.Items.BinOpLess(vm, r) }
func (o *TypedArray) BinOpGreater(vm *VM, r Object) (Object, error) {
	return o.Items.BinOpGreater(vm, r)
}

// Append appends checked items in place.
func (o *TypedArray) Append(vm *VM, items ...Object) error {
	if err := o.check(vm, items...); err != nil {
		return err
	}
	o.Items = append(o.Items, items...)
	return nil
}

// AppendObjects returns a new TypedArray with the checked items appended.
func (o *TypedArray) AppendObjects(vm *VM, items ...Object) (Object, error) {
	if err := o.check(vm, items...); err != nil {
		return nil, err
	}
	return o.with(append(o.Items.Copy().(Array), items...)), nil
}

// BinOpAdd `ta + v`: a new TypedArray with v appended (an iterable extends it),
// each new item checked.
func (o *TypedArray) BinOpAdd(vm *VM, right Object) (Object, error) {
	r, err := o.Items.Copy().(Array).BinOpAdd(vm, right)
	if err != nil {
		return nil, err
	}
	arr := r.(Array)
	if err = o.check(vm, arr[len(o.Items):]...); err != nil {
		return nil, err
	}
	return o.with(arr), nil
}

// BinOpInc `ta ++ v`: a new TypedArray with v's elements appended (checked).
func (o *TypedArray) BinOpInc(vm *VM, right Object) (Object, error) {
	r, err := o.Items.BinOpInc(vm, right)
	if err != nil {
		return nil, err
	}
	arr := r.(Array)
	if err = o.check(vm, arr[len(o.Items):]...); err != nil {
		return nil, err
	}
	return o.with(arr), nil
}

// SelfAssignOpAdd `ta += v`: append v (checked).
func (o *TypedArray) SelfAssignOpAdd(vm *VM, value Object) (Object, error) {
	if err := o.Append(vm, value); err != nil {
		return nil, err
	}
	return o, nil
}

// SelfAssignOpInc `ta ++= v`: append v's elements (checked).
func (o *TypedArray) SelfAssignOpInc(vm *VM, value Object) (Object, error) {
	other, err := ValuesOf(vm, value, &NamedArgs{})
	if err != nil {
		return nil, err
	}
	if err = o.Append(vm, other...); err != nil {
		return nil, err
	}
	return o, nil
}
