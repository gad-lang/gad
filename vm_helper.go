package gad

import (
	"fmt"
	"math"
)

func ToStr(vm *VM, o Object) (_ Str, err error) {
	var v Object
	if v, err = Val(vm.Builtins.Call(BuiltinStr, Call{VM: vm, Args: Args{Array{o}}})); err != nil {
		return
	}
	return v.(Str), nil
}

func ToRawStr(vm *VM, o Object) (_ RawStr, err error) {
	var v Object
	if v, err = Val(vm.Builtins.Call(BuiltinRawStr, Call{VM: vm, Args: Args{Array{o}}})); err != nil {
		return
	}
	return v.(RawStr), nil
}

func ToRepr(vm *VM, o Object) (_ Str, err error) {
	var v Object
	if v, err = Val(vm.Builtins.Call(BuiltinRepr, Call{VM: vm, Args: Args{Array{o}}})); err != nil {
		return
	}
	return v.(Str), nil
}

func ToReprTyped(vm *VM, typ ObjectType, o Object) (s Str, err error) {
	s = Str(typ.Name()) + ":"
	var v Object
	if v, err = Val(vm.Builtins.Call(BuiltinRepr, Call{VM: vm, Args: Args{Array{o}}})); err != nil {
		return
	}
	return s + v.(Str), nil
}

func ToReprTypedS(vm *VM, typ ObjectType, o Object) (_ string, err error) {
	var s Str
	s, err = ToReprTyped(vm, typ, o)
	return string(s), err
}

func ToReprTypedRS(vm *VM, typ ObjectType, o any) (s string, err error) {
	s = typ.Name() + ":"
	var v string
	switch t := o.(type) {
	case string:
		v = t
	case Representer:
		if v, err = t.Repr(vm); err != nil {
			return
		}
	case Object:
		var v_ Str
		if v_, err = ToRepr(vm, t); err != nil {
			return
		}
		v = string(v_)
	}
	return ReprQuote(s + v), nil
}

func DeepCopy(vm *VM, o Object) (Object, error) {
	return Val(vm.Builtins.Call(BuiltinDeepCopy, Call{VM: vm, Args: Args{Array{o}}}))
}

func Copy(o Object) Object {
	if cp, _ := o.(Copier); cp != nil {
		return cp.Copy()
	}
	return o
}

func ToIterator(vm *VM, obj Object, na *NamedArgs) (l int, it Iterator, err error) {
	l = -1
	switch t := obj.(type) {
	case ObjectIterator:
		it = t
		switch t2 := t.GetIterator().(type) {
		case LengthIterator:
			l = t2.Length()
		}
		return
	case LengthIterator:
		it = t
		l = t.Length()
	case Iterator:
		it = t
	case CanIterabler:
		if t.CanIterate() {
			it = t.Iterate(vm, na)
			if itl, _ := it.(LengthIterator); itl != nil {
				l = itl.Length()
			}
		}
	case Iterabler:
		it = t.Iterate(vm, na)
		if itl, _ := it.(LengthIterator); itl != nil {
			l = itl.Length()
		}
	default:
		mc := vm.Builtins.Get(BuiltinIterator).(MethodCaller)
		if startMethod := mc.CallerMethodOfArgsTypes(ObjectTypes{obj.Type()}); startMethod != nil {
			if nextMethod := mc.CallerMethodOfArgsTypes(ObjectTypes{obj.Type(), TAny}); nextMethod != nil {
				if lenMethod := vm.Builtins.Get(BuiltinLen).(MethodCaller).CallerMethodOfArgsTypes(ObjectTypes{obj.Type()}); lenMethod != nil {
					var lenValue Object
					if lenValue, err = NewInvoker(vm, startMethod).Invoke(Args{Array{obj}}, nil); err != nil {
						return
					}
					switch t := lenValue.(type) {
					case Int:
						if t > math.MaxInt32 {
							err = ErrType.NewError(fmt.Sprintf("length(_ %s) = %d is grether then "+
								"math.MaxInt32 value", obj.Type(), int64(t)))
							return
						}
						l = int(t)
					default:
						err = ErrType.NewError(
							fmt.Sprintf("length(%s) result type expected %s, found %s", obj.Type().Name(),
								TInt.Name(), lenValue.Type().Name()))
						return
					}
				}

				var (
					typeName string
					s        Str
				)

				if s, err = ToRepr(vm, startMethod); err != nil {
					return
				}
				typeName = "{start: " + string(s) + ", next: "

				if s, err = ToRepr(vm, nextMethod); err != nil {
					return
				}
				typeName += string(s) + "}"

				var (
					nextArgs   = Array{obj, nil}
					nextCaller VMCaller
				)

				if nextCaller, err = NewInvoker(vm, nextMethod).ValidArgs(true).Caller(Args{nextArgs}, nil); err != nil {
					return
				}

				vm.curFrame.Defer(nextCaller.Close)

				it = NewIterator(
					func(vm *VM) (state *IteratorState, err error) {
						state = &IteratorState{}
						var val Object
						if val, err = NewInvoker(vm, startMethod).Invoke(Args{Array{obj}}, nil); err == nil {
							if arr, ok := val.(Array); ok && len(arr) == 2 {
								state.Value = arr[0]
								if e, _ := arr[1].(*KeyValue); e != nil {
									state.Entry = *e
								} else {
									state.Entry.K = Nil
									state.Entry.V = arr[1]
								}
							} else {
								state.Mode = IteratorStateModeDone
							}
						}
						return
					},
					func(vm *VM, state *IteratorState) (err error) {
						nextArgs[1] = state.Value
						var val Object
						if val, err = nextCaller.Call(); err == nil {
							if val == Nil {
								state.Mode = IteratorStateModeDone
							} else if arr, ok := val.(Array); ok && len(arr) == 2 {
								state.Value = arr[0]
								if e, _ := arr[1].(*KeyValue); e != nil {
									state.Entry = *e
								} else {
									state.Entry.K = Nil
									state.Entry.V = arr[1]
								}
							}
						}
						return
					}).
					SetInput(obj).
					SetItType(&Type{Parent: TIterator, TypeName: typeName})
			}
		}
	}
	if err == nil && it == nil {
		err = ErrNotIterable.NewError(obj.Type().Name())
	}
	return
}

func ToStateIterator(vm *VM, obj Object, na *NamedArgs) (l int, sit *StateIteratorObject, err error) {
	var it Iterator
	if l, it, err = ToIterator(vm, obj, na); err == nil {
		sit = NewStateIteratorObject(vm, it)
	}
	return
}

func Iterate(vm *VM, it Iterator, init func(state *IteratorState) error, cb func(e *KeyValue) error) (err error) {
	var state *IteratorState
	state, err = it.Start(vm)
	if err == nil && init != nil {
		if err = init(state); err != nil {
			return
		}
	}
	for err == nil && state.Mode != IteratorStateModeDone {
		if state.Mode != IteratorStateModeContinue {
			if err = cb(&state.Entry); err != nil {
				return
			}
		}
		state.Mode = IteratorStateModeEntry
		err = it.Next(vm, state)
	}
	return
}

func IterateObject(vm *VM, o Object, na *NamedArgs, init func(state *IteratorState) error, cb func(e *KeyValue) error) (err error) {
	var it Iterator
	if _, it, err = ToIterator(vm, o, na); err != nil {
		return
	} else if it != nil {
		return Iterate(vm, it, init, cb)
	} else {
		err = ErrNotIterable.NewError(o.Type().Name())
	}
	return
}

func CollectCb(vm *VM, o Object, na *NamedArgs, cb func(e *KeyValue, i *Int) Object) (values Array, err error) {
	var (
		l  int
		it Iterator
	)
	if l, it, err = ToIterator(vm, o, na); err == nil {
		var (
			state *IteratorState
			i     Int
		)

		if l > 0 {
			values = make(Array, l)
			state, err = it.Start(vm)
			for err == nil && state.Mode != IteratorStateModeDone {
				if state.Mode != IteratorStateModeContinue {
					values[i] = cb(&state.Entry, &i)
				}
				i++
				err = it.Next(vm, state)
			}
		} else {
			state, err = it.Start(vm)
			for err == nil && state.Mode != IteratorStateModeDone {
				if state.Mode != IteratorStateModeContinue {
					values = append(values, cb(&state.Entry, &i))
				}
				i++
				err = it.Next(vm, state)
			}
		}
	}
	return
}

func ValuesOf(vm *VM, o Object, na *NamedArgs) (values Array, err error) {
	var ok bool

	if values, ok = o.(Array); ok {
		return values, nil
	}

	if g, _ := o.(ValuesGetter); g != nil {
		return g.Values(), nil
	}

	return CollectCb(vm, o, na, func(e *KeyValue, i *Int) Object {
		return e.V
	})
}

func itemsOfCb(vm *VM, o Object, na *NamedArgs, cb func(kv *KeyValue) error) (err error) {
start:
	if values, ok := o.(KeyValueArray); ok {
		for _, value := range values {
			if err = cb(value); err != nil {
				return
			}
		}

		return
	}

	if g, _ := o.(ItemsGetter); g != nil {
		if o, err = g.Items(vm); err != nil {
			return
		}
		goto start
	}

	err = IterateObject(vm, o, na, nil, cb)
	return
}

func ItemsOf(vm *VM, o Object, na *NamedArgs) (values KeyValueArray, err error) {
	err = itemsOfCb(vm, o, na, func(kv *KeyValue) error {
		values = append(values, kv)
		return nil
	})
	return
}

func DoCall(co CallerObject, c Call) (ret Object, err error) {
	var yc *yieldCall

	for {
		if ret, err = co.Call(c); err == nil {
			if yc, _ = ret.(*yieldCall); yc != nil {
				co, c = yc.CallerObject, *yc.c
				continue
			}
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
			}
		}
		return
	}
}
