package gad

import (
	"errors"
	"fmt"
	"go/ast"
	"io"
	"math"
	"reflect"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	_ "text/template"
	_ "unsafe"

	"github.com/gad-lang/gad/repr"
)

type ReflectMethod struct {
	baseType reflect.Type
	Method   reflect.Method
	i        int
}

func (r *ReflectMethod) Type() ObjectType {
	return TReflectMethod
}

func (r *ReflectMethod) ToString() string {
	return r.baseType.String() + "#" + r.Method.Name
}

func (r *ReflectMethod) IsFalsy() bool {
	return false
}

func (r *ReflectMethod) Equal(right Object) bool {
	if o, _ := right.(*ReflectMethod); o != nil {
		return o.baseType == r.baseType && o.Method == r.Method
	}
	return false
}

type ReflectField struct {
	BaseType reflect.Type
	IsPtr    bool
	Struct   IndexableStructField
	Value    reflect.Value
}

func (r *ReflectField) String() string {
	return r.Struct.Name + " " + r.Struct.Type.Name() + " " + fmt.Sprint(r.Struct.Index)
}

func (r *ReflectField) Type() ObjectType {
	return NewReflectType(r.Value.Type())
}

func (r *ReflectField) ToString() string {
	return fmt.Sprint(r.Value.Interface())
}

func (r *ReflectField) IsFalsy() bool {
	switch r.Value.Kind() {
	case reflect.Slice, reflect.Map, reflect.Array, reflect.String:
		return r.Value.Len() == 0
	case reflect.Bool:
		return !r.Value.Bool()
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return r.Value.Int() == 0
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return r.Value.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return r.Value.Float() == 0
	default:
		if z, _ := r.Value.Addr().Interface().(interface{ IsZero() bool }); z != nil {
			return z.IsZero()
		}
		return false
	}
}

func (r *ReflectField) Equal(right Object) bool {
	if o, _ := right.(*ReflectField); o != nil {
		return o.BaseType == r.BaseType && o.Value == r.Value
	}
	return false
}

func (r *ReflectField) Set(f reflect.Value, v Object) {

}

type ReflectType struct {
	RType        reflect.Type
	RMethods     map[string]*ReflectMethod
	FieldsNames  []string
	RFields      map[string]*ReflectField
	formatMethod *ReflectMethod
}

var _ ObjectType = (*ReflectType)(nil)

var (
	reflectTypeCache   = map[reflect.Type]*ReflectType{}
	reflectTypeCacheMu sync.Mutex
)

type IndexableStructField struct {
	reflect.StructField
	Index []int
	Names []string
}

func indirectType(reflectType reflect.Type) reflect.Type {
	for reflectType.Kind() == reflect.Ptr {
		reflectType = reflectType.Elem()
	}
	return reflectType
}

func FieldsOfReflectType(ityp reflect.Type) (result []*IndexableStructField) {
	ityp = indirectType(ityp)

	type item struct {
		fi, i int
		f     *IndexableStructField
	}
	var (
		walk    func(typ reflect.Type, path []int, name []string)
		nameMap = map[string]item{}
		fields  []*IndexableStructField
	)
	walk = func(typ reflect.Type, path []int, name []string) {
		typ = indirectType(typ)
		if typ.Kind() != reflect.Struct || (path != nil && typ == ityp) {
			return
		}

		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			path := append(append([]int{}, path...), i)
			name := append(append([]string{}, name...), field.Name)
			if ast.IsExported(field.Name) {
				fields = append(fields, &IndexableStructField{field, path, name})
				if field.Anonymous {
					walk(field.Type, path, name)
				}
			} else if field.Anonymous {
				walk(field.Type, path, name)
			}
		}
	}
	walk(ityp, nil, nil)

	for i := len(fields) - 1; i >= 0; i-- {
		f := fields[i]
		if old, ok := nameMap[f.Name]; ok {
			if len(old.f.Index) > len(f.Index) {
				old.f = f
				result[old.i] = f
			}
		} else {
			nameMap[f.Name] = item{i, len(result), f}
			result = append(result, f)
		}
	}
	return
}

func NewReflectType(typ reflect.Type) (rt *ReflectType) {
	reflectTypeCacheMu.Lock()
	defer reflectTypeCacheMu.Unlock()

	if typ.Kind() == reflect.Ptr || typ.Kind() == reflect.Interface {
		typ = typ.Elem()
	}

	if rt = reflectTypeCache[typ]; rt != nil {
		return
	}

	rt = &ReflectType{RType: typ}

	if typ.Kind() == reflect.Struct {
		fields := map[string]*ReflectField{}
		fields_ := FieldsOfReflectType(typ)
		for _, f := range fields_ {
			if old, ok := fields[f.Name]; ok {
				if len(f.Index) >= len(old.Struct.Index) {
					continue
				}
			}

			rf := &ReflectField{
				BaseType: typ,
				Value:    reflect.New(f.Type),
				Struct:   *f,
			}

			if rf.Value.Kind() == reflect.Ptr {
				rf.IsPtr = true
				rf.Value = rf.Value.Elem()
			}
			fields[f.Name] = rf
			rt.FieldsNames = append(rt.FieldsNames, f.Name)
		}

		rt.RFields = fields
	}
	reflectTypeCache[typ] = rt

	var (
		methods = map[string]*ReflectMethod{}
		ptrType = reflect.PtrTo(typ)
		n       = ptrType.NumMethod()
	)

	for i := 0; i < n; i++ {
		m := ptrType.Method(i)
		if !m.IsExported() {
			continue
		}

		methods[m.Name] = &ReflectMethod{
			baseType: ptrType,
			Method:   m,
			i:        i,
		}
	}

	if format := methods["Format"]; format != nil {
		t := format.Method.Type

		if t.NumOut() == 0 && t.NumIn() == 3 && t.In(1) == fmtStateType && t.In(2) == runeType {
			rt.formatMethod = format
		}
	}

	rt.RMethods = methods

	return
}

func (r *ReflectType) Type() ObjectType {
	return TBase
}

func (r *ReflectType) ToString() string {
	return r.RType.String()
}

func (r *ReflectType) IsFalsy() bool {
	return false
}

func (r *ReflectType) Equal(right Object) bool {
	if o, _ := right.(*ReflectType); o != nil {
		return o.RType == r.RType
	}
	return false
}

func (r *ReflectType) Call(c Call) (Object, error) {
	if c.NamedArgs.IsFalsy() {
		return r.New(c.VM, nil)
	}
	return r.New(c.VM, c.NamedArgs.Dict())
}

func (r *ReflectType) Name() string {
	return "reflect:" + r.Fqn()
}

func (r *ReflectType) Fqn() string {
	var n = r.RType.Name()
	if n == "" {
		n = r.RType.Kind().String()
	} else {
		n = r.RType.PkgPath() + "." + n
	}
	return n
}

func (r *ReflectType) Getters() Dict {
	return nil
}

func (r *ReflectType) Setters() Dict {
	return nil
}

func (r *ReflectType) Methods() (m Dict) {
	m = make(Dict, len(r.RMethods))
	for key := range r.RMethods {
		m[key] = Nil
	}
	return m
}

func (r *ReflectType) Fields() (fields Dict) {
	fields = Dict{}
	for _, f := range r.RFields {
		fields[f.Struct.Name] = f
	}
	return
}

func (r *ReflectType) GetRMethods() map[string]*ReflectMethod {
	return r.RMethods
}

func (r *ReflectType) New(vm *VM, m Dict) (_ Object, err error) {
	var rv reflect.Value
	switch r.RType.Kind() {
	case reflect.Struct:
		rv = reflect.New(r.RType).Elem()
		obj := &ReflectStruct{
			ReflectValue: ReflectValue{RType: r, RValue: rv, Options: &ReflectValueOptions{}},
		}
		for s, v := range m {
			if err = obj.indexSet(vm, s, v); err != nil {
				return
			}
		}
		return obj, nil
	case reflect.Map:
		rv = reflect.MakeMap(r.RType)
		for k, v := range m {
			rv.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(vm.ToInterface(v)))
		}
	}
	return &ReflectValue{RType: r, RValue: rv, Options: &ReflectValueOptions{}}, nil
}

func (r *ReflectType) IsChildOf(t ObjectType) bool {
	return t == TBase
}

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
		_, isnill = isNil(rv)
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
		// We allow functions with 0 or 1 result or 2 results where the second is an error.
		switch t.RType.NumOut() {
		case 0, 1:
		case 2:
			if t.RType.Out(1) != errorType {
				return nil, ErrIncompatibleReflectFuncType.NewError(fmt.Sprintf("out %d of function isn't error.", t.RType.NumOut()))
			}
		default:
			return nil, ErrIncompatibleReflectFuncType.NewError(fmt.Sprintf("function called with %d args; should be <= 2.", t.RType.NumOut()))
		}
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
}

var (
	_ ReflectValuer = (*ReflectValue)(nil)
	_ Niler         = (*ReflectValue)(nil)
	_ Copier        = (*ReflectValue)(nil)
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
	r.methodsGetter.ToStr = methodsName().ToString
	r.methodsGetter.It = func(vm *VM, na *NamedArgs) Iterator {
		return methodsName().Iterate(vm, na)
	}
	r.methodsGetter.GetIndex = func(vm *VM, index Object) (value Object, err error) {
		name := index.ToString()
		if tm := r.RType.RMethods[name]; tm != nil {
			return r.method(tm), nil
		}
		return nil, ErrInvalidIndex.NewError(name)
	}
	r.methodsGetter.CallNameHandler = func(name string, c Call) (Object, error) {
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
	var w strings.Builder
	w.WriteString(repr.QuotePrefix)
	w.WriteString("reflectValue:")
	if r.Options.ToStr == nil {
		fmt.Fprintf(&w, "%+v", r)
	} else {
		w.WriteString(r.Options.ToStr())
	}
	w.WriteString(repr.QuoteSufix)
	return w.String()
}

func (r *ReflectValue) Format(s fmt.State, verb rune) {
	if verb == 'v' && s.Flag('+') {
		s.Write([]byte(r.RType.Fqn()))
		s.Write([]byte(repr.QuotePrefix))
		if r.RType.RType.Name() == "" {
			s.Write([]byte(r.RType.RType.String()))
			s.Write([]byte{':', ' '})
		}
	}
	if r.RType.formatMethod != nil {
		r.RValue.Addr().Method(r.RType.formatMethod.i).Call([]reflect.Value{reflect.ValueOf(s), reflect.ValueOf(verb)})
	} else if verb == 'v' {
		s.Write([]byte(fmt.Sprint(r.ToInterface())))
	}

	if verb == 'v' && s.Flag('+') {
		s.Write([]byte(repr.QuoteSufix))
	}
}

func (r *ReflectValue) ToStringW(w io.Writer) {
	fmt.Fprintf(w, "%+v", r)
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
	return isZero(r.RValue.Addr())
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

func (r *ReflectFunc) ToString() string {
	return fmt.Sprintf(ReprQuote("reflectFunc: %s"), r.RType.RType.String())
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

	if err != nil {
		return
	}

	var ret reflect.Value
	ret, err = safeCall(r.RValue, argv)
	if err == nil {
		if ret.IsValid() {
			if mustIsNil(ret) {
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
	_ ReflectValuer     = (*ReflectArray)(nil)
	_ Iterabler         = (*ReflectArray)(nil)
	_ IndexGetSetter    = (*ReflectArray)(nil)
	_ LengthGetter      = (*ReflectArray)(nil)
	_ ObjectRepresenter = (*ReflectArray)(nil)
)

func (o *ReflectArray) Format(s fmt.State, verb rune) {
	if verb == 'v' {
		s.Write([]byte(repr.QuotePrefix + "reflectArray:"))
		o.ReflectValue.ToStringW(s)
		s.Write([]byte(repr.QuoteSufix))
		return
	}
	o.ReflectValue.Format(s, verb)
}

func (o *ReflectArray) ToString() string {
	return fmt.Sprintf("%v", o)
}

func (o *ReflectArray) IsFalsy() bool {
	return o.RValue.Len() == 0
}

func (o *ReflectArray) Get(vm *VM, i int) (value Object, err error) {
	return vm.ToObject(o.RValue.Index(i).Interface())
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

func (o *ReflectArray) Repr(vm *VM) (_ string, err error) {
	var w strings.Builder
	w.WriteString(repr.QuotePrefix)
	var (
		rpr  = vm.Builtins.ArgsInvoker(BuiltinRepr, Call{VM: vm})
		l    = o.RValue.Len()
		rpro Object
	)
	w.WriteString(o.Type().Name())
	w.WriteString(":[")
	for i := 0; i < l; i++ {
		if rpro, err = vm.ToObject(o.RValue.Index(i).Interface()); err != nil {
			return
		}
		if rpro, err = rpr(rpro); err != nil {
			return
		}
		if i > 0 {
			w.WriteString(", ")
		}
		w.WriteString(rpro.ToString())
	}
	w.WriteString("]")
	w.WriteString(repr.QuoteSufix)
	return w.String(), nil
}

type ReflectSlice struct {
	ReflectArray
}

func (o *ReflectSlice) Slice(low, high int) Object {
	cp := *o
	cp.RValue = cp.RValue.Slice(low, high)
	return &cp
}

var (
	_ ReflectValuer  = (*ReflectSlice)(nil)
	_ Iterabler      = (*ReflectSlice)(nil)
	_ IndexGetSetter = (*ReflectSlice)(nil)
	_ Appender       = (*ReflectSlice)(nil)
	_ Slicer         = (*ReflectSlice)(nil)
)

func (o *ReflectSlice) Append(vm *VM, items ...Object) (_ Object, err error) {
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

func (o *ReflectSlice) Format(s fmt.State, verb rune) {
	if verb == 's' {
		s.Write([]byte(repr.QuotePrefix + "reflectSlice:"))
		o.ReflectValue.ToStringW(s)
		s.Write([]byte(repr.QuoteSufix))
		return
	}
	o.ReflectValue.Format(s, verb)
}

func (o *ReflectSlice) ToString() string {
	return fmt.Sprintf("%s", o)
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
	_ ReflectValuer     = (*ReflectMap)(nil)
	_ Iterabler         = (*ReflectMap)(nil)
	_ Indexer           = (*ReflectMap)(nil)
	_ ObjectRepresenter = (*ReflectMap)(nil)
)

func (o *ReflectMap) Format(s fmt.State, verb rune) {
	if verb == 's' {
		s.Write([]byte(repr.QuotePrefix + "reflectMap:"))
		o.ReflectValue.ToStringW(s)
		s.Write([]byte(repr.QuoteSufix))
		return
	}
	o.ReflectValue.Format(s, verb)
}

func (o *ReflectMap) Length() int {
	return o.Value().Len()
}

func (o *ReflectMap) ToString() string {
	return fmt.Sprintf("%s", o)
}

func (o *ReflectMap) Repr(vm *VM) (_ string, err error) {
	var w strings.Builder
	w.WriteString(repr.QuotePrefix)
	var (
		values   []string
		rpr      = vm.Builtins.ArgsInvoker(BuiltinRepr, Call{VM: vm})
		ko, rpro Object
	)
	for _, k := range o.RValue.MapKeys() {
		if ko, err = NewReflectValue(k.Interface()); err != nil {
			return
		}
		if ko, err = rpr(ko); err != nil {
			return
		}
		if rpro, err = vm.ToObject(o.RValue.MapIndex(k).Interface()); err != nil {
			return
		}
		if rpro, err = rpr(rpro); err != nil {
			return
		}
		values = append(values, fmt.Sprintf("%s: %v", ko.ToString(), rpro.ToString()))
	}
	sort.Strings(values)
	w.WriteString(o.Type().Name())
	w.WriteString(":{")
	w.WriteString(strings.Join(values, ", "))
	w.WriteString("}")
	w.WriteString(repr.QuoteSufix)
	return w.String(), nil
}

func (o *ReflectMap) IndexDelete(vm *VM, index Object) (err error) {
	o.RValue.SetMapIndex(reflect.ValueOf(vm.ToInterface(index)), reflect.Value{})
	return nil
}

func (o *ReflectMap) IndexGet(vm *VM, index Object) (value Object, err error) {
	v := o.RValue.MapIndex(reflect.ValueOf(vm.ToInterface(index)))
	if !v.IsValid() || mustIsNil(v) {
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

type ReflectStruct struct {
	ReflectValue
	fieldHandler         func(vm *VM, s *ReflectStruct, name string, v any) any
	fallbackIndexHandler func(vm *VM, s *ReflectStruct, name string) (handled bool, value any, err error)
	Data                 Dict
	Interface            any
}

var (
	_ ReflectValuer     = (*ReflectStruct)(nil)
	_ Iterabler         = (*ReflectStruct)(nil)
	_ IndexGetSetter    = (*ReflectStruct)(nil)
	_ ObjectRepresenter = (*ReflectStruct)(nil)
)

func (s *ReflectStruct) Init() {
	s.Interface = s.ToInterface()
	s.ReflectValue.Init()
}

func (s *ReflectStruct) ToString() string {
	i := s.ToInterface()
	switch t := i.(type) {
	case fmt.Stringer:
		return t.String()
	case fmt.Formatter:
		return fmt.Sprintf("%v", t)
	default:
		if s.Options.ToStr == nil {
			var (
				values  []string
				value   any
				handled bool
				err     error
				w       strings.Builder
			)
			for _, name := range s.RType.FieldsNames {
				if handled, value, err = s.SafeField(nil, name); err == nil && handled {
					if !IsZero(value) {
						var (
							rv = reflect.ValueOf(value)
							s  string
						)
						if rv.Kind() == reflect.String {
							s = strconv.Quote(rv.String())
						} else {
							s = fmt.Sprint(value)
						}
						values = append(values, fmt.Sprintf("%s: %v", name, s))
					}
				}
			}
			sort.Strings(values)
			w.WriteString("{")
			w.WriteString(strings.Join(values, ", "))
			w.WriteString("}")
			return w.String()
		}
		return s.Options.ToStr()
	}
}

func (s *ReflectStruct) Repr(vm *VM) (_ string, err error) {
	var w strings.Builder
	w.WriteString(repr.QuotePrefix)

	var (
		values  []string
		value   any
		handled bool
		rpro    Object
	)

	for _, name := range s.RType.FieldsNames {
		if handled, value, err = s.SafeField(nil, name); err == nil && handled {
			if !IsZero(value) {
				if rpro, err = vm.ToObject(value); err != nil {
					return
				}
				values = append(values, fmt.Sprintf("%s: %v", name, ReprQuote(rpro.Type().Name()+": "+rpro.ToString())))
			}
		}
	}
	sort.Strings(values)
	w.WriteString(s.Type().Name())
	w.WriteString(":{")
	w.WriteString(strings.Join(values, ", "))
	w.WriteString("}")
	w.WriteString(repr.QuoteSufix)
	return w.String(), nil
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
		return s.methodsGetter.GetIndex(vm, Str(index))
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
			It: func(vm *VM, na *NamedArgs) Iterator {
				if s.Data == nil {
					return Dict{}.Iterate(vm, na)
				}
				return s.Data.Iterate(vm, na)
			},
			GetIndex: func(vm *VM, index Object) (value Object, err error) {
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

//go:linkname canBeNil text/template.canBeNil
func canBeNil(dest reflect.Type) bool

//go:linkname indirectInterface text/template.indirectInterface
func indirectInterface(v reflect.Value) reflect.Value

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
			if !mustIsNil(ret[l-1]) {
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
