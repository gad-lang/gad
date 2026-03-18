package gad

import (
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"unsafe"

	"github.com/gad-lang/gad/repr"
)

func ToCode(o Object) string {
	switch v := o.(type) {
	case Str:
		return strconv.Quote(v.ToString())
	case Char:
		return strconv.QuoteRune(rune(v))
	case Bytes:
		return fmt.Sprint([]byte(v))
	default:
		return v.ToString()
	}
}

func ObjectsStrW(w io.Writer, vm *VM, len int, get func(i int) Object) (err error) {
	var (
		last  = len - 1
		do    = vm.Builtins.ArgsInvoker(BuiltinStr, Call{VM: vm})
		repro Object
	)

	for i := 0; i <= last; i++ {
		if repro, err = do(get(i)); err != nil {
			return
		}
		w.Write([]byte(repro.ToString()))
		if i != last {
			w.Write([]byte{',', ' '})
		}
	}
	return
}

func ArrayToString(len int, get func(i int) Object) string {
	return ArrayToStringBraces("[", "]", len, get)
}

func ArrayToStringBraces(lb, rb string, len int, get func(i int) Object) string {
	var (
		sb   strings.Builder
		last = len - 1
	)

	sb.WriteString(lb)

	for i := 0; i <= last; i++ {
		sb.WriteString(ToCode(get(i)))
		if i != last {
			sb.WriteString(", ")
		}
	}

	sb.WriteString(rb)
	return sb.String()
}

func AnyMapToMap(src map[string]any) (m Dict, err error) {
	m = make(Dict, len(src))
	for k, v := range src {
		if m[k], err = ToObject(v); err != nil {
			return
		}
	}
	return
}

func NewArgCaller(vm *VM, co CallerObject, args Array, namedArgs NamedArgs) func() (ret Object, err error) {
	call := Call{
		VM:        vm,
		Args:      Args{args},
		NamedArgs: namedArgs,
	}
	return func() (ret Object, err error) {
		return DoCall(co, call)
	}
}

func (vm *VM) AddCallerMethodOverride(co CallerObject, types ParamsTypes, override bool, caller CallerObject, onAdd func(method *TypedCallerMethod) error) (err error) {
	return co.(MethodCaller).AddMethodByTypes(vm, types, caller, override, onAdd)
}

func (vm *VM) AddCallerMethod(co CallerObject, types ParamsTypes, caller CallerObject, onAdd func(method *TypedCallerMethod) error) (err error) {
	return co.(MethodCaller).AddMethodByTypes(vm, types, caller, false, onAdd)
}

func ObjectOrNil(v Object) Object {
	if v == nil {
		return Nil
	}
	return v
}

func SplitCaller(vm *VM, caller Object, cb func(co CallerObject, types ParamsTypes) error, fallback ...func(co CallerObject) error) (err error) {
	switch v := caller.(type) {
	case CallerObjectWithParamTypes:
		return cb(v, v.ParamTypes())
	case CallerObjectWithVMParamTypes:
		var types ParamsTypes
		if types, err = v.ParamTypes(vm); err != nil {
			return
		}
		return cb(v, types)
	case *Func:
		err, _ = v.Methods.Walk(func(m *TypedCallerMethod) any {
			types := make(ParamsTypes, len(m.types))
			for i, typ := range m.types {
				types[i] = ObjectTypes{typ}
			}
			return cb(m.CallerObject, types)
		}).(error)
		return err
	default:
		if Callable(caller) && len(fallback) == 1 {
			return fallback[0](caller.(CallerObject))
		}
	}
	return ErrType.NewError("object isn't Caller")
}

func ParamTypesOfRawCaller(vm *VM, caller Object) (types ParamsTypes, err error) {
	switch v := caller.(type) {
	case CallerObjectWithParamTypes:
		return v.ParamTypes(), nil
	case CallerObjectWithVMParamTypes:
		return v.ParamTypes(vm)
	default:
		return nil, ErrType.NewError("object isn't Raw Caller")
	}
}

func IsPrimitive(obj Object) bool {
	switch obj.(type) {
	case *NilType, Decimal:
		return true
	}

	val := reflect.ValueOf(obj)
try:
	// skip primitive values
	switch val.Type().Kind() {
	case reflect.Interface:
		val = val.Elem()
		goto try
	case reflect.Map, reflect.Ptr, reflect.Slice, reflect.Array, reflect.Func, reflect.Chan, reflect.Struct:
		return false
	default:
		return true
	}
}

func AddressOf(obj Object) unsafe.Pointer {
	type entry struct {
		object Object
	}
	if !IsPrimitive(obj) {
		entry := entry{obj}
		return reflect.ValueOf(entry.object).UnsafePointer()
	}
	return nil
}

func IsAssignableTo(typ, refType ObjectType) (ok bool) {
	if refType == TAny || typ.Equal(refType) {
		return true
	}
	if inferer, _ := typ.(ObjectTypeAssignersGetter); inferer != nil {
		inferer.GetTypeAssigners(func(t ObjectType) any {
			if ok = t.Equal(refType); ok {
				return true
			}
			return nil
		})
	}
	return
}

func TypeAssigners(t ObjectType, cb func(t ObjectType) any) (ret any) {
	if ret = cb(t); ret != nil {
		return
	}
	if inferer, _ := t.(ObjectTypeAssignersGetter); inferer != nil {
		if ret = inferer.GetTypeAssigners(cb); ret != nil {
			return
		}
	}
	return cb(TAny)
}

var (
	ReprQuote      = repr.Quote
	ReprQuoteTyped = repr.QuoteTyped
)
