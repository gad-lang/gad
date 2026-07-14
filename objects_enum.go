package gad

import "sort"

var (
	_ Object             = (*Enum)(nil)
	_ ObjectType         = (*Enum)(nil)
	_ ModuleGetter       = (*Enum)(nil)
	_ ModuleSetter       = (*Enum)(nil)
	_ ToDictConverter    = (*Enum)(nil)
	_ IndexSetterUpdater = (*Enum)(nil)
	_ Printabler         = (*Enum)(nil)
	_ ReprTypeNamer      = (*Enum)(nil)
	_ IndexGetter        = (*Enum)(nil)
	_ Iterabler          = (*Enum)(nil)
)

// Enum is an ordered, named set of integer constants produced by the `enum`
// syntax. It is also an ObjectType: each member is an EnumValue whose Type() is
// the owning Enum. An Enum is immutable after construction, indexable by member
// name and iterable in declaration order.
type Enum struct {
	EnumName string
	Values   map[string]*EnumValue
	Module   *ModuleSpec
}

// NewEnum returns an empty Enum; members are added in order with AddValue.
func NewEnum(enumName string, module *ModuleSpec) *Enum {
	return &Enum{EnumName: enumName, Module: module, Values: make(map[string]*EnumValue, 0)}
}

func (e *Enum) Call(c Call) (Object, error) {
	return nil, ErrNotCallable
}

func (e *Enum) CanCall() bool {
	return false
}

func (e *Enum) Name() string {
	return e.EnumName
}

func (e *Enum) String() string {
	return ReprQuoteTyped(TEnum.name, e.EnumName)
}

func (e *Enum) GadObjectType() {
}

func (e *Enum) AssignTo(_ *VM, obj Object, to TypeAssigner) (Object, error) {
	return assignByTypeChain(e, obj, to)
}

func (e *Enum) CanAssign(obj Object) (bool, error) {
	return canAssignByType(e, obj)
}

// AddValue appends a member with the given name and underlying int/uint value.
// Its Index is the current member count, so members added in source order keep
// that order.
func (e *Enum) AddValue(name string, value Object) {
	e.Values[name] = &EnumValue{
		Enum:  e,
		Index: len(e.Values),
		Name:  name,
		Value: value,
	}
}

func (e *Enum) Iterate(_ *VM, na *NamedArgs) Iterator {
	keys := make([]string, 0, len(e.Values))

	for k := range e.Values {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
		return e.Values[keys[i]].Index < e.Values[keys[j]].Index
	})

	return SliceEntryIteration(TDictIterator, e, keys, func(v string) (_, _ Object, _ error) {
		return Str(v), e.Values[v], nil
	}).ParseNamedArgs(na)
}

func (e *Enum) IndexGet(vm *VM, index Object) (value Object, err error) {
	key := index.ToString()
	if value, ok := e.Values[key]; ok {
		return value, nil
	}
	return nil, ErrInvalidIndex.NewError(key)
}

func (e *Enum) ReprTypeName() string {
	return "enum " + ReprQuote(e.FullName())
}

func (e *Enum) Print(state *PrinterState) error {
	defer state.WrapRepr(e)()

	var entries PrintStateDictEntries

	for name, value := range e.Values {
		entries = append(entries, &PrintStateDictEntry{name, value})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Value.(*EnumValue).Index < entries[j].Value.(*EnumValue).Index
	})

	return state.PrintDict(len(entries),
		func(i int) (Object, error) {
			return Str(entries[i].Name), nil
		}, func(i int) (Object, error) {
			return entries[i].Value, nil
		})
}

func (e *Enum) ToArray() (arr Array) {
	arr = make(Array, len(e.Values))
	var i int
	for _, v := range e.Values {
		arr[i] = v
		i++
	}
	sort.Slice(arr, func(i, j int) bool {
		return arr[i].(*EnumValue).Index < arr[j].(*EnumValue).Index
	})
	return
}

func (e *Enum) UpdateIndexSetter(out StringIndexSetter) {
	for k, v := range e.Values {
		out.Set(k, v)
	}
}

func (e *Enum) ToDict() (d Dict) {
	d = make(Dict, len(e.Values))
	e.UpdateIndexSetter(d)
	return
}

func (e *Enum) SetModule(m *ModuleSpec) {
	e.Module = m
}

func (e *Enum) GetModule() *ModuleSpec {
	return e.Module
}

func (e *Enum) FullName() string {
	if e.Module != nil {
		return e.Module.Name + "." + e.EnumName
	}
	return e.EnumName
}

func (e *Enum) IsFalsy() bool {
	return len(e.Values) == 0
}

func (e *Enum) Type() ObjectType {
	return TEnum
}

func (e *Enum) ToString() string {
	return ReprQuoteTyped("enum", e.FullName())
}

func (e *Enum) Equal(right Object) bool {
	if o, _ := right.(*Enum); o != nil {
		return o == e
	}
	return false
}

var (
	_ Object      = (*EnumValue)(nil)
	_ IndexGetter = (*EnumValue)(nil)
)

// EnumValue is a single member of an Enum: its declaration Index, owning Enum,
// Name and the underlying Int/Uint Value. Its members are reachable from Gad as
// `.name`, `.value`, `.index` and `.enum`.
type EnumValue struct {
	Index int
	Enum  *Enum
	Name  string
	Value Object
}

func (e *EnumValue) IsFalsy() bool {
	return e.Value.IsFalsy()
}

func (e *EnumValue) Type() ObjectType {
	return e.Enum
}

func (e *EnumValue) IsInt() (ok bool) {
	_, ok = e.Value.(Int)
	return
}

func (e *EnumValue) ToString() string {
	var typName = "uint"
	if e.IsInt() {
		typName = "int"
	}
	return ReprQuoteTyped("enum "+ReprQuote(e.Enum.FullName()), e.Name+" = "+ReprQuoteTyped(typName, e.Value.ToString()))
}

func (e *EnumValue) Equal(right Object) bool {
	if rv, _ := right.(*EnumValue); rv != nil {
		return rv == e
	}
	return false
}

// IndexGet exposes the value's members: `name` (the field name), `value` (the
// underlying int/uint), `index` (declaration order) and `enum` (the owning
// Enum).
func (e *EnumValue) IndexGet(_ *VM, index Object) (Object, error) {
	switch index.ToString() {
	case "name":
		return Str(e.Name), nil
	case "value":
		return e.Value, nil
	case "index":
		return Int(e.Index), nil
	case "enum":
		return e.Enum, nil
	}
	return nil, ErrInvalidIndex.NewError(index.ToString())
}
