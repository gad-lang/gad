package gad

import (
	"fmt"
	"go/ast"
	"reflect"
	"sync"
)

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

type ReflectValuePrinter interface {
	GadPrint(state *PrinterState) (err error)
}

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

var _ ObjectType = (*ReflectType)(nil)

type ReflectType struct {
	RType             reflect.Type
	RMethods          map[string]*ReflectMethod
	FieldsNames       []string
	RFields           map[string]*ReflectField
	formatMethod      *ReflectMethod
	CallObject        func(obj *ReflectStruct, c Call) (Object, error)
	InstancePrintFunc func(state *PrinterState, obj *ReflectValue) error
	*FuncSpec
}

func NewReflectType(t any) (rt *ReflectType) {
	var typ reflect.Type
	switch t := t.(type) {
	case reflect.Type:
		typ = t
	default:
		typ = reflect.TypeOf(t)
	}

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
		ptrType = reflect.PointerTo(typ)
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

	if rt.RType.Implements(reflect.TypeOf((*ReflectValuePrinter)(nil)).Elem()) {
		rt.InstancePrintFunc = func(state *PrinterState, obj *ReflectValue) (err error) {
			return obj.ToInterface().(ReflectValuePrinter).GadPrint(state)
		}
	} else if rt.RType.Implements(reflect.TypeOf((*fmt.Formatter)(nil)).Elem()) {
		rt.InstancePrintFunc = func(state *PrinterState, obj *ReflectValue) (err error) {
			_, err = fmt.Fprintf(state.writer, "%v", obj.ToInterface())
			return
		}
	} else if rt.RType.Implements(reflect.TypeOf((*fmt.Stringer)(nil)).Elem()) {
		rt.InstancePrintFunc = func(state *PrinterState, obj *ReflectValue) (err error) {
			_, err = state.Write([]byte(obj.ToInterface().(fmt.Stringer).String()))
			return
		}
	}

	rt.FuncSpec = NewFuncSpec(rt)
	rt.FuncSpec.defaul = &Function{
		FuncName: "#default",
		Value:    rt.NewDefault,
	}

	return
}

func (ReflectType) GadObjectType() {}

func (t *ReflectType) FuncSpecName() string {
	return "reflect type " + ReprQuote(t.Fqn())
}

func (t *ReflectType) Name() string {
	return t.Fqn()
}

func (t *ReflectType) FullName() string {
	return t.Fqn()
}

func (t *ReflectType) Type() ObjectType {
	return TBase
}

func (t *ReflectType) ToString() string {
	return t.String()
}

func (t *ReflectType) String() string {
	return string(MustToStr(nil, t))
}

func (t *ReflectType) Print(state *PrinterState) (err error) {
	return t.PrintFuncWrapper(state, t)
}

func (t *ReflectType) Equal(right Object) bool {
	if o, _ := right.(*ReflectType); o != nil {
		return o.RType == t.RType
	}
	return false
}

func (t *ReflectType) NewDefault(c Call) (Object, error) {
	if c.NamedArgs.IsFalsy() {
		return t.New(c.VM, nil)
	}
	return t.New(c.VM, c.NamedArgs.Dict())
}

func (t *ReflectType) Call(c Call) (_ Object, err error) {
	caller, validate := t.CallerMethodWithValidationCheckOfArgs(c.Args)
	c.SafeArgs = !validate
	if c.Args.IsFalsy() {
		c.Args = append(c.Args, Array{t})
	} else {
		c.Args = append(Args{Array{t}}, c.Args...)
	}
	return YieldCall(caller, &c), nil
}

func (t *ReflectType) Fqn() string {
	var n = t.RType.Name()
	if n == "" {
		n = t.RType.Kind().String()
	} else {
		n = t.RType.PkgPath() + "." + n
	}
	return n
}

func (t *ReflectType) Fields() (fields Dict) {
	fields = Dict{}
	for _, f := range t.RFields {
		fields[f.Struct.Name] = f
	}
	return
}

func (t *ReflectType) GetRMethods() map[string]*ReflectMethod {
	return t.RMethods
}

func (t *ReflectType) New(vm *VM, m Dict) (_ Object, err error) {
	var rv reflect.Value
	switch t.RType.Kind() {
	case reflect.Struct:
		rv = reflect.New(t.RType).Elem()
		obj := &ReflectStruct{
			ReflectValue: ReflectValue{
				RType:   t,
				RValue:  rv,
				Options: &ReflectValueOptions{},
				ptr:     true,
			},
		}

		obj.Init()

		for s, v := range m {
			if err = obj.indexSet(vm, s, v); err != nil {
				return
			}
		}
		return obj, nil
	case reflect.Map:
		rv = reflect.MakeMap(t.RType)
		for k, v := range m {
			rv.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(vm.ToInterface(v)))
		}
	}
	return &ReflectValue{RType: t, RValue: rv, Options: &ReflectValueOptions{}}, nil
}

func ReflectTypeOf(v any) (rt *ReflectType) {
	return NewReflectType(indirectInterface(reflect.ValueOf(v)).Type())
}
