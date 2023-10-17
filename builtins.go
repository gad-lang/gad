// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gad-lang/gad/token"
	"github.com/shopspring/decimal"
)

// BuiltinType represents a builtin type
type BuiltinType uint16

func (t BuiltinType) String() string {
	return BuiltinObjects[t].(*BuiltinFunction).Name
}

// Builtins
const (
	BuiltinTypesBegin_ BuiltinType = iota
	// types
	BuiltinNil
	BuiltinBool
	BuiltinInt
	BuiltinUint
	BuiltinFloat
	BuiltinDecimal
	BuiltinChar
	BuiltinString
	BuiltinBytes
	BuiltinArray
	BuiltinMap
	BuiltinSyncMap
	BuiltinKeyValue
	BuiltinKeyValueArray
	BuiltinError
	BuiltinBuffer
	BuiltinTypesEnd_

	BuiltinFunctionsBegin_
	BuiltinCast
	BuiltinAppend
	BuiltinDelete
	BuiltinCopy
	BuiltinDeepCopy
	BuiltinRepeat
	BuiltinContains
	BuiltinLen
	BuiltinSort
	BuiltinSortReverse
	BuiltinTypeName
	BuiltinChars
	BuiltinWrite
	BuiltinPrint
	BuiltinPrintf
	BuiltinPrintln
	BuiltinSprintf
	BuiltinGlobals
	BuiltinStdIO
	BuiltinWrap
	BuiltinNewType
	BuiltinTypeOf
	BuiltinMakeArray
	BuiltinCap
	BuiltinKeys
	BuiltinValues
	BuiltinItems
	BuiltinVMPushWriter
	BuiltinVMPopWriter

	BuiltinIs
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

	BuiltinFunctionsEnd_
	BuiltinErrorsBegin_
	// errors
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
	BuiltinErrorsEnd_

	BuiltinConstantsBegin_
	BuiltinDiscardWriter
	BuiltinConstantsEnd_
)

// BuiltinsMap is list of builtin types, exported for REPL.
var BuiltinsMap = map[string]BuiltinType{
	"cast":        BuiltinCast,
	"append":      BuiltinAppend,
	"delete":      BuiltinDelete,
	"copy":        BuiltinCopy,
	"dcopy":       BuiltinDeepCopy,
	"repeat":      BuiltinRepeat,
	"contains":    BuiltinContains,
	"len":         BuiltinLen,
	"sort":        BuiltinSort,
	"sortReverse": BuiltinSortReverse,
	"typeName":    BuiltinTypeName,
	"chars":       BuiltinChars,
	"write":       BuiltinWrite,
	"print":       BuiltinPrint,
	"printf":      BuiltinPrintf,
	"println":     BuiltinPrintln,
	"sprintf":     BuiltinSprintf,
	"globals":     BuiltinGlobals,
	"stdio":       BuiltinStdIO,
	"wrap":        BuiltinWrap,
	"newType":     BuiltinNewType,
	"typeof":      BuiltinTypeOf,

	"is":         BuiltinIs,
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

	"vmPushWriter":   BuiltinVMPushWriter,
	"vmPopWriter":    BuiltinVMPopWriter,
	"DISCARD_WRITER": BuiltinDiscardWriter,
}

// BuiltinObjects is list of builtins, exported for REPL.
var BuiltinObjects = map[BuiltinType]Object{
	// :makeArray is a private builtin function to help destructuring array assignments
	BuiltinMakeArray: &BuiltinFunction{
		Name:  ":makeArray",
		Value: funcPiOROe(builtinMakeArrayFunc),
	},
	BuiltinCast: &BuiltinFunction{
		Name:  "cast",
		Value: builtinCastFunc,
	},
	BuiltinChars: &BuiltinFunction{
		Name:  "chars",
		Value: funcPOROe(builtinCharsFunc),
	},
	BuiltinAppend: &BuiltinFunction{
		Name:  "append",
		Value: builtinAppendFunc,
	},
	BuiltinDelete: &BuiltinFunction{
		Name:  "delete",
		Value: builtinDeleteFunc,
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
	BuiltinTypeName: &BuiltinFunction{
		Name:  "typeName",
		Value: funcPORO(builtinTypeNameFunc),
	},
	BuiltinWrite: &BuiltinFunction{
		Name:  "write",
		Value: builtinWriteFunc,
	},
	BuiltinPrint: &BuiltinFunction{
		Name:  "print",
		Value: builtinPrintFunc,
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
	BuiltinIs: &BuiltinFunction{
		Name:  "is",
		Value: builtinIsFunc,
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
	BuiltinStdIO: &BuiltinFunction{
		Name:  "stdio",
		Value: builtinStdIO,
	},
	BuiltinWrap: &BuiltinFunction{
		Name:  "wrap",
		Value: builtinWrapFunc,
	},
	BuiltinNewType: &BuiltinFunction{
		Name:  "newType",
		Value: builtinNewTypeFunc,
	},
	BuiltinTypeOf: &BuiltinFunction{
		Name:  "typeof",
		Value: builtinTypeOfFunc,
	},
	BuiltinVMPushWriter: &BuiltinFunction{
		Name:  "vmPushWriter",
		Value: builtinVMPushWriterFunc,
	},
	BuiltinVMPopWriter: &BuiltinFunction{
		Name:  "vmPopWriter",
		Value: builtinVMPopWriterFunc,
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

	BuiltinDiscardWriter: DiscardWriter,
}

var Types = map[reflect.Type]ObjectType{}

func RegisterBuiltinType(typ BuiltinType, name string, val any, init CallableFunc) *BuiltinObjType {
	ot := &BuiltinObjType{NameValue: name, Value: init}
	BuiltinObjects[typ] = ot
	BuiltinsMap[name] = typ

	rt := reflect.TypeOf(val)
	for rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	Types[rt] = ot
	return ot
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
						vv.Type().Name(),
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
					arg.Type().Name(),
				)
				return nil, err
			}
			arg, valid = c.Args.ShiftOk()
			i++
		}
		return obj, nil
	case Appender:
		return obj.Append(c.Args.Values()...)
	default:
		return Nil, NewArgumentTypeError(
			"1st",
			"array",
			obj.Type().Name(),
		)
	}
}

func builtinDeleteFunc(c Call) (_ Object, err error) {
	var (
		target = &Arg{
			Name: "target",
			Accept: func(v Object) error {
				if _, ok := v.(IndexDeleter); !ok {
					return ErrNotIndexDeletable
				}
				return nil
			},
		}
		key = &Arg{}
	)
	if err = c.Args.Destructure(target, key); err != nil {
		return
	}
	return Nil, target.Value.(IndexDeleter).IndexDelete(c.VM, key.Value)
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
			arg.Type().Name(),
		)
	}
	return
}

func builtinContainsFunc(arg0, arg1 Object) (Object, error) {
	var ok bool
	switch obj := arg0.(type) {
	case Map:
		_, ok = obj[arg1.ToString()]
	case *SyncMap:
		_, ok = obj.Get(arg1.ToString())
	case Array:
		for _, item := range obj {
			if item.Equal(arg1) {
				ok = true
				break
			}
		}
	case *NamedArgs:
		ok = obj.Contains(arg1.ToString())
	case String:
		ok = strings.Contains(string(obj), arg1.ToString())
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
				arg1.Type().Name(),
			)
		}
	case *NilType:
	default:
		return Nil, NewArgumentTypeError(
			"1st",
			"map|array|string|bytes|namedArgs",
			arg0.Type().Name(),
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
			arg.Type().Name(),
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
		arg.Type().Name(),
	)
}

func builtinErrorFunc(arg Object) Object {
	return &Error{Name: "error", Message: arg.ToString()}
}

func builtinTypeNameFunc(arg Object) Object { return String(arg.Type().Name()) }

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
		arg.Type().Name(),
	)
}

func builtinStringFunc(c Call) (ret Object, err error) {
	if err := c.Args.CheckLen(1); err != nil {
		return Nil, err
	}

	o := c.Args.Get(0)

	if ts, _ := o.(ToStringer); ts != nil {
		return ts.Stringer(c)
	}

	ret = String(o.ToString())
	return
}

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
					args[i].Type().Name(),
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
			arg.Type().Name(),
		)
	}
	return
}

func builtinPrintfFunc(c Call) (_ Object, err error) {
	var (
		out = &NamedArgVar{Value: c.VM.StdOut, AcceptTypes: []ObjectType{TWriter}}
		n   int
	)

	if err = c.NamedArgs.Get(out); err != nil {
		return
	}

	w := out.Value.(Writer)

	switch size := c.Args.Len(); size {
	case 0:
		err = ErrWrongNumArguments.NewError("want>=1 got=0")
	case 1:
		n, err = fmt.Fprint(w, c.Args.Get(0).ToString())
	default:
		format, _ := c.Args.ShiftOk()
		vargs := make([]any, 0, size-1)
		for i := 0; i < size-1; i++ {
			vargs = append(vargs, c.Args.Get(i))
		}
		n, err = fmt.Fprintf(w, format.ToString(), vargs...)
	}
	return Int(n), err
}

func builtinWriteFunc(c Call) (ret Object, err error) {
	var (
		w     io.Writer = c.VM.StdOut
		total Int
		n     int
	)

	if err = c.Args.CheckMinLen(1); err != nil {
		return
	}

	arg := c.Args.Get(0)
	if w2, ok := arg.(Writer); ok {
		w = w2
		c.Args.Shift()
	}

	c.Args.Walk(func(i int, arg Object) (continueLoop bool) {
		switch t := arg.(type) {
		case String:
			n, err = w.Write([]byte(t))
		case Bytes:
			n, err = w.Write(t)
		case BytesConverter:
			var b Bytes
			if b, err = t.ToBytes(); err == nil {
				n, err = w.Write(b)
			}
		case io.WriterTo:
			var i64 int64
			i64, err = t.WriteTo(w)
			total = Int(i64)
		default:
			n, err = fmt.Fprint(w, arg)
		}
		total += Int(n)
		return err == nil
	})

	return total, err
}

func builtinBufferFunc(c Call) (ret Object, err error) {
	var (
		w = &Buffer{}
	)

	c.Args.Walk(func(i int, arg Object) (continueLoop bool) {
		switch t := arg.(type) {
		case String:
			_, err = w.Write([]byte(t))
		case Bytes:
			_, err = w.Write(t)
		case BytesConverter:
			var b Bytes
			if b, err = t.ToBytes(); err == nil {
				_, err = w.Write(b)
			}
		case ToWriter:
			_, err = t.WriteTo(w)
		default:
			_, err = fmt.Fprint(w, arg)
		}
		return err == nil
	})

	return w, err
}

func builtinPrintFunc(c Call) (_ Object, err error) {
	var (
		w     io.Writer = c.VM.StdOut
		total Int
		n     int
	)

	if err = c.Args.CheckMinLen(1); err != nil {
		return
	}

	arg := c.Args.Get(0)
	if w2, ok := arg.(Writer); ok {
		w = w2
		c.Args.Shift()
	}

	switch size := c.Args.Len(); size {
	case 0:
	default:
		vargs := make([]any, 0, size)
		for i := 0; i < size; i++ {
			vargs = append(vargs, c.Args.Get(i))
		}
		n, err = fmt.Fprint(w, vargs...)
		return Int(n), err
	}

	return total, err
}

func builtinPrintlnFunc(c Call) (ret Object, err error) {
	var (
		w io.Writer = c.VM.StdOut
		n int
	)

	switch size := c.Args.Len(); size {
	case 0:
		n, err = w.Write([]byte("\n"))
	case 1:
		arg := c.Args.Get(0)
		if w2, ok := arg.(Writer); ok {
			n, err = w2.Write([]byte("\n"))
		} else {
			n, err = fmt.Fprintln(w, c.Args.Get(0))
		}
	default:
		arg := c.Args.Get(0)
		if w2, ok := arg.(Writer); ok {
			w = w2
			c.Args.Shift()
			size--
		}

		vargs := make([]any, 0, size)
		for i := 0; i < size; i++ {
			vargs = append(vargs, c.Args.Get(i))
		}
		n, err = fmt.Fprintln(w, vargs...)
	}
	return Int(n), err
}

func builtinSprintfFunc(c Call) (ret Object, err error) {
	ret = Nil
	switch size := c.Args.Len(); size {
	case 0:
		err = ErrWrongNumArguments.NewError("want>=1 got=0")
	case 1:
		ret = String(c.Args.Get(0).ToString())
	default:
		format, _ := c.Args.ShiftOk()
		vargs := make([]any, 0, size-1)
		for i := 0; i < size-1; i++ {
			vargs = append(vargs, c.Args.Get(i))
		}
		ret = String(fmt.Sprintf(format.ToString(), vargs...))
	}
	return
}

func builtinGlobalsFunc(c Call) (Object, error) {
	return c.VM.GetGlobals(), nil
}

func builtinIsFunc(c Call) (ok Object, err error) {
	if err = c.Args.CheckMinLen(2); err != nil {
		return
	}
	ok = True
	var (
		types []ObjectType
		t     = c.Args.Shift()
		argt  ObjectType
	)

	if arr, ok_ := t.(Array); ok_ {
		types = make([]ObjectType, len(arr))
		for i, t := range arr {
			if ot, _ := t.(ObjectType); ok_ {
				types[i] = ot
			} else {
				return nil, NewArgumentTypeError(fmt.Sprintf("1st [%d]", i), "type", "object")
			}
		}

		c.Args.Walk(func(i int, arg Object) bool {
			argt = arg.Type()
			for _, t := range types {
				if t.Equal(argt) {
					return true
				}
			}
			ok = False
			return false
		})
	} else {
		c.Args.Walk(func(i int, arg Object) bool {
			if !t.Equal(arg.Type()) {
				ok = False
				return false
			}
			return true
		})
	}
	return
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

func builtinIsIterableFunc(arg Object) Object { return Bool(Iterable(arg)) }

func builtinKeysFunc(c Call) (_ Object, err error) {
	if err := c.Args.CheckLen(1); err != nil {
		return nil, err
	}
	var arr Array
	switch v := c.Args.Get(0).(type) {
	case KeysGetter:
		arr = v.Keys()
	default:
		if Iterable(v) {
			it := v.(Iterabler).Iterate(c.VM)
			for it.Next() {
				arr = append(arr, it.Key())
			}
		}
	}
	return arr, nil
}

func builtinValuesFunc(c Call) (_ Object, err error) {
	if err := c.Args.CheckLen(1); err != nil {
		return nil, err
	}
	var arr Array
	switch v := c.Args.Get(0).(type) {
	case Array:
		arr = v
	case ValuesGetter:
		arr = v.Values()
	default:
		if Iterable(v) {
			var (
				it = v.(Iterabler).Iterate(c.VM)
				v  Object
			)
			for it.Next() {
				if v, err = it.Value(); err != nil {
					return nil, err
				}
				arr = append(arr, v)
			}
		}
	}
	return arr, nil
}

func builtinItemsFunc(c Call) (_ Object, err error) {
	if err := c.Args.CheckLen(1); err != nil {
		return nil, err
	}
	var arr KeyValueArray
	switch v := c.Args.Get(0).(type) {
	case ItemsGetter:
		arr = v.Items()
	default:
		if Iterable(v) {
			var (
				it = v.(Iterabler).Iterate(c.VM)
				v  Object
			)
			for it.Next() {
				if v, err = it.Value(); err != nil {
					return nil, err
				}
				arr = append(arr, KeyValue{it.Key(), v})
			}
		}
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

func builtinStdIO(c Call) (ret Object, err error) {
	ret = Nil
	l := c.Args.Len()
	switch l {
	case 1:
		// get
		var arg = &Arg{AcceptTypes: []ObjectType{TString, TInt, TUint}}
		if err = c.Args.Destructure(arg); err != nil {
			return
		}
		switch t := arg.Value.(type) {
		case String:
			switch t {
			case "IN":
				ret = c.VM.StdIn
			case "OUT":
				ret = c.VM.StdOut
			case "ERR":
				ret = c.VM.StdErr
			default:
				err = ErrUnexpectedArgValue.NewError("string(" + string(t) + ")")
			}
		case Int:
			switch t {
			case 0:
				ret = c.VM.StdIn
			case 1:
				ret = c.VM.StdOut
			case 2:
				ret = c.VM.StdErr
			default:
				err = ErrUnexpectedArgValue.NewError("int(" + t.ToString() + ")")
			}
		case Uint:
			switch t {
			case 0:
				ret = c.VM.StdIn
			case 1:
				ret = c.VM.StdOut
			case 2:
				ret = c.VM.StdErr
			default:
				err = ErrUnexpectedArgValue.NewError("uint(" + t.ToString() + ")")
			}
		}
	case 2:
		var code = -1
		var codeArg = &Arg{AcceptTypes: []ObjectType{TString, TInt, TUint}}
		if err = c.Args.DestructureValue(codeArg); err != nil {
			return
		}
		switch t := codeArg.Value.(type) {
		case String:
			switch t {
			case "IN":
				code = 0
			case "OUT":
				code = 1
			case "ERR":
				code = 2
			default:
				err = ErrUnexpectedArgValue.NewError("string(" + string(t) + ")")
			}
		case Int:
			switch t {
			case 0, 1, 2:
				code = int(t)
			default:
				err = ErrUnexpectedArgValue.NewError("int(" + t.ToString() + ")")
			}
		case Uint:
			switch t {
			case 0, 1, 2:
				code = int(t)
			default:
				err = ErrUnexpectedArgValue.NewError("uint(" + t.ToString() + ")")
			}
		}

		switch code {
		case 0:
			var v = &Arg{AcceptTypes: []ObjectType{TReader}}
			if err = c.Args.DestructureValue(v); err != nil {
				return
			}
			c.VM.StdIn = NewStackReader(v.Value.(Reader))
		case 1, 2:
			var v = &Arg{AcceptTypes: []ObjectType{TWriter}}
			if err = c.Args.DestructureValue(v); err != nil {
				return
			}
			if code == 1 {
				c.VM.StdOut = NewStackWriter(v.Value.(Writer))
			} else {
				c.VM.StdErr = NewStackWriter(v.Value.(Writer))
			}
		}
	// set
	default:
		err = ErrWrongNumArguments.NewError(fmt.Sprintf("want=1|2 got=%d", l))
	}
	return
}

func builtinVMPushWriterFunc(c Call) (ret Object, err error) {
	if c.Args.Len() == 0 {
		buf := &Buffer{}
		c.VM.StdOut.Push(buf)
		return buf, nil
	}

	if err := c.Args.CheckMaxLen(1); err != nil {
		return nil, err
	}

	arg := c.Args.Get(0)
	if arg == Nil {
		arg = DiscardWriter
	}
	if w, ok := arg.(Writer); ok {
		c.VM.StdOut.Push(w)
		return w, nil
	}

	return nil, NewArgumentTypeError(
		"1st",
		"writer",
		arg.Type().Name(),
	)
}

func builtinVMPopWriterFunc(c Call) (ret Object, err error) {
	c.VM.StdOut.Pop()
	return Nil, nil
}

func builtinWrapFunc(c Call) (ret Object, err error) {
	if err = c.Args.CheckMinLen(1); err != nil {
		return
	}
	caller := c.Args.Shift()
	if !Callable(caller) {
		err = ErrNotCallable.NewError("1st arg")
	}
	return &CallWrapper{
		Caller:    caller.(CallerObject),
		Args:      c.Args.Copy().(Args),
		NamedArgs: c.NamedArgs.UnreadPairs(),
	}, nil
}

func builtinNewTypeFunc(c Call) (ret Object, err error) {
	var t ObjType
	var name = &Arg{
		Name:        "name",
		AcceptTypes: []ObjectType{TString},
	}
	if err = c.Args.Destructure(name); err != nil {
		return
	}
	t.TypeName = string(name.Value.(String))
	var (
		get = &NamedArgVar{
			Name:        "get",
			AcceptTypes: []ObjectType{TMap},
		}
		set = &NamedArgVar{
			Name:        "set",
			AcceptTypes: []ObjectType{TMap},
		}
		fields = &NamedArgVar{
			Name:        "fields",
			AcceptTypes: []ObjectType{TMap},
		}
		methods = &NamedArgVar{
			Name:        "methods",
			AcceptTypes: []ObjectType{TMap},
		}
		init = &NamedArgVar{
			Name: "init",
			Accept: func(v Object) error {
				if !Callable(v) {
					return ErrNotCallable
				}
				return nil
			},
		}
		toString = &NamedArgVar{
			Name: "toString",
			Accept: func(v Object) error {
				if !Callable(v) {
					return ErrNotCallable
				}
				return nil
			},
		}
		extends = &NamedArgVar{
			Name:        "extends",
			AcceptTypes: []ObjectType{TArray},
		}
	)

	if err = c.NamedArgs.Get(init, get, set, fields, methods, toString, extends); err != nil {
		return
	}

	if init.Value != nil {
		t.Init = init.Value.(CallerObject)
	}

	if toString.Value != nil {
		t.Stringer = toString.Value.(CallerObject)
	}

	if fields.Value != nil {
		t.fields = fields.Value.(Map)
	}

	if get.Value != nil {
		t.getters = Map{}
		for name, v := range get.Value.(Map) {
			if !Callable(v) {
				return nil, NewArgumentTypeError(
					"get["+name+"]st",
					"callable",
					v.Type().Name(),
				)
			}
			t.getters[name] = v
		}
	}

	if set.Value != nil {
		t.setters = Map{}
		for name, v := range set.Value.(Map) {
			if !Callable(v) {
				return nil, NewArgumentTypeError(
					"set["+name+"]st",
					"callable",
					v.Type().Name(),
				)
			}
			t.setters[name] = v
		}
	}
	if methods.Value != nil {
		t.methods = Map{}
		for name, v := range methods.Value.(Map) {
			if !Callable(v) {
				return nil, NewArgumentTypeError(
					"method["+name+"]st",
					"callable",
					v.Type().Name(),
				)
			}
			t.methods[name] = v
		}
	}
	if extends.Value != nil {
		arr := methods.Value.(Array)
		t.Inherits = make(ObjectTypeArray, len(arr))
		for i, v := range arr {
			if ot, _ := v.(ObjectType); ot == nil {
				return nil, NewArgumentTypeError(
					"extends["+strconv.Itoa(i)+"]st",
					"ObjectType",
					v.Type().Name(),
				)
			} else {
				t.Inherits = append(t.Inherits, ot)
				for name, f := range ot.Fields() {
					if _, ok := t.fields[name]; !ok {
						t.fields[name] = f
					}
				}
				for name, f := range ot.Getters() {
					if _, ok := t.getters[name]; !ok {
						t.getters[name] = f
					}
				}
				for name, f := range ot.Setters() {
					if _, ok := t.setters[name]; !ok {
						t.setters[name] = f
					}
				}
				for name, f := range ot.Methods() {
					if _, ok := t.methods[name]; !ok {
						t.methods[name] = f
					}
				}
			}
		}
	}
	return &t, nil
}

func builtinTypeOfFunc(c Call) (_ Object, err error) {
	if err = c.Args.CheckLen(1); err != nil {
		return
	}

	return TypeOf(c.Args.Get(0)), nil
}

func builtinSyncMapFunc(c Call) (ret Object, err error) {
	if err = c.Args.CheckMaxLen(1); err != nil {
		return
	}
	if c.Args.Len() == 0 {
		return &SyncMap{Value: map[string]Object{}}, nil
	}
	arg := c.Args.Get(0)
	switch t := arg.(type) {
	case Map:
		return &SyncMap{Value: t}, nil
	case *SyncMap:
		return t, nil
	default:
		err = NewArgumentTypeError(
			"0st",
			"map|syncMap",
			arg.Type().Name(),
		)
	}
	return
}

func TypeOf(arg Object) ObjectType {
	ot := arg.Type()
	if ot == nil {
		return typeOf(arg)
	}
	return ot
}

func typeOf(arg Object) ObjectType {
	rt := reflect.TypeOf(arg)
	for rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	ot := Types[rt]
	if ot == nil {
		ot = Nil.Type()
	}
	return ot
}

func builtinCastFunc(c Call) (ret Object, err error) {
	if err = c.Args.CheckLen(2); err != nil {
		return
	}

	var (
		typ = &Arg{Accept: func(v Object) (err error) {
			if ot, _ := v.(ObjectType); ot == nil {
				err = NewArgumentTypeError(
					"1st",
					"objectType",
					v.Type().Name(),
				)
			}
			return
		}}
		obj = &Arg{Accept: func(v Object) (err error) {
			if ot, _ := v.(Objector); ot == nil {
				err = NewArgumentTypeError(
					"2st",
					"objector",
					v.Type().Name(),
				)
			}
			return
		}}
	)
	if err = c.Args.Destructure(typ, obj); err != nil {
		return
	}
	curFields := obj.Value.(Objector).Fields()
	ot2 := typ.Value.(ObjectType)
	for f := range ot2.Fields() {
		if curFields[f] == nil {
			err = ErrIncompatibleCast.NewError(fmt.Sprintf("field %q not found in %s", f, ot2.ToString()))
			return
		}
	}
	return ot2.New(c.VM, obj.Value.(Objector).Fields())
}
