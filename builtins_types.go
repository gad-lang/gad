package gad

import (
	"reflect"
)

var (
	TAny                    = &Type{TypeName: "Any"}
	TSymbol                 = &Type{Parent: TAny, TypeName: "Symbol"}
	TIterationStateFlag     = &Type{Parent: TAny, TypeName: "IterationStateFlag"}
	IterationStop           = &Type{Parent: TIterationStateFlag, TypeName: "IterationStop"}
	IterationSkip           = &Type{Parent: TIterationStateFlag, TypeName: "IterationSkip"}
	TBase                   = &Type{Parent: TAny, TypeName: "Base"}
	TIterator               = &Type{Parent: TAny, TypeName: "Iterator"}
	TIterabler              = &Type{Parent: TAny, TypeName: "Iterabler"}
	TNilIterator            = &Type{Parent: TIterator, TypeName: "NilIterator"}
	TStateIterator          = &Type{Parent: TIterator, TypeName: "StateIterator"}
	TStrIterator            = &Type{Parent: TIterator, TypeName: "StrIterator"}
	TRawStrIterator         = &Type{Parent: TIterator, TypeName: "RawStrIterator"}
	TArrayIterator          = &Type{Parent: TIterator, TypeName: "ArrayIterator"}
	TDictIterator           = &Type{Parent: TIterator, TypeName: "DictIterator"}
	TBytesIterator          = &Type{Parent: TIterator, TypeName: "BytesIterator"}
	TKeyValueArrayIterator  = &Type{Parent: TIterator, TypeName: "KeyValueArrayIterator"}
	TKeyValueArraysIterator = &Type{Parent: TIterator, TypeName: "KeyValueArraysIterator"}
	TArgsIterator           = &Type{Parent: TIterator, TypeName: "ArgsIterator"}
	TReflectArrayIterator   = &Type{Parent: TIterator, TypeName: "ReflectArrayIterator"}
	TReflectMapIterator     = &Type{Parent: TIterator, TypeName: "ReflectMapIterator"}
	TReflectStructIterator  = &Type{Parent: TIterator, TypeName: "ReflectStructIterator"}
	TKeysIterator           = &Type{Parent: TIterator, TypeName: "KeysIterator"}
	TValuesIterator         = &Type{Parent: TIterator, TypeName: "ValuesIterator"}
	TEnumerateIterator      = &Type{Parent: TIterator, TypeName: "EnumerateIterator"}
	TItemsIterator          = &Type{Parent: TIterator, TypeName: "ItemsIterator"}
	TCallbackIterator       = &Type{Parent: TIterator, TypeName: "CallbackIterator"}
	TEachIterator           = &Type{Parent: TIterator, TypeName: "EachIterator"}
	TMapIterator            = &Type{Parent: TIterator, TypeName: "MapIterator"}
	TFilterIterator         = &Type{Parent: TIterator, TypeName: "FilterIterator"}
	TZipIterator            = &Type{Parent: TIterator, TypeName: "ZipIterator"}
	TPipedInvokeIterator    = &Type{Parent: TIterator, TypeName: "PipedInvokeIterator"}
)

var (
	_ Object       = (*Type)(nil)
	_ ObjectType   = (*Type)(nil)
	_ CallerObject = (*Type)(nil)
	_ MethodCaller = (*Type)(nil)
)

type Type struct {
	TypeName       string
	Parent         ObjectType
	calllerMethods MethodArgType
	Constructor    CallerObject
	Static         Dict
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

func (t *Type) AddCallerMethod(vm *VM, types MultipleObjectTypes, handler CallerObject, override bool) error {
	if len(types) == 0 {
		// overrides default constructor. uses Type.new to instantiate.
		override = true
	}
	return t.calllerMethods.Add(types, &CallerMethod{
		CallerObject: handler,
	}, override)
}

func (t *Type) WithMethod(types MultipleObjectTypes, handler CallerObject, override bool) *Type {
	if len(types) == 0 {
		// overrides default constructor. uses Type.new to instantiate.
		override = true
	}
	t.calllerMethods.Add(types, &CallerMethod{
		CallerObject: handler,
	}, override)
	return t
}

func (t *Type) WithConstructor(handler CallerObject) *Type {
	t.Constructor = handler
	return t
}

func (t *Type) HasCallerMethods() bool {
	return !t.calllerMethods.IsZero()
}

func (t *Type) CallerMethods() *MethodArgType {
	return &t.calllerMethods
}

func (t *Type) CallerOf(args Args) (co CallerObject, ok bool) {
	var types []ObjectType
	args.Walk(func(i int, arg Object) any {
		if t, ok := arg.(ObjectType); ok {
			types = append(types, t)
		} else {
			types = append(types, arg.Type())
		}
		return nil
	})
	return t.CallerOfTypes(types)
}

func (t *Type) GetMethod(types []ObjectType) (co CallerObject) {
	return t.calllerMethods.GetMethod(types).Caller()
}

func (t *Type) CallerOfTypes(types []ObjectType) (co CallerObject, validate bool) {
	if method := t.calllerMethods.GetMethod(types); method != nil {
		return method.CallerObject, false
	}
	return t, validate
}

func (t *Type) Caller() CallerObject {
	return t.Constructor
}

func (t *Type) Call(c Call) (_ Object, err error) {
	caller, validate := t.CallerOf(c.Args)
	if caller == nil {
		if t.Constructor == nil {
			return nil, ErrNotInitializable
		}
		caller = t.Constructor
	}
	c.SafeArgs = !validate
	return YieldCall(caller, &c), nil
}

func (t *Type) IsFalsy() bool {
	return t.TypeName == ""
}

func (t *Type) Type() ObjectType {
	return t.Parent
}

func (t *Type) ToString() string {
	return t.TypeName
}

func (t *Type) Equal(right Object) bool {
	rt, _ := right.(*Type)
	return rt == t
}

func (t Type) Name() string {
	return t.TypeName
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

func (t *Type) IsChildOf(ot ObjectType) bool {
	return ot == t.Parent
}

var Types = map[reflect.Type]ObjectType{}

func RegisterBuiltinType(typ BuiltinType, name string, val any, init CallableFunc) *BuiltinObjType {
	if init == nil {
		init = func(call Call) (ret Object, err error) {
			return nil, ErrNotInitializable
		}
	}
	ot := &BuiltinObjType{NameValue: name, Value: init}
	BuiltinObjects[typ] = ot
	BuiltinsMap[name] = typ

	rt := reflect.TypeOf(val)
	for rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	Types[rt] = ot
	return ot
}

func TypeOf(arg Object) ObjectType {
	ot := arg.Type()
	if ot == nil {
		return DetectTypeOf(arg)
	}
	return ot
}

func DetectTypeOf(arg Object) ObjectType {
	rt := reflect.TypeOf(arg)
	for rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	ot := Types[rt]
	if ot == nil {
		ot = Nil.Type()
	}
	return ot
}

func TypesOf(obj Object) (types []ObjectType) {
	types = append(types, obj.Type())

	var ok bool
	if _, ok = obj.(Iterabler); ok {
		types = append(types, TIterabler)
	}
	return types
}

func init() {
	TIterator.WithMethod(
		MultipleObjectTypes{{nil}},
		&Function{
			Value: func(c Call) (o Object, err error) {
				if err = c.Args.CheckLen(1); err != nil {
					return
				}
				_, o, err = ToStateIterator(c.VM, c.Args.GetOnly(0), &c.NamedArgs)
				return
			},
		},
		true)
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
