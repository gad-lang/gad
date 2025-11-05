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

	if rs, ok := o.(RawStr); ok {
		return rs, nil
	}

	if ret, err = Val(c.VM.Builtins.Call(BuiltinStr, c)); err != nil {
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
		)

		if _, err = c.VM.Builtins.Call(
			BuiltinPrint, Call{
				VM:      c.VM,
				Args:    Args{Array{state, arg}},
				Context: c.Context,
			}); err != nil {
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
	} else {
		d := Dict{}
		err = c.Args.WalkE(func(i int, arg Object) error {
			switch t := arg.(type) {
			case ToDictConveter:
				t.ToDict(d)
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
		maxDepth, _ = c.NamedArgs.GetValue(PrintStateOptionMaxDepth).(Int)
		raw         = !c.NamedArgs.GetValue(PrintStateOptionRaw).IsFalsy()
		indent      = c.NamedArgs.GetValue(PrintStateOptionIndent)
	)

	if err = c.Args.Destructure(wArg); err != nil {
		return
	}

	return NewPrinterState(
		c.VM,
		wArg.Value.(Writer).GoWriter(),
		PrinterStateWithRaw(raw),
		PrinterStateWithMaxDepth(int(maxDepth)),
		PrinterStateWithIndent(indent),
		PrinterStateWithOptions(c.NamedArgs.unreadDict()),
		PrinterStateWithContext(c.Context),
	), nil
}
