package gad

import (
	"context"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gad-lang/gad/token"
	"github.com/shopspring/decimal"
)

var (
	NewFlagFunc = funcPORO(func(arg Object) Object {
		return Flag(!arg.IsFalsy())
	})

	NewBoolFunc = funcPORO(func(arg Object) Object {
		return Bool(!arg.IsFalsy())
	})

	NewIntFunc = funcPi64RO(func(v int64) Object {
		return Int(v)
	})

	NewUintFunc = funcPu64RO(func(v uint64) Object {
		return Uint(v)
	})

	NewFloatFunc = funcPf64RO(func(v float64) Object {
		return Float(v)
	})

	NewDecimalFunc = funcPpVM_OROe(func(vm *VM, v Object) (Object, error) {
		return Decimal(decimal.Zero).BinaryOp(vm, token.Add, v)
	})

	NewCharFunc = funcPOROe(func(arg Object) (Object, error) {
		v, ok := ToChar(arg)
		if ok && v != utf8.RuneError {
			return v, nil
		}
		if v == utf8.RuneError || arg == Nil {
			return Nil, nil
		}
		return Nil, NewArgumentTypeError(
			"1st",
			"numeric|string|bool",
			arg.Type().Name(),
		)
	})
)

func NewErrorFunc(c Call) (ret Object, err error) {
	if err = c.Args.CheckLen(1); err != nil {
		return
	}
	arg := c.Args.Get(0)
	if e, _ := arg.(*Error); e != nil {
		return e, nil
	}

	var s Str
	if s, err = ToStr(c.VM, arg); err != nil {
		return
	}

	return &Error{Name: "error", Message: string(s)}, nil
}

func NewRawStrFunc(c Call) (ret Object, err error) {
	if err := c.Args.CheckLen(1); err != nil {
		return Nil, err
	}

	o := c.Args.Get(0)

	switch v := o.(type) {
	case RawStr:
		return v, nil
	case Str:
		return RawStr(v), nil
	}

	if ret, err = c.VM.Builtins.Call(BuiltinStr, c); err != nil {
		return
	}

	if c.NamedArgs.GetValue("cast").IsFalsy() {
		ret = c.VM.ToRawStrHandler(c.VM, ret.(Str))
	} else {
		ret = RawStr(ret.(Str))
	}

	return
}

func NewStrFunc(c Call) (_ Object, err error) {
	if err := c.Args.CheckMinLen(1); err != nil {
		return nil, err
	}

	l := c.Args.Length()
	switch l {
	case 1:
		type mustToString string
		const mustToStringKey mustToString = "mustToString"
		type sval struct {
			v Object
		}
		arg := c.Args.Get(0)
		newSv := &sval{v: arg}

		if c.Context != nil {
			if sv, _ := c.Context.Value(mustToStringKey).(*sval); sv != nil {
				if !IsPrimitive(sv.v) {
					if !IsPrimitive(arg) {
						a := reflect.ValueOf(sv.v).UnsafePointer()
						b := reflect.ValueOf(newSv.v).UnsafePointer()

						if a == b {
							return Str(arg.ToString()), nil
						}
					}
				} else if sv.v == arg {
					return Str(arg.ToString()), nil
				}
			}
		} else {
			c.Context = context.Background()
		}

		c.Context = context.WithValue(c.Context, mustToStringKey, newSv)

		var (
			w     strings.Builder
			state = PrinterStateFromCall(&Call{
				VM:        c.VM,
				Args:      Args{{NewWriter(&w)}},
				NamedArgs: c.NamedArgs,
				Context:   c.Context,
			})

			strCall = Call{
				VM:      c.VM,
				Args:    Args{Array{state, arg}},
				Context: c.Context,
			}
		)

		if c.VM == nil {
			if _, err = BuiltinPrintFunc(strCall); err != nil {
				return
			}
		} else if _, err = c.VM.Builtins.Call(BuiltinPrint, strCall); err != nil {
			return
		}
		return Str(w.String()), nil
	default:
		var (
			w     strings.Builder
			state = PrinterStateFromCall(&Call{
				VM:        c.VM,
				Args:      Args{{NewWriter(&w)}},
				NamedArgs: c.NamedArgs,
				Context:   c.Context,
			})
		)

		c.Args.Walk(func(_ int, arg Object) any {
			_, err = c.VM.Builtins.Call(
				BuiltinPrint, Call{
					VM:      c.VM,
					Args:    Args{Array{state, arg}},
					Context: c.Context,
				})
			return err
		})
		if err != nil {
			return
		}
		return Str(w.String()), nil
	}
}

func NewTypedIdentFunc(c Call) (ret Object, err error) {
	var (
		nameArg = &Arg{
			Name:          "name",
			TypeAssertion: TypeAssertionFromTypes(TStr),
		}
		typesArg = &Arg{
			Name: "types",
			TypeAssertion: NewTypeAssertion(TypeAssertionHandlers{
				"arrayOfTypes": func(v Object) (ok bool) {
					var arr Array
					if arr, ok = v.(Array); !ok {
						return false
					}
					for _, o := range arr {
						if !IsType(o) {
							return false
						}
					}
					return true
				},
			}),
		}
	)

	if err = c.Args.Destructure(nameArg, typesArg); err != nil {
		return
	}

	ret = &TypedIdent{
		Name:  string(nameArg.Value.(Str)),
		Types: typesArg.Value.(Array),
	}
	return
}

func NewBytesFunc(c Call) (_ Object, err error) {
	size := c.Args.Length()

	switch size {
	case 0:
		length := NamedArgVar{
			Name:          "length",
			Value:         Int(0),
			TypeAssertion: TypeAssertionFromTypes(TInt),
		}
		if err = c.NamedArgs.Get(&length); err != nil {
			return
		}
		return make(Bytes, int(length.Value.(Int))), nil
	case 1:
		if v, ok := ToBytes(c.Args.Get(0)); ok {
			return v, nil
		}
	}

	out := make(Bytes, 0, size)
	for _, args := range c.Args {
		for i, obj := range args {
			switch v := obj.(type) {
			case Int:
				out = append(out, byte(v))
			case Uint:
				out = append(out, byte(v))
			case Char:
				out = append(out, byte(v))
			default:
				return Nil, NewArgumentTypeError(
					strconv.Itoa(i+1),
					"int|uint|char",
					args[i].Type().Name(),
				)
			}
		}
	}
	return out, nil
}

func NewBufferFunc(c Call) (ret Object, err error) {
	var w = &Buffer{}
	if !c.Args.IsFalsy() {
		_, err = BuiltinWriteFunc(Call{
			VM:        c.VM,
			Args:      append(Args{Array{w}}, c.Args...),
			NamedArgs: c.NamedArgs,
		})
	}
	return w, err
}

func NewArrayFunc(c Call) (ret Object, err error) {
	return c.Args.Values(), nil
}

func NewDictFunc(c Call) (ret Object, err error) {
	if c.Args.IsFalsy() {
		ret = c.NamedArgs.AllDict()
	} else if c.Args.Length() == 1 {
		arg := c.Args.Get(0)
		switch t := arg.(type) {
		case ToDictConverter:
			return t.ToDict(), nil
		case DictUpdator:
			d := Dict{}
			t.UpdateDict(d)
			return d, nil
		default:
			d := Dict{}
			if err = ItemsOfCb(c.VM, &c.NamedArgs, func(kv *KeyValue) error {
				d[kv.K.ToString()] = kv.V
				return nil
			}, arg); err != nil {
				return
			}
			return d, nil
		}
	} else {
		d := Dict{}
		err = c.Args.WalkE(func(i int, arg Object) error {
			switch t := arg.(type) {
			case DictUpdator:
				t.UpdateDict(d)
				return nil
			default:
				return ItemsOfCb(c.VM, &c.NamedArgs, func(kv *KeyValue) error {
					d[kv.K.ToString()] = kv.V
					return nil
				}, arg)
			}
		})
		ret = d
	}
	return
}

func NewSyncDictFunc(c Call) (ret Object, err error) {
	if c.Args.Length() == 1 {
		arg := c.Args.Get(0)
		switch t := arg.(type) {
		case *SyncDict:
			return t, nil
		case Dict:
			return &SyncDict{Value: t}, nil
		}
	}

	if ret, err = NewDictFunc(c); err != nil {
		return
	}

	ret = &SyncDict{Value: ret.(Dict)}
	return
}

func NewKeyValueFunc(c Call) (ret Object, err error) {
	if err := c.Args.CheckLen(2); err != nil {
		return nil, err
	}
	return &KeyValue{c.Args.Get(0), c.Args.Get(1)}, nil
}

func NewKeyValueArrayFunc(c Call) (ret Object, err error) {
	if c.Args.IsFalsy() {
		ret = c.NamedArgs.Join()
	} else {
		var arr KeyValueArray
		err = c.Args.WalkE(func(i int, arg Object) error {
			if kv, _ := arg.(*KeyValue); kv != nil {
				arr = append(arr, kv)
				return nil
			}
			return ItemsOfCb(c.VM, &c.NamedArgs, func(kv *KeyValue) error {
				arr = append(arr, kv)
				return nil
			}, arg)
		})
		if err == nil {
			ret = arr
		}
	}
	return
}

func NewRegexpFunc(c Call) (_ Object, err error) {
	var (
		input = Arg{
			Name:          "input",
			TypeAssertion: TypeAssertionFromTypes(TStr, TRawStr),
		}
	)

	if err = c.Args.Destructure(&input); err != nil {
		return
	}

	var re *regexp.Regexp
	if re, err = regexp.Compile(input.Value.ToString()); err != nil {
		return
	}

	return (*Regexp)(re), err
}

func NewMixedParamsFunc(c Call) (ret Object, err error) {
	return &MixedParams{
		Positional: c.Args.Array(),
		Named:      c.NamedArgs.Join(),
	}, nil
}

func NewPrinterStateFunc(c Call) (ret Object, err error) {
	var (
		wArg = &Arg{
			Name: "writer",
			TypeAssertion: NewTypeAssertion(TypeAssertionHandlers{
				"writer": Writeable,
			}),
		}
	)

	if err = c.Args.Destructure(wArg); err != nil {
		return
	}

	return NewPrinterState(
		c.VM,
		wArg.Value.(Writer).GoWriter(),
		PrinterStateWithContext(c.Context),
	).ParseOptions(&c.NamedArgs), nil
}

func NewClassFunc(c Call) (ret Object, err error) {
	var (
		nameArg = &Arg{
			Name:          "name",
			TypeAssertion: TypeAssertionFromTypes(TStr),
		}
	)

	if err = c.Args.Destructure(nameArg); err != nil {
		return
	}

	t := NewClass(string(nameArg.Value.(Str)), c.VM.CurrentModule())

	var (
		kvaTA  = TypeAssertionFromTypes(TKeyValueArray)
		dictTA = TypeAssertionFromTypes(TDict)

		fields = &NamedArgVar{
			Name:          "fields",
			TypeAssertion: kvaTA,
			Do: func(value Object) error {
				return t.CallAddFields(Call{VM: c.VM, Args: Args{Array{value}}})
			},
		}

		methods = &NamedArgVar{
			Name:          "methods",
			TypeAssertion: TypeAssertionFromTypes(TDict, TKeyValueArray, TArray),
			Do: func(value Object) (err error) {
				return t.CallAddMethods(Call{VM: c.VM, Args: Args{Array{value}}})
			},
		}

		properties = &NamedArgVar{
			Name:          "properties",
			TypeAssertion: dictTA,
			Do: func(value Object) (err error) {
				return t.CallAddProperties(Call{VM: c.VM, Args: Args{Array{value}}})
			},
		}

		constructor = &NamedArgVar{
			Name:          "new",
			TypeAssertion: NewTypeAssertion(TypeAssertions(WithCallable(), WithArray())),
			Do: func(value Object) (err error) {
				return t.CallAddNewHandlers(Call{VM: c.VM, Args: Args{Array{value}}})
			},
		}

		extends = &NamedArgVar{
			Name:          "extends",
			TypeAssertion: TypeAssertionFromTypes(TArray),
			Do: func(value Object) (err error) {
				return t.CallExtends(Call{VM: c.VM, Args: Args{Array{value}}})
			},
		}
	)

	if err = c.NamedArgs.GetDo(constructor, fields, methods, properties, extends); err != nil {
		return
	}
	return t, nil
}

func NewFuncFunc(c Call) (_ Object, err error) {
	const anonymous = "‹anonymous›"

	if err = c.Args.CheckMinLen(1); err != nil {
		return
	}

	var (
		arg       = c.Args.Get(0)
		moduleVar = &NamedArgVar{
			Name:          "module",
			TypeAssertion: TypeAssertionFromTypes(TModule),
		}
		name string
	)

	switch t := arg.(type) {
	case Str:
		name = string(t)
		c.Args.Shift()
	case CallerObject:
		name = t.Name()
	default:
		err = NewArgumentTypeError("1st (nameOrCaller)", "str|CallerObject", arg.Type().Name())
		return
	}

	if name == "" {
		name = anonymous
	}

	if err = c.NamedArgs.Get(moduleVar); err != nil {
		return
	}

	module, _ := moduleVar.Value.(*Module)
	if module == nil {
		module = c.VM.CurrentModule()
	}
	
	f := NewFunc(name, module)

	err = c.Args.WalkE(func(i int, arg Object) (err error) {
		return SplitCaller(c.VM, arg, func(co CallerObject, types ParamsTypes) (err error) {
			return f.AddMethodByTypes(c.VM, types, co, false, nil)
		})
	})

	return f, err
}

func NewComputedValue(c Call) (_ Object, err error) {
	f := &Arg{
		Name:          "func",
		TypeAssertion: NewTypeAssertion(TypeAssertions(WithRawCallable())),
	}
	if err = c.Args.Destructure(f); err != nil {
		return
	}

	if cv, ok := f.Value.(*ComputedValue); ok {
		return cv, nil
	}

	return &ComputedValue{f.Value.(CallerObject)}, err
}
