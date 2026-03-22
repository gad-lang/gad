package gad

import (
	"reflect"

	"github.com/gad-lang/gad/repr"
)

var (
	TAny                         = NewType("any")
	TModule                      = NewType("Module")
	TSymbol                      = NewType("Symbol", TAny)
	TIterationStateFlag          = NewType("IterationStateFlag", TAny)
	IterationStop                = NewType("IterationStop", TIterationStateFlag)
	IterationSkip                = NewType("IterationSkip", TIterationStateFlag)
	TBase                        = NewType("Base", TAny)
	TClass                       = NewType("Class", TBase)
	TClassConstructor            = NewType("ClassConstructor", TBase)
	TClassProperty               = NewType("ClassProperty", TBase)
	TClassMethod                 = NewType("ClassMethod", TBase)
	TClassField                  = NewType("ClassField", TBase)
	TClassInstanceMethod         = NewType("ClassInstanceMethod", TBase)
	TClassInstancePropertyGetter = NewType("ClassInstancePropertyGetter", TBase)
	TClassInstancePropertySetter = NewType("ClassInstancePropertySetter", TBase)
	TIterator                    = NewType("Iterator", TAny)
	TIterabler                   = NewType("Iterabler", TAny)
	TNilIterator                 = NewType("NilIterator", TIterator)
	TStateIterator               = NewType("StateIterator", TIterator)
	TStrIterator                 = NewType("StrIterator", TIterator)
	TRawStrIterator              = NewType("RawStrIterator", TIterator)
	TArrayIterator               = NewType("ArrayIterator", TIterator)
	TDictIterator                = NewType("DictIterator", TIterator)
	TBytesIterator               = NewType("BytesIterator", TIterator)
	TKeyValueArrayIterator       = NewType("KeyValueArrayIterator", TIterator)
	TKeyValueArraysIterator      = NewType("KeyValueArraysIterator", TIterator)
	TArgsIterator                = NewType("ArgsIterator", TIterator)
	TReflectArrayIterator        = NewType("ReflectArrayIterator", TIterator)
	TReflectMapIterator          = NewType("ReflectMapIterator", TIterator)
	TReflectStructIterator       = NewType("ReflectStructIterator", TIterator)
	TKeysIterator                = NewType("KeysIterator", TIterator)
	TValuesIterator              = NewType("ValuesIterator", TIterator)
	TEnumerateIterator           = NewType("EnumerateIterator", TIterator)
	TItemsIterator               = NewType("ItemsIterator", TIterator)
	TCallbackIterator            = NewType("CallbackIterator", TIterator)
	TEachIterator                = NewType("EachIterator", TIterator)
	TMapIterator                 = NewType("MapIterator", TIterator)
	TFilterIterator              = NewType("FilterIterator", TIterator)
	TZipIterator                 = NewType("ZipIterator", TIterator)
	TPipedInvokeIterator         = NewType("PipedInvokeIterator", TIterator)
)

var (
	_ Object       = (*Type)(nil)
	_ ObjectType   = (*Type)(nil)
	_ CallerObject = (*Type)(nil)
	_ MethodCaller = (*Type)(nil)
)

func TypeToString(typeName string) string {
	return repr.Quote("Type " + repr.Quote(typeName))
}

type Type struct {
	Parent ObjectType
	Static Dict
	Module *Module
	name   string
	*FuncSpec
}

func NewType(typeName string, parent ...ObjectType) (t *Type) {
	t = &Type{name: typeName}
	if len(parent) > 0 {
		t.Parent = parent[0]
	}
	t.FuncSpec = NewFuncSpec(t)
	return
}

func (Type) GadObjectType() {}

func (t Type) Copy() Object {
	cp := &t
	cp.Static = Copy(t.Static)
	cp.FuncSpec = cp.FuncSpec.CopyWithTarget(cp)
	return cp
}

func (t *Type) GetModule() *Module {
	return t.Module
}

func (t *Type) FuncSpecName() string {
	return "type " + ReprQuote(t.FullName())
}

func (t *Type) String() string {
	return string(MustToStr(nil, t))
}

func (t *Type) Print(state *PrinterState) (err error) {
	if ok, _ := state.options.TypesAsFullNames(); ok {
		return state.WriteString(t.FullName())
	}
	return t.PrintFuncWrapper(state, t)
}

func (t *Type) IndexGet(vm *VM, index Object) (value Object, err error) {
	if t.Static == nil {
		return Dict{}.IndexGet(vm, index)
	}
	return t.Static.IndexGet(vm, index)
}

func (t *Type) IndexSet(vm *VM, index, value Object) (err error) {
	if t.Static == nil {
		t.Static = make(Dict)
	}
	return t.Static.IndexSet(vm, index, value)
}

func (t *Type) IndexDelete(vm *VM, index Object) (err error) {
	if t.Static == nil {
		return
	}
	return t.Static.IndexDelete(vm, index)
}

func (t *Type) WithConstructor(handler CallerObject) *Type {
	t.defaul = handler
	return t
}

func (t *Type) WithStatic(d Dict) *Type {
	t.Static = d
	return t
}

func (t *Type) Constructor() CallerObject {
	return t.defaul
}

func (t *Type) Caller() CallerObject {
	return t.defaul
}

func (t *Type) IsFalsy() bool {
	return false
}

func (t *Type) Type() ObjectType {
	return t.Parent
}

func (t *Type) ToString() string {
	return t.FullName()
}

func (t *Type) Equal(right Object) bool {
	rt, _ := right.(*Type)
	return rt == t
}

func (t Type) Name() string {
	return t.name
}

func (t Type) FullName() string {
	if t.Module != nil {
		return t.Module.Info.Name + "." + t.name
	}
	return t.name
}

func (Type) Getters() Dict {
	return nil
}

func (Type) Setters() Dict {
	return nil
}

func (Type) Fields() Dict {
	return nil
}

func (Type) Methods() Dict {
	return nil
}

func (Type) New(*VM, Dict) (Object, error) {
	return nil, ErrNotInitializable
}

var Types = map[reflect.Type]ObjectType{}

func RegisterBuiltinType(typ BuiltinType, name string, val any, init CallableFunc) *BuiltinObjType {
	if init == nil {
		init = func(call Call) (ret Object, err error) {
			return nil, ErrNotInitializable
		}
	}
	ot := NewBuiltinObjType(name).WithNew(init)
	ot.builtinType = typ
	BuiltinObjects[typ] = ot
	BuiltinsMap[name] = typ

	rt := reflect.TypeOf(val)
	for rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	Types[rt] = ot
	return ot
}

func init() {
	AddMethod(TIterator, NewFunction(
		"",
		func(c Call) (o Object, err error) {
			if err = c.Args.CheckLen(1); err != nil {
				return
			}
			_, o, err = ToStateIterator(c.VM, c.Args.GetOnly(0), &c.NamedArgs)
			return
		},
		FunctionWithParams(func(p func(name string) *ParamBuilder) {
			p("iterable").Type(TAny).Usage("An iterable object")
		}),
	))

	TZipIterator.WithConstructor(
		&Function{
			Value: func(c Call) (o Object, err error) {
				var it = make([]Iterator, c.Args.Length())
				c.Args.Walk(func(i int, arg Object) any {
					if _, it[i], err = ToIterator(c.VM, arg, &c.NamedArgs); err != nil {
						return err
					}
					return nil
				})
				if err != nil {
					return
				}

				o = IteratorObject(ZipIterator(it...))
				return
			},
		})
}
