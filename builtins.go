// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import (
	"fmt"
	"sync"
)

//go:generate go run ./cmd/mkcallable -output builtins_zfuncs.go builtins.go

// BuiltinType represents a builtin type
type BuiltinType uint16

func (t BuiltinType) String() string {
	switch bt := BuiltinObjects[t].(type) {
	case *BuiltinFunction:
		return bt.Name
	case *BuiltinObjType:
		return bt.NameValue
	case fmt.Stringer:
		return bt.String()
	default:
		return fmt.Sprintf("<unknown built-in type: %d>", t)
	}
}

// Builtins
const (
	BuiltinTypesBegin_ BuiltinType = iota
	// types
	BuiltinNil
	BuiltinFlag
	BuiltinBool
	BuiltinInt
	BuiltinUint
	BuiltinFloat
	BuiltinDecimal
	BuiltinChar
	BuiltinRawStr
	BuiltinStr
	BuiltinBytes
	BuiltinArray
	BuiltinDict
	BuiltinSyncDic
	BuiltinKeyValue
	BuiltinKeyValueArray
	BuiltinError
	BuiltinBuffer
	BuiltinRegexp
	BuiltinRegexpStrsResult
	BuiltinRegexpStrsSliceResult
	BuiltinRegexpBytesResult
	BuiltinRegexpBytesSliceResult
	BuiltinIterator
	BuiltinZipIterator
	BuiltinTypesEnd_

	BuiltinFunctionsBegin_
	BuiltinBinaryOp
	BuiltinRepr
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
	BuiltinFilter
	BuiltinMap
	BuiltinEach
	BuiltinReduce
	BuiltinTypeName
	BuiltinChars
	BuiltinClose
	BuiltinRead
	BuiltinWrite
	BuiltinPrint
	BuiltinPrintf
	BuiltinPrintln
	BuiltinSprintf
	BuiltinGlobals
	BuiltinStdIO
	BuiltinWrap
	BuiltinStruct
	BuiltinNew
	BuiltinTypeOf
	BuiltinAddCallMethod
	BuiltinRawCaller
	BuiltinMakeArray
	BuiltinCap
	BuiltinIterate
	BuiltinKeys
	BuiltinValues
	BuiltinItems
	BuiltinCollect
	BuiltinEnumerate
	BuiltinIteratorInput
	BuiltinVMPushWriter
	BuiltinVMPopWriter
	BuiltinOBStart
	BuiltinOBEnd
	BuiltinFlush
	BuiltinUserData
	BuiltinNamedParamTypeCheck

	BuiltinIs
	BuiltinIsError
	BuiltinIsInt
	BuiltinIsUint
	BuiltinIsFloat
	BuiltinIsChar
	BuiltinIsBool
	BuiltinIsStr
	BuiltinIsRawStr
	BuiltinIsBytes
	BuiltinIsDict
	BuiltinIsSyncDict
	BuiltinIsArray
	BuiltinIsNil
	BuiltinIsFunction
	BuiltinIsCallable
	BuiltinIsIterable
	BuiltinIsIterator

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

	BuiltinBinOperatorsBegin_
	BuiltinBinOpAdd
	BuiltinBinOpSub
	BuiltinBinOpMul
	BuiltinBinOpQuo
	BuiltinBinOpRem
	BuiltinBinOpAnd
	BuiltinBinOpOr
	BuiltinBinOpXor
	BuiltinBinOpShl
	BuiltinBinOpShr
	BuiltinBinOpAndNot
	BuiltinBinOpLAnd
	BuiltinBinOpEqual
	BuiltinBinOpNotEqual
	BuiltinBinOpLess
	BuiltinBinOpGreater
	BuiltinBinOpLessEq
	BuiltinBinOpGreaterEq
	BuiltinBinOpTilde
	BuiltinBinOpDoubleTilde
	BuiltinBinOpTripleTilde
	BuiltinBinOperatorsEnd_
)

var (
	lastBuiltinType = BuiltinBinOperatorsEnd_
	lastBuiltinMux  = sync.Mutex{}
)

func NewBuiltinType() (t BuiltinType) {
	lastBuiltinMux.Lock()
	defer lastBuiltinMux.Unlock()
	lastBuiltinType++
	t = lastBuiltinType
	return t
}

// BuiltinsMap is list of builtin types, exported for REPL.
var BuiltinsMap = map[string]BuiltinType{
	"binaryOp":            BuiltinBinaryOp,
	"cast":                BuiltinCast,
	"append":              BuiltinAppend,
	"delete":              BuiltinDelete,
	"copy":                BuiltinCopy,
	"dcopy":               BuiltinDeepCopy,
	"repeat":              BuiltinRepeat,
	"contains":            BuiltinContains,
	"len":                 BuiltinLen,
	"sort":                BuiltinSort,
	"sortReverse":         BuiltinSortReverse,
	"filter":              BuiltinFilter,
	"map":                 BuiltinMap,
	"each":                BuiltinEach,
	"reduce":              BuiltinReduce,
	"typeName":            BuiltinTypeName,
	"chars":               BuiltinChars,
	"close":               BuiltinClose,
	"read":                BuiltinRead,
	"write":               BuiltinWrite,
	"print":               BuiltinPrint,
	"printf":              BuiltinPrintf,
	"println":             BuiltinPrintln,
	"sprintf":             BuiltinSprintf,
	"globals":             BuiltinGlobals,
	"stdio":               BuiltinStdIO,
	"wrap":                BuiltinWrap,
	"struct":              BuiltinStruct,
	"new":                 BuiltinNew,
	"typeof":              BuiltinTypeOf,
	"addCallMethod":       BuiltinAddCallMethod,
	"rawCaller":           BuiltinRawCaller,
	"repr":                BuiltinRepr,
	"userData":            BuiltinUserData,
	"namedParamTypeCheck": BuiltinNamedParamTypeCheck,

	"is":         BuiltinIs,
	"isError":    BuiltinIsError,
	"isInt":      BuiltinIsInt,
	"isUint":     BuiltinIsUint,
	"isFloat":    BuiltinIsFloat,
	"isChar":     BuiltinIsChar,
	"isBool":     BuiltinIsBool,
	"isStr":      BuiltinIsStr,
	"isRawStr":   BuiltinIsRawStr,
	"isBytes":    BuiltinIsBytes,
	"isDict":     BuiltinIsDict,
	"isSyncDict": BuiltinIsSyncDict,
	"isArray":    BuiltinIsArray,
	"isNil":      BuiltinIsNil,
	"isFunction": BuiltinIsFunction,
	"isCallable": BuiltinIsCallable,
	"isIterable": BuiltinIsIterable,
	"isIterator": BuiltinIsIterator,

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

	"iterate":       BuiltinIterate,
	"keys":          BuiltinKeys,
	"values":        BuiltinValues,
	"items":         BuiltinItems,
	"collect":       BuiltinCollect,
	"enumerate":     BuiltinEnumerate,
	"iterator":      BuiltinIterator,
	"iteratorInput": BuiltinIteratorInput,
	"zip":           BuiltinZipIterator,
	"keyValue":      BuiltinKeyValue,
	"keyValueArray": BuiltinKeyValueArray,

	"vmPushWriter":   BuiltinVMPushWriter,
	"vmPopWriter":    BuiltinVMPopWriter,
	"obstart":        BuiltinOBStart,
	"obend":          BuiltinOBEnd,
	"flush":          BuiltinFlush,
	"DISCARD_WRITER": BuiltinDiscardWriter,
}

type Builtins struct {
	Objects BuiltinObjectsMap
	Map     map[string]BuiltinType
	last    BuiltinType
}

func NewBuiltins() *Builtins {
	return &Builtins{Objects: BuiltinObjects, Map: BuiltinsMap, last: NewBuiltinType()}
}

func (s *Builtins) SetType(typ ObjectType) *Builtins {
	return s.Set(typ.Name(), typ)
}

func (s *Builtins) Set(name string, obj Object) *Builtins {
	if s.last == lastBuiltinType {
		newObjects := make(BuiltinObjectsMap, len(s.Objects))
		newMap := make(map[string]BuiltinType, len(s.Objects))
		for t, o := range s.Objects {
			newObjects[t] = o
		}
		for name, t := range s.Map {
			newMap[name] = t
		}
		s.Objects = newObjects
		s.Map = newMap
	}
	s.last++
	s.Map[name] = s.last
	s.Objects[s.last] = obj
	return s
}

func (s *Builtins) Call(t BuiltinType, c Call) (Object, error) {
	return DoCall(s.Objects[t].(CallerObject), c)
}

func (s *Builtins) Caller(t BuiltinType) CallerObject {
	return s.Objects[t].(CallerObject)
}

func (s *Builtins) Invoker(t BuiltinType, c Call) func() (Object, error) {
	caller := s.Objects[t].(CallerObject)
	return func() (Object, error) {
		return caller.Call(c)
	}
}

func (s *Builtins) ArgsInvoker(t BuiltinType, c Call) func(arg ...Object) (Object, error) {
	caller := s.Objects[t].(CallerObject)
	c.Args = Args{nil}
	return func(arg ...Object) (Object, error) {
		c.Args[0] = arg
		return Val(caller.Call(c))
	}
}

func (s *Builtins) Get(t BuiltinType) Object {
	return s.Objects[t]
}

func (s *Builtins) AppendMap(m map[string]Object) {
	for name, o := range m {
		s.Set(name, o)
	}
}

type BuiltinObjectsMap map[BuiltinType]Object

func (m BuiltinObjectsMap) Build() BuiltinObjectsMap {
	cp := make(BuiltinObjectsMap, len(m))
	for key, value := range m {
		if Callable(value) {
			if cma, _ := value.(CanCallerObjectMethodsEnabler); cma == nil || !cma.MethodsDisabled() {
				if cwm, _ := value.(*CallerObjectWithMethods); cwm == nil {
					value = NewCallerObjectWithMethods(value.(CallerObject))
				}
			}
		}
		cp[key] = value
	}
	return cp
}

func (m BuiltinObjectsMap) Append(obj ...Object) BuiltinObjectsMap {
	var (
		cp  = make(BuiltinObjectsMap, len(m))
		max BuiltinType
	)

	for t, obj := range m {
		cp[t] = obj
		if t > max {
			max = t
		}
	}
	for i, object := range obj {
		cp[max+BuiltinType(i)] = object
	}
	return cp
}

// BuiltinObjects is list of builtins, exported for REPL.
var BuiltinObjects = BuiltinObjectsMap{
	// :makeArray is a private builtin function to help destructuring array assignments
	BuiltinMakeArray: &BuiltinFunction{
		Name:                  ":makeArray",
		Value:                 funcPiOROe(BuiltinMakeArrayFunc),
		AcceptMethodsDisabled: true,
	},
	BuiltinBinaryOp: &BuiltinFunction{
		Name:  "binaryOp",
		Value: BuiltinBinaryOpFunc,
	},
	BuiltinCast: &BuiltinFunction{
		Name:  "cast",
		Value: BuiltinCastFunc,
	},
	BuiltinChars: &BuiltinFunction{
		Name:  "chars",
		Value: funcPOROe(BuiltinCharsFunc),
	},
	BuiltinAppend: &BuiltinFunction{
		Name:  "append",
		Value: BuiltinAppendFunc,
	},
	BuiltinDelete: &BuiltinFunction{
		Name:  "delete",
		Value: BuiltinDeleteFunc,
	},
	BuiltinCopy: &BuiltinFunction{
		Name:  "copy",
		Value: BuiltinCopyFunc,
	},
	BuiltinDeepCopy: &BuiltinFunction{
		Name:  "dcopy",
		Value: BuiltinDeepCopyFunc,
	},
	BuiltinRepeat: &BuiltinFunction{
		Name:  "repeat",
		Value: funcPOiROe(BuiltinRepeatFunc),
	},
	BuiltinContains: &BuiltinFunction{
		Name:  "contains",
		Value: funcPOOROe(BuiltinContainsFunc),
	},
	BuiltinLen: &BuiltinFunction{
		Name:  "len",
		Value: funcPORO(BuiltinLenFunc),
	},
	BuiltinCap: &BuiltinFunction{
		Name:  "cap",
		Value: funcPORO(BuiltinCapFunc),
	},
	BuiltinSort: &BuiltinFunction{
		Name:  "sort",
		Value: funcPpVM_OCo_less_ROe(BuiltinSortFunc),
	},
	BuiltinSortReverse: &BuiltinFunction{
		Name:  "sortReverse",
		Value: funcPpVM_OCo_less_ROe(BuiltinSortReverseFunc),
	},
	BuiltinTypeName: &BuiltinFunction{
		Name:                  "typeName",
		Value:                 funcPORO(BuiltinTypeNameFunc),
		AcceptMethodsDisabled: true,
	},
	BuiltinPrint: &BuiltinFunction{
		Name:  "print",
		Value: BuiltinPrintFunc,
	},
	BuiltinPrintf: &BuiltinFunction{
		Name:  "printf",
		Value: BuiltinPrintfFunc,
	},
	BuiltinPrintln: &BuiltinFunction{
		Name:  "println",
		Value: BuiltinPrintlnFunc,
	},
	BuiltinSprintf: &BuiltinFunction{
		Name:                  "sprintf",
		Value:                 BuiltinSprintfFunc,
		AcceptMethodsDisabled: true,
	},
	BuiltinGlobals: &BuiltinFunction{
		Name:                  "globals",
		Value:                 BuiltinGlobalsFunc,
		AcceptMethodsDisabled: true,
	},
	BuiltinRepr: &BuiltinFunction{
		Name:  "repr",
		Value: BuiltinReprFunc,
	},
	BuiltinNamedParamTypeCheck: &BuiltinFunction{
		Name:                  "namedParamTypeCheck",
		Value:                 BuiltinNamedParamTypeCheckFunc,
		AcceptMethodsDisabled: true,
	},
	BuiltinIs: &BuiltinFunction{
		Name:                  "is",
		Value:                 BuiltinIsFunc,
		AcceptMethodsDisabled: true,
	},
	BuiltinIsError: &BuiltinFunction{
		Name:                  "isError",
		Value:                 BuiltinIsErrorFunc,
		AcceptMethodsDisabled: true,
	},
	BuiltinIsInt: &BuiltinFunction{
		Name:                  "isInt",
		Value:                 funcPORO(BuiltinIsIntFunc),
		AcceptMethodsDisabled: true,
	},
	BuiltinIsUint: &BuiltinFunction{
		Name:                  "isUint",
		Value:                 funcPORO(BuiltinIsUintFunc),
		AcceptMethodsDisabled: true,
	},
	BuiltinIsFloat: &BuiltinFunction{
		Name:                  "isFloat",
		Value:                 funcPORO(BuiltinIsFloatFunc),
		AcceptMethodsDisabled: true,
	},
	BuiltinIsChar: &BuiltinFunction{
		Name:                  "isChar",
		Value:                 funcPORO(BuiltinIsCharFunc),
		AcceptMethodsDisabled: true,
	},
	BuiltinIsBool: &BuiltinFunction{
		Name:                  "isBool",
		Value:                 funcPORO(BuiltinIsBoolFunc),
		AcceptMethodsDisabled: true,
	},
	BuiltinIsStr: &BuiltinFunction{
		Name:                  "isStr",
		Value:                 funcPORO(BuiltinIsStrFunc),
		AcceptMethodsDisabled: true,
	},
	BuiltinIsRawStr: &BuiltinFunction{
		Name:                  "isRawStr",
		Value:                 funcPORO(BuiltinIsRawStrFunc),
		AcceptMethodsDisabled: true,
	},
	BuiltinIsBytes: &BuiltinFunction{
		Name:                  "isBytes",
		Value:                 funcPORO(BuiltinIsBytesFunc),
		AcceptMethodsDisabled: true,
	},
	BuiltinIsDict: &BuiltinFunction{
		Name:                  "isDict",
		Value:                 funcPORO(BuiltinIsDictFunc),
		AcceptMethodsDisabled: true,
	},
	BuiltinIsSyncDict: &BuiltinFunction{
		Name:                  "isSyncDict",
		Value:                 funcPORO(BuiltinIsSyncDictFunc),
		AcceptMethodsDisabled: true,
	},
	BuiltinIsArray: &BuiltinFunction{
		Name:                  "isArray",
		Value:                 funcPORO(BuiltinIsArrayFunc),
		AcceptMethodsDisabled: true,
	},
	BuiltinIsNil: &BuiltinFunction{
		Name:                  "isNil",
		Value:                 funcPORO(BuiltinIsNilFunc),
		AcceptMethodsDisabled: true,
	},
	BuiltinIsFunction: &BuiltinFunction{
		Name:                  "isFunction",
		Value:                 funcPORO(BuiltinIsFunctionFunc),
		AcceptMethodsDisabled: true,
	},
	BuiltinIsCallable: &BuiltinFunction{
		Name:  "isCallable",
		Value: funcPORO(BuiltinIsCallableFunc),
	},
	BuiltinIsIterable: &BuiltinFunction{
		Name:  "isIterable",
		Value: funcPpVM_ORO(BuiltinIsIterableFunc),
	},
	BuiltinIsIterator: &BuiltinFunction{
		Name:  "isIterator",
		Value: funcPORO(BuiltinIsIteratorFunc),
	},
	BuiltinStdIO: &BuiltinFunction{
		Name:  "stdio",
		Value: BuiltinStdIOFunc,
	},
	BuiltinWrap: &BuiltinFunction{
		Name:  "wrap",
		Value: BuiltinWrapFunc,
	},
	BuiltinStruct: &BuiltinFunction{
		Name:                  "struct",
		Value:                 BuiltinStructFunc,
		AcceptMethodsDisabled: true,
	},
	BuiltinNew: &BuiltinFunction{
		Name:  "new",
		Value: BuiltinNewFunc,
	},
	BuiltinTypeOf: &BuiltinFunction{
		Name:                  "typeof",
		Value:                 BuiltinTypeOfFunc,
		AcceptMethodsDisabled: true,
	},
	BuiltinAddCallMethod: &BuiltinFunction{
		Name:                  "addCallMethod",
		Value:                 funcPpVM_CoCob_override_Re(BuiltinAddCallMethodFunc),
		AcceptMethodsDisabled: true,
	},
	BuiltinRawCaller: &BuiltinFunction{
		Name:                  "rawCaller",
		Value:                 BuiltinRawCallerFunc,
		AcceptMethodsDisabled: true,
	},
	BuiltinVMPushWriter: &BuiltinFunction{
		Name:                  "vmPushWriter",
		Value:                 BuiltinPushWriterFunc,
		AcceptMethodsDisabled: true,
	},
	BuiltinVMPopWriter: &BuiltinFunction{
		Name:  "vmPopWriter",
		Value: BuiltinPopWriterFunc,
	},
	BuiltinOBStart: &BuiltinFunction{
		Name:                  "obstart",
		Value:                 BuiltinOBStartFunc,
		AcceptMethodsDisabled: true,
	},
	BuiltinOBEnd: &BuiltinFunction{
		Name:                  "obend",
		Value:                 BuiltinOBEndFunc,
		AcceptMethodsDisabled: true,
	},
	BuiltinFlush: &BuiltinFunction{
		Name:  "flush",
		Value: BuiltinFlushFunc,
	},
	BuiltinUserData: &BuiltinFunction{
		Name:  "userData",
		Value: BuiltinUserDataFunc,
	},
	BuiltinClose: &BuiltinFunction{
		Name:  "close",
		Value: BuiltinCloseFunc,
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

func init() {
	BuiltinObjects[BuiltinRead] = &BuiltinFunction{
		Name:  "read",
		Value: BuiltinReadFunc,
	}
	BuiltinObjects[BuiltinWrite] = &BuiltinFunction{
		Name:  "write",
		Value: BuiltinWriteFunc,
	}
	BuiltinObjects[BuiltinFilter] = &BuiltinFunction{
		Name:  "filter",
		Value: BuiltinFilterFunc,
	}
	BuiltinObjects[BuiltinMap] = &BuiltinFunction{
		Name:  "map",
		Value: BuiltinMapFunc,
	}
	BuiltinObjects[BuiltinEach] = &BuiltinFunction{
		Name:  "each",
		Value: BuiltinEachFunc,
	}
	BuiltinObjects[BuiltinReduce] = &BuiltinFunction{
		Name:  "reduce",
		Value: BuiltinReduceFunc,
	}
	BuiltinObjects[BuiltinEach] = &BuiltinFunction{
		Name:  "each",
		Value: BuiltinEachFunc,
	}

	BuiltinObjects[BuiltinIterate] = &BuiltinFunction{
		Name:  "iterate",
		Value: BuiltinIterateFunc,
	}
	BuiltinObjects[BuiltinKeys] = &BuiltinFunction{
		Name:  "keys",
		Value: BuiltinKeysFunc,
	}
	BuiltinObjects[BuiltinValues] = &BuiltinFunction{
		Name:  "values",
		Value: BuiltinValuesFunc,
	}
	BuiltinObjects[BuiltinItems] = &BuiltinFunction{
		Name:  "items",
		Value: BuiltinItemsFunc,
	}
	BuiltinObjects[BuiltinCollect] = &BuiltinFunction{
		Name:  "collect",
		Value: BuiltinCollectFunc,
	}
	BuiltinObjects[BuiltinEnumerate] = &BuiltinFunction{
		Name:  "enumerate",
		Value: BuiltinEnumerateFunc,
	}
	BuiltinObjects[BuiltinIterator] = TIterator
	BuiltinObjects[BuiltinZipIterator] = TZipIterator
	BuiltinObjects[BuiltinIteratorInput] = &BuiltinFunction{
		Name:  "iteratorInput",
		Value: funcPORO(BuiltinIteratorInputFunc),
	}
}

// functions to generate with mkcallable

// builtin delete
//
//gad:callable func(o Object, k string) (err error)

// builtin copy, dcopy, len, error, typeName, bool, string, isInt, isUint
// isFloat, isChar, isBool, isString, isBytes, isMap, isSyncMap, isArray
// isNil, isFunction, isCallable, isIterable
//
//gad:callable func(o Object) (ret Object)

// builtin repeat
//
//gad:callable func(o Object, n int) (ret Object, err error)

// builtin array
//
//gad:callable func(n int, o Object) (ret Object, err error)

// builtin contains
//
//gad:callable func(o Object, v Object) (ret Object, err error)

// builtin sort, sortReverse, int, uint, float, char, chars
//
//gad:callable func(o Object) (ret Object, err error)

// builtin int
//
//gad:callable func(v int64) (ret Object)

// builtin uint
//
//gad:callable func(v uint64) (ret Object)

// builtin float
//
//gad:callable func(v float64) (ret Object)

// builtin addMethod
//
//gad:callable func(o CallerObject, argsType Array, handler CallerObject, override=bool) (err error)

// builtin addMethod
//
//gad:callable func(vm *VM, o CallerObject, handler CallerObject, override=bool) (err error)

// builtin sort, sortReverse
//
//gad:callable func(vm *VM, v Object, less=CallerObject) (ret Object, err error)

// builtin decimal
//
//gad:callable func(vm *VM, v Object) (ret Object, err error)

// builtin isIterable
//
//gad:callable func(vm *VM, v Object) (ret Object)
