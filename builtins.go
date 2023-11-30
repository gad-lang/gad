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
	"filter":      BuiltinFilter,
	"map":         BuiltinMap,
	"reduce":      BuiltinReduce,
	"foreach":     BuiltinForEach,
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

// BuiltinObjects is list of builtins, exported for REPL.
var BuiltinObjects = map[BuiltinType]Object{
	// :makeArray is a private builtin function to help destructuring array assignments
	BuiltinMakeArray: &BuiltinFunction{
		Name:  ":makeArray",
		Value: funcPiOROe(BuiltinMakeArrayFunc),
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
		Name:  "typeName",
		Value: funcPORO(BuiltinTypeNameFunc),
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
		Name:  "sprintf",
		Value: BuiltinSprintfFunc,
	},
	BuiltinGlobals: &BuiltinFunction{
		Name:  "globals",
		Value: BuiltinGlobalsFunc,
	},
	BuiltinIs: &BuiltinFunction{
		Name:  "is",
		Value: BuiltinIsFunc,
	},
	BuiltinIsError: &BuiltinFunction{
		Name:  "isError",
		Value: BuiltinIsErrorFunc,
	},
	BuiltinIsInt: &BuiltinFunction{
		Name:  "isInt",
		Value: funcPORO(BuiltinIsIntFunc),
	},
	BuiltinIsUint: &BuiltinFunction{
		Name:  "isUint",
		Value: funcPORO(BuiltinIsUintFunc),
	},
	BuiltinIsFloat: &BuiltinFunction{
		Name:  "isFloat",
		Value: funcPORO(BuiltinIsFloatFunc),
	},
	BuiltinIsChar: &BuiltinFunction{
		Name:  "isChar",
		Value: funcPORO(BuiltinIsCharFunc),
	},
	BuiltinIsBool: &BuiltinFunction{
		Name:  "isBool",
		Value: funcPORO(BuiltinIsBoolFunc),
	},
	BuiltinIsString: &BuiltinFunction{
		Name:  "isString",
		Value: funcPORO(BuiltinIsStringFunc),
	},
	BuiltinIsBytes: &BuiltinFunction{
		Name:  "isBytes",
		Value: funcPORO(BuiltinIsBytesFunc),
	},
	BuiltinIsDict: &BuiltinFunction{
		Name:  "isDict",
		Value: funcPORO(BuiltinIsDictFunc),
	},
	BuiltinIsSyncDict: &BuiltinFunction{
		Name:  "isSyncDict",
		Value: funcPORO(BuiltinIsSyncDictFunc),
	},
	BuiltinIsArray: &BuiltinFunction{
		Name:  "isArray",
		Value: funcPORO(BuiltinIsArrayFunc),
	},
	BuiltinIsNil: &BuiltinFunction{
		Name:  "isNil",
		Value: funcPORO(BuiltinIsNilFunc),
	},
	BuiltinIsFunction: &BuiltinFunction{
		Name:  "isFunction",
		Value: funcPORO(BuiltinIsFunctionFunc),
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
	BuiltinVMPushWriter: &BuiltinFunction{
		Name:  "vmPushWriter",
		Value: BuiltinPushWriterFunc,
	},
	BuiltinVMPopWriter: &BuiltinFunction{
		Name:  "vmPopWriter",
		Value: BuiltinPopWriterFunc,
	},
	BuiltinOBStart: &BuiltinFunction{
		Name:  "obstart",
		Value: BuiltinOBStartFunc,
	},
	BuiltinOBEnd: &BuiltinFunction{
		Name:  "obend",
		Value: BuiltinOBEndFunc,
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
