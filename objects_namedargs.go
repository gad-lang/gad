package gad

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/gad-lang/gad/runehelper"
	"github.com/gad-lang/gad/token"
)

type TypeAssertionHandler func(v Object) bool

type TypeAssertionHandlers map[string]TypeAssertionHandler

type TypeAsssertionOption func(a TypeAssertionHandlers)

func WithCallable() TypeAsssertionOption {
	return func(a TypeAssertionHandlers) {
		a["callable"] = Callable
	}
}

func WithMethodCaller() TypeAsssertionOption {
	return func(a TypeAssertionHandlers) {
		a["methodCaller"] = func(v Object) (ok bool) {
			_, ok = v.(MethodCaller)
			return
		}
	}
}

func WithMethodAdder() TypeAsssertionOption {
	return func(a TypeAssertionHandlers) {
		a["methodAdder"] = func(v Object) (ok bool) {
			_, ok = v.(MethodAdder)
			return
		}
	}
}

func WithIsAssignable(types ...ObjectType) TypeAsssertionOption {
	tarr := ObjectTypeArray(types)
	return func(a TypeAssertionHandlers) {
		a[ObjectTypes(types).String()] = func(v Object) bool {
			return tarr.Assign(v.Type())
		}
	}
}

func WithArray() TypeAsssertionOption {
	return func(a TypeAssertionHandlers) {
		a["array"] = func(v Object) (ok bool) {
			_, ok = v.(Array)
			return
		}
	}
}

func WithRawCallable() TypeAsssertionOption {
	return func(a TypeAssertionHandlers) {
		a["rawcallable"] = IsFunction
	}
}

func WithFlag() TypeAsssertionOption {
	return func(a TypeAssertionHandlers) {
		a["flag"] = func(v Object) bool {
			switch v.(type) {
			case Flag, Bool:
				return true
			default:
				return false
			}
		}
	}
}

func TypeAssertions(opt ...TypeAsssertionOption) TypeAssertionHandlers {
	ta := make(TypeAssertionHandlers)
	for _, option := range opt {
		option(ta)
	}
	return ta
}

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

func (a *TypeAssertion) Options(opt ...TypeAsssertionOption) *TypeAssertion {
	if a.Handlers == nil {
		a.Handlers = make(TypeAssertionHandlers)
	}
	for _, o := range opt {
		o(a.Handlers)
	}
	return a
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

type KeyValue struct {
	K, V Object
}

var (
	_ Object         = (*KeyValue)(nil)
	_ DeepCopier     = (*KeyValue)(nil)
	_ Copier         = (*KeyValue)(nil)
	_ IndexGetSetter = (*KeyValue)(nil)
	_ Printabler     = (*KeyValue)(nil)
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
	return TKeyValue
}

func (o *KeyValue) Print(state *PrinterState) (err error) {
	var open, close []byte
	prev := state.stack.PrevValue()

	escape := func(v Object) bool {
		switch v.(type) {
		case *KeyValue, Array:
			return true
		default:
			return false
		}
	}

	if prev == nil || escape(prev) {
		open, close = []byte{'['}, []byte{']'}
	}

	state.Write(open)
	defer state.Write(close)

	switch t := o.K.(type) {
	case Str:
		state.PrintKey(t)
	default:
		Print(state, o.K)
	}

	if o.V != Yes {
		state.WriteString("=")
		state.QuoteNextStr(0)
		state.Print(o.V)
	}
	return nil
}

func (o *KeyValue) ToString() string {
	var sb strings.Builder
	switch t := o.K.(type) {
	case Str:
		if runehelper.IsIdentifierOrDigitRunes([]rune(t)) {
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

// A *KeyValue orders after nil and against another *KeyValue by IsLess; these
// implement the comparison ObjectWith{Op}BinOperator interfaces.
func (o *KeyValue) BinOpLess(vm *VM, right Object) (Object, error) {
	if right == Nil {
		return False, nil
	}
	if kv, ok := right.(*KeyValue); ok {
		return Bool(o.IsLess(vm, kv)), nil
	}
	return nil, NewOperandTypeError(token.Less.String(), o.Type().Name(), right.Type().Name())
}

func (o *KeyValue) BinOpLessEq(vm *VM, right Object) (Object, error) {
	if right == Nil {
		return False, nil
	}
	if kv, ok := right.(*KeyValue); ok {
		return Bool(o.IsLess(vm, kv) || o.Equal(kv)), nil
	}
	return nil, NewOperandTypeError(token.LessEq.String(), o.Type().Name(), right.Type().Name())
}

// Greater and GreaterEq both report !IsLess (matching the original behavior).
func (o *KeyValue) BinOpGreater(vm *VM, right Object) (Object, error) {
	return o.greater(token.Greater, vm, right)
}

func (o *KeyValue) BinOpGreaterEq(vm *VM, right Object) (Object, error) {
	return o.greater(token.GreaterEq, vm, right)
}

func (o *KeyValue) greater(tok token.Token, vm *VM, right Object) (Object, error) {
	if right == Nil {
		return True, nil
	}
	if kv, ok := right.(*KeyValue); ok {
		return Bool(!o.IsLess(vm, kv)), nil
	}
	return nil, NewOperandTypeError(tok.String(), o.Type().Name(), right.Type().Name())
}

func (o *KeyValue) IsLess(vm *VM, other *KeyValue) bool {
	if o.K.ToString() < other.K.ToString() {
		return true
	}
	v, _ := BinaryOp(vm, token.Less, o.V, other.V)
	return v == nil || !v.IsFalsy()
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
	_ Object       = (*KeyValueArray)(nil)
	_ DeepCopier   = (*KeyValueArray)(nil)
	_ Copier       = (*KeyValueArray)(nil)
	_ LengthGetter = (*KeyValueArray)(nil)
	_ Sorter       = (*KeyValueArray)(nil)
	_ KeysGetter   = (*KeyValueArray)(nil)
	_ ItemsGetter  = (*KeyValueArray)(nil)
	_ Printabler   = (*KeyValueArray)(nil)
)

func (o *KeyValueArray) Append(vm *VM, items ...Object) (err error) {
	cp := *o
	for _, item := range items {
		switch t := item.(type) {
		case *KeyValue:
			cp = append(cp, t)
		case Array:
			err = cp.Append(vm, t...)
		case *NamedParamsVar:
			err = ItemsOfCb(vm, &NamedArgs{}, func(kv *KeyValue) error {
				cp = append(cp, kv)
				return nil
			}, t.Object)
		default:
			err = ItemsOfCb(vm, &NamedArgs{}, func(kv *KeyValue) error {
				cp = append(cp, kv)
				return nil
			}, t)
		}
		if err != nil {
			return
		}
	}
	*o = cp
	return
}

func (o KeyValueArray) AppendObjects(vm *VM, items ...Object) (this Object, err error) {
	err = ItemsOfCb(vm, &NamedArgs{}, func(kv *KeyValue) error {
		o = append(o, kv)
		return nil
	}, items...)
	return o, err
}

func (o KeyValueArray) Type() ObjectType {
	return TKeyValueArray
}

func (o KeyValueArray) UpdateIndexSetter(out StringIndexSetter) {
	for _, v := range o {
		out.Set(v.K.ToString(), v.V)
	}
}

func (o KeyValueArray) Print(state *PrinterState) (err error) {
	if state.IsRepr {
		defer state.WrapRepr(o)()
	}
	return state.PrintValues(len(o), []byte{'(', ';'}, []byte{')'}, []byte{','},
		func(i int) (Object, error) {
			return o[i], nil
		})
}

func (o KeyValueArray) ToString() string {
	return string(MustToStr(nil, o))
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

func (o *KeyValueArray) AppendArrayOfPairs(arr Array) error {
	kva := make(KeyValueArray, len(*o)+len(arr))
	copy(kva, *o)
	i := len(*o)

	for j, v := range arr {
		switch na := v.(type) {
		case *KeyValue:
			kva[i] = na
			i++
		default:
			return NewIndexValueTypeError(strconv.Itoa(j), "keyValue", v.Type().Name())
		}
	}
	*o = kva
	return nil
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

	for x, arr := range arr {
		for j, v := range arr {
			switch na := v.(type) {
			case *KeyValue:
				o2[i] = na
				i++
			case Array:
				if len(na) == 2 {
					o2[i] = &KeyValue{na[0], na[1]}
					i++
				} else {
					return nil, NewIndexValueTypeError(fmt.Sprintf("%d[%d]", x, j), "keyValue|[2]array",
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

// BinOpAdd appends right's entries (ObjectWithAddBinOperator); the comparison
// operators order the array after nil and are otherwise unsupported.
func (o KeyValueArray) BinOpAdd(vm *VM, right Object) (Object, error) {
	return o.AppendObjects(vm, right)
}

func (o KeyValueArray) BinOpLess(_ *VM, right Object) (Object, error) {
	return binCmpAfterNil(token.Less, o, right)
}

func (o KeyValueArray) BinOpLessEq(_ *VM, right Object) (Object, error) {
	return binCmpAfterNil(token.LessEq, o, right)
}

func (o KeyValueArray) BinOpGreater(_ *VM, right Object) (Object, error) {
	return binCmpAfterNil(token.Greater, o, right)
}

func (o KeyValueArray) BinOpGreaterEq(_ *VM, right Object) (Object, error) {
	return binCmpAfterNil(token.GreaterEq, o, right)
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

// BinOpIn implements the `in` operator (ObjectWithInBinOperator): reports
// whether v is a key of the key-value array (`v in kva`).
func (o KeyValueArray) BinOpIn(_ *VM, v Object) (Object, error) {
	for _, e := range o {
		if e.K.Equal(v) {
			return True, nil
		}
	}
	return False, nil
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

func (o KeyValueArray) Items(_ *VM, cb ItemsGetterCallback) (err error) {
	for i, value := range o {
		if err = cb(i, value); err != nil {
			return
		}
	}
	return
}

func (o KeyValueArray) ToArray() (arr Array) {
	arr = make(Array, len(o))
	for i, value := range o {
		arr[i] = value
	}
	return
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

var (
	_ Object                    = (*KeyValueArrays)(nil)
	_ DeepCopier                = (*KeyValueArrays)(nil)
	_ Copier                    = (*KeyValueArrays)(nil)
	_ Printabler                = (*KeyValueArrays)(nil)
	_ IndexGetter               = (*KeyValueArrays)(nil)
	_ ObjectWithLessBinOperator = (*KeyValueArrays)(nil)
)

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

func (o KeyValueArrays) Print(state *PrinterState) (err error) {
	return state.PrintArray(len(o),
		func(i int) (Object, error) {
			return o[i], nil
		})
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

// KeyValueArrays orders after nil and is otherwise not comparable.
func (o KeyValueArrays) BinOpLess(_ *VM, right Object) (Object, error) {
	return binCmpAfterNil(token.Less, o, right)
}

func (o KeyValueArrays) BinOpLessEq(_ *VM, right Object) (Object, error) {
	return binCmpAfterNil(token.LessEq, o, right)
}

func (o KeyValueArrays) BinOpGreater(_ *VM, right Object) (Object, error) {
	return binCmpAfterNil(token.Greater, o, right)
}

func (o KeyValueArrays) BinOpGreaterEq(_ *VM, right Object) (Object, error) {
	return binCmpAfterNil(token.GreaterEq, o, right)
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

// NamedArgVar is a struct to destructure named arguments from Call object.
type NamedArgVar struct {
	Name   string
	Value  Object
	ValueF func() Object
	Do     func(value Object) error
	*TypeAssertion
}

func (v *NamedArgVar) IsFalsy() bool {
	return v.Value == nil || v.Value.IsFalsy()
}

var (
	_ Object           = (*NamedArgs)(nil)
	_ NameCallerObject = (*NamedArgs)(nil)
	_ IndexGetter      = (*NamedArgs)(nil)
)

// NamedArgs holds the keyword (name=value) arguments of a call.
//
// Values are lazily materialised from sources into the m map on first access
// (see check). By default reading a value with GetValue/GetValueOrNil
// *consumes* it — the key is removed from m and recorded in ready — so a
// function only sees each named argument once and leftover names can be
// detected. When the NamedArgs is read-only (see WithReadOnly), reads do not
// consume, so the same NamedArgs can be reused across several calls; this is
// useful when reusing one instance in a loop and mutating a backing Dict
// between calls (Dict.ToNamedArgs keeps a live reference to that Dict in m).
type NamedArgs struct {
	ro      bool           // when true, reads do not consume values
	sources KeyValueArrays // original name=value pairs (last wins on duplicate)
	m       Dict           // materialised, not-yet-consumed values
	ready   Dict           // keys already consumed
}

// NewNamedArgs returns a NamedArgs from the given name=value pair lists. Later
// pairs take precedence over earlier ones for duplicate keys.
func NewNamedArgs(pairs ...KeyValueArray) *NamedArgs {
	return &NamedArgs{sources: pairs}
}

// WithReadOnly sets the read-only flag and returns o. While read-only, reads do
// not consume values and Add returns ErrNotWriteable.
func (o *NamedArgs) WithReadOnly(v bool) *NamedArgs {
	o.ro = v
	return o
}

// SetReadOnly sets the read-only flag (see WithReadOnly).
func (o *NamedArgs) SetReadOnly(v bool) {
	o.ro = v
}

// IsReadOnly reports whether reads consume values.
func (o *NamedArgs) IsReadOnly() bool {
	return o.ro
}

func (o *NamedArgs) Contains(key string) bool {
	if _, ok := o.ready[key]; ok {
		return false
	}
	o.check()
	_, ok := o.m[key]
	return ok
}

func (o *NamedArgs) Add(obj Object) (err error) {
	if o.ro {
		return ErrNotWriteable
	}
	var arr KeyValueArray
	if err = arr.Append(nil, obj); err != nil {
		return
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
	return string(MustToStr(nil, o))
}

// BinOpAdd merges a nil right operand (a no-op add); NamedArgs otherwise orders
// after nil and is not comparable to other operands.
func (o *NamedArgs) BinOpAdd(_ *VM, right Object) (Object, error) {
	if right == Nil {
		if err := o.Add(right); err != nil {
			return nil, err
		}
		return o, nil
	}
	return nil, NewOperandTypeError(token.Add.String(), o.Type().Name(), right.Type().Name())
}

func (o *NamedArgs) BinOpLess(_ *VM, right Object) (Object, error) {
	return binCmpAfterNil(token.Less, o, right)
}

func (o *NamedArgs) BinOpLessEq(_ *VM, right Object) (Object, error) {
	return binCmpAfterNil(token.LessEq, o, right)
}

func (o *NamedArgs) BinOpGreater(_ *VM, right Object) (Object, error) {
	return binCmpAfterNil(token.Greater, o, right)
}

func (o *NamedArgs) BinOpGreaterEq(_ *VM, right Object) (Object, error) {
	return binCmpAfterNil(token.GreaterEq, o, right)
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

// check lazily materialises sources into m (and initialises ready) on first
// use. It iterates sources from last to first so that, with later-wins
// precedence, the first write of a key is the effective one. It is a no-op once
// m is set — including when m is supplied directly (e.g. by Dict.ToNamedArgs),
// in which case that Dict is used live.
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

// GetValueOrNil returns the value for key, or nil if absent. Unless the
// NamedArgs is read-only, the key is consumed (removed from m and recorded in
// ready) so it is reported only once.
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

// GetDo destructure and call dst.Do if is valid.
// Return errors:
// - ArgumentTypeError if type check of arg is fail.
// - UnexpectedNamedArg if have unexpected arg.
// - other error returned by dst.Do.
func (o *NamedArgs) GetDo(dst ...*NamedArgVar) (err error) {
	args := o.unreadDict()

	if err = o.getOneOf(args, dst...); err != nil {
		return
	}

	for key := range args {
		return ErrUnexpectedNamedArg.NewError(strconv.Quote(key))
	}

	for _, d := range dst {
		if d.Value != nil && d.Do != nil {
			if err = d.Do(d.Value); err != nil {
				return
			}
		}
	}

	return nil
}

// GetOne get one value.
// Return errors:
// - ArgumentTypeError if type check of arg is fail.
func (o *NamedArgs) GetOne(dst ...*NamedArgVar) (err error) {
	return o.getOneOf(o.unreadDict(), dst...)
}

// GetOneDo get one value and call dst.Do handler if is valid.
// Return errors:
// - ArgumentTypeError if type check of arg is fail.
// - other error returned by dst.Do.
func (o *NamedArgs) GetOneDo(dst ...*NamedArgVar) (err error) {
	if err = o.getOneOf(o.unreadDict(), dst...); err != nil {
		return
	}
	for _, d := range dst {
		if d.Value != nil && d.Do != nil {
			if err = d.Do(d.Value); err != nil {
				return
			}
		}
	}
	return
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

func (o *NamedArgs) Items(_ *VM, cb ItemsGetterCallback) (err error) {
	var i int
	if len(o.ready) == 0 {
		return o.Walk(func(kv *KeyValue) (err error) {
			err = cb(i, kv)
			i++
			return
		})
	}
	return o.Walk(func(kv *KeyValue) (err error) {
		if _, ok := o.ready[kv.K.ToString()]; !ok {
			err = cb(i, kv)
			i++
		}
		return
	})
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

func (o *NamedArgs) Print(state *PrinterState) (err error) {
	defer state.WrapRepr(o)()
	if len(o.ready) == 0 {
		return o.Join().Print(state)
	}
	return o.UnreadPairs().Print(state)
}

type NamedParamsVar struct {
	Object
}
