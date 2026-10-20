package gad

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// DictType[K, V] (issue #19): the type of a dict whose keys are of K (a type of
// strings: str, an enum — a key a member's name —, a union of them) and whose
// values are of V. It is structural, as an array type is: a plain dict of those
// keys and values is of it; nothing is bound to a value. `DictType[V]` is
// `DictType[str, V]`, and `DictType[str, any]` is the builtin `dict` itself.

// DictOfType is an instance of the generic DictType: `DictType[K, V]`.
type DictOfType struct {
	Key, Value Object
}

var (
	_ Object        = (*DictOfType)(nil)
	_ IndexGetter   = (*DictOfType)(nil)
	_ TypeAssigner  = (*DictOfType)(nil)
	_ vmCanAssigner = (*DictOfType)(nil)
)

// dictTypes caches the instances by their arguments: `DictType[str, int] ===
// DictType[str, int]`.
var dictTypes sync.Map

// typeIdentity is a type's identity for the cache: a builtin by its key (a VM
// holds copies of the builtins), any other by its pointer.
func typeIdentity(o Object) string {
	switch t := o.(type) {
	case *BuiltinObjType:
		return "b" + strconv.Itoa(int(t.builtinType))
	case BuiltinObjTypeKey:
		return "b" + strconv.Itoa(int(t))
	}
	if o == TAny {
		return "b" + strconv.Itoa(int(BuiltinAny))
	}
	if u, ok := o.(*TypeUnion); ok {
		parts := make([]string, len(u.Types))
		for i, m := range u.Types {
			parts[i] = typeIdentity(m)
		}
		return "u(" + strings.Join(parts, "|") + ")"
	}
	return fmt.Sprintf("p%p", o)
}

// dictTypeOf is `DictType[index]`: index the value type, or an array [K, V]
// (or [V]).
func dictTypeOf(vm *VM, index Object) (Object, error) {
	key, val := Object(TStr), index
	if a, ok := index.(Array); ok {
		switch len(a) {
		case 1:
			val = a[0]
		case 2:
			key, val = a[0], a[1]
		default:
			return nil, ErrType.NewErrorf("DictType: [K, V] or [V], not %d type arguments", len(a))
		}
	}
	return newDictOfType(vm, key, val)
}

// newDictOfType is DictType[key, val]: `dict` for [str, any], else the cached
// instance.
func newDictOfType(vm *VM, key, val Object) (Object, error) {
	if !isDictKeyType(key) {
		return nil, ErrType.NewErrorf("DictType: a key is a string: %s is not a type of strings", ReprQuote(key.ToString()))
	}
	if _, ok := val.(TypeAssigner); !ok && val != TAny {
		return nil, ErrType.NewErrorf("DictType: the value type: %s is not a type", ReprQuote(val.ToString()))
	}
	if sameType(key, TStr) && sameType(val, TAny) {
		if vm != nil {
			return vm.Builtins.builtins.Objects[BuiltinDict], nil
		}
		return BuiltinObjects[BuiltinDict], nil
	}
	id := typeIdentity(key) + "," + typeIdentity(val)
	if t, ok := dictTypes.Load(id); ok {
		return t.(*DictOfType), nil
	}
	t, _ := dictTypes.LoadOrStore(id, &DictOfType{Key: key, Value: val})
	return t.(*DictOfType), nil
}

// isDictKeyType reports whether t is a type of strings: str, rawstr, any, an
// enum, a union of them.
func isDictKeyType(t Object) bool {
	switch x := t.(type) {
	case *Enum:
		return true
	case *TypeUnion:
		for _, m := range x.Types {
			if !isDictKeyType(m) {
				return false
			}
		}
		return len(x.Types) > 0
	}
	return sameType(t, TStr) || sameType(t, TRawStr) || sameType(t, TAny)
}

// keyOK reports whether the key k is of the key type t.
func keyOK(t Object, k string) bool {
	switch x := t.(type) {
	case *Enum:
		_, ok := x.Values[k]
		return ok
	case *TypeUnion:
		for _, m := range x.Types {
			if keyOK(m, k) {
				return true
			}
		}
		return false
	}
	return true
}

// keyOf is the key k stands for in a dict of key type t: a str (a rawstr) as
// it is, a member of an enum key type (or its value) its name; ok false when
// k is not one.
func keyOf(t Object, k Object, convert bool) (string, bool) {
	switch x := t.(type) {
	case *Enum:
		if m, ok := x.Member(k); ok {
			return m.Name, true
		}
		return "", false
	case *TypeUnion:
		for _, m := range x.Types {
			if s, ok := keyOf(m, k, convert); ok {
				return s, true
			}
		}
		return "", false
	}
	switch v := k.(type) {
	case Str:
		return string(v), true
	case RawStr:
		return string(v), true
	case *EnumValue:
		return v.Name, true
	}
	if convert {
		return k.ToString(), true
	}
	return "", false
}

func (t *DictOfType) Type() ObjectType { return TDictType }

func (t *DictOfType) Name() string {
	return "DictType[" + typeName(t.Key) + ", " + typeName(t.Value) + "]"
}

func typeName(o Object) string {
	if n, ok := o.(interface{ Name() string }); ok {
		return n.Name()
	}
	return o.ToString()
}

func (t *DictOfType) ToString() string { return t.Name() }
func (t *DictOfType) String() string   { return t.Name() }
func (t *DictOfType) IsFalsy() bool    { return false }

// Equal is structural: the same key and value types.
func (t *DictOfType) Equal(right Object) bool {
	r, ok := right.(*DictOfType)
	return ok && (r == t || (sameType(t.Key, r.Key) && sameType(t.Value, r.Value)))
}

// IndexGet reflects the type: `@key`, `@value`, `@name`.
func (t *DictOfType) IndexGet(_ *VM, index Object) (Object, error) {
	switch index.ToString() {
	case "@key":
		return t.Key, nil
	case "@value":
		return t.Value, nil
	case "@name":
		return Str(t.Name()), nil
	}
	return nil, ErrInvalidIndex.NewError(index.ToString())
}

// dictItems are the items of a dict or a syncDict.
func dictItems(obj Object) (Dict, bool) {
	switch d := obj.(type) {
	case Dict:
		return d, true
	case *SyncDict:
		d.mu.RLock()
		defer d.mu.RUnlock()
		return d.Value, true
	case *TypedDict:
		return d.Value, true
	}
	return nil, false
}

func (t *DictOfType) CanAssign(obj Object) (bool, error) { return t.CanAssignVM(nil, obj) }

// CanAssignVM reports whether obj is a dict (a syncDict, a typed dict) every key
// of which is of K and every value of V.
func (t *DictOfType) CanAssignVM(vm *VM, obj Object) (bool, error) {
	return t.mismatch(vm, obj) == "", nil
}

// mismatch says why obj is not of t ("" when it is).
func (t *DictOfType) mismatch(vm *VM, obj Object) string {
	d, ok := dictItems(obj)
	if !ok {
		return ReprQuote(obj.Type().Name()) + " is not a dict"
	}
	keys := make([]string, 0, len(d))
	for k := range d {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !keyOK(t.Key, k) {
			return "key " + strconv.Quote(k) + " is not of " + ReprQuote(typeName(t.Key))
		}
		if why := t.valueMismatch(vm, d[k]); why != "" {
			return "value " + strconv.Quote(k) + ": " + why
		}
	}
	return ""
}

func (t *DictOfType) valueMismatch(vm *VM, v Object) string {
	if t.Value == TAny {
		return ""
	}
	if vm == nil {
		if ta, ok := t.Value.(TypeAssigner); ok {
			if ok, _ := ta.CanAssign(v); ok {
				return ""
			}
		}
		return ReprQuote(v.Type().Name()) + " is not of " + ReprQuote(typeName(t.Value))
	}
	if _, err := AssignToType(vm, v, t.Value); err != nil {
		return ReprQuote(v.Type().Name()) + " is not of " + ReprQuote(typeName(t.Value))
	}
	return ""
}

// AssignTo is `obj :: DictType[K, V]` (the receiver is `to`).
func (t *DictOfType) AssignTo(vm *VM, obj Object, to TypeAssigner) (Object, error) {
	if to != TypeAssigner(t) {
		return nil, ErrIncompatibleAssign
	}
	if why := t.mismatch(vm, obj); why != "" {
		return nil, ErrIncompatibleAssign.NewErrorf("%s is not assignable to %s: %s", ReprQuote(obj.Type().Name()), ReprQuote(t.Name()), why)
	}
	return obj, nil
}

// assignMismatch says why obj is refused (mismatchSayer).
func (t *DictOfType) assignMismatch(vm *VM, obj Object) string { return t.mismatch(vm, obj) }

// CanCall reports that a dict type is a constructor.
func (t *DictOfType) CanCall() bool { return true }

// Call builds a dict of the type — its entries checked, not converted (`::>`
// converts) —: the named args, and the positional ones — a keyValue, a
// [key, value] pair, a dict or a key-value array (their entries) —, or one
// array alone: its indexes the keys (`i ::> K`), its items the values.
func (t *DictOfType) Call(c Call) (Object, error) {
	d, err := buildDictEntries(c, t.Key, func(k string, v Object) error {
		if !keyOK(t.Key, k) {
			return ErrType.NewErrorf("%s: key %q is not of %s", t.Name(), k, ReprQuote(typeName(t.Key)))
		}
		if why := t.valueMismatch(c.VM, v); why != "" {
			return ErrType.NewErrorf("%s: value %q: %s", t.Name(), k, why)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return d, nil
}

// buildDictEntries are the entries of a dict-type call (DictOfType.Call),
// each checked by check.
func buildDictEntries(c Call, keyType Object, check func(k string, v Object) error) (Dict, error) {
	d := Dict{}
	put := func(k, v Object) error {
		ks, ok := keyOf(keyType, k, false)
		if !ok {
			return ErrType.NewErrorf("a key %s is not of %s", ReprQuote(k.ToString()), ReprQuote(typeName(keyType)))
		}
		if err := check(ks, v); err != nil {
			return err
		}
		d[ks] = v
		return nil
	}
	n := c.Args.Length()
	if n == 1 {
		if a, ok := c.Args.Get(0).(Array); ok {
			// an array alone: its indexes the keys
			for i, v := range a {
				k, ok := keyOf(keyType, Int(i), true)
				if !ok {
					return nil, ErrType.NewErrorf("index %d is not a key of %s", i, ReprQuote(typeName(keyType)))
				}
				if err := check(k, v); err != nil {
					return nil, err
				}
				d[k] = v
			}
			return d, nil
		}
	}
	for i := 0; i < n; i++ {
		switch e := c.Args.Get(i).(type) {
		case *KeyValue:
			if err := put(e.K, e.V); err != nil {
				return nil, err
			}
		case KeyValueArray:
			for _, kv := range e {
				if err := put(kv.K, kv.V); err != nil {
					return nil, err
				}
			}
		case Dict:
			keys := make([]string, 0, len(e))
			for k := range e {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				if err := put(Str(k), e[k]); err != nil {
					return nil, err
				}
			}
		case Array:
			if len(e) != 2 {
				return nil, ErrType.NewErrorf("argument %d: a [key, value] pair, not %d items", i, len(e))
			}
			if err := put(e[0], e[1]); err != nil {
				return nil, err
			}
		default:
			return nil, ErrType.NewErrorf("argument %d: a keyValue, a [key, value] pair, a dict or a key-value array, not %s", i, e.Type().Name())
		}
	}
	var err error
	c.NamedArgs.Walk(func(kv *KeyValue) error {
		if err == nil {
			err = put(kv.K, kv.V)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return d, nil
}

// transform is `obj ::> DictType[K, V]`: each key into K (an enum's member by its
// name or value → its name), each value into V.
func (t *DictOfType) transform(vm *VM, obj Object) (Object, error) {
	var pairs []*KeyValue
	switch x := obj.(type) {
	case KeyValueArray:
		pairs = x
	default:
		d, ok := dictItems(obj)
		if !ok {
			return nil, ErrIncompatibleAssign.NewErrorf("%s is not assignable to %s: not a dict", ReprQuote(obj.Type().Name()), ReprQuote(t.Name()))
		}
		for k, v := range d {
			pairs = append(pairs, &KeyValue{K: Str(k), V: v})
		}
	}
	out := Dict{}
	for _, kv := range pairs {
		k, ok := keyOf(t.Key, kv.K, true)
		if !ok || !keyOK(t.Key, k) {
			return nil, ErrIncompatibleAssign.NewErrorf("%s: key %s is not of %s", t.Name(), ReprQuote(kv.K.ToString()), ReprQuote(typeName(t.Key)))
		}
		v := kv.V
		if t.Value != TAny {
			var err error
			if v, err = AssignToTypeTransform(vm, v, t.Value); err != nil {
				return nil, err
			}
		}
		out[k] = v
	}
	return out, nil
}

// NewDictTypeFunc is the constructor of `DictType`: `DictType(K, V)` (or
// `DictType(V)`) the instance of those types — `DictType[K, V]` from types
// held in variables.
func NewDictTypeFunc(c Call) (Object, error) {
	switch c.Args.Length() {
	case 1:
		return newDictOfType(c.VM, TStr, c.Args.Get(0))
	case 2:
		return newDictOfType(c.VM, c.Args.Get(0), c.Args.Get(1))
	}
	return nil, ErrWrongNumArguments.NewError("DictType(K, V) or DictType(V)")
}

// anyDictReflect are the keys `dict` — `DictType[str, any]` — answers.
func anyDictReflect(index Object) (Object, bool) {
	switch index.ToString() {
	case "@key":
		return TStr, true
	case "@value":
		return TAny, true
	case "@name":
		return Str("DictType[str, any]"), true
	}
	return nil, false
}
