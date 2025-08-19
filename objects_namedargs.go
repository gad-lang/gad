package gad

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/gad-lang/gad/repr"
	"github.com/gad-lang/gad/runehelper"
	"github.com/gad-lang/gad/token"
)

type TypeAssertionHandler func(v Object) bool

type TypeAssertionHandlers map[string]TypeAssertionHandler

type TypeAssertion struct {
	Types    []ObjectType
	Handlers TypeAssertionHandlers
}

func NewTypeAssertion(handlers TypeAssertionHandlers, types ...ObjectType) *TypeAssertion {
	return &TypeAssertion{Types: types, Handlers: handlers}
}

func TypeAssertionFromTypes(types ...ObjectType) *TypeAssertion {
	return &TypeAssertion{Types: types}
}

func TypeAssertionFlag() *TypeAssertion {
	return TypeAssertionFromTypes(TFlag)
}

func (a *TypeAssertion) AcceptHandler(name string, handler TypeAssertionHandler) *TypeAssertion {
	if a == nil {
		*a = TypeAssertion{}
	}
	if a.Handlers == nil {
		a.Handlers = TypeAssertionHandlers{}
	}
	a.Handlers[name] = handler
	return a
}

func (a *TypeAssertion) Accept(value Object) (expectedNames string) {
	if a == nil || len(a.Handlers) == 0 && len(a.Types) == 0 {
		return
	}

	var (
		names []string
		vt    = value.Type()
	)

	for _, t := range a.Types {
		if t.Equal(vt) {
			return
		}
		names = append(names, t.Name())
	}

	for name, handler := range a.Handlers {
		if handler(value) {
			return
		}
		names = append(names, name)
	}

	expectedNames = strings.Join(names, "|")
	return
}

func (a *TypeAssertion) AcceptType(value ObjectType) (expectedNames string) {
	if a == nil || len(a.Handlers) == 0 && len(a.Types) == 0 {
		return
	}
	var names []string

	for _, t := range a.Types {
		if t.Equal(value) {
			return
		}
		names = append(names, t.Name())
	}

	for name, handler := range a.Handlers {
		if handler(value) {
			return
		}
		names = append(names, name)
	}

	expectedNames = strings.Join(names, "|")
	return
}

// Arg is a struct to destructure arguments from Call object.
type Arg struct {
	Name  string
	Value Object
	*TypeAssertion
}

type ArgValue struct {
	Arg   Arg
	Value any
}

// NamedArgVar is a struct to destructure named arguments from Call object.
type NamedArgVar struct {
	Name   string
	Value  Object
	ValueF func() Object
	*TypeAssertion
}

type KeyValue struct {
	K, V Object
}

var (
	_ Object         = &KeyValue{}
	_ DeepCopier     = &KeyValue{}
	_ Copier         = &KeyValue{}
	_ IndexGetSetter = &KeyValue{}
)

func (o *KeyValue) IndexSet(vm *VM, index, value Object) error {
	switch t := index.(type) {
	case Str:
		switch t {
		case "v":
			o.V = value
			return nil
		case "k":
			o.K = value
			return nil
		}
	}

	ret, err := ToRepr(vm, value)
	if err != nil {
		return err
	}
	return ErrInvalidIndex.NewError(ret.ToString())
}

func (o *KeyValue) Type() ObjectType {
	return DetectTypeOf(o)
}

func (o *KeyValue) ToString() string {
	var sb strings.Builder
	switch t := o.K.(type) {
	case Str:
		if runehelper.IsLetterOrDigitRunes([]rune(t)) {
			sb.WriteString(string(t))
		} else {
			sb.WriteString(strconv.Quote(string(t)))
		}
	default:
		sb.WriteString(o.K.ToString())
	}
	if o.V != Yes {
		sb.WriteString("=")
		switch t := o.V.(type) {
		case Str:
			sb.WriteString(t.Quoted())
		case RawStr:
			sb.WriteString(t.Quoted())
		case *KeyValue:
			sb.WriteByte('[')
			sb.WriteString(t.ToString())
			sb.WriteByte(']')
		default:
			sb.WriteString(t.ToString())
		}
	}
	return sb.String()
}

func (o *KeyValue) Repr(vm *VM) (_ string, err error) {
	var (
		sb    strings.Builder
		do    = vm.Builtins.ArgsInvoker(BuiltinRepr, Call{VM: vm})
		repro Object
	)

	if repro, err = do(o.K); err != nil {
		return
	}
	sb.WriteString(repr.QuotePrefix)
	sb.WriteString(o.Type().Name())
	sb.WriteString(":")
	sb.WriteString(string(repro.(Str)))

	if o.V != Yes {
		sb.WriteString("=")
		if repro, err = do(o.K); err != nil {
			return
		}
		sb.WriteString(string(repro.(Str)))
	}
	sb.WriteString(repr.QuoteSufix)
	return sb.String(), nil
}

// DeepCopy implements DeepCopier interface.
func (o KeyValue) DeepCopy(vm *VM) (_ Object, err error) {
	if o.V, err = DeepCopy(nil, o.V); err != nil {
		return
	}
	return &o, nil
}

// Copy implements Copier interface.
func (o KeyValue) Copy() Object {
	return &o
}

// Equal implements Object interface.
func (o *KeyValue) Equal(right Object) bool {
	v, ok := right.(*KeyValue)
	if !ok {
		return false
	}

	return o.K.Equal(v.K) && o.V.Equal(v.V)
}

// IsFalsy implements Object interface.
func (o *KeyValue) IsFalsy() bool { return o.K == Nil && o.V == Nil }

// CanCall implements Object interface.
func (KeyValue) CanCall() bool { return false }

// Call implements Object interface.
func (KeyValue) Call(*NamedArgs, ...Object) (Object, error) {
	return nil, ErrNotCallable
}

// BinaryOp implements Object interface.
func (o *KeyValue) BinaryOp(vm *VM, tok token.Token, right Object) (Object, error) {
	switch tok {
	case token.Less, token.LessEq:
		if right == Nil {
			return False, nil
		}
		if kv, ok := right.(*KeyValue); ok {
			if o.IsLess(vm, kv) {
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

		if kv, ok := right.(*KeyValue); ok {
			return Bool(!o.IsLess(vm, kv)), nil
		}
	}
	return nil, NewOperandTypeError(
		tok.String(),
		o.Type().Name(),
		right.Type().Name())
}

func (o *KeyValue) IsLess(vm *VM, other *KeyValue) bool {
	if o.K.ToString() < other.K.ToString() {
		return true
	}
	if bo, _ := o.V.(BinaryOperatorHandler); bo != nil {
		v, _ := bo.BinaryOp(vm, token.Less, other.V)
		return v == nil || !v.IsFalsy()
	}
	return false
}

// CanIterate implements Object interface.
func (KeyValue) CanIterate() bool { return true }

func (o *KeyValue) IndexGet(vm *VM, index Object) (value Object, err error) {
	value = Nil
	switch t := index.(type) {
	case Str:
		switch t {
		case "k":
			return o.K, nil
		case "v":
			return o.V, nil
		case "array":
			return Array{o.K, o.V}, nil
		}
	default:
		err = NewArgumentTypeError(
			"1st",
			"str",
			index.Type().Name(),
		)
		return
	}
	err = ErrInvalidIndex.NewError(index.ToString())
	return
}

type KeyValueArray []*KeyValue

var (
	_ Object       = KeyValueArray{}
	_ DeepCopier   = KeyValueArray{}
	_ Copier       = KeyValueArray{}
	_ LengthGetter = KeyValueArray{}
	_ Sorter       = KeyValueArray{}
	_ KeysGetter   = KeyValueArray{}
	_ ItemsGetter  = KeyValueArray{}
)

func (o *KeyValueArray) Add(_ *VM, items ...Object) (err error) {
	for i, item := range items {
		if kv, _ := item.(*KeyValue); kv != nil {
			*o = append(*o, kv)
		} else {
			return NewArgumentTypeErrorT(fmt.Sprint(i+1), item.Type(), TKeyValue)
		}
	}
	return
}

func (o KeyValueArray) Append(_ *VM, items ...Object) (this Object, err error) {
	return o, o.Add(nil, items...)
}

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

func (o KeyValueArray) ToDict() (ret Dict) {
	ret = make(Dict, len(o))
	for _, v := range o {
		ret[v.K.ToString()] = v.V
	}
	return
}

func (o KeyValueArray) MDict() (ret Dict) {
	ret = make(Dict)
	for _, v := range o {
		k := v.K.ToString()
		if prev, ok := ret[k]; ok {
			ret[k] = append(prev.(Array), v.V)
		} else {
			ret[k] = Array{v.V}
		}
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

func (o KeyValueArray) Repr(vm *VM) (_ string, err error) {
	var (
		sb    strings.Builder
		do    = vm.Builtins.ArgsInvoker(BuiltinRepr, Call{VM: vm})
		repro Object
	)
	sb.WriteString(repr.QuotePrefix)
	sb.WriteString(o.Type().Name())
	sb.WriteString(":")
	sb.WriteString("(;")
	last := len(o) - 1

	for i, v := range o {
		if repro, err = do(v); err != nil {
			return
		}
		sb.WriteString(repro.ToString())
		if i != last {
			sb.WriteString(", ")
		}
	}

	sb.WriteString(")")
	sb.WriteString(repr.QuoteSufix)
	return sb.String(), nil
}

// DeepCopy implements DeepCopier interface.
func (o KeyValueArray) DeepCopy(vm *VM) (r Object, err error) {
	cp := make(KeyValueArray, len(o))
	for i, v := range o {
		if r, err = v.DeepCopy(vm); err != nil {
			return
		}
		cp[i] = r.(*KeyValue)
	}
	return cp, nil
}

// Copy implements Copier interface.
func (o KeyValueArray) Copy() Object {
	cp := make(KeyValueArray, len(o))
	copy(cp, o)
	return cp
}

func (o KeyValueArray) ToArray() (ret Array) {
	ret = make(Array, len(o))
	for i, v := range o {
		ret[i] = Array{v.K, v.V}
	}
	return
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
			case *KeyValue:
				o2[i] = na
				i++
			case Array:
				if len(na) == 2 {
					o2[i] = &KeyValue{na[0], na[1]}
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
		arr[i] = &KeyValue{Str(k), v}
		i++
	}

	return arr
}

func (o KeyValueArray) AddItems(arg ...*KeyValue) KeyValueArray {
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
	case *KeyValue:
		return append(o, v), nil
	case Dict:
		return o.AppendMap(v), nil
	case KeyValueArray:
		return o.AddItems(v...), nil
	case *NamedArgs:
		return o.AddItems(v.UnreadPairs()...), nil
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
func (o KeyValueArray) BinaryOp(_ *VM, tok token.Token, right Object) (Object, error) {
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
			return o[i].IsLess(vm, o[j])
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

func (o KeyValueArray) SortReverse(vm *VM) (Object, error) {
	sort.Slice(o, func(i, j int) bool {
		return !o[i].IsLess(vm, o[j])
	})
	return o, nil
}

func (o KeyValueArray) Get(keys ...Object) Object {
	if len(keys) == 0 {
		return Array{}
	}

	var e *KeyValue
	if len(keys) > 1 {
		var arr Array
	keys:
		for _, key := range keys {
			for l := len(o); l > 0; l-- {
				e = o[l-1]
				if e.K.Equal(key) {
					arr = append(arr, e.V)
					continue keys
				}
			}
			arr = append(arr, Nil)
		}
		return arr
	}
	for l := len(o); l > 0; l-- {
		e = o[l-1]
		if e.K.Equal(keys[0]) {
			return e.V
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
			if kv.K.Equal(k) {
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
		var e *KeyValue
		for l := len(o); l > 0; l-- {
			e = o[l-1]
			if e.K.Equal(keyArg) && !e.V.IsFalsy() {
				return True, nil
			}
		}
		return False, nil
	case "get":
		return o.Get(c.Args.Values()...), nil
	case "delete":
		return o.Delete(c.Args.Values()...), nil
	case "values":
		if c.Args.Length() == 0 {
			return o.Values(), nil
		}

		var ret Array

		c.Args.Walk(func(i int, arg Object) any {
			for _, e := range o {
				if e.K.Equal(arg) {
					ret = append(ret, e.V)
				}
			}
			return nil
		})
		return ret, nil
	case "sort":
		if err = c.Args.CheckMaxLen(1); err != nil {
			return
		}
		switch c.Args.Length() {
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
		switch c.Args.Length() {
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
		return o.SortReverse(c.VM)
	default:
		return nil, ErrInvalidIndex.NewError(name)
	}
}

// CanIterate implements Object interface.
func (KeyValueArray) CanIterate() bool { return true }

// Length implements LengthGetter interface.
func (o KeyValueArray) Length() int {
	return len(o)
}

func (o KeyValueArray) Items(*VM) (KeyValueArray, error) {
	return o, nil
}

func (o KeyValueArray) Keys() (arr Array) {
	arr = make(Array, len(o))
	for i, v := range o {
		arr[i] = v.K
	}
	return
}

func (o KeyValueArray) Values() (arr Array) {
	arr = make(Array, len(o))
	for i, v := range o {
		arr[i] = v.V
	}
	return
}

type KeyValueArrays []KeyValueArray

func (o KeyValueArrays) Repr(vm *VM) (_ string, err error) {
	return ArrayRepr(o.Type().Name(), vm, len(o), func(i int) Object {
		return o[i]
	})
}

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
func (o KeyValueArrays) DeepCopy(vm *VM) (_ Object, err error) {
	var (
		cp = make(KeyValueArrays, len(o))
		vo Object
	)
	for i, v := range o {
		if vo, err = DeepCopy(vm, v); err != nil {
			return
		}
		cp[i] = vo.(KeyValueArray)
	}
	return cp, nil
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
func (o KeyValueArrays) BinaryOp(_ *VM, tok token.Token, right Object) (Object, error) {
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

// Length implements LengthGetter interface.
func (o KeyValueArrays) Length() int {
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
				ret.AddItems(arr...)
			}
			return ret, nil
		}
	default:
		return nil, ErrInvalidIndex.NewError(name)
	}
}

var EmptyNamedArgs = &NamedArgs{ro: true}

type NamedArgs struct {
	ro      bool
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
	if o.ro {
		return ErrNotWriteable
	}
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
		arg := &Arg{TypeAssertion: TypeAssertionFromTypes(TStr)}
		if err := c.Args.Destructure(arg); err != nil {
			return nil, err
		}
		return o.GetValue(string(arg.Value.(Str))), nil
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

func (o *NamedArgs) BinaryOp(_ *VM, tok token.Token, right Object) (Object, error) {
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
	var (
		name = &Arg{
			TypeAssertion: TypeAssertionFromTypes(TStr),
		}
		checkType = &Arg{
			TypeAssertion: TypeAssertionFromTypes(TBool),
		}
	)
	if c.Args.Length() == 2 {
		if err := c.Args.Destructure(name, checkType); err != nil {
			return nil, err
		}
		val := o.GetValue(string(name.Value.(Str)))
		return val, nil
	} else if err := c.Args.Destructure(name); err != nil {
		return nil, err
	}
	return o.GetValue(string(name.Value.(Str))), nil
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

	o.Walk(func(na *KeyValue) error {
		if _, ok := o.ready[na.K.ToString()]; ok {
			arr = append(arr, na)
		}
		return nil
	})
	return
}

func (o *NamedArgs) IndexGet(vm *VM, index Object) (value Object, err error) {
	switch t := index.(type) {
	case Str:
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
			return o.AllDict().IndexGet(vm, t)
		}
	default:
		err = NewArgumentTypeError(
			"1st",
			"str",
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
				o.m[v.K.ToString()] = v.V
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

// MustGetValue Must return value from key but not takes as read
func (o *NamedArgs) MustGetValue(key string) (val Object) {
	if val = o.MustGetValueOrNil(key); val == nil {
		val = Nil
	}
	return
}

// GetPassedValue Get passed value
func (o *NamedArgs) GetPassedValue(key string) (val Object) {
	o.Walk(func(na *KeyValue) error {
		if na.K.ToString() == key {
			val = na.V
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
		if !o.ro {
			delete(o.m, key)
			o.ready[key] = nil
		}
		return
	}
	return nil
}

// MustGetValueOrNil Must return value from key nut not takes as read
func (o *NamedArgs) MustGetValueOrNil(key string) (val Object) {
	o.check()

	if val = o.m[key]; val != nil {
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
	for i, d := range dst {
		if v, ok := args[d.Name]; ok && v != Nil {
			if expectedNames := d.TypeAssertion.Accept(v); expectedNames != "" {
				return NewArgumentTypeError(
					d.Name+"["+strconv.Itoa(i)+"]st",
					expectedNames,
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

	for i, d := range dst {
		if v, ok := args[d.Name]; ok && v != Nil {
			delete(args, d.Name)

			if expectedNames := d.TypeAssertion.Accept(v); expectedNames != "" {
				err = NewArgumentTypeError(
					d.Name+"["+strconv.Itoa(i)+"]st",
					expectedNames,
					v.Type().Name(),
				)
				return nil, err
			}
			d.Value = v
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
		o.Walk(func(na *KeyValue) error {
			ret = append(ret, na)
			return nil
		})
		return
	}
	o.Walk(func(na *KeyValue) error {
		if _, ok := o.ready[na.K.ToString()]; !ok {
			ret = append(ret, na)
		}
		return nil
	})
	return
}

// Walk pass over all pairs and call `cb` function.
// if `cb` function returns any error, stop iterator and return then.
func (o *NamedArgs) Walk(cb func(na *KeyValue) error) (err error) {
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
	return o.Walk(func(na *KeyValue) error {
		for _, name := range accept {
			if name == na.K.ToString() {
				return nil
			}
		}
		return ErrUnexpectedNamedArg.NewError(strconv.Quote(na.K.ToString()))
	})
}

func (o *NamedArgs) CheckNamesFromSet(set map[string]int) error {
	if set == nil {
		return nil
	}
	return o.Walk(func(na *KeyValue) error {
		if _, ok := set[na.K.ToString()]; !ok {
			return ErrUnexpectedNamedArg.NewError(strconv.Quote(na.K.ToString()))
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

func (o NamedArgs) DeepCopy(vm *VM) (_ Object, err error) {
	var r Object
	if o.m != nil {
		if r, err = o.m.DeepCopy(vm); err != nil {
			return
		}
		o.m = r.(Dict)
	}
	if r, err = o.sources.DeepCopy(vm); err != nil {
		return
	}
	o.sources = r.(KeyValueArrays)
	return &o, nil
}
