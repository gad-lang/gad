package gad

import (
	"io"
	"strings"
)

func (vm *VM) CallBuiltin(t BuiltinType, namedArgs *NamedArgs, args ...Object) (Object, error) {
	c := Call{VM: vm, Args: Args{args}}
	if namedArgs != nil {
		c.NamedArgs = *namedArgs
	}
	return vm.Builtins.Call(t, c)
}

func MustToStr(vm *VM, o Object) (_ Str) {
	s, err := ToStr(vm, o)
	if err != nil {
		panic(err)
	}
	return s
}

func ToStr(vm *VM, o Object) (_ Str, err error) {
	switch v := o.(type) {
	case Str:
		return v, nil
	}
	var w strings.Builder
	if err = ToStrW(&w, vm, o); err != nil {
		return
	}
	return Str(w.String()), nil
}

func ToStrW(w io.Writer, vm *VM, o Object) (err error) {
	switch v := o.(type) {
	case Str:
		_, err = w.Write([]byte(v))
	default:
		err = Print(NewPrinterState(vm, w), o)
	}
	return
}

func ToRawStr(vm *VM, o Object) (_ RawStr, err error) {
	var w strings.Builder
	if err = ToRawStrW(&w, vm, o); err != nil {
		return
	}
	return RawStr(w.String()), nil
}

func ToRawStrW(w io.Writer, vm *VM, o Object) (err error) {
	ps := NewPrinterState(vm, w)
	ps.options.SetRaw(true)
	return Print(ps, o)
}

func Print(state *PrinterState, o ...Object) (err error) {
	if state.VM == nil {
		_, err = BuiltinPrintFunc(Call{Args: Args{append(Array{state}, o...)}})
	} else {
		_, err = state.VM.CallBuiltin(BuiltinPrint, nil, append(Array{state}, o...)...)
	}
	return
}

func ToRepr(vm *VM, o Object, opts ...PrinterStateOptions) (_ Str, err error) {
	var (
		v  Object
		na NamedArgs
	)

	if len(opts) > 0 {
		na = *opts[0].Dict().ToNamedArgs()
	}

	if vm == nil {
		if v, err = BuiltinReprFunc(Call{Args: Args{Array{o}}, NamedArgs: na}); err != nil {
			return
		}
	} else if v, err = vm.Builtins.Call(BuiltinRepr, Call{VM: vm, Args: Args{Array{o}}, NamedArgs: na}); err != nil {
		return
	}
	return v.(Str), nil
}

func DeepCopy[T Object](vm *VM, o T) (_ T, err error) {
	var r Object
	r, err = vm.Builtins.Call(BuiltinDeepCopy, Call{VM: vm, Args: Args{Array{o}}})
	if err != nil {
		return
	}
	return r.(T), nil
}

func Copy[T Object](o T) T {
	if cp, _ := Object(o).(Copier); cp != nil {
		return cp.Copy().(T)
	}
	return o
}

func DoCall(co CallerObject, c Call) (ret Object, err error) {
	var yc *yieldCall

	for {
		if ret, err = co.Call(c); err == nil {
			if yc, _ = ret.(*yieldCall); yc != nil {
				co, c = yc.CallerObject, *yc.c
				continue
			}
		} else {
			err = ErrCall.Wrap(err, co.ToString())
		}
		return
	}
}

func Val(v Object, e error) (ret Object, err error) {
	if e != nil {
		return nil, e
	}

	ret = v

	var yc *yieldCall

	for {
		if yc, _ = ret.(*yieldCall); yc != nil {
			if ret, err = yc.CallerObject.Call(*yc.c); err == nil {
				continue
			} else {
				err = ErrCall.Wrap(err, yc.CallerObject.ToString())
			}
		}
		return
	}
}

func MustVal(v Object, _ error) (ret Object) {
	return v
}

func NamedParamTypeCheck(name string, typeso, value Object) (badTypes string, err error) {
	var (
		types     []ObjectType
		assertion = &TypeAssertion{
			Handlers: map[string]TypeAssertionHandler{
				"ObjectType|[]ObjectType": func(v Object) bool {
					switch t := v.(type) {
					case ObjectType:
						types = append(types, t)
						return true
					case Array:
						for _, object := range t {
							switch t2 := object.(type) {
							case ObjectType:
								types = append(types, t2)
							default:
								return false
							}
						}
						return true
					default:
						switch t2 := t.(type) {
						case ObjectType:
							types = append(types, t2)
							return true
						default:
							return false
						}
					}
				},
			},
		}
	)

	if badTypes = assertion.Accept(typeso); badTypes != "" {
		return
	}

	err = NamedParamTypeCheckAssertion(name, NewTypeAssertion(TypeAssertions(WithIsAssignable(types...))), value)
	return
}

func NamedParamTypeCheckAssertion(name string, assertion *TypeAssertion, value Object) (err error) {
	if expectedNames := assertion.Accept(value); expectedNames != "" {
		err = NewNamedArgumentTypeError(
			string(name),
			expectedNames,
			value.Type().Name(),
		)
		return
	}
	return
}
