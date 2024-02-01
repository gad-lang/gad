package gad

import (
	"fmt"
	"strconv"
	"strings"

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

func ArrayToString(len int, get func(i int) Object) string {
	var (
		sb   strings.Builder
		last = len - 1
	)
	sb.WriteString("[")

	for i := 0; i <= last; i++ {
		sb.WriteString(ToCode(get(i)))
		if i != last {
			sb.WriteString(", ")
		}
	}

	sb.WriteString("]")
	return sb.String()
}

func ArrayRepr(typName string, vm *VM, len int, get func(i int) Object) (_ string, err error) {
	var (
		sb    strings.Builder
		last  = len - 1
		do    = vm.Builtins.ArgsInvoker(BuiltinRepr, Call{VM: vm})
		repro Object
	)
	sb.WriteString(repr.QuotePrefix)
	sb.WriteString(typName + ":[")

	for i := 0; i <= last; i++ {
		if repro, err = do(get(i)); err != nil {
			return
		}
		sb.WriteString(repro.ToString())
		if i != last {
			sb.WriteString(", ")
		}
	}

	sb.WriteString("]")
	sb.WriteString(repr.QuoteSufix)
	return sb.String(), nil
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
		return Val(co.Call(call))
	}
}

func (vm *VM) AddCallerMethodOverride(co CallerObject, types MultipleObjectTypes, override bool, caller CallerObject) error {
	return co.(MethodCaller).AddCallerMethod(vm, types, caller, override)
}

func (vm *VM) AddCallerMethod(co CallerObject, types MultipleObjectTypes, caller CallerObject) error {
	return co.(MethodCaller).AddCallerMethod(vm, types, caller, false)
}

var (
	ReprQuote      = repr.Quote
	ReprQuoteTyped = repr.QuoteTyped
)
