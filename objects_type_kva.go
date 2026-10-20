package gad

import "sync"

// KeyValueArrayType[K, V] (issue #20): the type of a key-value array whose keys
// are of K (any type: a key-value array's keys are) and whose values are of V.
// Structural, as DictType[K, V] (#19); `KeyValueArrayType[V]` is
// `KeyValueArrayType[any, V]`, `keyValueArray` is `KeyValueArrayType[any, any]`.
// The pairs keep their order and their duplicates; a reset (`key=no`) is no
// value: never checked.

// KVAOfType is an instance of the generic KeyValueArrayType.
type KVAOfType struct {
	Key, Value Object
}

var (
	_ Object        = (*KVAOfType)(nil)
	_ IndexGetter   = (*KVAOfType)(nil)
	_ TypeAssigner  = (*KVAOfType)(nil)
	_ vmCanAssigner = (*KVAOfType)(nil)
)

var kvaTypes sync.Map

// kvaTypeOf is `KeyValueArrayType[index]`: [K, V], [V] or V.
func kvaTypeOf(vm *VM, index Object) (Object, error) {
	key, val := Object(TAny), index
	if a, ok := index.(Array); ok {
		switch len(a) {
		case 1:
			val = a[0]
		case 2:
			key, val = a[0], a[1]
		default:
			return nil, ErrType.NewErrorf("KeyValueArrayType: [K, V] or [V], not %d type arguments", len(a))
		}
	}
	return newKVAOfType(vm, key, val)
}

func newKVAOfType(vm *VM, key, val Object) (Object, error) {
	for _, t := range []Object{key, val} {
		if _, ok := t.(TypeAssigner); !ok && t != TAny {
			return nil, ErrType.NewErrorf("KeyValueArrayType: %s is not a type", ReprQuote(t.ToString()))
		}
	}
	if sameType(key, TAny) && sameType(val, TAny) {
		if vm != nil {
			return vm.Builtins.builtins.Objects[BuiltinKeyValueArray], nil
		}
		return BuiltinObjects[BuiltinKeyValueArray], nil
	}
	id := typeIdentity(key) + "," + typeIdentity(val)
	if t, ok := kvaTypes.Load(id); ok {
		return t.(*KVAOfType), nil
	}
	t, _ := kvaTypes.LoadOrStore(id, &KVAOfType{Key: key, Value: val})
	return t.(*KVAOfType), nil
}

func (t *KVAOfType) Type() ObjectType { return TKeyValueArrayType }

func (t *KVAOfType) Name() string {
	return "KeyValueArrayType[" + typeName(t.Key) + ", " + typeName(t.Value) + "]"
}

func (t *KVAOfType) ToString() string { return t.Name() }
func (t *KVAOfType) String() string   { return t.Name() }
func (t *KVAOfType) IsFalsy() bool    { return false }

func (t *KVAOfType) Equal(right Object) bool {
	r, ok := right.(*KVAOfType)
	return ok && (r == t || (sameType(t.Key, r.Key) && sameType(t.Value, r.Value)))
}

func (t *KVAOfType) IndexGet(_ *VM, index Object) (Object, error) {
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

// kvaPairs are the pairs of a key-value array (a typed one's too).
func kvaPairs(obj Object) (KeyValueArray, bool) {
	switch x := obj.(type) {
	case KeyValueArray:
		return x, true
	case *TypedKeyValueArray:
		return x.Value, true
	}
	return nil, false
}

// pairMismatch says why the pair at position i is not of the types ("" when it
// is).
func pairMismatch(vm *VM, key, val Object, kv *KeyValue, i int) string {
	if key != TAny {
		if _, err := AssignToType(vm, kv.K, key); err != nil {
			return "pair " + itoa(i) + ": key " + ReprQuote(kv.K.ToString()) + " is not of " + ReprQuote(typeName(key))
		}
	}
	if val != TAny && !IsReset(kv.V) {
		if _, err := AssignToType(vm, kv.V, val); err != nil {
			return "pair " + itoa(i) + " (" + kv.K.ToString() + "): " + ReprQuote(kv.V.Type().Name()) + " is not of " + ReprQuote(typeName(val))
		}
	}
	return ""
}

func (t *KVAOfType) mismatch(vm *VM, obj Object) string {
	pairs, ok := kvaPairs(obj)
	if !ok {
		return ReprQuote(obj.Type().Name()) + " is not a key-value array"
	}
	for i, kv := range pairs {
		if why := pairMismatch(vm, t.Key, t.Value, kv, i); why != "" {
			return why
		}
	}
	return ""
}

func (t *KVAOfType) CanAssign(obj Object) (bool, error) { return t.CanAssignVM(nil, obj) }

func (t *KVAOfType) CanAssignVM(vm *VM, obj Object) (bool, error) {
	if vm == nil {
		pairs, ok := kvaPairs(obj)
		return ok && len(pairs) == 0 || ok && t.Key == TAny && t.Value == TAny, nil
	}
	return t.mismatch(vm, obj) == "", nil
}

func (t *KVAOfType) AssignTo(vm *VM, obj Object, to TypeAssigner) (Object, error) {
	if to != TypeAssigner(t) {
		return nil, ErrIncompatibleAssign
	}
	if why := t.mismatch(vm, obj); why != "" {
		return nil, ErrIncompatibleAssign.NewErrorf("%s is not assignable to %s: %s", ReprQuote(obj.Type().Name()), ReprQuote(t.Name()), why)
	}
	return obj, nil
}

func (t *KVAOfType) assignMismatch(vm *VM, obj Object) string { return t.mismatch(vm, obj) }

func (t *KVAOfType) CanCall() bool { return true }

// Call builds a key-value array of the type, its pairs checked (in order,
// duplicates kept): the named args, and the positional ones — a keyValue, a
// [key, value] pair, a key-value array or a dict —, or one array alone: its
// indexes the keys.
func (t *KVAOfType) Call(c Call) (Object, error) {
	return buildKVAPairs(c, t.Name(), t.Key, t.Value)
}

func buildKVAPairs(c Call, name string, key, val Object) (KeyValueArray, error) {
	var out KeyValueArray
	add := func(k, v Object) error {
		kv := &KeyValue{K: k, V: v}
		if why := pairMismatch(c.VM, key, val, kv, len(out)); why != "" {
			return ErrType.NewErrorf("%s: %s", name, why)
		}
		out = append(out, kv)
		return nil
	}
	n := c.Args.Length()
	if n == 1 {
		if a, ok := c.Args.Get(0).(Array); ok {
			for i, v := range a {
				var k Object = Int(i)
				if key != TAny {
					var err error
					if k, err = AssignToTypeTransform(c.VM, Int(i), key); err != nil {
						return nil, err
					}
				}
				if err := add(k, v); err != nil {
					return nil, err
				}
			}
			return out, nil
		}
	}
	for i := 0; i < n; i++ {
		switch e := c.Args.Get(i).(type) {
		case *KeyValue:
			if err := add(e.K, e.V); err != nil {
				return nil, err
			}
		case KeyValueArray:
			for _, kv := range e {
				if err := add(kv.K, kv.V); err != nil {
					return nil, err
				}
			}
		case Dict:
			for _, k := range sortedKeys(e) {
				if err := add(Str(k), e[k]); err != nil {
					return nil, err
				}
			}
		case Array:
			if len(e) != 2 {
				return nil, ErrType.NewErrorf("argument %d: a [key, value] pair, not %d items", i, len(e))
			}
			if err := add(e[0], e[1]); err != nil {
				return nil, err
			}
		default:
			return nil, ErrType.NewErrorf("argument %d: a keyValue, a [key, value] pair, a key-value array or a dict, not %s", i, e.Type().Name())
		}
	}
	var err error
	c.NamedArgs.Walk(func(kv *KeyValue) error {
		if err == nil {
			err = add(kv.K, kv.V)
		}
		return nil
	})
	return out, err
}

// transform is `obj ::> KeyValueArrayType[K, V]`: each key into K, each value
// into V (a reset kept as it is).
func (t *KVAOfType) transform(vm *VM, obj Object) (Object, error) {
	var pairs KeyValueArray
	if p, ok := kvaPairs(obj); ok {
		pairs = p
	} else if d, ok := dictItems(obj); ok {
		for _, k := range sortedKeys(d) {
			pairs = append(pairs, &KeyValue{K: Str(k), V: d[k]})
		}
	} else {
		return nil, ErrIncompatibleAssign.NewErrorf("%s is not assignable to %s: not a key-value array", ReprQuote(obj.Type().Name()), ReprQuote(t.Name()))
	}
	out := make(KeyValueArray, len(pairs))
	for i, kv := range pairs {
		k, v := kv.K, kv.V
		var err error
		if t.Key != TAny {
			if k, err = AssignToTypeTransform(vm, k, t.Key); err != nil {
				return nil, err
			}
		}
		if t.Value != TAny && !IsReset(v) {
			if v, err = AssignToTypeTransform(vm, v, t.Value); err != nil {
				return nil, err
			}
		}
		out[i] = &KeyValue{K: k, V: v}
	}
	return out, nil
}

// NewKVATypeFunc is the constructor of `KeyValueArrayType`: `(K, V)` or `(V)`.
func NewKVATypeFunc(c Call) (Object, error) {
	switch c.Args.Length() {
	case 1:
		return newKVAOfType(c.VM, TAny, c.Args.Get(0))
	case 2:
		return newKVAOfType(c.VM, c.Args.Get(0), c.Args.Get(1))
	}
	return nil, ErrWrongNumArguments.NewError("KeyValueArrayType(K, V) or KeyValueArrayType(V)")
}

func anyKVAReflect(index Object) (Object, bool) {
	switch index.ToString() {
	case "@key", "@value":
		return TAny, true
	case "@name":
		return Str("KeyValueArrayType[any, any]"), true
	}
	return nil, false
}

func itoa(i int) string { return Int(i).ToString() }
