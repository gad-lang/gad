// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

//go:generate go run ./cmd/mkcallable -output builtins_zfuncs.go builtins.go

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
	BuiltinText
	BuiltinString
	BuiltinBytes
	BuiltinArray
	BuiltinDict
	BuiltinSyncDic
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
	BuiltinFilter
	BuiltinMap
	BuiltinReduce
	BuiltinForEach
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
	BuiltinAddCallMethod
	BuiltinRawCaller
	BuiltinMakeArray
	BuiltinCap
	BuiltinKeys
	BuiltinValues
	BuiltinItems
	BuiltinVMPushWriter
	BuiltinVMPopWriter
	BuiltinOBStart
	BuiltinOBEnd
	BuiltinFlush

	BuiltinIs
	BuiltinIsError
	BuiltinIsInt
	BuiltinIsUint
	BuiltinIsFloat
	BuiltinIsChar
	BuiltinIsBool
	BuiltinIsString
	BuiltinIsBytes
	BuiltinIsDict
	BuiltinIsSyncDict
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
	"cast":          BuiltinCast,
	"append":        BuiltinAppend,
	"delete":        BuiltinDelete,
	"copy":          BuiltinCopy,
	"dcopy":         BuiltinDeepCopy,
	"repeat":        BuiltinRepeat,
	"contains":      BuiltinContains,
	"len":           BuiltinLen,
	"sort":          BuiltinSort,
	"sortReverse":   BuiltinSortReverse,
	"filter":        BuiltinFilter,
	"map":           BuiltinMap,
	"reduce":        BuiltinReduce,
	"foreach":       BuiltinForEach,
	"typeName":      BuiltinTypeName,
	"chars":         BuiltinChars,
	"write":         BuiltinWrite,
	"print":         BuiltinPrint,
	"printf":        BuiltinPrintf,
	"println":       BuiltinPrintln,
	"sprintf":       BuiltinSprintf,
	"globals":       BuiltinGlobals,
	"stdio":         BuiltinStdIO,
	"wrap":          BuiltinWrap,
	"newType":       BuiltinNewType,
	"typeof":        BuiltinTypeOf,
	"addCallMethod": BuiltinAddCallMethod,
	"rawCaller":     BuiltinRawCaller,

	"is":         BuiltinIs,
	"isError":    BuiltinIsError,
	"isInt":      BuiltinIsInt,
	"isUint":     BuiltinIsUint,
	"isFloat":    BuiltinIsFloat,
	"isChar":     BuiltinIsChar,
	"isBool":     BuiltinIsBool,
	"isString":   BuiltinIsString,
	"isBytes":    BuiltinIsBytes,
	"isDict":     BuiltinIsDict,
	"isSyncDict": BuiltinIsSyncDict,
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
	"obstart":        BuiltinOBStart,
	"obend":          BuiltinOBEnd,
	"flush":          BuiltinFlush,
	"DISCARD_WRITER": BuiltinDiscardWriter,
}

type BuiltinObjectsMap map[BuiltinType]Object

func (m BuiltinObjectsMap) Build() BuiltinObjectsMap {
	cp := make(BuiltinObjectsMap, len(m))
	for key, value := range m {
		if co, _ := value.(CallerObject); co != nil {
			if cma, _ := co.(CanCallerObjectMethodsEnabler); cma == nil || !cma.MethodsDisabled() {
				if cwm, _ := value.(*CallerObjectWithMethods); cwm == nil {
					value = NewCallerObjectWithMethods(co)
				}
			}
		}
		cp[key] = value
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
		Value: funcPORO(BuiltinCopyFunc),
	},
	BuiltinDeepCopy: &BuiltinFunction{
		Name:  "dcopy",
		Value: funcPORO(BuiltinDeepCopyFunc),
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
		Value: funcPOROe(BuiltinSortFunc),
	},
	BuiltinSortReverse: &BuiltinFunction{
		Name:  "sortReverse",
		Value: funcPOROe(BuiltinSortReverseFunc),
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
	BuiltinIsString: &BuiltinFunction{
		Name:                  "isString",
		Value:                 funcPORO(BuiltinIsStringFunc),
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
		Value: funcPORO(BuiltinIsIterableFunc),
	},
	BuiltinKeys: &BuiltinFunction{
		Name:  "keys",
		Value: BuiltinKeysFunc,
	},
	BuiltinValues: &BuiltinFunction{
		Name:  "values",
		Value: BuiltinValuesFunc,
	},
	BuiltinItems: &BuiltinFunction{
		Name:  "items",
		Value: BuiltinItemsFunc,
	},
	BuiltinStdIO: &BuiltinFunction{
		Name:  "stdio",
		Value: BuiltinStdIOFunc,
	},
	BuiltinWrap: &BuiltinFunction{
		Name:  "wrap",
		Value: BuiltinWrapFunc,
	},
	BuiltinNewType: &BuiltinFunction{
		Name:  "newType",
		Value: BuiltinNewTypeFunc,
	},
	BuiltinTypeOf: &BuiltinFunction{
		Name:  "typeof",
		Value: BuiltinTypeOfFunc,
	},
	BuiltinAddCallMethod: &BuiltinFunction{
		Name:                  "addCallMethod",
		Value:                 funcPpVM_CoCobRe(BuiltinAddCallMethodFunc),
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
	BuiltinObjects[BuiltinReduce] = &BuiltinFunction{
		Name:  "reduce",
		Value: BuiltinReduceFunc,
	}
	BuiltinObjects[BuiltinForEach] = &BuiltinFunction{
		Name:  "foreach",
		Value: BuiltinForEachFunc,
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
