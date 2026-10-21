package gad

import (
	"fmt"
	"reflect"
)

// objectToReflect is o as a Go value of type t, for a Go function's argument,
// a slice's or map's item, a struct's field:
//
//   - a reflected value: its Go value (prepareArg's conversions);
//   - a callable of gad, for a Go function type: a Go function that calls it
//     through vm (reflect.MakeFunc) — a callback; its results converted back,
//     an error of the call returned when the type's last result is an error;
//   - an array for a slice or an array type, a dict for a map type: made of
//     the items, each converted;
//   - anything else: its Go value (vm.ToInterface), converted when it is
//     convertible (int64 to int, …).
func objectToReflect(vm *VM, o Object, t reflect.Type) (reflect.Value, error) {
	if rv, ok := o.(ReflectValuer); ok && rv.GetRValue() != nil {
		return prepareArg(indirectInterface(rv.GetRValue().RValue), t)
	}
	// ArrayOf[T], DictOf[K, V]: bound to the gad collection itself
	if v, ok, err := bindGoCollection(vm, o, t); ok {
		return v, err
	}
	// a typed collection's items, for a slice or a map
	switch x := o.(type) {
	case *TypedArray:
		if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
			o = x.Items
		}
	case *TypedDict:
		if t.Kind() == reflect.Map {
			o = x.Value
		}
	}
	switch t.Kind() {
	case reflect.Func:
		if _, isGoFunc := o.(*ReflectFunc); !isGoFunc && Callable(o) {
			return reflectCallback(vm, o, t), nil
		}
	case reflect.Slice:
		if arr, ok := o.(Array); ok {
			s := reflect.MakeSlice(t, len(arr), len(arr))
			for i, item := range arr {
				v, err := objectToReflect(vm, item, t.Elem())
				if err != nil {
					return reflect.Value{}, fmt.Errorf("item %d: %w", i, err)
				}
				s.Index(i).Set(v)
			}
			return s, nil
		}
	case reflect.Array:
		if arr, ok := o.(Array); ok {
			if len(arr) > t.Len() {
				return reflect.Value{}, fmt.Errorf("%d items for %s", len(arr), t)
			}
			a := reflect.New(t).Elem()
			for i, item := range arr {
				v, err := objectToReflect(vm, item, t.Elem())
				if err != nil {
					return reflect.Value{}, fmt.Errorf("item %d: %w", i, err)
				}
				a.Index(i).Set(v)
			}
			return a, nil
		}
	case reflect.Map:
		if d, ok := o.(Dict); ok {
			m := reflect.MakeMapWithSize(t, len(d))
			for k, item := range d {
				kv, err := objectToReflect(vm, Str(k), t.Key())
				if err != nil {
					return reflect.Value{}, fmt.Errorf("key %q: %w", k, err)
				}
				v, err := objectToReflect(vm, item, t.Elem())
				if err != nil {
					return reflect.Value{}, fmt.Errorf("key %q: %w", k, err)
				}
				m.SetMapIndex(kv, v)
			}
			return m, nil
		}
	}
	var v reflect.Value
	if o != nil && o != Nil {
		v = indirectInterface(reflect.ValueOf(vm.ToInterface(o)))
	}
	return prepareArg(v, t)
}

// reflectCallback is fn, a callable of gad, as a Go function of type t: it
// calls fn through vm, synchronously — a callback of the Go call running it.
func reflectCallback(vm *VM, fn Object, t reflect.Type) reflect.Value {
	return reflect.MakeFunc(t, func(in []reflect.Value) []reflect.Value {
		args := make(Array, 0, len(in))
		for i, v := range in {
			if t.IsVariadic() && i == len(in)-1 {
				for j := 0; j < v.Len(); j++ {
					o, err := vm.ToObject(v.Index(j).Interface())
					if err != nil {
						return reflectResults(vm, t, nil, err)
					}
					args = append(args, o)
				}
				continue
			}
			o, err := vm.ToObject(v.Interface())
			if err != nil {
				return reflectResults(vm, t, nil, err)
			}
			args = append(args, o)
		}
		ret, err := vm.Call(fn, Args{args}, nil)
		return reflectResults(vm, t, ret, err)
	})
}

// reflectResults are the results of a Go function of type t that returned
// ret, or failed with err: returned as its last result when it is an error,
// a panic otherwise (the Go call running the callback returns it).
func reflectResults(vm *VM, t reflect.Type, ret Object, err error) []reflect.Value {
	n := t.NumOut()
	out := make([]reflect.Value, n)
	for i := range out {
		out[i] = reflect.Zero(t.Out(i))
	}
	hasErr := n > 0 && t.Out(n-1) == errorType
	values := n
	if hasErr {
		values--
	}
	if err == nil && values > 0 {
		vals := Array{ret}
		if values > 1 {
			arr, ok := ret.(Array)
			if !ok || len(arr) != values {
				err = ErrType.NewErrorf("callback of %s: %d results, given %s", t, values, ret.Type().Name())
			}
			vals = arr
		}
		for i := 0; err == nil && i < values; i++ {
			var v reflect.Value
			if v, err = objectToReflect(vm, vals[i], t.Out(i)); err == nil {
				out[i] = v
			}
		}
	}
	if err != nil {
		if !hasErr {
			panic(err)
		}
		out[n-1] = reflect.ValueOf(&err).Elem()
	}
	return out
}

// ReflectChan is a Go channel: iterated (`for v in ch`), its values received
// until it is closed; `ch.send(v)`, `ch.recv()` (`[value, ok]`), `ch.close()`,
// `len(ch)`, `ch.cap()`.
type ReflectChan struct {
	ReflectValue
}

var (
	_ Iterabler    = (*ReflectChan)(nil)
	_ LengthGetter = (*ReflectChan)(nil)
)

func (o *ReflectChan) CanIterate() bool { return true }

func (o *ReflectChan) Length() int { return o.RValue.Len() }

// Iterate receives the channel's values until it is closed, keyed by their
// order.
func (o *ReflectChan) Iterate(vm *VM, _ *NamedArgs) Iterator {
	var i int
	recv := func(vm *VM, state *IteratorState) error {
		v, ok := o.RValue.Recv()
		if !ok {
			state.Mode = IteratorStateModeDone
			return nil
		}
		val, err := vm.ToObject(v.Interface())
		if err != nil {
			return err
		}
		state.Mode = IteratorStateModeEntry
		state.Entry.K, state.Entry.V = Int(i), val
		i++
		return nil
	}
	return NewIterator(
		func(vm *VM) (state *IteratorState, err error) {
			state = &IteratorState{}
			i = 0
			err = recv(vm, state)
			return
		},
		recv,
	).SetInput(o).SetItType(o.Type())
}

// IndexGet gives the channel's methods: send, recv, close, cap.
func (o *ReflectChan) IndexGet(vm *VM, index Object) (Object, error) {
	switch index.ToString() {
	case "send":
		return &Function{FuncName: "send", Value: func(c Call) (Object, error) {
			if err := c.Args.CheckLen(1); err != nil {
				return nil, err
			}
			v, err := objectToReflect(c.VM, c.Args.Get(0), o.RValue.Type().Elem())
			if err != nil {
				return nil, ErrType.NewErrorf("send: %v", err)
			}
			o.RValue.Send(v)
			return Nil, nil
		}}, nil
	case "recv":
		return &Function{FuncName: "recv", Value: func(c Call) (Object, error) {
			v, ok := o.RValue.Recv()
			if !ok {
				return Array{Nil, False}, nil
			}
			val, err := c.VM.ToObject(v.Interface())
			if err != nil {
				return nil, err
			}
			return Array{val, True}, nil
		}}, nil
	case "close":
		return &Function{FuncName: "close", Value: func(c Call) (_ Object, err error) {
			defer func() {
				if r := recover(); r != nil {
					err = ErrType.NewErrorf("close: %v", r)
				}
			}()
			o.RValue.Close()
			return Nil, nil
		}}, nil
	case "cap":
		return &Function{FuncName: "cap", Value: func(Call) (Object, error) {
			return Int(o.RValue.Cap()), nil
		}}, nil
	}
	return nil, ErrInvalidIndex.NewError(index.ToString())
}

// CallName calls the channel's method name.
func (o *ReflectChan) CallName(name string, c Call) (Object, error) {
	f, err := o.IndexGet(c.VM, Str(name))
	if err != nil {
		return nil, err
	}
	return f.(CallerObject).Call(c)
}

// BinOpAdd is `s + v`: the slice with v appended (converted to the item
// type); of a pointer to a slice (`&ids`), the Go slice itself.
func (o *ReflectSlice) BinOpAdd(vm *VM, right Object) (Object, error) {
	return o.AppendObjects(vm, right)
}

// BinOpInc is `s ++ v`: the slice with the values of v appended.
func (o *ReflectSlice) BinOpInc(vm *VM, right Object) (Object, error) {
	values, err := ValuesOf(vm, right, &NamedArgs{})
	if err != nil {
		return nil, err
	}
	return o.AppendObjects(vm, values...)
}
