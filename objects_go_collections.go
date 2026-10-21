package gad

import (
	"reflect"
	"strconv"
)

// Gad collections seen from Go. A Go function whose parameter is an
// ArrayOf[T] or a DictOf[K, V] (or a pointer to one) gets the gad collection
// given — not a copy —: what it changes, gad sees. T, K and V are Go types
// (int, uint, string, a struct, gad.Object, …), each item converted when read
// and written (objectToReflect); a typed collection (`type ints []int`,
// `type scores dict[str, int]`) checks what is written as its type does.
//
// A parameter of a plain slice or map type gets a copy instead, synchronized
// back when the Go function returns (syncBack): the items it changed, as Go's
// semantics are — a slice's growth is not seen, a map's changes are.

// goCollection is what a Go parameter bound to a gad collection is.
type goCollection interface {
	bindGad(vm *VM, o Object) error
}

// ArrayOf is a gad array (or typed array) of items T, seen from Go.
type ArrayOf[T any] struct {
	// Key and Value are the gad types of its indexes (int) and of its items
	// (the typed array's element type; any for an array).
	Key, Value Object

	vm    *VM
	ptr   Pointer
	arr   Array
	typed *TypedArray
}

// NewArrayOf is the gad array o seen from Go: an array, a typed array, or a
// pointer to one (`&a`, `&obj.list`: Append then grows what it points to).
func NewArrayOf[T any](vm *VM, o Object) (*ArrayOf[T], error) {
	a := &ArrayOf[T]{}
	if err := a.bindGad(vm, o); err != nil {
		return nil, err
	}
	return a, nil
}

func (a *ArrayOf[T]) bindGad(vm *VM, o Object) error {
	*a = ArrayOf[T]{vm: vm, Key: TInt, Value: TAny}
	if p, ok := o.(Pointer); ok {
		v, err := p.PtrGet(vm)
		if err != nil {
			return err
		}
		switch v.(type) {
		case Array:
			a.ptr = p
			return nil
		case *TypedArray:
			o = v
		}
	}
	switch t := o.(type) {
	case Array:
		a.arr = t
	case *TypedArray:
		a.typed = t
		a.Value = t.ArrayType.Elem
	default:
		return ErrType.NewErrorf("an array, not %s", o.Type().Name())
	}
	return nil
}

// Object is the gad array.
func (a *ArrayOf[T]) Object() Object {
	switch {
	case a.typed != nil:
		return a.typed
	case a.ptr != nil:
		return a.items()
	}
	return a.arr
}

func (a *ArrayOf[T]) items() Array {
	switch {
	case a.typed != nil:
		return a.typed.Items
	case a.ptr != nil:
		v, _ := a.ptr.PtrGet(a.vm)
		arr, _ := v.(Array)
		return arr
	}
	return a.arr
}

// Len is the number of items.
func (a *ArrayOf[T]) Len() int { return len(a.items()) }

// Get is the item i.
func (a *ArrayOf[T]) Get(i int) (T, error) {
	items := a.items()
	if i < 0 || i >= len(items) {
		var zero T
		return zero, ErrIndexOutOfBounds
	}
	return ValueOf[T](a.vm, items[i])
}

// Set writes v as the item i (checked by a typed array).
func (a *ArrayOf[T]) Set(i int, v T) error {
	o, err := ObjectOf(a.vm, v)
	if err != nil {
		return err
	}
	if a.typed != nil {
		return a.typed.IndexSet(a.vm, Int(i), o)
	}
	items := a.items()
	if i < 0 || i >= len(items) {
		return ErrIndexOutOfBounds
	}
	items[i] = o
	return nil
}

// Append adds vs at the end: of a typed array, or of the variable of an
// array given by its pointer (`&a`) — an array given by value cannot grow
// (its holder would not see it), an error.
func (a *ArrayOf[T]) Append(vs ...T) error {
	objs := make(Array, len(vs))
	for i, v := range vs {
		o, err := ObjectOf(a.vm, v)
		if err != nil {
			return err
		}
		objs[i] = o
	}
	switch {
	case a.typed != nil:
		if err := a.typed.checkAdded(a.vm, objs...); err != nil {
			return err
		}
		a.typed.Items = append(a.typed.Items, objs...)
	case a.ptr != nil:
		return a.ptr.PtrSet(a.vm, append(a.items(), objs...))
	default:
		return ErrType.NewError("an array given by value cannot grow: give its pointer (`&a`)")
	}
	return nil
}

// Delete removes the item i: of a typed array, or of the variable of an
// array given by its pointer.
func (a *ArrayOf[T]) Delete(i int) error {
	switch {
	case a.typed != nil:
		return a.typed.IndexDelete(a.vm, Int(i))
	case a.ptr != nil:
		items := a.items()
		if i < 0 || i >= len(items) {
			return ErrIndexOutOfBounds
		}
		return a.ptr.PtrSet(a.vm, append(items[:i:i], items[i+1:]...))
	}
	return ErrType.NewError("an array given by value cannot shrink: give its pointer (`&a`)")
}

// Values are the items, as Go values.
func (a *ArrayOf[T]) Values() ([]T, error) {
	items := a.items()
	out := make([]T, len(items))
	for i, o := range items {
		v, err := ValueOf[T](a.vm, o)
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}

// DictOf is a gad dict (or typed dict) of keys K and values V, seen from Go.
type DictOf[K comparable, V any] struct {
	// Key and Value are the gad types of its keys (str; the typed dict's K)
	// and of its values (any; the typed dict's V).
	Key, Value Object

	vm    *VM
	d     Dict
	typed *TypedDict
}

// NewDictOf is the gad dict o seen from Go: a dict, a typed dict, or a
// pointer to a variable holding one.
func NewDictOf[K comparable, V any](vm *VM, o Object) (*DictOf[K, V], error) {
	d := &DictOf[K, V]{}
	if err := d.bindGad(vm, o); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *DictOf[K, V]) bindGad(vm *VM, o Object) error {
	*d = DictOf[K, V]{vm: vm, Key: TStr, Value: TAny}
	if p, ok := o.(Pointer); ok {
		v, err := p.PtrGet(vm)
		if err != nil {
			return err
		}
		o = v
	}
	switch t := o.(type) {
	case Dict:
		d.d = t
	case *TypedDict:
		d.typed = t
		d.Key, d.Value = t.DictType.Key, t.DictType.Value
	default:
		return ErrType.NewErrorf("a dict, not %s", o.Type().Name())
	}
	return nil
}

// Object is the gad dict.
func (d *DictOf[K, V]) Object() Object {
	if d.typed != nil {
		return d.typed
	}
	return d.d
}

func (d *DictOf[K, V]) dict() Dict {
	if d.typed != nil {
		return d.typed.Value
	}
	return d.d
}

// key is k as a key of the dict.
func (d *DictOf[K, V]) key(k K) (string, error) {
	o, err := ObjectOf(d.vm, k)
	if err != nil {
		return "", err
	}
	if d.typed != nil {
		s, ok := keyOf(d.typed.DictType.Key, o, false)
		if !ok {
			return "", ErrType.NewErrorf("%s: a key %s is not of %s", d.typed.DictType.Name(),
				ReprQuote(o.ToString()), ReprQuote(typeName(d.typed.DictType.Key)))
		}
		return s, nil
	}
	return o.ToString(), nil
}

// Len is the number of entries.
func (d *DictOf[K, V]) Len() int { return len(d.dict()) }

// Get is the value of k, and whether it has one.
func (d *DictOf[K, V]) Get(k K) (V, bool, error) {
	var zero V
	s, err := d.key(k)
	if err != nil {
		return zero, false, err
	}
	o, ok := d.dict()[s]
	if !ok {
		return zero, false, nil
	}
	v, err := ValueOf[V](d.vm, o)
	return v, true, err
}

// Has reports whether the dict has k.
func (d *DictOf[K, V]) Has(k K) bool {
	s, err := d.key(k)
	if err != nil {
		return false
	}
	_, ok := d.dict()[s]
	return ok
}

// Set writes v as the value of k (checked by a typed dict).
func (d *DictOf[K, V]) Set(k K, v V) error {
	s, err := d.key(k)
	if err != nil {
		return err
	}
	o, err := ObjectOf(d.vm, v)
	if err != nil {
		return err
	}
	if d.typed != nil {
		return d.typed.IndexSet(d.vm, Str(s), o)
	}
	d.d[s] = o
	return nil
}

// Delete removes k.
func (d *DictOf[K, V]) Delete(k K) error {
	s, err := d.key(k)
	if err != nil {
		return err
	}
	delete(d.dict(), s)
	return nil
}

// Keys are the keys, as Go values (in no order).
func (d *DictOf[K, V]) Keys() ([]K, error) {
	out := make([]K, 0, len(d.dict()))
	for s := range d.dict() {
		k, err := keyFromString[K](d.vm, s)
		if err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, nil
}

// Range calls fn with each entry (in no order) while it returns true.
func (d *DictOf[K, V]) Range(fn func(k K, v V) bool) error {
	for s, o := range d.dict() {
		k, err := keyFromString[K](d.vm, s)
		if err != nil {
			return err
		}
		v, err := ValueOf[V](d.vm, o)
		if err != nil {
			return err
		}
		if !fn(k, v) {
			break
		}
	}
	return nil
}

// keyFromString is the key s of a dict as a K: a string, or a number parsed.
func keyFromString[K comparable](vm *VM, s string) (K, error) {
	var zero K
	t := reflect.TypeFor[K]()
	v := reflect.New(t).Elem()
	switch t.Kind() {
	case reflect.String:
		v.SetString(s)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(s, 10, t.Bits())
		if err != nil {
			return zero, ErrType.NewErrorf("key %q: %v", s, err)
		}
		v.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(s, 10, t.Bits())
		if err != nil {
			return zero, ErrType.NewErrorf("key %q: %v", s, err)
		}
		v.SetUint(n)
	default:
		return ValueOf[K](vm, Str(s))
	}
	return v.Interface().(K), nil
}

// ObjectOf is the Go value v as a gad value (vm's converters): a gad value
// as it is.
func ObjectOf[T any](vm *VM, v T) (Object, error) {
	if o, ok := any(v).(Object); ok {
		return o, nil
	}
	return vm.ToObject(any(v))
}

// ValueOf is the gad value o as a T: o itself when it is one, else
// converted as an argument of a Go function is (objectToReflect).
func ValueOf[T any](vm *VM, o Object) (T, error) {
	if v, ok := any(o).(T); ok {
		return v, nil
	}
	var zero T
	rv, err := objectToReflect(vm, o, reflect.TypeFor[T]())
	if err != nil {
		return zero, err
	}
	return rv.Interface().(T), nil
}

var goCollectionType = reflect.TypeFor[goCollection]()

// bindGoCollection is o bound to t when t is a Go collection (ArrayOf,
// DictOf) or a pointer to one.
func bindGoCollection(vm *VM, o Object, t reflect.Type) (reflect.Value, bool, error) {
	switch {
	case t.Kind() == reflect.Ptr && t.Implements(goCollectionType):
		w := reflect.New(t.Elem())
		err := w.Interface().(goCollection).bindGad(vm, o)
		return w, true, err
	case reflect.PointerTo(t).Implements(goCollectionType):
		w := reflect.New(t)
		err := w.Interface().(goCollection).bindGad(vm, o)
		return w.Elem(), true, err
	}
	return reflect.Value{}, false, nil
}

// syncBack writes into o, the gad array or dict a Go function was given a
// copy of (v), what the function changed: the items of a slice (not its
// growth, as Go's semantics are), the entries of a map. A typed collection
// checks what is written.
func syncBack(vm *VM, o Object, v reflect.Value) error {
	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		var items Array
		switch t := o.(type) {
		case Array:
			items = t
		case *TypedArray:
			items = t.Items
		default:
			return nil
		}
		n := min(len(items), v.Len())
		for i := 0; i < n; i++ {
			nv, err := vm.ToObject(v.Index(i).Interface())
			if err != nil {
				return err
			}
			if items[i].Equal(nv) {
				continue
			}
			if t, ok := o.(*TypedArray); ok {
				if err = t.IndexSet(vm, Int(i), nv); err != nil {
					return err
				}
				continue
			}
			items[i] = nv
		}
	case reflect.Map:
		var d Dict
		switch t := o.(type) {
		case Dict:
			d = t
		case *TypedDict:
			d = t.Value
		default:
			return nil
		}
		seen := make(map[string]bool, v.Len())
		iter := v.MapRange()
		for iter.Next() {
			k, err := vm.ToObject(iter.Key().Interface())
			if err != nil {
				return err
			}
			nv, err := vm.ToObject(iter.Value().Interface())
			if err != nil {
				return err
			}
			ks := k.ToString()
			seen[ks] = true
			if old, ok := d[ks]; ok && old.Equal(nv) {
				continue
			}
			if t, ok := o.(*TypedDict); ok {
				if err = t.IndexSet(vm, k, nv); err != nil {
					return err
				}
				continue
			}
			d[ks] = nv
		}
		for k := range d {
			if !seen[k] {
				delete(d, k)
			}
		}
	}
	return nil
}
