package gad

import (
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
	"runtime/debug"
	"sort"
	"strings"
	_ "text/template"
	_ "unsafe"

	"github.com/gad-lang/gad/repr"
	"github.com/gad-lang/gad/zeroer"
)

type ReflectValuer interface {
	Object
	Copier
	NameCallerObject
	ToIterfaceConverter
	Value() reflect.Value
	GetRValue() *ReflectValue
	GetRType() *ReflectType
	IsPtr() bool
	IsNil() bool
}

type ReflectValueOptions struct {
	ToStr    func() string
	ItValuer func(value interface{}) (Object, error)
}

func NewReflectValue(v any, opts ...*ReflectValueOptions) (ReflectValuer, error) {
	if rv, _ := v.(ReflectValuer); rv != nil {
		return rv, nil
	}

	var (
		rv  = reflect.ValueOf(v)
		rvr interface {
			ReflectValuer
			Init()
		}
		isnill bool
		opt    *ReflectValueOptions
	)

	for _, opt = range opts {
	}
	if opt == nil {
		opt = &ReflectValueOptions{}
	}

	func() {
		defer func() {
			recover()
		}()
		_, isnill = zeroer.IsNilValue(rv)
	}()

	rv = indirectInterface(rv)

	var (
		t   = NewReflectType(rv.Type())
		ptr = rv.Kind() == reflect.Ptr
	)

	if ptr {
		if isnill {
			return nil, nil
		}
		rv = rv.Elem()
	} else if t.RType.Kind() != reflect.Func {
		ptrv := reflect.New(t.RType).Elem()
		ptrv.Set(rv)
		rv = ptrv
	}

	orv := ReflectValue{RType: t, RValue: rv, ptr: ptr, nil: isnill, Options: opt}
	switch rv.Kind() {
	case reflect.Struct:
		rvr = &ReflectStruct{ReflectValue: orv}
	case reflect.Map:
		rvr = &ReflectMap{orv}
	case reflect.Slice:
		rvr = &ReflectSlice{ReflectArray{orv}}
	case reflect.Array:
		rvr = &ReflectArray{orv}
	case reflect.Func:
		return &ReflectFunc{orv}, nil
	default:
		return &orv, nil
	}
	rvr.Init()
	return rvr, nil
}

func MustNewReflectValue(v any, opts ...*ReflectValueOptions) ReflectValuer {
	rv, err := NewReflectValue(v, opts...)
	if err != nil {
		panic(err)
	}
	return rv
}

type ReflectValue struct {
	RType                     *ReflectType
	RValue                    reflect.Value
	ptr                       bool
	nil                       bool
	Options                   *ReflectValueOptions
	fallbackNameCallerHandler func(s ReflectValuer, name string, c Call) (handled bool, value Object, err error)
	methodsGetter             IndexGetProxy
	*FuncSpec
}

var (
	_ ReflectValuer = (*ReflectValue)(nil)
	_ Niler         = (*ReflectValue)(nil)
	_ Copier        = (*ReflectValue)(nil)
	_ Printer       = (*ReflectValue)(nil)
)

func (r *ReflectValue) Init() {
	methodsName := func() Array {
		names := make(Array, len(r.RType.RMethods))
		var i int
		for name := range r.RType.RMethods {
			names[i] = Str(name)
			i++
		}
		sort.Slice(names, func(i, j int) bool {
			return names[i].(Str) < names[j].(Str)
		})
		return names
	}
	r.methodsGetter.ToStrFunc = methodsName().ToString
	r.methodsGetter.IterateFunc = func(vm *VM, na *NamedArgs) Iterator {
		return methodsName().Iterate(vm, na)
	}
	r.methodsGetter.GetIndexFunc = func(vm *VM, index Object) (value Object, err error) {
		name := index.ToString()
		if tm := r.RType.RMethods[name]; tm != nil {
			return r.method(tm), nil
		}
		return nil, ErrInvalidIndex.NewError(name)
	}
	r.methodsGetter.CallNameFunc = func(name string, c Call) (Object, error) {
		if tm := r.RType.RMethods[name]; tm != nil {
			return r.method(tm).Call(c)
		}
		return nil, ErrInvalidIndex.NewError(name)
	}
}

func (r *ReflectValue) Methods() *IndexGetProxy {
	return &r.methodsGetter
}

func (r *ReflectValue) Method(name string) *ReflectFunc {
	if tm := r.RType.RMethods[name]; tm != nil {
		return r.method(tm)
	}
	return nil
}

func (r *ReflectValue) method(tm *ReflectMethod) *ReflectFunc {
	return &ReflectFunc{ReflectValue{
		RType:  &ReflectType{RType: tm.Method.Type},
		RValue: r.RValue.Addr().Method(tm.i),
	}}
}

func (r *ReflectValue) FalbackNameCallerHandler(handler func(s ReflectValuer, name string, c Call) (handled bool, value Object, err error)) *ReflectValue {
	r.fallbackNameCallerHandler = handler
	return r
}

func (r *ReflectValue) Value() reflect.Value {
	if r.ptr {
		return r.RValue.Addr()
	}
	return r.RValue
}

func (r *ReflectValue) PtrValue() reflect.Value {
	return r.RValue.Addr()
}

func (r *ReflectValue) GetRType() *ReflectType {
	return r.RType
}

func (r *ReflectValue) IsPtr() bool {
	return r.ptr
}

func (r *ReflectValue) ToInterface() any {
	return r.Value().Interface()
}

func (r *ReflectValue) Type() ObjectType {
	return r.RType
}

func (r *ReflectValue) ToString() string {
	return string(MustToStr(nil, r))
}

func (r *ReflectValue) Format(s fmt.State, verb rune) {
	if verb == 'v' && s.Flag('+') {
		s.Write([]byte(r.RType.Fqn()))
		s.Write([]byte(repr.QuotePrefix))
		if r.RType.RType.Name() == "" {
			s.Write([]byte(r.RType.RType.String()))
			s.Write([]byte{':', ' '})
		}
		defer func() {
			s.Write([]byte(repr.QuoteSufix))
		}()
	}

	if r.RType.formatMethod != nil {
		r.RValue.Addr().Method(r.RType.formatMethod.i).Call([]reflect.Value{reflect.ValueOf(s), reflect.ValueOf(verb)})
	} else {
		s.Write([]byte(fmt.Sprint(r.ToInterface())))
	}
}

func (r *ReflectValue) Print(state *PrinterState) (err error) {
	if r.RType.InstancePrintFunc != nil {
		return r.RType.InstancePrintFunc(state, r)
	}
	defer state.WrapReprString(r.RType.Fqn())()
	if r.RType.RType.Name() == "" {
		state.WriteString(r.RType.RType.String())
		state.WriteString(": ")
	}
	return state.WriteString(fmt.Sprint(r.ToInterface()))
}

func (r *ReflectValue) IsNil() bool {
	return r.nil
}

func (r *ReflectValue) Copy() (obj Object) {
	rv, _ := NewReflectValue(r.RValue.Interface())
	if r.ptr {
		rv.GetRValue().ptr = true
	}
	obj = rv
	return
}

func (r *ReflectValue) GetRValue() *ReflectValue {
	return r
}

func (r *ReflectValue) IsFalsy() bool {
	if r.nil || !r.RValue.IsValid() {
		return true
	}
	return zeroer.IsZeroValue(r.RValue.Addr())
}

func (r *ReflectValue) Equal(right Object) bool {
	if o, _ := right.(ReflectValuer); o != nil {
		return o.Value() == r.Value()
	} else if right == Nil {
		return r.nil
	}
	return false
}

func (r *ReflectValue) CallName(name string, c Call) (Object, error) {
	return r.CallNameOf(r, name, c)
}

func (r *ReflectValue) CallNameOf(this ReflectValuer, name string, c Call) (Object, error) {
	if tm := r.RType.RMethods[name]; tm != nil {
		return r.method(tm).Call(c)
	}
	if r.fallbackNameCallerHandler != nil {
		if handled, value, err := r.fallbackNameCallerHandler(this, name, c); handled || err != nil {
			return value, err
		}
	}
	return nil, ErrInvalidIndex.NewError(name)
}

type ReflectFunc struct {
	ReflectValue
}

var (
	_ ReflectValuer = (*ReflectFunc)(nil)
	_ CallerObject  = (*ReflectFunc)(nil)
)

func (r *ReflectFunc) Name() string {
	return r.RType.Name()
}

func (r *ReflectFunc) ToString() string {
	var (
		t    = r.RType.RType
		name = t.PkgPath() + "." + t.Name()
		toS  = func(l int, get func(i int) reflect.Type) string {
			r := make([]string, l)

			for i := 0; i < l; i++ {
				name := get(i).PkgPath()
				if len(name) > 0 {
					name += "."
				}
				name += t.Name()
				r[i] = name
			}
			return strings.Join(r, ", ")
		}

		inArgs  = "(" + toS(t.NumIn(), t.In) + ")"
		outArgs = toS(t.NumOut(), t.Out)
	)

	if t.NumOut() > 1 {
		outArgs = "(" + outArgs + ")"
	}

	if len(outArgs) > 0 {
		outArgs = " " + outArgs
	}

	return fmt.Sprintf(ReprQuote("reflectFunc: %s(%s)%s"), strings.TrimPrefix(name, "."), inArgs, outArgs)
}

func (r *ReflectFunc) Call(c Call) (_ Object, err error) {
	typ := r.RValue.Type()

	var (
		numIn = typ.NumIn()
		argv  []reflect.Value
		argc  int
	)

	if typ.IsVariadic() {
		if argc = c.Args.Length(); argc < numIn-1 {
			return nil, ErrType.NewError(fmt.Sprintf("wrong number of args: got %d want at least %d", argc, numIn-1))
		}
		dddType := typ.In(numIn - 1).Elem()
		argv = make([]reflect.Value, argc)
		if err = reflectCallArgsToValues(c.VM, typ, dddType, numIn, c.Args, argv); err != nil {
			return
		}
	} else if numIn == 1 && reflectCallTypeToArgv(typ.In(0), &c, &argv) {
		// argv was populated
	} else if argc = c.Args.Length(); argc != numIn {
		return nil, ErrType.NewError(fmt.Sprintf("wrong number of args: got %d want %d", argc, numIn))
	} else {
		argv = make([]reflect.Value, argc)
		if err = reflectCallArgsToValues(c.VM, typ, nil, numIn, c.Args, argv); err != nil {
			return
		}
	}

	var ret reflect.Value
	ret, err = safeCall(r.RValue, argv)
	if err == nil {
		if ret.IsValid() {
			if zeroer.MustIsNil(ret) {
				return Nil, nil
			}
			return c.VM.ToObject(ret.Interface())
		}
		return Nil, nil
	}
	return
}

func reflectCallArgsToValues(vm *VM, typ, dddType reflect.Type, numIn int, args Args, argv []reflect.Value) (err error) {
	args.Walk(func(i int, arg Object) any {
		var rarg reflect.Value
		switch t := arg.(type) {
		case *ReflectValue:
			rarg = t.RValue
		default:
			rarg = reflect.ValueOf(vm.ToInterface(arg))
		}

		rarg = indirectInterface(rarg)
		// Compute the expected type. Clumsy because of variadics.
		argType := dddType
		if !typ.IsVariadic() || i < numIn-1 {
			argType = typ.In(i)
		}

		if argv[i], err = prepareArg(rarg, argType); err != nil {
			err = ErrType.NewError(fmt.Sprintf("arg %d: %s", i, err))
		}
		return err
	})
	return
}

type ReflectArray struct {
	ReflectValue
}

var (
	_ ReflectValuer  = (*ReflectArray)(nil)
	_ Iterabler      = (*ReflectArray)(nil)
	_ IndexGetSetter = (*ReflectArray)(nil)
	_ LengthGetter   = (*ReflectArray)(nil)
	_ Printer        = (*ReflectArray)(nil)
)

func (o *ReflectArray) ToString() string {
	return string(MustToStr(nil, o))
}

func (o *ReflectArray) IsFalsy() bool {
	return o.RValue.Len() == 0
}

func (o *ReflectArray) Get(vm *VM, i int) (value Object, err error) {
	e := o.RValue.Index(i).Interface()
	if value, _ = e.(Object); value == nil {
		if vm == nil {
			value, err = ToObject(e)
		} else {
			value, err = vm.ToObject(e)
		}
	}
	return
}

func (o *ReflectArray) IndexGet(vm *VM, index Object) (value Object, err error) {
	var ix int
	switch t := index.(type) {
	case Int:
		ix = int(t)
	case Uint:
		ix = int(t)
	default:
		if index.ToString() == ObjectMethodsGetterFieldName {
			return o.Methods(), nil
		}
		return nil, ErrUnexpectedArgValue.NewError("expected index types: int|uint|\"" + ObjectMethodsGetterFieldName + "\"")
	}

	if ix >= o.RValue.Len() {
		return nil, ErrIndexOutOfBounds
	}

	return o.Get(vm, ix)
}

func (o *ReflectArray) IndexSet(vm *VM, index, value Object) (err error) {
	var ix int
	switch t := index.(type) {
	case Int:
		ix = int(t)
		if t > math.MaxInt {
			return ErrUnexpectedArgValue.NewError(fmt.Sprintf("%u > %d", t, math.MaxInt))
		}
	case Uint:
		if t > math.MaxInt {
			return ErrUnexpectedArgValue.NewError(fmt.Sprintf("%u > %d", t, math.MaxInt))
		}
		ix = int(t)
	default:
		return ErrUnexpectedArgValue.NewError("expected index types: int|uint")
	}

	var v reflect.Value
	if rv, _ := value.(*ReflectValue); rv != nil {
		v = rv.RValue
	} else {
		v = reflect.ValueOf(vm.ToInterface(value))
	}

	if ix >= o.RValue.Len() {
		return ErrIndexOutOfBounds
	}

	o.RValue.Index(ix).Set(v)
	return
}

func (o *ReflectArray) Length() int {
	return o.RValue.Len()
}

func (o *ReflectArray) Copy() (obj Object) {
	v := reflect.New(o.RType.RType).Elem()
	reflect.Copy(v, o.RValue)
	rv, _ := NewReflectValue(v.Interface())
	rv.GetRValue().ptr = o.ptr
	obj = rv
	return
}

func (o *ReflectArray) Print(state *PrinterState) (err error) {
	if o.RType.InstancePrintFunc != nil {
		return o.RType.InstancePrintFunc(state, &o.ReflectValue)
	}
	return state.PrintArray(o.Length(), func(i int) (Object, error) {
		return o.Get(state.VM, i)
	})
}

type ReflectSlice struct {
	ReflectArray
}

var (
	_ ReflectValuer  = (*ReflectSlice)(nil)
	_ Iterabler      = (*ReflectSlice)(nil)
	_ IndexGetSetter = (*ReflectSlice)(nil)
	_ Appender       = (*ReflectSlice)(nil)
	_ Slicer         = (*ReflectSlice)(nil)
)

func (o *ReflectSlice) ToString() string {
	return string(MustToStr(nil, o))
}

func (o *ReflectSlice) Print(state *PrinterState) (err error) {
	if o.RType.InstancePrintFunc != nil {
		return o.RType.InstancePrintFunc(state, &o.ReflectValue)
	}
	return state.PrintArray(o.Length(), func(i int) (Object, error) {
		return o.Get(state.VM, i)
	})
}

func (o *ReflectSlice) Slice(low, high int) Object {
	cp := *o
	cp.RValue = cp.RValue.Slice(low, high)
	return &cp
}

func (o *ReflectSlice) AppendObjects(vm *VM, items ...Object) (_ Object, err error) {
	var (
		itemType = o.RType.RType.Elem()
		values   = reflect.MakeSlice(o.RType.RType, len(items), len(items))
		itemV    reflect.Value
	)

	for i, item := range items {
		if itemV, err = prepareArg(reflect.ValueOf(vm.ToInterface(item)), itemType); err != nil {
			return
		}
		values.Index(i).Set(itemV)
	}

	v := reflect.AppendSlice(o.RValue, values)

	if o.ptr {
		o.RValue.Set(v)
		return o, nil
	}

	cp := *o
	cp.RValue = v
	return &cp, nil
}

func (o *ReflectSlice) Insert(vm *VM, at int, items ...Object) (_ Object, err error) {
	var (
		itemType = o.RType.RType.Elem()
		values   = reflect.MakeSlice(o.RType.RType, len(items), len(items))
		itemV    reflect.Value
	)

	if at < 0 {
		at = o.RValue.Len() + at
		if at < 0 {
			return nil, ErrInvalidIndex.NewError("negative position is greather then slice length")
		}
	}

	for i, item := range items {
		if itemV, err = prepareArg(reflect.ValueOf(vm.ToInterface(item)), itemType); err != nil {
			return
		}
		values.Index(i).Set(itemV)
	}

	var v reflect.Value
	if at == 0 {
		v = reflect.AppendSlice(values, o.RValue)
	} else {
		begging := o.RValue.Slice(0, at)
		ending := o.RValue.Slice(at, o.RValue.Len())
		v = reflect.AppendSlice(begging, reflect.AppendSlice(values, ending))
	}

	if o.ptr {
		o.RValue.Set(v)
		return o, nil
	}

	cp := *o
	cp.RValue = v
	return &cp, nil
}

func (o *ReflectSlice) Copy() (obj Object) {
	v := reflect.MakeSlice(o.RType.RType, o.RValue.Len(), o.RValue.Len())
	reflect.Copy(v, o.RValue)
	rv, _ := NewReflectValue(v.Interface())
	rv.GetRValue().ptr = o.ptr
	obj = rv
	return
}

type ReflectMap struct {
	ReflectValue
}

var (
	_ ReflectValuer = (*ReflectMap)(nil)
	_ Iterabler     = (*ReflectMap)(nil)
	_ Indexer       = (*ReflectMap)(nil)
	_ Printer       = (*ReflectMap)(nil)
)

func (o *ReflectMap) Length() int {
	return o.Value().Len()
}

func (o *ReflectMap) ToString() string {
	return string(MustToStr(nil, o))
}

func (o *ReflectMap) Print(state *PrinterState) (err error) {
	if o.RType.InstancePrintFunc != nil {
		return o.RType.InstancePrintFunc(state, &o.ReflectValue)
	}

	var (
		keys            = o.RValue.MapKeys()
		sortKeysType, _ = state.options.SortKeys()
		getKey          = func(i int) (Object, error) {
			return state.VM.ToObject(keys[i].Interface())
		}
		getKeyValue = func(i int) reflect.Value {
			return keys[i]
		}
		keysO Array
		keysM map[string]int
		keysS []string
	)

	if sortKeysType == 0 {
		sortKeysType = PrintStateOptionSortTypeAscending
	}

	keysO = make(Array, len(keys))
	keysM = make(map[string]int, len(keys))
	keysS = make([]string, len(keys))

	var (
		k  Object
		ks string
	)

	if state.VM == nil {
		getKey = func(i int) (Object, error) {
			return ToObject(keys[i].Interface())
		}
	}

	for i := range keys {
		if k, err = getKey(i); err != nil {
			return
		}
		keysO[i] = k
		ks = k.ToString()
		keysM[ks] = i
		keysS[i] = ks
	}

	getKey = func(i int) (Object, error) {
		return keysO[keysM[keysS[i]]], nil
	}
	getKeyValue = func(i int) reflect.Value {
		return keys[keysM[keysS[i]]]
	}

	switch sortKeysType {
	case PrintStateOptionSortTypeAscending:
		sort.Strings(keysS)
	case PrintStateOptionSortTypeDescending:
		sort.Slice(keysS, func(i, j int) bool {
			return keysS[i] > keysS[j]
		})
	}

	if state.IsRepr {
		defer state.WrapRepr(o)()
	}
	return state.PrintDict(o.Length(), getKey, func(i int) (value Object, err error) {
		e := o.RValue.MapIndex(getKeyValue(i)).Interface()
		if value, _ = e.(Object); value == nil {
			if state.VM == nil {
				value, err = ToObject(e)
			} else {
				value, err = state.VM.ToObject(e)
			}
		}
		return
	})
}

func (o *ReflectMap) IndexDelete(vm *VM, index Object) (err error) {
	o.RValue.SetMapIndex(reflect.ValueOf(vm.ToInterface(index)), reflect.Value{})
	return nil
}

func (o *ReflectMap) IndexGet(vm *VM, index Object) (value Object, err error) {
	v := o.RValue.MapIndex(reflect.ValueOf(vm.ToInterface(index)))
	if !v.IsValid() || zeroer.MustIsNil(v) {
		if index.ToString() == ObjectMethodsGetterFieldName {
			return o.Methods(), nil
		}
		return Nil, nil
	}
	return vm.ToObject(v.Interface())
}

func (o *ReflectMap) IndexSet(vm *VM, index, value Object) (err error) {
	var v reflect.Value
	if rv, _ := value.(*ReflectValue); rv != nil {
		v = rv.RValue
	} else {
		v = reflect.ValueOf(vm.ToInterface(value))
	}

	o.RValue.SetMapIndex(reflect.ValueOf(vm.ToInterface(index)), v)
	return
}

func (o *ReflectMap) IsFalsy() bool {
	return o.RValue.Len() == 0
}

func (o *ReflectMap) Copy() (obj Object) {
	v := reflect.MakeMapWithSize(o.RType.RType, o.RValue.Len())
	for _, k := range o.RValue.MapKeys() {
		v.SetMapIndex(k, o.RValue.MapIndex(k))
	}
	rv, _ := NewReflectValue(v.Interface())
	rv.GetRValue().ptr = o.ptr
	obj = rv
	return
}

var (
	_ CallerObject    = (*ReflectStruct)(nil)
	_ CanCallerObject = (*ReflectStruct)(nil)
	_ Iterabler       = (*ReflectStruct)(nil)
	_ IndexGetSetter  = (*ReflectStruct)(nil)
	_ Printer         = (*ReflectStruct)(nil)
	_ ReflectValuer   = (*ReflectStruct)(nil)
	_ Iterabler       = (*ReflectStruct)(nil)
	_ IndexGetSetter  = (*ReflectStruct)(nil)
	_ Printer         = (*ReflectStruct)(nil)
)

type ReflectStruct struct {
	ReflectValue
	fieldHandler         func(vm *VM, s *ReflectStruct, name string, v any) any
	fallbackIndexHandler func(vm *VM, s *ReflectStruct, name string) (handled bool, value any, err error)
	Data                 Dict
	Interface            any
}

func (s *ReflectStruct) Name() string {
	return s.RType.Name()
}

func (s *ReflectStruct) Call(c Call) (Object, error) {
	if s.RType.CallObject == nil {
		return Nil, ErrNotCallable.NewError(s.Type().ToString())
	}
	return s.RType.CallObject(s, c)
}

func (s *ReflectStruct) CanCall() bool {
	return s.RType.CallObject != nil
}

func (s *ReflectStruct) Reader() Reader {
	if r, ok := s.Interface.(io.Reader); ok {
		return NewReader(r)
	}
	return nil
}

func (s *ReflectStruct) Writer() Writer {
	if w, ok := s.Interface.(io.Writer); ok {
		return NewWriter(w)
	}
	return nil
}

func (s *ReflectStruct) CanClose() bool {
	return s.Interface.(io.Closer) != nil
}

func (s *ReflectStruct) Close() error {
	return s.Interface.(io.Closer).Close()
}

func (s *ReflectStruct) CanIterationDone() (ok bool) {
	return ToIterationDoner(s.Interface) != nil
}

func (s *ReflectStruct) IterationDone(vm *VM) error {
	return ToIterationDoner(s.Interface).IterationDone(vm)
}

func (s *ReflectStruct) Init() {
	s.Interface = s.ToInterface()
	s.ReflectValue.Init()
}

func (s *ReflectStruct) ReprTypeName() string {
	n := s.RType.FullName()
	if s.IsPtr() {
		n = "*" + n
	}
	return "reflect instance of " + ReprQuote(n)
}

func (s *ReflectStruct) ToString() string {
	return string(MustToStr(nil, s))
}

// Print prints object writing output to out writer.
// Options:
// - anonymous flag: include anonymous fields.
// - zeros flag: include zero fields.
// - sortKeys int = 0: fields sorting. 1: ASC, 2: DESC.
func (s *ReflectStruct) Print(state *PrinterState) (err error) {
	if !state.IsRepr {
		i := s.ToInterface()
		switch t := i.(type) {
		case fmt.Stringer:
			return state.WriteString(t.String())
		case fmt.Formatter:
			return state.WriteString(fmt.Sprintf("%v", t))
		default:
			if s.Options.ToStr != nil {
				return state.WriteString(s.Options.ToStr())
			}

			if s.RType.InstancePrintFunc != nil {
				return s.RType.InstancePrintFunc(state, &s.ReflectValue)
			}
		}
	}

	type entry struct {
		name  string
		value Object
	}

	var (
		value        any
		handled      bool
		o            Object
		entries      []entry
		zeros        = state.options.IsZeros()
		anonymous    = state.options.IsAnonymous()
		sortKeysType = state.options.IsSortKeys()

		toObj = func(v any) (Object, error) {
			return state.VM.ToObject(v)
		}
	)

	if state.VM == nil {
		toObj = func(v any) (Object, error) {
			return ToObject(v)
		}
	}

	for _, name := range s.RType.FieldsNames {
		isa := s.RType.RFields[name].Struct.Anonymous
		if anonymous || !isa {
			if handled, value, err = s.SafeField(nil, name); err == nil && handled {
				if zeros || !zeroer.IsZero(value) {
					if isa {
						entries = append(entries, entry{name, nil})
					} else {
						if o, err = toObj(value); err != nil {
							return
						}
						entries = append(entries, entry{name, o})
					}
				}
			}
		}
	}

	switch sortKeysType {
	case 2:
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].name > entries[j].name
		})
	default:
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].name < entries[j].name
		})
	}

	if state.IsRepr {
		defer state.WrapRepr(s)()
	}
	return state.PrintDict(len(entries),
		func(i int) (Object, error) {
			return Str(entries[i].name), nil
		}, func(i int) (Object, error) {
			return entries[i].value, nil
		})
}

func (s *ReflectStruct) FieldHandler(handler func(vm *VM, s *ReflectStruct, name string, v any) any) *ReflectStruct {
	s.fieldHandler = handler
	return s
}

func (s *ReflectStruct) FalbackIndexHandler(handler func(vm *VM, s *ReflectStruct, name string) (handled bool, value any, err error)) *ReflectStruct {
	s.fallbackIndexHandler = handler
	return s
}

func (s *ReflectStruct) CallName(name string, c Call) (Object, error) {
	return s.CallNameOf(s, name, c)
}

func (s *ReflectStruct) IndexGet(vm *VM, index Object) (value Object, err error) {
	return s.IndexGetS(vm, index.ToString())
}

func (s *ReflectStruct) IndexGetS(vm *VM, index string) (value Object, err error) {
	if index == ObjectMethodsGetterFieldName {
		return &s.methodsGetter, nil
	}

	var (
		vi      any
		handled bool
	)

	if handled, vi, err = s.Field(vm, index); !handled {
		// check if is getter of private field
		if _, ok := s.RType.RType.FieldByName(strings.ToLower(index[:1]) + index[1:]); ok {
			if method := s.Method(index); method != nil && method.RType.RType.NumIn() == 1 {
				value, err = method.Call(Call{VM: vm})
				handled = true
				if err == nil {
					return
				}
			}
		}

		if !handled && s.fallbackIndexHandler != nil {
			handled, vi, err = s.fallbackIndexHandler(vm, s, index)
		}
	}

	if err != nil {
		return
	}

	if !handled {
		return s.methodsGetter.GetIndexFunc(vm, Str(index))
	}

	if value, _ = vi.(Object); value != nil {
		return value, nil
	}
	return vm.ToObject(vi)
}

func (s *ReflectStruct) Field(vm *VM, name string) (handled bool, value any, err error) {
	if field := s.RType.RFields[name]; field != nil {
		handled = true
		value = s.RValue.FieldByIndex(field.Struct.Index).Interface()
		if vm != nil && s.fieldHandler != nil {
			value = s.fieldHandler(vm, s, name, value)
		}
	}
	return
}

func (s *ReflectStruct) SafeField(vm *VM, name string) (handled bool, value any, err error) {
	defer func() {
		if r := recover(); r != nil {
			if err, _ = r.(error); err != nil {
				return
			}
			err = errors.New(fmt.Sprint(r))
		}
	}()
	return s.Field(vm, name)
}

func (s *ReflectStruct) IndexSet(vm *VM, index, value Object) (err error) {
	return s.indexSet(vm, index.ToString(), value)
}

func (s *ReflectStruct) SetValues(vm *VM, values Dict) (err error) {
	for k, v := range values {
		if err = s.indexSet(vm, k, v); err != nil {
			return
		}
	}
	return
}

func (s *ReflectStruct) indexSet(vm *VM, index string, value Object) (err error) {
	field := s.RType.RFields[index]
	if field != nil {
		return s.SetFieldValue(vm, field, value)
	}

	if m := s.RType.RMethods["Set"+strings.ToUpper(index[0:1])+index[1:]]; m != nil && m.Method.Type.NumIn() == 2 {
		_, err = s.method(m).Call(Call{VM: vm, Args: Args{{value}}})
		return
	}

	err = ErrInvalidIndex.NewError(index)
	return
}

func (s *ReflectStruct) SetField(vm *VM, index string, value any) (handled bool, err error) {
	var field *ReflectField
	field, handled = s.RType.RFields[index]

	if handled {
		err = s.SetFieldValue(vm, field, value)
	}
	return
}

func (s *ReflectStruct) SetFieldValue(vm *VM, df *ReflectField, value any) (err error) {
	field := s.RValue.FieldByIndex(df.Struct.Index)
	if value == nil || value == Nil {
		field.Set(reflect.Zero(field.Type()))
		return
	} else {
		var v reflect.Value
		switch t := value.(type) {
		case ReflectValuer:
			v = t.Value()
		case Object:
			v = reflect.ValueOf(vm.ToInterface(t))
		default:
			v = reflect.ValueOf(t)
		}

		if df.IsPtr {
			if !v.IsValid() {
				field.Set(reflect.Zero(field.Type()))
			} else if v.CanAddr() {
				v = v.Addr()
			} else if v.Kind() != reflect.Ptr {
				ptrv := reflect.New(v.Type())
				ptrv.Elem().Set(v)
				v = ptrv.Elem()
			}
		}

		var (
			t    = field.Type()
			indt = t
			ptr  = t.Kind() == reflect.Ptr
		)

		if ptr {
			indt = indt.Elem()
		}

		if m, _ := value.(Dict); m != nil && indt.Kind() == reflect.Struct {
			if v, err = mapToReflectStruct(vm, indt, m); err != nil {
				return
			}
		}

		if t.Kind() == reflect.Ptr {
			if v.Kind() == reflect.Ptr {
				v = v.Elem()
			}

			if v, err = prepareArg(v, t.Elem()); err == nil {
				v = v.Addr()
			}
		} else if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}

		if v, err = prepareArg(v, t); err == nil {
			field.Set(v)
		}
	}
	return
}

func (s *ReflectStruct) Copy() (obj Object) {
	v := reflect.New(s.RType.RType)
	v.Elem().Set(reflect.ValueOf(s.RValue.Interface()))
	if s.ptr {
		obj, _ = NewReflectValue(v.Interface())
	} else {
		obj, _ = NewReflectValue(v.Elem().Interface())
	}
	return
}

func (s *ReflectStruct) UserData() Indexer {
	if s.Data != nil {
		return s.Data
	}
	return &IndexDeleteProxy{
		IndexGetProxy: IndexGetProxy{
			IterateFunc: func(vm *VM, na *NamedArgs) Iterator {
				if s.Data == nil {
					return Dict{}.Iterate(vm, na)
				}
				return s.Data.Iterate(vm, na)
			},
			GetIndexFunc: func(vm *VM, index Object) (value Object, err error) {
				if s.Data == nil {
					return nil, ErrInvalidIndex.NewError(index.ToString())
				}
				return s.Data.IndexGet(vm, index)
			},
		},
		IndexDelProxy: IndexDelProxy{
			Del: func(vm *VM, key Object) error {
				if s.Data == nil {
					return nil
				}
				delete(s.Data, key.ToString())
				return nil
			},
		},
		IndexSetProxy: IndexSetProxy{
			Set: func(vm *VM, key, value Object) error {
				if s.Data == nil {
					s.Data = make(Dict)
				}
				s.Data.Set(key.ToString(), value)
				return nil
			},
		},
	}
}

var reflectValueType = reflect.TypeOf((*reflect.Value)(nil)).Elem()

// canBeNil reports whether an untyped nil can be assigned to the type. See reflect.Zero.
func canBeNil(typ reflect.Type) bool {
	switch typ.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return true
	case reflect.Struct:
		return typ == reflectValueType
	}
	return false
}

// indirectInterface returns the concrete value in an interface value,
// or else the zero reflect.Value.
// That is, if v represents the interface value x, the result is the same as reflect.ValueOf(x):
// the fact that x was an interface value is forgotten.
func indirectInterface(v reflect.Value) reflect.Value {
	if v.Kind() != reflect.Interface {
		return v
	}
	if v.IsNil() {
		return reflect.Value{}
	}
	return v.Elem()
}

// mapToReflectStruct create new struct instance from Dict.
func mapToReflectStruct(vm *VM, indirectType reflect.Type, m Dict) (v reflect.Value, err error) {
	var vlr ReflectValuer
	if vlr, err = NewReflectValue(reflect.New(indirectType).Interface()); err != nil {
		return
	}
	s := vlr.(*ReflectStruct)
	if err = s.SetValues(vm, m); err != nil {
		return
	}
	return s.RValue.Addr(), nil
}

// safeCall runs fun.Call(args), and returns the resulting value and error, if
// any. If the call panics, the panic value is returned as an error.
func safeCall(fun reflect.Value, args []reflect.Value) (val reflect.Value, err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				trace := debug.Stack()
				e2 := ErrReflectCallPanicsType.NewError(fmt.Sprintf("%s: %v\n%s", fun, err, string(trace)))
				e2.Cause = e
				err = e2
			} else {
				err = ErrReflectCallPanicsType.NewError(fmt.Sprintf("%v: %v", fun.Type(), r))
			}
		}
	}()
	ret := fun.Call(args)
	l := len(ret)
	switch l {
	case 0:
	default:
		if ret[l-1].Type() == errorType {
			if !zeroer.MustIsNil(ret[l-1]) {
				err = ret[l-1].Interface().(error)
			}
			ret = ret[:l-1]
			l--
		}
		if l == 1 {
			val = ret[0]
		} else {
			arr := make([]any, l)
			val = reflect.ValueOf(arr)
			for i, value := range ret {
				val.Index(i).Set(value)
			}
		}
	}
	return
}

// prepareArg checks if value can be used as an argument of type argType, and
// converts an invalid value to appropriate zero if possible.
func prepareArg(value reflect.Value, argType reflect.Type) (v reflect.Value, err error) {
	if !value.IsValid() {
		if !canBeNil(argType) {
			return reflect.Value{}, fmt.Errorf("value is nil; should be of type %s", argType)
		}
		value = reflect.Zero(argType)
	}

	vt := value.Type()

	if vt.AssignableTo(argType) {
		return value, nil
	}

	if argType.Kind() == reflect.Ptr && reflect.PtrTo(vt).AssignableTo(argType) {
		if value.CanAddr() {
			return value.Addr(), nil
		} else {
			ptrv := reflect.New(value.Type())
			ptrv.Elem().Set(value)
			return ptrv, nil
		}
	} else if vt.Kind() == reflect.Ptr && reflect.PtrTo(argType).AssignableTo(vt) {
		return value.Elem(), nil
	}

	if vt.ConvertibleTo(argType) {
		value = value.Convert(argType)
		return value, nil
	}

	if vt == mapType {
		switch argType.Kind() {
		case reflect.Struct:
			var m Dict
			if m, err = AnyMapToMap(value.Interface().(map[string]any)); err != nil {
				return
			} else if v, err = mapToReflectStruct(nil, argType, m); err != nil {
				return
			}
			return prepareArg(v.Elem(), argType)
		case reflect.Ptr:
			if argType.Elem().Kind() == reflect.Struct {
				var m Dict
				if m, err = AnyMapToMap(value.Interface().(map[string]any)); err != nil {
					return
				} else if v, err = mapToReflectStruct(nil, argType.Elem(), m); err != nil {
					return
				}
				return prepareArg(v, argType)
			}
		}
	}

	return reflect.Value{}, fmt.Errorf("value has type %s; should be %s", value.Type(), argType)
}

var (
	callType     = reflect.TypeOf((*Call)(nil)).Elem()
	errorType    = reflect.TypeOf((*error)(nil)).Elem()
	fmtStateType = reflect.TypeOf((*fmt.State)(nil)).Elem()
	runeType     = reflect.TypeOf((*rune)(nil)).Elem()
	mapType      = reflect.TypeOf((*map[string]any)(nil)).Elem()
)

func reflectIsCallType(t reflect.Type) (ok, ptr bool) {
	if ok = t == callType; !ok {
		if ptr = t.Kind() == reflect.Ptr && t.Elem() == callType; ptr {
			ok = true
		}
	}
	return
}

func reflectCallTypeToArgv(t reflect.Type, c *Call, argv *[]reflect.Value) bool {
	if ok, ptr := reflectIsCallType(t); ok {
		*argv = make([]reflect.Value, 1)
		if ptr {
			(*argv)[0] = reflect.ValueOf(c)
		} else {
			(*argv)[0] = reflect.ValueOf(*c)
		}
		return true
	}
	return false
}
