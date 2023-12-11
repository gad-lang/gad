package gad

import (
	"fmt"
	"io"
	"math"
	"reflect"
	"strings"
	"sync"
	_ "text/template"
	_ "unsafe"
)

type ReflectMethod struct {
	baseType reflect.Type
	m        reflect.Method
	i        int
}

func (r *ReflectMethod) Type() ObjectType {
	return TReflectMethod
}

func (r *ReflectMethod) ToString() string {
	return r.baseType.String() + "#" + r.m.Name
}

func (r *ReflectMethod) IsFalsy() bool {
	return false
}

func (r *ReflectMethod) Equal(right Object) bool {
	if o, _ := right.(*ReflectMethod); o != nil {
		return o.baseType == r.baseType && o.m == r.m
	}
	return false
}

type ReflectField struct {
	baseType reflect.Type
	ptr      bool
	f        reflect.StructField
	v        reflect.Value
}

func (r *ReflectField) Type() ObjectType {
	return NewReflectType(r.v.Type())
}

func (r *ReflectField) ToString() string {
	return fmt.Sprint(r.v.Interface())
}

func (r *ReflectField) IsFalsy() bool {
	switch r.v.Kind() {
	case reflect.Slice, reflect.Map, reflect.Array, reflect.String:
		return r.v.Len() == 0
	case reflect.Bool:
		return !r.v.Bool()
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return r.v.Int() == 0
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return r.v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return r.v.Float() == 0
	default:
		if z, _ := r.v.Addr().Interface().(interface{ IsZero() bool }); z != nil {
			return z.IsZero()
		}
		return false
	}
}

func (r *ReflectField) Equal(right Object) bool {
	if o, _ := right.(*ReflectField); o != nil {
		return o.baseType == r.baseType && o.v == r.v
	}
	return false
}

func (r *ReflectField) Set(f reflect.Value, v Object) {

}

type ReflectType struct {
	typ          reflect.Type
	methods      map[string]*ReflectMethod
	fieldsNames  []string
	fields       map[string]*ReflectField
	formatMethod *ReflectMethod
}

var _ ObjectType = (*ReflectType)(nil)

var (
	reflectTypeCache   = map[reflect.Type]*ReflectType{}
	reflectTypeCacheMu sync.Mutex
)

func NewReflectType(typ reflect.Type) (rt *ReflectType) {
	reflectTypeCacheMu.Lock()
	defer reflectTypeCacheMu.Unlock()

	if typ.Kind() == reflect.Ptr || typ.Kind() == reflect.Interface {
		typ = typ.Elem()
	}

	if rt = reflectTypeCache[typ]; rt != nil {
		return
	}

	rt = &ReflectType{typ: typ}
	reflectTypeCache[typ] = rt

	if typ.Kind() == reflect.Struct {
		n := typ.NumField()
		fields := map[string]*ReflectField{}

		for i := 0; i < n; i++ {
			f := typ.Field(i)
			if f.Anonymous || !f.IsExported() {
				continue
			}

			if old, ok := fields[f.Name]; ok {
				if len(f.Index) >= len(old.f.Index) {
					continue
				}
			}

			rf := &ReflectField{
				baseType: typ,
				v:        reflect.New(f.Type),
				f:        f,
			}

			if rf.v.Kind() == reflect.Ptr {
				rf.ptr = true
				rf.v = rf.v.Elem()
			}
			fields[f.Name] = rf
			rt.fieldsNames = append(rt.fieldsNames, f.Name)
		}

		rt.fields = fields
	}

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
			m:        m,
			i:        i,
		}
	}

	if format := methods["Format"]; format != nil {
		t := format.m.Type

		if t.NumOut() == 0 && t.NumIn() == 3 && t.In(1) == fmtStateType && t.In(2) == runeType {
			rt.formatMethod = format
		}
	}

	rt.methods = methods

	return
}

func (r *ReflectType) Type() ObjectType {
	return TNil
}

func (r *ReflectType) ToString() string {
	return r.typ.String()
}

func (r *ReflectType) IsFalsy() bool {
	return false
}

func (r *ReflectType) Equal(right Object) bool {
	if o, _ := right.(*ReflectType); o != nil {
		return o.typ == r.typ
	}
	return false
}

func (r *ReflectType) Call(c Call) (obj Object, err error) {
	if c.NamedArgs.IsFalsy() {
		obj, _ = r.New(c.VM, nil)
	} else {
		obj, _ = r.New(c.VM, c.NamedArgs.Dict())
	}
	return
}

func (r *ReflectType) Name() string {
	return "reflect:" + r.Fqn()
}

func (r *ReflectType) Fqn() string {
	var n = r.typ.Name()
	if n == "" {
		n = r.typ.Kind().String()
	} else {
		n = r.typ.PkgPath() + "." + n
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
	m = make(Dict, len(r.methods))
	for key := range r.methods {
		m[key] = Nil
	}
	return m
}

func (r *ReflectType) Fields() (fields Dict) {
	n := r.typ.NumField()
	fields = Dict{}

	for i := 0; i < n; i++ {
		f := r.typ.Field(i)
		if f.Anonymous || !f.IsExported() {
			continue
		}
		if old, ok := fields[f.Name]; ok {
			old := old.(*ReflectField)
			if len(f.Index) >= len(old.f.Index) {
				continue
			}
		}

		rf := &ReflectField{
			baseType: r.typ,
			v:        reflect.New(f.Type),
			f:        f,
		}

		if rf.v.Kind() == reflect.Ptr {
			rf.ptr = true
			rf.v = rf.v.Elem()
		}
		fields[f.Name] = rf
	}
	return
}

func (r *ReflectType) New(vm *VM, m Dict) (Object, error) {
	var rv reflect.Value
	switch r.typ.Kind() {
	case reflect.Struct:
		rv = reflect.New(r.typ).Elem()
		for k, v := range m {
			if f := r.fields[k]; f != nil {
				r.fields[k].Set(rv.FieldByIndex(r.fields[k].f.Index), v)
			}
		}
	case reflect.Map:
		rv = reflect.MakeMap(r.typ)
		for k, v := range m {
			rv.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(ToInterface(v)))
		}
	}
	return &ReflectValue{typ: r, v: rv}, nil
}

func (r *ReflectType) IsChildOf(t ObjectType) bool {
	return false
}

type ReflectValuer interface {
	Object
	Copier
	NameCallerObject
	ToIterfaceConverter
	Value() reflect.Value
	GetReflectValue() *ReflectValue
}

func NewReflectValue(v any) (ReflectValuer, error) {
	var (
		rv     = reflect.ValueOf(v)
		isnill bool
	)
	func() {
		defer func() {
			recover()
		}()
		isnill = rv.IsNil()
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
	} else if t.typ.Kind() != reflect.Func {
		ptrv := reflect.New(t.typ).Elem()
		ptrv.Set(rv)
		rv = ptrv
	}

	orv := ReflectValue{typ: t, v: rv, ptr: ptr, nil: isnill}
	switch rv.Kind() {
	case reflect.Struct:
		return &ReflectStruct{orv}, nil
	case reflect.Map:
		return &ReflectMap{orv}, nil
	case reflect.Slice:
		return &ReflectSlice{ReflectArray{orv}}, nil
	case reflect.Array:
		return &ReflectArray{orv}, nil
	case reflect.Func:
		// We allow functions with 0 or 1 result or 2 results where the second is an error.
		switch t.typ.NumOut() {
		case 0, 1:
		case 2:
			if t.typ.Out(1) != errorType {
				return nil, ErrIncompatibleReflectFuncType.NewError(fmt.Sprintf("out %d of function isn't error.", t.typ.NumOut()))
			}
		default:
			return nil, ErrIncompatibleReflectFuncType.NewError(fmt.Sprintf("function called with %d args; should be <= 2.", t.typ.NumOut()))
		}
		return &ReflectFunc{orv}, nil
	default:
		return &orv, nil
	}
}

func MustNewReflectValue(v any) ReflectValuer {
	rv, err := NewReflectValue(v)
	if err != nil {
		panic(err)
	}
	return rv
}

type ReflectValue struct {
	typ *ReflectType
	v   reflect.Value
	ptr bool
	nil bool
}

var (
	_ ReflectValuer = (*ReflectValue)(nil)
	_ Niler         = (*ReflectValue)(nil)
	_ Copier        = (*ReflectValue)(nil)
)

func (r *ReflectValue) Value() reflect.Value {
	if r.ptr {
		return r.v.Addr()
	}
	return r.v
}

func (r *ReflectValue) IsPtr() bool {
	return r.ptr
}

func (r *ReflectValue) ToInterface() any {
	return r.Value().Interface()
}

func (r *ReflectValue) Type() ObjectType {
	return r.typ
}

func (r *ReflectValue) ToString() string {
	var w strings.Builder
	w.WriteString("<reflectValue:")
	fmt.Fprintf(&w, "%+v", r)
	w.WriteString(">")
	return w.String()
}

func (r *ReflectValue) Format(s fmt.State, verb rune) {
	if verb == 'v' && s.Flag('+') {
		s.Write([]byte(r.typ.Fqn()))
		s.Write([]byte{'<'})
		if r.typ.typ.Name() == "" {
			s.Write([]byte(r.typ.typ.String()))
			s.Write([]byte{':', ' '})
		}
	}
	if r.typ.formatMethod != nil {
		r.v.Addr().Method(r.typ.formatMethod.i).Call([]reflect.Value{reflect.ValueOf(s), reflect.ValueOf(verb)})
	} else if verb == 'v' {
		s.Write([]byte(fmt.Sprint(r.ToInterface())))
	}

	if verb == 'v' && s.Flag('+') {
		s.Write([]byte{'>'})
	}
}

func (r *ReflectValue) ToStringW(w io.Writer) {
	fmt.Fprintf(w, "%+v", r)
}

func (r *ReflectValue) IsNil() bool {
	return r.nil
}

func (r *ReflectValue) Copy() (obj Object) {
	rv, _ := NewReflectValue(r.v.Interface())
	if r.ptr {
		rv.GetReflectValue().ptr = true
	}
	obj = rv
	return
}

func (r *ReflectValue) GetReflectValue() *ReflectValue {
	return r
}

func (r *ReflectValue) IsFalsy() bool {
	if r.nil || !r.v.IsValid() {
		return true
	}

	if z, _ := r.v.Addr().Interface().(interface{ IsZero() bool }); z != nil {
		return z.IsZero()
	}

	switch r.v.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return r.v.Uint() == 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return r.v.Int() == 0
	case reflect.Bool:
		return !r.v.Bool()
	case reflect.String:
		return r.v.Len() == 0
	}
	return false
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
	if tm := r.typ.methods[name]; tm != nil {
		return r.callName(tm, c)
	}
	return nil, ErrInvalidIndex.NewError(name)
}

func (r *ReflectValue) callName(tm *ReflectMethod, c Call) (Object, error) {
	return (&ReflectFunc{ReflectValue{
		typ: &ReflectType{typ: tm.m.Type},
		v:   r.v.Addr().Method(tm.i),
	}}).Call(c)
}

type ReflectFunc struct {
	ReflectValue
}

var (
	_ ReflectValuer = (*ReflectFunc)(nil)
	_ CallerObject  = (*ReflectFunc)(nil)
)

func (r *ReflectFunc) ToString() string {
	return fmt.Sprintf("<reflectFunc: %s>", r.typ.typ.String())
}

func (r *ReflectFunc) Call(c Call) (_ Object, err error) {
	typ := r.v.Type()

	var (
		numIn = typ.NumIn()
		argv  []reflect.Value
		argc  int
	)

	if typ.IsVariadic() {
		if argc = c.Args.Len(); argc < numIn-1 {
			return nil, ErrType.NewError(fmt.Sprintf("wrong number of args: got %d want at least %d", argc, numIn-1))
		}
		dddType := typ.In(numIn - 1).Elem()
		argv = make([]reflect.Value, argc)
		if err = reflectCallArgsToValues(typ, dddType, numIn, c.Args, argv); err != nil {
			return
		}
	} else if numIn == 1 && reflectCallTypeToArgv(typ.In(0), &c, &argv) {
		// argv was populated
	} else if argc = c.Args.Len(); argc != numIn {
		return nil, ErrType.NewError(fmt.Sprintf("wrong number of args: got %d want %d", argc, numIn))
	} else {
		argv = make([]reflect.Value, argc)
		if err = reflectCallArgsToValues(typ, nil, numIn, c.Args, argv); err != nil {
			return
		}
	}

	if err != nil {
		return
	}

	var ret reflect.Value
	ret, err = safeCall(r.v, argv)
	if err == nil {
		if ret.IsValid() {
			return ToObject(ret.Interface())
		}
		return Nil, nil
	}
	return
}

func reflectCallArgsToValues(typ, dddType reflect.Type, numIn int, args Args, argv []reflect.Value) (err error) {
	args.Walk(func(i int, arg Object) any {
		var rarg reflect.Value
		switch t := arg.(type) {
		case *ReflectValue:
			rarg = t.v
		default:
			rarg = reflect.ValueOf(ToInterface(arg))
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
)

func (o *ReflectArray) Format(s fmt.State, verb rune) {
	if verb == 'v' {
		s.Write([]byte("<reflectArray:"))
		o.ReflectValue.ToStringW(s)
		s.Write([]byte(">"))
		return
	}
	o.ReflectValue.Format(s, verb)
}

func (o *ReflectArray) ToString() string {
	return fmt.Sprintf("%v", o)
}

func (o *ReflectArray) IsFalsy() bool {
	return o.v.Len() == 0
}

func (o *ReflectArray) IndexGet(_ *VM, index Object) (value Object, err error) {
	var ix int
	switch t := index.(type) {
	case Int:
		ix = int(t)
		if t > math.MaxInt {
			return nil, ErrUnexpectedArgValue.NewError(fmt.Sprintf("%u > %d", t, math.MaxInt))
		}
	case Uint:
		if t > math.MaxInt {
			return nil, ErrUnexpectedArgValue.NewError(fmt.Sprintf("%u > %d", t, math.MaxInt))
		}
		ix = int(t)
	default:
		return nil, ErrUnexpectedArgValue.NewError("expected index types: int|uint")
	}

	if ix >= o.v.Len() {
		return nil, ErrIndexOutOfBounds
	}

	return ToObject(o.v.Index(ix).Interface())
}

func (o *ReflectArray) IndexSet(_ *VM, index, value Object) (err error) {
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
		v = rv.v
	} else {
		v = reflect.ValueOf(ToInterface(value))
	}

	if ix >= o.v.Len() {
		return ErrIndexOutOfBounds
	}

	o.v.Index(ix).Set(v)
	return
}

func (o *ReflectArray) Len() int {
	return o.v.Len()
}

func (o *ReflectArray) Iterate(*VM) Iterator {
	return &ReflectArrayIterator{v: o.v, l: o.v.Len()}
}

func (o *ReflectArray) Copy() (obj Object) {
	v := reflect.New(o.typ.typ).Elem()
	reflect.Copy(v, o.v)
	rv, _ := NewReflectValue(v.Interface())
	rv.GetReflectValue().ptr = o.ptr
	obj = rv
	return
}

type ReflectSlice struct {
	ReflectArray
}

func (o *ReflectSlice) Slice(low, high int) Object {
	cp := *o
	cp.v = cp.v.Slice(low, high)
	return &cp
}

var (
	_ ReflectValuer  = (*ReflectSlice)(nil)
	_ Iterabler      = (*ReflectSlice)(nil)
	_ IndexGetSetter = (*ReflectSlice)(nil)
	_ Appender       = (*ReflectSlice)(nil)
	_ Slicer         = (*ReflectSlice)(nil)
)

func (o *ReflectSlice) Append(items ...Object) (_ Object, err error) {
	var (
		itemType = o.typ.typ.Elem()
		values   = reflect.MakeSlice(o.typ.typ, len(items), len(items))
		itemV    reflect.Value
	)

	for i, item := range items {
		if itemV, err = prepareArg(reflect.ValueOf(ToInterface(item)), itemType); err != nil {
			return
		}
		values.Index(i).Set(itemV)
	}

	v := reflect.AppendSlice(o.v, values)

	if o.ptr {
		o.v.Set(v)
		return o, nil
	}

	cp := *o
	cp.v = v
	return &cp, nil
}

func (o *ReflectSlice) Format(s fmt.State, verb rune) {
	if verb == 's' {
		s.Write([]byte("<reflectSlice:"))
		o.ReflectValue.ToStringW(s)
		s.Write([]byte(">"))
		return
	}
	o.ReflectValue.Format(s, verb)
}

func (o *ReflectSlice) ToString() string {
	return fmt.Sprintf("%s", o)
}

func (o *ReflectSlice) Copy() (obj Object) {
	v := reflect.MakeSlice(o.typ.typ, o.v.Len(), o.v.Len())
	reflect.Copy(v, o.v)
	rv, _ := NewReflectValue(v.Interface())
	rv.GetReflectValue().ptr = o.ptr
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
)

func (o *ReflectMap) Format(s fmt.State, verb rune) {
	if verb == 's' {
		s.Write([]byte("<reflectMap:"))
		o.ReflectValue.ToStringW(s)
		s.Write([]byte(">"))
		return
	}
	o.ReflectValue.Format(s, verb)
}

func (o *ReflectMap) ToString() string {
	return fmt.Sprintf("%s", o)
}

func (o *ReflectMap) IndexDelete(_ *VM, index Object) (err error) {
	o.v.SetMapIndex(reflect.ValueOf(ToInterface(index)), reflect.Value{})
	return nil
}

func (o *ReflectMap) IndexGet(_ *VM, index Object) (value Object, err error) {
	v := o.v.MapIndex(reflect.ValueOf(ToInterface(index)))
	if !v.IsValid() || v.IsNil() {
		return Nil, nil
	}
	return ToObject(v.Interface())
}

func (o *ReflectMap) IndexSet(_ *VM, index, value Object) (err error) {
	var v reflect.Value
	if rv, _ := value.(*ReflectValue); rv != nil {
		v = rv.v
	} else {
		v = reflect.ValueOf(ToInterface(value))
	}

	o.v.SetMapIndex(reflect.ValueOf(ToInterface(index)), v)
	return
}

func (o *ReflectMap) Iterate(*VM) Iterator {
	return &ReflectMapIterator{v: o.v, keys: o.v.MapKeys()}
}

func (o *ReflectMap) IsFalsy() bool {
	return o.v.Len() == 0
}

func (o *ReflectMap) Copy() (obj Object) {
	v := reflect.MakeMapWithSize(o.typ.typ, o.v.Len())
	for _, k := range o.v.MapKeys() {
		v.SetMapIndex(k, o.v.MapIndex(k))
	}
	rv, _ := NewReflectValue(v.Interface())
	rv.GetReflectValue().ptr = o.ptr
	obj = rv
	return
}

type ReflectStruct struct {
	ReflectValue
}

func (r *ReflectStruct) Iterate(vm *VM) Iterator {
	return &ReflectStructIterator{vm, r, 0}
}

var (
	_ ReflectValuer  = (*ReflectStruct)(nil)
	_ Iterabler      = (*ReflectStruct)(nil)
	_ IndexGetSetter = (*ReflectStruct)(nil)
)

func (r *ReflectStruct) IndexGet(vm *VM, index Object) (value Object, err error) {
	return r.IndexGetS(vm, index.ToString())
}

func (r *ReflectStruct) IndexGetS(vm *VM, index string) (value Object, err error) {
	if field := r.typ.fields[index]; field != nil {
		value, err = ToObject(r.v.FieldByIndex(field.f.Index).Interface())
	} else if m := r.typ.methods[index]; m != nil && m.m.Type.NumIn() == 1 {
		_, err = r.callName(m, Call{VM: vm})
	} else {
		err = ErrInvalidIndex.NewError(index)
	}
	return
}

func (r *ReflectStruct) IndexSet(vm *VM, index, value Object) (err error) {
	return r.indexSet(vm, index.ToString(), value)
}

func (r *ReflectStruct) SetValues(vm *VM, values Dict) (err error) {
	for k, v := range values {
		if err = r.indexSet(vm, k, v); err != nil {
			return
		}
	}
	return
}

func (r *ReflectStruct) indexSet(vm *VM, index string, value Object) (err error) {
	field := r.typ.fields[index]
	if field != nil {
		return r.SetFieldValue(vm, field, value)
	}

	if m := r.typ.methods["Set"+strings.ToUpper(index[0:1])+index[1:]]; m != nil && m.m.Type.NumIn() == 2 {
		_, err = r.callName(m, Call{VM: vm, Args: Args{{value}}})
		return
	}

	err = ErrInvalidIndex.NewError(index)
	return
}

func (r *ReflectStruct) SetFieldValue(vm *VM, df *ReflectField, value Object) (err error) {
	field := r.v.FieldByIndex(df.f.Index)
	if value == Nil {
		field.Set(reflect.Zero(field.Type()))
		return
	} else {
		var v reflect.Value
		if rv, _ := value.(*ReflectValue); rv != nil {
			v = rv.v
		} else {
			v = reflect.ValueOf(ToInterface(value))
		}

		if df.ptr {
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

func (r *ReflectStruct) Copy() (obj Object) {
	v := reflect.New(r.typ.typ)
	v.Elem().Set(reflect.ValueOf(r.v.Interface()))
	if r.ptr {
		obj, _ = NewReflectValue(v.Interface())
	} else {
		obj, _ = NewReflectValue(v.Elem().Interface())
	}
	return
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
	return s.v.Addr(), nil
}

// safeCall runs fun.Call(args), and returns the resulting value and error, if
// any. If the call panics, the panic value is returned as an error.
func safeCall(fun reflect.Value, args []reflect.Value) (val reflect.Value, err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				e2 := ErrReflectCallPanicsType.NewError(fmt.Sprintf("%s: %v", fun, err))
				e2.Cause = e
				err = e2
			} else {
				for _, arg := range args {
					fmt.Println(arg.Type().String())
				}
				err = ErrReflectCallPanicsType.NewError(fmt.Sprintf("%s: %v", fun.Type(), r))
			}
		}
	}()
	ret := fun.Call(args)
	switch len(ret) {
	case 0:
	case 1:
		if ret[0].Type() == errorType {
			if !ret[0].IsNil() {
				err = ret[0].Interface().(error)
			}
		} else {
			val = ret[0]
		}
	case 2:
		val = ret[0]
		if !ret[1].IsNil() {
			err = ret[1].Interface().(error)
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
