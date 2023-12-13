package gad

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gad-lang/gad/token"
)

// Arg is a struct to destructure arguments from Call object.
type Arg struct {
	Name        string
	Value       Object
	AcceptTypes []ObjectType
	Accept      func(v Object) string
}

// NamedArgVar is a struct to destructure named arguments from Call object.
type NamedArgVar struct {
	Name        string
	Value       Object
	ValueF      func() Object
	AcceptTypes []ObjectType
	Accept      func(v Object) error
}

// NewNamedArgVar creates a new NamedArgVar struct with the given arguments.
func NewNamedArgVar(name string, value Object, types ...ObjectType) *NamedArgVar {
	return &NamedArgVar{Name: name, Value: value, AcceptTypes: types}
}

// NewNamedArgVarF creates a new NamedArgVar struct with the given arguments and value creator func.
func NewNamedArgVarF(name string, value func() Object, types ...ObjectType) *NamedArgVar {
	return &NamedArgVar{Name: name, ValueF: value, AcceptTypes: types}
}

type KeyValue [2]Object

var (
	_ Object       = KeyValue{}
	_ DeepCopier   = KeyValue{}
	_ Copier       = KeyValue{}
	_ LengthGetter = KeyValue{}
)

func (o KeyValue) Type() ObjectType {
	return DetectTypeOf(o)
}

func (o KeyValue) ToString() string {
	var sb strings.Builder
	switch t := o[0].(type) {
	case String:
		if isLetterOrDigitRunes([]rune(t)) {
			sb.WriteString(string(t))
		} else {
			sb.WriteString(strconv.Quote(string(t)))
		}
	default:
		sb.WriteString(o[0].ToString())
	}
	if o[1] != True {
		sb.WriteString("=")
		switch t := o[1].(type) {
		case String:
			sb.WriteString(strconv.Quote(string(t)))
		default:
			sb.WriteString(t.ToString())
		}
	}
	return sb.String()
}

// DeepCopy implements DeepCopier interface.
func (o KeyValue) DeepCopy() Object {
	var cp KeyValue
	for i, v := range o[:] {
		if vv, ok := v.(DeepCopier); ok {
			cp[i] = vv.DeepCopy()
		} else {
			cp[i] = v
		}
	}
	return cp
}

// Copy implements Copier interface.
func (o KeyValue) Copy() Object {
	return KeyValue{o[0], o[1]}
}

// Equal implements Object interface.
func (o KeyValue) Equal(right Object) bool {
	v, ok := right.(KeyValue)
	if !ok {
		return false
	}

	return o[0].Equal(v[0]) && o[1].Equal(v[1])
}

// IsFalsy implements Object interface.
func (o KeyValue) IsFalsy() bool { return o[0] == Nil && o[1] == Nil }

// CanCall implements Object interface.
func (KeyValue) CanCall() bool { return false }

// Call implements Object interface.
func (KeyValue) Call(*NamedArgs, ...Object) (Object, error) {
	return nil, ErrNotCallable
}

// BinaryOp implements Object interface.
func (o KeyValue) BinaryOp(tok token.Token, right Object) (Object, error) {
	switch tok {
	case token.Less, token.LessEq:
		if right == Nil {
			return False, nil
		}
		if kv, ok := right.(KeyValue); ok {
			if o.IsLess(kv) {
				return True, nil
			}
			if tok == token.LessEq {
				return Bool(o.Equal(kv)), nil
			}
			return False, nil
		}
	case token.Greater, token.GreaterEq:
		if right == Nil {
			return True, nil
		}

		if tok == token.GreaterEq {
			if o.Equal(right) {
				return True, nil
			}
		}

		if kv, ok := right.(KeyValue); ok {
			return Bool(!o.IsLess(kv)), nil
		}
	}
	return nil, NewOperandTypeError(
		tok.String(),
		o.Type().Name(),
		right.Type().Name())
}

func (o KeyValue) IsLess(other KeyValue) bool {
	if o.Key().ToString() < other.Key().ToString() {
		return true
	}
	if bo, _ := o.Value().(BinaryOperatorHandler); bo != nil {
		v, _ := bo.BinaryOp(token.Less, other.Value())
		return v == nil || !v.IsFalsy()
	}
	return false
}

// CanIterate implements Object interface.
func (KeyValue) CanIterate() bool { return true }

// Iterate implements Iterable interface.
func (o KeyValue) Iterate(*VM) Iterator {
	return &ArrayIterator{V: o[:]}
}

// Len implements LengthGetter interface.
func (o KeyValue) Len() int {
	return len(o)
}

func (o KeyValue) Key() Object {
	return o[0]
}

func (o KeyValue) Value() Object {
	return o[1]
}

func (o KeyValue) IndexGet(_ *VM, index Object) (value Object, err error) {
	value = Nil
	switch t := index.(type) {
	case String:
		switch t {
		case "k":
			return o[0], nil
		case "v":
			return o[1], nil
		case "array":
			return Array(o[:]), nil
		}
	case Int:
		switch t {
		case 0, 1:
			return o[t], nil
		}
	case Uint:
		switch t {
		case 0, 1:
			return o[t], nil
		}
	default:
		err = NewArgumentTypeError(
			"1st",
			"string|int|uint",
			index.Type().Name(),
		)
		return
	}
	err = ErrInvalidIndex
	return
}

type KeyValueArray []KeyValue

var (
	_ Object       = KeyValueArray{}
	_ DeepCopier   = KeyValueArray{}
	_ Copier       = KeyValueArray{}
	_ LengthGetter = KeyValueArray{}
	_ Sorter       = KeyValueArray{}
	_ KeysGetter   = KeyValueArray{}
	_ ItemsGetter  = KeyValueArray{}
)

func (o KeyValueArray) Type() ObjectType {
	return DetectTypeOf(o)
}

func (o KeyValueArray) Array() (ret Array) {
	ret = make(Array, len(o))
	for i, v := range o {
		ret[i] = v
	}
	return
}

func (o KeyValueArray) Map() (ret Dict) {
	ret = make(Dict, len(o))
	for _, v := range o {
		ret[v.Key().ToString()] = v.Value()
	}
	return
}

func (o KeyValueArray) ToString() string {
	var sb strings.Builder
	sb.WriteString("(;")
	last := len(o) - 1

	for i, v := range o {
		sb.WriteString(v.ToString())
		if i != last {
			sb.WriteString(", ")
		}
	}

	sb.WriteString(")")
	return sb.String()
}

// DeepCopy implements DeepCopier interface.
func (o KeyValueArray) DeepCopy() Object {
	cp := make(KeyValueArray, len(o))
	for i, v := range o {
		cp[i] = v.DeepCopy().(KeyValue)
	}
	return cp
}

// Copy implements Copier interface.
func (o KeyValueArray) Copy() Object {
	cp := make(KeyValueArray, len(o))
	copy(cp, o)
	return cp
}

// IndexGet implements Object interface.
func (o KeyValueArray) IndexGet(_ *VM, index Object) (Object, error) {
	switch v := index.(type) {
	case Int:
		idx := int(v)
		if idx >= 0 && idx < len(o) {
			return o[v], nil
		}
		return nil, ErrIndexOutOfBounds
	case Uint:
		idx := int(v)
		if idx >= 0 && idx < len(o) {
			return o[v], nil
		}
		return nil, ErrIndexOutOfBounds
	case String:
		switch v {
		case "arrays":
			ret := make(Array, len(o))
			for i, v := range o {
				ret[i] = Array(v[:])
			}
			return ret, nil
		case "map":
			return o.Map(), nil
		default:
			return nil, ErrInvalidIndex.NewError(string(v))
		}
	}
	return nil, NewIndexTypeError("int|uint", index.Type().Name())
}

// Equal implements Object interface.
func (o KeyValueArray) Equal(right Object) bool {
	v, ok := right.(KeyValueArray)
	if !ok {
		return false
	}

	if len(o) != len(v) {
		return false
	}

	for i := range o {
		if !o[i].Equal(v[i]) {
			return false
		}
	}
	return true
}

// IsFalsy implements Object interface.
func (o KeyValueArray) IsFalsy() bool { return len(o) == 0 }

// CanCall implements Object interface.
func (KeyValueArray) CanCall() bool { return false }

// Call implements Object interface.
func (KeyValueArray) Call(*NamedArgs, ...Object) (Object, error) {
	return nil, ErrNotCallable
}

func (o KeyValueArray) AppendArray(arr ...Array) (KeyValueArray, error) {
	var (
		i  = len(o)
		nl = i
		o2 KeyValueArray
	)

	for _, arr := range arr {
		nl += len(arr)
	}

	o2 = make(KeyValueArray, nl)
	copy(o2, o)

	for _, arr := range arr {
		for _, v := range arr {
			switch na := v.(type) {
			case KeyValue:
				o2[i] = na
				i++
			case Array:
				if len(na) == 2 {
					o2[i] = KeyValue{na[0], na[1]}
					i++
				} else {
					return nil, NewIndexValueTypeError("keyValue|[2]array",
						fmt.Sprintf("[%d]%s", len(na), v.Type().Name()))
				}
			default:
				return nil, NewIndexTypeError("keyValue", v.Type().Name())
			}
		}
	}
	return o2, nil
}

func (o KeyValueArray) AppendMap(m Dict) KeyValueArray {
	var (
		i   = len(o)
		arr = make(KeyValueArray, i+len(m))
	)

	copy(arr, o)

	for k, v := range m {
		arr[i] = KeyValue{String(k), v}
		i++
	}

	return arr
}

func (o KeyValueArray) Append(arg ...KeyValue) KeyValueArray {
	if len(o) == 0 {
		return arg
	}
	var (
		i   = len(o)
		arr = make(KeyValueArray, i+len(arg))
	)

	copy(arr, o)
	copy(arr[i:], arg)
	return arr
}

func (o KeyValueArray) AppendObject(obj Object) (KeyValueArray, error) {
	switch v := obj.(type) {
	case KeyValue:
		return append(o, v), nil
	case Dict:
		return o.AppendMap(v), nil
	case KeyValueArray:
		return o.Append(v...), nil
	case *NamedArgs:
		return o.Append(v.UnreadPairs()...), nil
	case Array:
		if o, err := o.AppendArray(v); err != nil {
			return nil, err
		} else {
			return o, nil
		}
	default:
		return nil, NewIndexTypeError("array|map|keyValue|keyValueArray", v.Type().Name())
	}
}

// BinaryOp implements Object interface.
func (o KeyValueArray) BinaryOp(tok token.Token, right Object) (Object, error) {
	switch tok {
	case token.Add:
		return o.AppendObject(right)
	case token.Less, token.LessEq:
		if right == Nil {
			return False, nil
		}
	case token.Greater, token.GreaterEq:
		if right == Nil {
			return True, nil
		}
	}
	return nil, NewOperandTypeError(
		tok.String(),
		o.Type().Name(),
		right.Type().Name())
}

func (o KeyValueArray) Sort(vm *VM, less CallerObject) (_ Object, err error) {
	if less == nil {
		sort.Slice(o, func(i, j int) bool {
			return o[i].IsLess(o[j])
		})
	} else {
		var (
			args   = Array{Nil, Nil}
			caller VMCaller
		)

		if caller, err = NewInvoker(vm, less).Caller(Args{args}, nil); err != nil {
			return
		}

		sort.Slice(o, func(i, j int) bool {
			args[0] = o[i]
			args[1] = o[j]
			ret, _ := caller.Call()
			return !ret.IsFalsy()
		})
	}
	return o, err
}

func (o KeyValueArray) SortReverse() (Object, error) {
	sort.Slice(o, func(i, j int) bool {
		return !o[i].IsLess(o[j])
	})
	return o, nil
}

func (o KeyValueArray) Get(keys ...Object) Object {
	if len(keys) == 0 {
		return Array{}
	}

	var e KeyValue
	if len(keys) > 1 {
		var arr Array
	keys:
		for _, key := range keys {
			for l := len(o); l > 0; l-- {
				e = o[l-1]
				if e[0].Equal(key) {
					arr = append(arr, e[1])
					continue keys
				}
			}
			arr = append(arr, Nil)
		}
		return arr
	}
	for l := len(o); l > 0; l-- {
		e = o[l-1]
		if e[0].Equal(keys[0]) {
			return e[1]
		}
	}
	return Nil
}

func (o KeyValueArray) Delete(keys ...Object) Object {
	if len(keys) == 0 {
		return o
	}

	var ret KeyValueArray
l:
	for _, kv := range o {
		for _, k := range keys {
			if kv[0].Equal(k) {
				continue l
			}
		}
		ret = append(ret, kv)
	}

	return ret
}

func (o KeyValueArray) CallName(name string, c Call) (_ Object, err error) {
	switch name {
	case "flag":
		if err = c.Args.CheckLen(1); err != nil {
			return
		}
		keyArg := c.Args.Get(0)
		var e KeyValue
		for l := len(o); l > 0; l-- {
			e = o[l-1]
			if e[0].Equal(keyArg) && !e.Value().IsFalsy() {
				return True, nil
			}
		}
		return False, nil
	case "get":
		return o.Get(c.Args.Values()...), nil
	case "delete":
		return o.Delete(c.Args.Values()...), nil
	case "values":
		if c.Args.Len() == 0 {
			return o.Values(), nil
		}

		var ret Array

		c.Args.Walk(func(i int, arg Object) any {
			for _, e := range o {
				if e[0].Equal(arg) {
					ret = append(ret, e.Value())
				}
			}
			return nil
		})
		return ret, nil
	case "sort":
		if err = c.Args.CheckMaxLen(1); err != nil {
			return
		}
		switch c.Args.Len() {
		case 0:
		case 1:
			switch t := c.Args.Get(0).(type) {
			case Bool:
				if t {
					o2 := make(KeyValueArray, len(o))
					copy(o2, o)
					o = o2
				}
			default:
				return nil, NewArgumentTypeError(
					"1st",
					"bool",
					t.Type().Name(),
				)
			}
		}
		return o.Sort(c.VM, nil)
	case "sortReverse":
		if err = c.Args.CheckMaxLen(1); err != nil {
			return
		}
		switch c.Args.Len() {
		case 0:
		case 1:
			switch t := c.Args.Get(0).(type) {
			case Bool:
				if t {
					o2 := make(KeyValueArray, len(o))
					copy(o2, o)
					o = o2
				}
			default:
				return nil, NewArgumentTypeError(
					"1st",
					"bool",
					t.Type().Name(),
				)
			}
		}
		return o.SortReverse()
	default:
		return nil, ErrInvalidIndex.NewError(name)
	}
}

// CanIterate implements Object interface.
func (KeyValueArray) CanIterate() bool { return true }

// Iterate implements Iterable interface.
func (o KeyValueArray) Iterate(*VM) Iterator {
	return &KeyValueArrayIterator{V: o}
}

// Len implements LengthGetter interface.
func (o KeyValueArray) Len() int {
	return len(o)
}

func (o KeyValueArray) Items() KeyValueArray {
	return o
}

func (o KeyValueArray) Keys() (arr Array) {
	arr = make(Array, len(o))
	for i, v := range o {
		arr[i] = v[0]
	}
	return
}

func (o KeyValueArray) Values() (arr Array) {
	arr = make(Array, len(o))
	for i, v := range o {
		arr[i] = v[1]
	}
	return
}

// KeyValueArrayIterator represents an iterator for the array.
type KeyValueArrayIterator struct {
	V KeyValueArray
	i int
}

var _ Iterator = (*KeyValueArrayIterator)(nil)

// Next implements Iterator interface.
func (it *KeyValueArrayIterator) Next() bool {
	it.i++
	return it.i-1 < len(it.V)
}

// Key implements Iterator interface.
func (it *KeyValueArrayIterator) Key() Object {
	return Int(it.i - 1)
}

// Value implements Iterator interface.
func (it *KeyValueArrayIterator) Value() (Object, error) {
	i := it.i - 1
	if i > -1 && i < len(it.V) {
		return it.V[i], nil
	}
	return Nil, nil
}

type KeyValueArrays []KeyValueArray

func (KeyValueArrays) Type() ObjectType {
	return TKeyValueArrays
}

func (o KeyValueArrays) Array() (ret Array) {
	ret = make(Array, len(o))
	for i, v := range o {
		ret[i] = v
	}
	return
}

func (o KeyValueArrays) ToString() string {
	var sb strings.Builder
	sb.WriteString("[")
	last := len(o) - 1

	for i, v := range o {
		sb.WriteString(v.ToString())
		if i != last {
			sb.WriteString(", ")
		}
	}

	sb.WriteString("]")
	return sb.String()
}

// DeepCopy implements DeepCopier interface.
func (o KeyValueArrays) DeepCopy() Object {
	cp := make(KeyValueArrays, len(o))
	for i, v := range o {
		cp[i] = v.DeepCopy().(KeyValueArray)
	}
	return cp
}

// Copy implements Copier interface.
func (o KeyValueArrays) Copy() Object {
	cp := make(KeyValueArrays, len(o))
	copy(cp, o)
	return cp
}

// IndexGet implements Object interface.
func (o KeyValueArrays) IndexGet(_ *VM, index Object) (Object, error) {
	switch v := index.(type) {
	case Int:
		idx := int(v)
		if idx >= 0 && idx < len(o) {
			return o[v], nil
		}
		return nil, ErrIndexOutOfBounds
	case Uint:
		idx := int(v)
		if idx >= 0 && idx < len(o) {
			return o[v], nil
		}
		return nil, ErrIndexOutOfBounds
	}
	return nil, NewIndexTypeError("int|uint", index.Type().Name())
}

// Equal implements Object interface.
func (o KeyValueArrays) Equal(right Object) bool {
	v, ok := right.(KeyValueArrays)
	if !ok {
		return false
	}

	if len(o) != len(v) {
		return false
	}

	for i := range o {
		if !o[i].Equal(v[i]) {
			return false
		}
	}
	return true
}

// IsFalsy implements Object interface.
func (o KeyValueArrays) IsFalsy() bool { return len(o) == 0 }

// CanCall implements Object interface.
func (KeyValueArrays) CanCall() bool { return false }

// Call implements Object interface.
func (KeyValueArrays) Call(*NamedArgs, ...Object) (Object, error) {
	return nil, ErrNotCallable
}

// BinaryOp implements Object interface.
func (o KeyValueArrays) BinaryOp(tok token.Token, right Object) (Object, error) {
	switch tok {
	case token.Less, token.LessEq:
		if right == Nil {
			return False, nil
		}
	case token.Greater, token.GreaterEq:
		if right == Nil {
			return True, nil
		}
	}
	return nil, NewOperandTypeError(
		tok.String(),
		o.Type().Name(),
		right.Type().Name())
}

// CanIterate implements Object interface.
func (KeyValueArrays) CanIterate() bool { return true }

// Iterate implements Iterable interface.
func (o KeyValueArrays) Iterate(*VM) Iterator {
	return &NamedArgArraysIterator{V: o}
}

// Len implements LengthGetter interface.
func (o KeyValueArrays) Len() int {
	return len(o)
}
func (o KeyValueArrays) CallName(name string, c Call) (Object, error) {
	switch name {
	case "merge":
		l := len(o)
		switch l {
		case 0, 1:
			return o, nil
		default:
			var ret KeyValueArray
			for _, arr := range o {
				ret.Append(arr...)
			}
			return ret, nil
		}
	default:
		return nil, ErrInvalidIndex.NewError(name)
	}
}

// NamedArgArraysIterator represents an iterator for the array.
type NamedArgArraysIterator struct {
	V KeyValueArrays
	i int
}

var _ Iterator = (*NamedArgArraysIterator)(nil)

// Next implements Iterator interface.
func (it *NamedArgArraysIterator) Next() bool {
	it.i++
	return it.i-1 < len(it.V)
}

// Key implements Iterator interface.
func (it *NamedArgArraysIterator) Key() Object {
	return Int(it.i - 1)
}

// Value implements Iterator interface.
func (it *NamedArgArraysIterator) Value() (Object, error) {
	i := it.i - 1
	if i > -1 && i < len(it.V) {
		return it.V[i], nil
	}
	return Nil, nil
}

type NamedArgs struct {
	sources KeyValueArrays
	m       Dict
	ready   Dict
}

func NewNamedArgs(pairs ...KeyValueArray) *NamedArgs {
	return &NamedArgs{sources: pairs}
}

func (o *NamedArgs) Contains(key string) bool {
	if _, ok := o.ready[key]; ok {
		return false
	}
	o.check()
	_, ok := o.m[key]
	return ok
}

func (o *NamedArgs) Add(obj Object) error {
	arr, err := KeyValueArray{}.AppendObject(obj)
	if err != nil {
		return err
	}
	o.sources = append(o.sources, arr)
	return nil
}

func (o *NamedArgs) CallName(name string, c Call) (Object, error) {
	switch name {
	case "get":
		arg := &Arg{AcceptTypes: []ObjectType{TString}}
		if err := c.Args.Destructure(arg); err != nil {
			return nil, err
		}
		return o.GetValue(string(arg.Value.(String))), nil
	default:
		return Nil, ErrInvalidIndex.NewError(name)
	}
}

func (o *NamedArgs) Type() ObjectType {
	return TNamedArgs
}

func (o *NamedArgs) Join() KeyValueArray {
	switch len(o.sources) {
	case 0:
		return KeyValueArray{}
	case 1:
		return o.sources[0]
	default:
		ret := make(KeyValueArray, 0)
		for _, t := range o.sources {
			ret = append(ret, t...)
		}
		return ret
	}
}

func (o *NamedArgs) ToString() string {
	if len(o.ready) == 0 {
		return o.Join().ToString()
	}
	return o.UnreadPairs().ToString()
}

func (o *NamedArgs) BinaryOp(tok token.Token, right Object) (Object, error) {
	if right == Nil {
		switch tok {
		case token.Add:
			if err := o.Add(right); err != nil {
				return nil, err
			}
			return o, nil
		case token.Less, token.LessEq:
			return False, nil
		case token.Greater, token.GreaterEq:
			return True, nil
		}
	}

	return nil, NewOperandTypeError(
		tok.String(),
		o.Type().Name(),
		right.Type().Name())
}

func (o *NamedArgs) IsFalsy() bool {
	for _, s := range o.sources {
		if len(s) > 0 {
			return false
		}
	}
	return true
}

func (o *NamedArgs) Equal(right Object) bool {
	v, ok := right.(*NamedArgs)
	if !ok {
		return false
	}
	if len(o.sources) != len(v.sources) {
		return false
	}
	for i, p := range o.sources {
		if !p.Equal(v.sources[i]) {
			return false
		}
	}
	return true
}

func (o *NamedArgs) Call(c Call) (Object, error) {
	arg := &Arg{AcceptTypes: []ObjectType{TString}}
	if err := c.Args.Destructure(arg); err != nil {
		return nil, err
	}
	return o.GetValue(string(arg.Value.(String))), nil
}

func (o *NamedArgs) Iterate(vm *VM) Iterator {
	return o.Join().Iterate(vm)
}

func (o *NamedArgs) CanIterate() bool {
	return true
}

func (o *NamedArgs) UnReady() *NamedArgs {
	return &NamedArgs{
		sources: KeyValueArrays{
			o.UnreadPairs(),
		},
	}
}

func (o *NamedArgs) Ready() (arr KeyValueArray) {
	if len(o.ready) == 0 {
		return
	}

	o.Walk(func(na KeyValue) error {
		if _, ok := o.ready[na.Key().ToString()]; ok {
			arr = append(arr, na)
		}
		return nil
	})
	return
}

func (o *NamedArgs) IndexGet(_ *VM, index Object) (value Object, err error) {
	switch t := index.(type) {
	case String:
		switch t {
		case "src":
			return o.sources, nil
		case "dict":
			return o.Dict(), nil
		case "unread":
			return o.UnReady(), nil
		case "ready":
			return o.Ready(), nil
		case "array":
			return o.Join(), nil
		case "readyNames":
			return o.ready.Keys(), nil
		default:
			return Nil, ErrInvalidIndex.NewError(string(t))
		}
	default:
		err = NewArgumentTypeError(
			"1st",
			"string",
			index.Type().Name(),
		)
		return
	}
}

func (o *NamedArgs) check() {
	if o.m == nil {
		o.m = Dict{}
		o.ready = Dict{}

		for i := len(o.sources) - 1; i >= 0; i-- {
			for _, v := range o.sources[i] {
				o.m[v.Key().ToString()] = v[1]
			}
		}
	}
}

// GetValue Must return value from key
func (o *NamedArgs) GetValue(key string) (val Object) {
	if val = o.GetValueOrNil(key); val == nil {
		val = Nil
	}
	return
}

// GetPassedValue Get passed value
func (o *NamedArgs) GetPassedValue(key string) (val Object) {
	o.Walk(func(na KeyValue) error {
		if na.Key().ToString() == key {
			val = na[1]
			return io.EOF
		}
		return nil
	})
	return
}

// GetValueOrNil Must return value from key
func (o *NamedArgs) GetValueOrNil(key string) (val Object) {
	o.check()

	if val = o.m[key]; val != nil {
		delete(o.m, key)
		o.ready[key] = nil
		return
	}
	return nil
}

func (o *NamedArgs) unreadDict() Dict {
	o.check()
	args := o.m.Copy().(Dict)
	for k := range o.ready {
		delete(args, k)
	}
	return args
}

// Get destructure.
// Return errors:
// - ArgumentTypeError if type check of arg is fail.
// - UnexpectedNamedArg if have unexpected arg.
func (o *NamedArgs) Get(dst ...*NamedArgVar) (err error) {
	args := o.unreadDict()

	if err = o.getOneOf(args, dst...); err != nil {
		return
	}

	for key := range args {
		return ErrUnexpectedNamedArg.NewError(strconv.Quote(key))
	}
	return nil
}

// GetOne get one value.
// Return errors:
// - ArgumentTypeError if type check of arg is fail.
func (o *NamedArgs) GetOne(dst ...*NamedArgVar) (err error) {
	return o.getOneOf(o.unreadDict(), dst...)
}

func (o *NamedArgs) getOneOf(args Dict, dst ...*NamedArgVar) (err error) {
read:
	for i, d := range dst {
		if v, ok := args[d.Name]; ok && v != Nil {
			if d.Accept != nil {
				if err = d.Accept(v); err != nil {
					return NewArgumentTypeError(
						d.Name+"["+strconv.Itoa(i)+"]st",
						err.Error(),
						v.Type().Name(),
					)
				}
			} else if len(d.AcceptTypes) > 0 {
				for _, t := range d.AcceptTypes {
					if t.Equal(v.Type()) {
						d.Value = v
						delete(args, d.Name)
						continue read
					}
				}

				var s = make([]string, len(d.AcceptTypes))
				for i, acceptType := range d.AcceptTypes {
					s[i] = acceptType.ToString()
				}
				return NewArgumentTypeError(
					d.Name+"["+strconv.Itoa(i)+"]st",
					strings.Join(s, "|"),
					v.Type().Name(),
				)
			}

			d.Value = v
			delete(args, d.Name)
		}

		if d.ValueF != nil {
			d.Value = d.ValueF()
		}
	}
	return
}

// GetVar destructure and return others.
// Returns ArgumentTypeError if type check of arg is fail.
func (o *NamedArgs) GetVar(dst ...*NamedArgVar) (args Dict, err error) {
	o.check()
	args = o.m
dst:
	for i, d := range dst {
		if v, ok := args[d.Name]; ok && v != Nil {
			if len(d.AcceptTypes) == 0 {
				d.Value = v
				delete(args, d.Name)
				continue
			}

			for _, t := range d.AcceptTypes {
				if t.Equal(v.Type()) {
					d.Value = v
					delete(args, d.Name)
					continue dst
				}
			}

			var s = make([]string, len(d.AcceptTypes))
			for i, acceptType := range d.AcceptTypes {
				s[i] = acceptType.ToString()
			}
			return nil, NewArgumentTypeError(
				strconv.Itoa(i)+"st",
				strings.Join(s, "|"),
				v.Type().Name(),
			)
		}

		if d.ValueF != nil {
			d.Value = d.ValueF()
		}
	}

	return
}

// Empty return if is empty
func (o *NamedArgs) Empty() bool {
	return o.IsFalsy()
}

// Dict return unread keys as Dict
func (o *NamedArgs) Dict() (ret Dict) {
	o.check()
	return o.m.Copy().(Dict)
}

func (o *NamedArgs) AllDict() (ret Dict) {
	o.check()
	return o.m
}

func (o *NamedArgs) UnreadPairs() (ret KeyValueArray) {
	if len(o.ready) == 0 {
		o.Walk(func(na KeyValue) error {
			ret = append(ret, na)
			return nil
		})
		return
	}
	o.Walk(func(na KeyValue) error {
		if _, ok := o.ready[na.Key().ToString()]; !ok {
			ret = append(ret, na)
		}
		return nil
	})
	return
}

// Walk pass over all pairs and call `cb` function.
// if `cb` function returns any error, stop iterator and return then.
func (o *NamedArgs) Walk(cb func(na KeyValue) error) (err error) {
	o.check()
	for _, arr := range o.sources {
		for _, item := range arr {
			if err = cb(item); err != nil {
				return
			}
		}
	}
	return
}

func (o *NamedArgs) CheckNames(accept ...string) error {
	return o.Walk(func(na KeyValue) error {
		for _, name := range accept {
			if name == na.Key().ToString() {
				return nil
			}
		}
		return ErrUnexpectedNamedArg.NewError(strconv.Quote(na.Key().ToString()))
	})
}

func (o *NamedArgs) CheckNamesFromSet(set map[string]int) error {
	if set == nil {
		return nil
	}
	return o.Walk(func(na KeyValue) error {
		if _, ok := set[na.Key().ToString()]; !ok {
			return ErrUnexpectedNamedArg.NewError(strconv.Quote(na.Key().ToString()))
		}
		return nil
	})
}

func (o *NamedArgs) Copy() Object {
	var cp NamedArgs
	cp.sources = make(KeyValueArrays, len(o.sources))
	for i, s := range o.sources {
		cp.sources[i] = s.Copy().(KeyValueArray)
	}
	if o.m != nil {
		cp.m = o.m.Copy().(Dict)
	}
	return &cp
}

func (o NamedArgs) DeepCopy() Object {
	if o.m != nil {
		o.m = o.m.DeepCopy().(Dict)
	}
	o.sources = o.sources.DeepCopy().(KeyValueArrays)
	return &o
}

func isLetterOrDigit(ch rune) bool {
	return '0' <= ch && ch <= '9' || 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_' ||
		ch >= utf8.RuneSelf && (unicode.IsLetter(ch) || unicode.IsDigit(ch))
}

func isLetterOrDigitRunes(chs []rune) bool {
	for _, r := range chs {
		if !isLetterOrDigit(r) {
			return false
		}
	}
	return true
}
