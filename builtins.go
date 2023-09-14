// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gad-lang/gad/token"
	"github.com/shopspring/decimal"
)

var (
	// PrintWriter is the default writer for printf and println builtins.
	PrintWriter io.Writer = os.Stdout
)

// BuiltinType represents a builtin type
type BuiltinType byte

// Builtins
const (
	BuiltinAppend BuiltinType = iota
	BuiltinDelete
	BuiltinCopy
	BuiltinDeepCopy
	BuiltinRepeat
	BuiltinContains
	BuiltinLen
	BuiltinSort
	BuiltinSortReverse
	BuiltinError
	BuiltinTypeName
	BuiltinBool
	BuiltinInt
	BuiltinUint
	BuiltinFloat
	BuiltinDecimal
	BuiltinChar
	BuiltinString
	BuiltinBytes
	BuiltinChars
	BuiltinPrintf
	BuiltinPrintln
	BuiltinSprintf
	BuiltinGlobals

	BuiltinIsError
	BuiltinIsInt
	BuiltinIsUint
	BuiltinIsFloat
	BuiltinIsChar
	BuiltinIsBool
	BuiltinIsString
	BuiltinIsBytes
	BuiltinIsMap
	BuiltinIsSyncMap
	BuiltinIsArray
	BuiltinIsNil
	BuiltinIsFunction
	BuiltinIsCallable
	BuiltinIsIterable

	BuiltinWrongNumArgumentsError
	BuiltinInvalidOperatorError
	BuiltinIndexOutOfBoundsError
	BuiltinNotIterableError
	BuiltinNotIndexableError
	BuiltinNotIndexAssignableError
	BuiltinNotCallableError
	BuiltinNotImplementedError
	BuiltinZeroDivisionError
	BuiltinTypeError

	BuiltinMakeArray
	BuiltinCap

	BuiltinKeys
	BuiltinValues
	BuiltinItems
	BuiltinKeyValue
	BuiltinKeyValueArray
)

// BuiltinsMap is list of builtin types, exported for REPL.
var BuiltinsMap = map[string]BuiltinType{
	"append":      BuiltinAppend,
	"delete":      BuiltinDelete,
	"copy":        BuiltinCopy,
	"dcopy":       BuiltinDeepCopy,
	"repeat":      BuiltinRepeat,
	"contains":    BuiltinContains,
	"len":         BuiltinLen,
	"sort":        BuiltinSort,
	"sortReverse": BuiltinSortReverse,
	"error":       BuiltinError,
	"typeName":    BuiltinTypeName,
	"bool":        BuiltinBool,
	"int":         BuiltinInt,
	"uint":        BuiltinUint,
	"float":       BuiltinFloat,
	"decimal":     BuiltinDecimal,
	"char":        BuiltinChar,
	"string":      BuiltinString,
	"bytes":       BuiltinBytes,
	"chars":       BuiltinChars,
	"printf":      BuiltinPrintf,
	"println":     BuiltinPrintln,
	"sprintf":     BuiltinSprintf,
	"globals":     BuiltinGlobals,

	"isError":    BuiltinIsError,
	"isInt":      BuiltinIsInt,
	"isUint":     BuiltinIsUint,
	"isFloat":    BuiltinIsFloat,
	"isChar":     BuiltinIsChar,
	"isBool":     BuiltinIsBool,
	"isString":   BuiltinIsString,
	"isBytes":    BuiltinIsBytes,
	"isMap":      BuiltinIsMap,
	"isSyncMap":  BuiltinIsSyncMap,
	"isArray":    BuiltinIsArray,
	"isNil":      BuiltinIsNil,
	"isFunction": BuiltinIsFunction,
	"isCallable": BuiltinIsCallable,
	"isIterable": BuiltinIsIterable,

	"WrongNumArgumentsError":  BuiltinWrongNumArgumentsError,
	"InvalidOperatorError":    BuiltinInvalidOperatorError,
	"IndexOutOfBoundsError":   BuiltinIndexOutOfBoundsError,
	"NotIterableError":        BuiltinNotIterableError,
	"NotIndexableError":       BuiltinNotIndexableError,
	"NotIndexAssignableError": BuiltinNotIndexAssignableError,
	"NotCallableError":        BuiltinNotCallableError,
	"NotImplementedError":     BuiltinNotImplementedError,
	"ZeroDivisionError":       BuiltinZeroDivisionError,
	"TypeError":               BuiltinTypeError,

	":makeArray": BuiltinMakeArray,
	"cap":        BuiltinCap,

	"keys":          BuiltinKeys,
	"values":        BuiltinValues,
	"items":         BuiltinItems,
	"keyValue":      BuiltinKeyValue,
	"keyValueArray": BuiltinKeyValueArray,
}

// BuiltinObjects is list of builtins, exported for REPL.
var BuiltinObjects = [...]Object{
	// :makeArray is a private builtin function to help destructuring array assignments
	BuiltinMakeArray: &BuiltinFunction{
		Name:  ":makeArray",
		Value: funcPiOROe(builtinMakeArrayFunc),
	},
	BuiltinAppend: &BuiltinFunction{
		Name:  "append",
		Value: builtinAppendFunc,
	},
	BuiltinDelete: &BuiltinFunction{
		Name:  "delete",
		Value: funcPOsRe(builtinDeleteFunc),
	},
	BuiltinCopy: &BuiltinFunction{
		Name:  "copy",
		Value: funcPORO(builtinCopyFunc),
	},
	BuiltinDeepCopy: &BuiltinFunction{
		Name:  "dcopy",
		Value: funcPORO(builtinDeepCopyFunc),
	},
	BuiltinRepeat: &BuiltinFunction{
		Name:  "repeat",
		Value: funcPOiROe(builtinRepeatFunc),
	},
	BuiltinContains: &BuiltinFunction{
		Name:  "contains",
		Value: funcPOOROe(builtinContainsFunc),
	},
	BuiltinLen: &BuiltinFunction{
		Name:  "len",
		Value: funcPORO(builtinLenFunc),
	},
	BuiltinCap: &BuiltinFunction{
		Name:  "cap",
		Value: funcPORO(builtinCapFunc),
	},
	BuiltinSort: &BuiltinFunction{
		Name:  "sort",
		Value: funcPOROe(builtinSortFunc),
	},
	BuiltinSortReverse: &BuiltinFunction{
		Name:  "sortReverse",
		Value: funcPOROe(builtinSortReverseFunc),
	},
	BuiltinError: &BuiltinFunction{
		Name:  "error",
		Value: funcPORO(builtinErrorFunc),
	},
	BuiltinTypeName: &BuiltinFunction{
		Name:  "typeName",
		Value: funcPORO(builtinTypeNameFunc),
	},
	BuiltinBool: &BuiltinFunction{
		Name:  "bool",
		Value: funcPORO(builtinBoolFunc),
	},
	BuiltinInt: &BuiltinFunction{
		Name:  "int",
		Value: funcPi64RO(builtinIntFunc),
	},
	BuiltinUint: &BuiltinFunction{
		Name:  "uint",
		Value: funcPu64RO(builtinUintFunc),
	},
	BuiltinFloat: &BuiltinFunction{
		Name:  "float",
		Value: funcPf64RO(builtinFloatFunc),
	},
	BuiltinDecimal: &BuiltinFunction{
		Name:  "decimal",
		Value: funcPOROe(builtinDecimalFunc),
	},
	BuiltinChar: &BuiltinFunction{
		Name:  "char",
		Value: funcPOROe(builtinCharFunc),
	},
	BuiltinString: &BuiltinFunction{
		Name:  "string",
		Value: funcPORO(builtinStringFunc),
	},
	BuiltinBytes: &BuiltinFunction{
		Name:  "bytes",
		Value: builtinBytesFunc,
	},
	BuiltinChars: &BuiltinFunction{
		Name:  "chars",
		Value: funcPOROe(builtinCharsFunc),
	},
	BuiltinPrintf: &BuiltinFunction{
		Name:  "printf",
		Value: builtinPrintfFunc,
	},
	BuiltinPrintln: &BuiltinFunction{
		Name:  "println",
		Value: builtinPrintlnFunc,
	},
	BuiltinSprintf: &BuiltinFunction{
		Name:  "sprintf",
		Value: builtinSprintfFunc,
	},
	BuiltinGlobals: &BuiltinFunction{
		Name:  "globals",
		Value: builtinGlobalsFunc,
	},
	BuiltinIsError: &BuiltinFunction{
		Name:  "isError",
		Value: builtinIsErrorFunc,
	},
	BuiltinIsInt: &BuiltinFunction{
		Name:  "isInt",
		Value: funcPORO(builtinIsIntFunc),
	},
	BuiltinIsUint: &BuiltinFunction{
		Name:  "isUint",
		Value: funcPORO(builtinIsUintFunc),
	},
	BuiltinIsFloat: &BuiltinFunction{
		Name:  "isFloat",
		Value: funcPORO(builtinIsFloatFunc),
	},
	BuiltinIsChar: &BuiltinFunction{
		Name:  "isChar",
		Value: funcPORO(builtinIsCharFunc),
	},
	BuiltinIsBool: &BuiltinFunction{
		Name:  "isBool",
		Value: funcPORO(builtinIsBoolFunc),
	},
	BuiltinIsString: &BuiltinFunction{
		Name:  "isString",
		Value: funcPORO(builtinIsStringFunc),
	},
	BuiltinIsBytes: &BuiltinFunction{
		Name:  "isBytes",
		Value: funcPORO(builtinIsBytesFunc),
	},
	BuiltinIsMap: &BuiltinFunction{
		Name:  "isMap",
		Value: funcPORO(builtinIsMapFunc),
	},
	BuiltinIsSyncMap: &BuiltinFunction{
		Name:  "isSyncMap",
		Value: funcPORO(builtinIsSyncMapFunc),
	},
	BuiltinIsArray: &BuiltinFunction{
		Name:  "isArray",
		Value: funcPORO(builtinIsArrayFunc),
	},
	BuiltinIsNil: &BuiltinFunction{
		Name:  "isNil",
		Value: funcPORO(builtinIsNilFunc),
	},
	BuiltinIsFunction: &BuiltinFunction{
		Name:  "isFunction",
		Value: funcPORO(builtinIsFunctionFunc),
	},
	BuiltinIsCallable: &BuiltinFunction{
		Name:  "isCallable",
		Value: funcPORO(builtinIsCallableFunc),
	},
	BuiltinIsIterable: &BuiltinFunction{
		Name:  "isIterable",
		Value: funcPORO(builtinIsIterableFunc),
	},
	BuiltinKeys: &BuiltinFunction{
		Name:  "keys",
		Value: builtinKeysFunc,
	},
	BuiltinValues: &BuiltinFunction{
		Name:  "values",
		Value: builtinValuesFunc,
	},
	BuiltinItems: &BuiltinFunction{
		Name:  "items",
		Value: builtinItemsFunc,
	},
	BuiltinKeyValue: &BuiltinFunction{
		Name:  "keyValue",
		Value: builtinKeyValueFunc,
	},
	BuiltinKeyValueArray: &BuiltinFunction{
		Name:  "keyValueArray",
		Value: builtinKeyValueArrayFunc,
	},

	BuiltinWrongNumArgumentsError:  ErrWrongNumArguments,
	BuiltinInvalidOperatorError:    ErrInvalidOperator,
	BuiltinIndexOutOfBoundsError:   ErrIndexOutOfBounds,
	BuiltinNotIterableError:        ErrNotIterable,
	BuiltinNotIndexableError:       ErrNotIndexable,
	BuiltinNotIndexAssignableError: ErrNotIndexAssignable,
	BuiltinNotCallableError:        ErrNotCallable,
	BuiltinNotImplementedError:     ErrNotImplemented,
	BuiltinZeroDivisionError:       ErrZeroDivision,
	BuiltinTypeError:               ErrType,
}

func builtinMakeArrayFunc(n int, arg Object) (Object, error) {
	if n <= 0 {
		return arg, nil
	}

	arr, ok := arg.(Array)
	if !ok {
		ret := make(Array, n)
		for i := 1; i < n; i++ {
			ret[i] = Nil
		}
		ret[0] = arg
		return ret, nil
	}

	length := len(arr)
	if n <= length {
		return arr[:n], nil
	}

	ret := make(Array, n)
	x := copy(ret, arr)
	for i := x; i < n; i++ {
		ret[i] = Nil
	}
	return ret, nil
}

func builtinAppendFunc(c Call) (Object, error) {
	target, ok := c.Args.ShiftOk()
	if !ok {
		return Nil, ErrWrongNumArguments.NewError("want>=1 got=0")
	}

	switch obj := target.(type) {
	case Array:
		for _, arg := range c.Args {
			arg.AppendToArray(&obj)
		}
		return obj, nil
	case Bytes:
		n := 0
		for _, args := range c.Args {
			for _, v := range args {
				n++
				switch vv := v.(type) {
				case Int:
					obj = append(obj, byte(vv))
				case Uint:
					obj = append(obj, byte(vv))
				case Char:
					obj = append(obj, byte(vv))
				default:
					return Nil, NewArgumentTypeError(
						strconv.Itoa(n),
						"int|uint|char",
						vv.TypeName(),
					)
				}
			}
		}
		return obj, nil
	case *NilType:
		ret := make(Array, 0, c.Args.Len())
		for _, arg := range c.Args {
			arg.AppendToArray(&ret)
		}
		return ret, nil
	case KeyValueArray:
		var (
			err        error
			i          = 1
			arg, valid = c.Args.ShiftOk()
		)

		for valid {
			if obj, err = obj.AppendObject(arg); err != nil {
				err = NewArgumentTypeError(
					strconv.Itoa(i)+"st",
					err.Error(),
					arg.TypeName(),
				)
				return nil, err
			}
			arg, valid = c.Args.ShiftOk()
			i++
		}
		return obj, nil
	default:
		return Nil, NewArgumentTypeError(
			"1st",
			"array",
			obj.TypeName(),
		)
	}
}

func builtinDeleteFunc(arg Object, key string) (err error) {
	if v, ok := arg.(IndexDeleter); ok {
		err = v.IndexDelete(String(key))
	} else {
		err = NewArgumentTypeError(
			"1st",
			"map|syncMap|IndexDeleter",
			arg.TypeName(),
		)
	}
	return
}

func builtinCopyFunc(arg Object) Object {
	if v, ok := arg.(Copier); ok {
		return v.Copy()
	}
	return arg
}

func builtinDeepCopyFunc(arg Object) Object {
	if v, ok := arg.(DeepCopier); ok {
		return v.DeepCopy()
	} else if v, ok := arg.(Copier); ok {
		return v.Copy()
	}
	return arg
}

func builtinRepeatFunc(arg Object, count int) (ret Object, err error) {
	if count < 0 {
		return nil, NewArgumentTypeError(
			"2nd",
			"non-negative integer",
			"negative integer",
		)
	}

	switch v := arg.(type) {
	case Array:
		out := make(Array, 0, len(v)*count)
		for i := 0; i < count; i++ {
			out = append(out, v...)
		}
		ret = out
	case String:
		ret = String(strings.Repeat(string(v), count))
	case Bytes:
		ret = Bytes(bytes.Repeat(v, count))
	default:
		err = NewArgumentTypeError(
			"1st",
			"array|string|bytes",
			arg.TypeName(),
		)
	}
	return
}

func builtinContainsFunc(arg0, arg1 Object) (Object, error) {
	var ok bool
	switch obj := arg0.(type) {
	case Map:
		_, ok = obj[arg1.String()]
	case *SyncMap:
		_, ok = obj.Get(arg1.String())
	case Array:
		for _, item := range obj {
			if item.Equal(arg1) {
				ok = true
				break
			}
		}
	case *NamedArgs:
		ok = obj.Contains(arg1.String())
	case String:
		ok = strings.Contains(string(obj), arg1.String())
	case Bytes:
		switch v := arg1.(type) {
		case Int:
			ok = bytes.Contains(obj, []byte{byte(v)})
		case Uint:
			ok = bytes.Contains(obj, []byte{byte(v)})
		case Char:
			ok = bytes.Contains(obj, []byte{byte(v)})
		case String:
			ok = bytes.Contains(obj, []byte(v))
		case Bytes:
			ok = bytes.Contains(obj, v)
		default:
			return Nil, NewArgumentTypeError(
				"2nd",
				"int|uint|string|char|bytes",
				arg1.TypeName(),
			)
		}
	case *NilType:
	default:
		return Nil, NewArgumentTypeError(
			"1st",
			"map|array|string|bytes|namedArgs",
			arg0.TypeName(),
		)
	}
	return Bool(ok), nil
}

func builtinLenFunc(arg Object) Object {
	var n int
	if v, ok := arg.(LengthGetter); ok {
		n = v.Len()
	}
	return Int(n)
}

func builtinCapFunc(arg Object) Object {
	var n int
	switch v := arg.(type) {
	case Array:
		n = cap(v)
	case Bytes:
		n = cap(v)
	}
	return Int(n)
}

func builtinSortFunc(arg Object) (ret Object, err error) {
	switch obj := arg.(type) {
	case Sorter:
		ret, err = obj.Sort()
	case String:
		s := []rune(obj)
		sort.Slice(s, func(i, j int) bool {
			return s[i] < s[j]
		})
		ret = String(s)
	case Bytes:
		sort.Slice(obj, func(i, j int) bool {
			return obj[i] < obj[j]
		})
		ret = arg
	case *NilType:
		ret = Nil
	default:
		ret = Nil
		err = NewArgumentTypeError(
			"1st",
			"array|string|bytes",
			arg.TypeName(),
		)
	}
	return
}

func builtinSortReverseFunc(arg Object) (Object, error) {
	switch obj := arg.(type) {
	case ReverseSorter:
		return obj.SortReverse()
	case String:
		s := []rune(obj)
		sort.Slice(s, func(i, j int) bool {
			return s[j] < s[i]
		})
		return String(s), nil
	case Bytes:
		sort.Slice(obj, func(i, j int) bool {
			return obj[j] < obj[i]
		})
		return obj, nil
	case *NilType:
		return Nil, nil
	}

	return Nil, NewArgumentTypeError(
		"1st",
		"array|string|bytes",
		arg.TypeName(),
	)
}

func builtinErrorFunc(arg Object) Object {
	return &Error{Name: "error", Message: arg.String()}
}

func builtinTypeNameFunc(arg Object) Object { return String(arg.TypeName()) }

func builtinBoolFunc(arg Object) Object { return Bool(!arg.IsFalsy()) }

func builtinIntFunc(v int64) Object { return Int(v) }

func builtinUintFunc(v uint64) Object { return Uint(v) }

func builtinFloatFunc(v float64) Object { return Float(v) }

func builtinDecimalFunc(v Object) (Object, error) {
	return Decimal(decimal.Zero).BinaryOp(token.Add, v)
}

func builtinCharFunc(arg Object) (Object, error) {
	v, ok := ToChar(arg)
	if ok && v != utf8.RuneError {
		return v, nil
	}
	if v == utf8.RuneError || arg == Nil {
		return Nil, nil
	}
	return Nil, NewArgumentTypeError(
		"1st",
		"numeric|string|bool",
		arg.TypeName(),
	)
}

func builtinStringFunc(arg Object) Object { return String(arg.String()) }

func builtinBytesFunc(c Call) (Object, error) {
	size := c.Args.Len()

	switch size {
	case 0:
		return Bytes{}, nil
	case 1:
		if v, ok := ToBytes(c.Args.Get(0)); ok {
			return v, nil
		}
	}

	out := make(Bytes, 0, size)
	for _, args := range c.Args {
		for i, obj := range args {
			switch v := obj.(type) {
			case Int:
				out = append(out, byte(v))
			case Uint:
				out = append(out, byte(v))
			case Char:
				out = append(out, byte(v))
			default:
				return Nil, NewArgumentTypeError(
					strconv.Itoa(i+1),
					"int|uint|char",
					args[i].TypeName(),
				)
			}
		}
	}
	return out, nil
}

func builtinCharsFunc(arg Object) (ret Object, err error) {
	switch obj := arg.(type) {
	case String:
		s := string(obj)
		ret = make(Array, 0, utf8.RuneCountInString(s))
		sz := len(obj)
		i := 0

		for i < sz {
			r, w := utf8.DecodeRuneInString(s[i:])
			if r == utf8.RuneError {
				return Nil, nil
			}
			ret = append(ret.(Array), Char(r))
			i += w
		}
	case Bytes:
		ret = make(Array, 0, utf8.RuneCount(obj))
		sz := len(obj)
		i := 0

		for i < sz {
			r, w := utf8.DecodeRune(obj[i:])
			if r == utf8.RuneError {
				return Nil, nil
			}
			ret = append(ret.(Array), Char(r))
			i += w
		}
	default:
		ret = Nil
		err = NewArgumentTypeError(
			"1st",
			"string|bytes",
			arg.TypeName(),
		)
	}
	return
}

func builtinPrintfFunc(c Call) (ret Object, err error) {
	ret = Nil
	switch size := c.Args.Len(); size {
	case 0:
		err = ErrWrongNumArguments.NewError("want>=1 got=0")
	case 1:
		_, err = fmt.Fprint(PrintWriter, c.Args.Get(0).String())
	default:
		format, _ := c.Args.ShiftOk()
		vargs := make([]interface{}, 0, size-1)
		for i := 0; i < size-1; i++ {
			vargs = append(vargs, c.Args.Get(i))
		}
		_, err = fmt.Fprintf(PrintWriter, format.String(), vargs...)
	}
	return
}

func builtinPrintlnFunc(c Call) (ret Object, err error) {
	ret = Nil
	switch size := c.Args.Len(); size {
	case 0:
		_, err = fmt.Fprintln(PrintWriter)
	case 1:
		_, err = fmt.Fprintln(PrintWriter, c.Args.Get(0))
	default:
		vargs := make([]interface{}, 0, size)
		for i := 0; i < size; i++ {
			vargs = append(vargs, c.Args.Get(i))
		}
		_, err = fmt.Fprintln(PrintWriter, vargs...)
	}
	return
}

func builtinSprintfFunc(c Call) (ret Object, err error) {
	ret = Nil
	switch size := c.Args.Len(); size {
	case 0:
		err = ErrWrongNumArguments.NewError("want>=1 got=0")
	case 1:
		ret = String(c.Args.Get(0).String())
	default:
		format, _ := c.Args.ShiftOk()
		vargs := make([]interface{}, 0, size-1)
		for i := 0; i < size-1; i++ {
			vargs = append(vargs, c.Args.Get(i))
		}
		ret = String(fmt.Sprintf(format.String(), vargs...))
	}
	return
}

func builtinGlobalsFunc(c Call) (Object, error) {
	return c.VM().GetGlobals(), nil
}

func builtinIsErrorFunc(c Call) (ret Object, err error) {
	ret = False
	switch c.Args.Len() {
	case 1:
		// We have Error, BuiltinError and also user defined error types.
		if _, ok := c.Args.Get(0).(error); ok {
			ret = True
		}
	case 2:
		if err, ok := c.Args.Get(0).(error); ok {
			if target, ok := c.Args.Get(1).(error); ok {
				ret = Bool(errors.Is(err, target))
			}
		}
	default:
		err = ErrWrongNumArguments.NewError(
			"want=1..2 got=", strconv.Itoa(c.Args.Len()))
	}
	return
}

func builtinIsIntFunc(arg Object) Object {
	_, ok := arg.(Int)
	return Bool(ok)
}

func builtinIsUintFunc(arg Object) Object {
	_, ok := arg.(Uint)
	return Bool(ok)
}

func builtinIsFloatFunc(arg Object) Object {
	_, ok := arg.(Float)
	return Bool(ok)
}

func builtinIsCharFunc(arg Object) Object {
	_, ok := arg.(Char)
	return Bool(ok)
}

func builtinIsBoolFunc(arg Object) Object {
	_, ok := arg.(Bool)
	return Bool(ok)
}

func builtinIsStringFunc(arg Object) Object {
	_, ok := arg.(String)
	return Bool(ok)
}

func builtinIsBytesFunc(arg Object) Object {
	_, ok := arg.(Bytes)
	return Bool(ok)
}

func builtinIsMapFunc(arg Object) Object {
	_, ok := arg.(Map)
	return Bool(ok)
}

func builtinIsSyncMapFunc(arg Object) Object {
	_, ok := arg.(*SyncMap)
	return Bool(ok)
}

func builtinIsArrayFunc(arg Object) Object {
	_, ok := arg.(Array)
	return Bool(ok)
}

func builtinIsNilFunc(arg Object) Object {
	_, ok := arg.(*NilType)
	return Bool(ok)
}

func builtinIsFunctionFunc(arg Object) Object {
	_, ok := arg.(*CompiledFunction)
	if ok {
		return True
	}

	_, ok = arg.(*BuiltinFunction)
	if ok {
		return True
	}

	_, ok = arg.(*Function)
	return Bool(ok)
}

func builtinIsCallableFunc(arg Object) Object {
	return Bool(Callable(arg))
}

func builtinIsIterableFunc(arg Object) Object { return Bool(arg.CanIterate()) }

func builtinKeysFunc(c Call) (Object, error) {
	if err := c.Args.CheckLen(1); err != nil {
		return nil, err
	}
	var arr Array
	switch v := c.Args.Get(0).(type) {
	case KeysGetter:
		arr = v.Keys()
	}
	return arr, nil
}

func builtinValuesFunc(c Call) (Object, error) {
	if err := c.Args.CheckLen(1); err != nil {
		return nil, err
	}
	var arr Array
	switch v := c.Args.Get(0).(type) {
	case Array:
		arr = v
	case ValuesGetter:
		arr = v.Values()
	}
	return arr, nil
}

func builtinItemsFunc(c Call) (Object, error) {
	if err := c.Args.CheckLen(1); err != nil {
		return nil, err
	}
	var arr KeyValueArray
	switch v := c.Args.Get(0).(type) {
	case ItemsGetter:
		arr = v.Items()
	}
	return arr, nil
}

func builtinKeyValueFunc(c Call) (ret Object, err error) {
	if err := c.Args.CheckLen(2); err != nil {
		return nil, err
	}
	return KeyValue{c.Args.Get(0), c.Args.Get(1)}, nil
}

func builtinKeyValueArrayFunc(c Call) (Object, error) {
	var (
		arr        KeyValueArray
		arg, valid = c.Args.ShiftOk()
	)

	for valid {
		switch t := arg.(type) {
		case KeyValueArray:
			arr = append(arr, t...)
		case KeyValue:
			arr = append(arr, t)
		case Array:
			if len(t) == 2 {
				arr = append(arr, KeyValue{t[0], t[1]})
			}
		case ItemsGetter:
			arr = append(arr, t.Items()...)
		}
		arg, valid = c.Args.ShiftOk()
	}
	return arr, nil
}
