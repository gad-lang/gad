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
		return bt.FuncName
	case *BuiltinObjType:
		return bt.Name()
	case *BuiltinFunctionWithMethods:
		return bt.name
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
	BuiltinAny
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
	BuiltinMixedParams
	BuiltinPrinterState
	BuiltinTypedIdent
	BuiltinFunc
	BuiltinComputedValue
	BuiltinTypesEnd_

	BuiltinStaticTypesStart_
	BuiltinStaticTypeBuiltinFunction
	BuiltinStaticTypeCallWrapper
	BuiltinStaticTypeCompiledFunction
	BuiltinStaticTypeFunction
	BuiltinStaticTypeKeyValueArrays
	BuiltinStaticTypeArgs
	BuiltinStaticTypeNamedArgs
	BuiltinStaticTypeObjectPtr
	BuiltinStaticTypeReader
	BuiltinStaticTypeWriter
	BuiltinStaticTypeDiscardWriter
	BuiltinStaticTypeObjectTypeArray
	BuiltinStaticTypeReflectMethod
	BuiltinStaticTypeIndexGetProxy
	BuiltinStaticTypesEnd_

	BuiltinFunctionsBegin_
	BuiltinBinaryOperator
	BuiltinSelfAssignOperator
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
	BuiltinNewClass
	BuiltinTypeOf
	BuiltinAddMethod
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
	BuiltinToArray

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

	GroupBuiltinBinaryOperatorsBegin
	BuiltinBinaryOperatorAdd
	BuiltinBinaryOperatorSub
	BuiltinBinaryOperatorMul
	BuiltinBinaryOperatorPow
	BuiltinBinaryOperatorQuo
	BuiltinBinaryOperatorRem
	BuiltinBinaryOperatorAnd
	BuiltinBinaryOperatorOr
	BuiltinBinaryOperatorXor
	BuiltinBinaryOperatorShl
	BuiltinBinaryOperatorShr
	BuiltinBinaryOperatorAndNot
	BuiltinBinaryOperatorLAnd
	BuiltinBinaryOperatorEqual
	BuiltinBinaryOperatorNotEqual
	BuiltinBinaryOperatorLess
	BuiltinBinaryOperatorGreater
	BuiltinBinaryOperatorLessEq
	BuiltinBinaryOperatorGreaterEq
	BuiltinBinaryOperatorTilde
	BuiltinBinaryOperatorDoubleTilde
	BuiltinBinaryOperatorTripleTilde
	BuiltinBinaryOperatorLambda
	GroupBuiltinBinaryOperatorsEnd

	GroupBuiltinSelfAssignOperatorsBegin
	BuiltinSelfAssignOperatorAdd
	BuiltinSelfAssignOperatorInc
	BuiltinSelfAssignOperatorSub
	BuiltinSelfAssignOperatorDec
	BuiltinSelfAssignOperatorMul
	BuiltinSelfAssignOperatorPow
	BuiltinSelfAssignOperatorQuo
	BuiltinSelfAssignOperatorRem
	BuiltinSelfAssignOperatorAnd
	BuiltinSelfAssignOperatorOr
	BuiltinSelfAssignOperatorXor
	BuiltinSelfAssignOperatorShl
	BuiltinSelfAssignOperatorShr
	BuiltinSelfAssignOperatorAndNot
	BuiltinSelfAssignOperatorLOr
	GroupBuiltinSelfAssignOperatorsEnd

	BuiltinEnd_
)

var (
	lastBuiltinType = GroupBuiltinSelfAssignOperatorsEnd
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
	"@binaryOperator":     BuiltinBinaryOperator,
	"@selfAssignOperator": BuiltinSelfAssignOperator,
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
	"Class":               BuiltinNewClass,
	"typeof":              BuiltinTypeOf,
	"addMethod":           BuiltinAddMethod,
	"rawCaller":           BuiltinRawCaller,
	"repr":                BuiltinRepr,
	"userData":            BuiltinUserData,
	"namedParamTypeCheck": BuiltinNamedParamTypeCheck,
	"toArray":             BuiltinToArray,

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

type StaticBuiltins struct {
	builtins *Builtins
}

func (b *StaticBuiltins) Builtins() *Builtins {
	return b.builtins
}

func (b *StaticBuiltins) Set(name string, obj Object) BuiltinType {
	b.builtins.last++
	b.builtins.Map[name] = b.builtins.last
	b.builtins.Objects[b.builtins.last] = obj
	return b.builtins.last
}

func (b *StaticBuiltins) Update(key BuiltinType, value Object) {
	b.builtins.Objects[key] = value
}

func (b *StaticBuiltins) Call(t BuiltinType, c Call) (Object, error) {
	return b.builtins.Call(t, c)
}

func (b *StaticBuiltins) Caller(t BuiltinType) CallerObject {
	return b.builtins.Caller(t)
}

func (b *StaticBuiltins) Invoker(t BuiltinType, c Call) func() (Object, error) {
	return b.builtins.Invoker(t, c)
}

func (b *StaticBuiltins) ArgsInvoker(t BuiltinType, c Call) func(arg ...Object) (Object, error) {
	return b.builtins.ArgsInvoker(t, c)
}

func (b *StaticBuiltins) Get(t BuiltinType) Object {
	return b.builtins.Get(t)
}

type Builtins struct {
	Objects BuiltinObjectsMap
	Map     map[string]BuiltinType
	last    BuiltinType
}

func NewBuiltins() *Builtins {
	return &Builtins{Objects: BuiltinObjects, Map: BuiltinsMap, last: NewBuiltinType()}
}

func (b *Builtins) SetType(typ ObjectType) *Builtins {
	return b.Set(typ.Name(), typ)
}

func (b *Builtins) Set(name string, obj Object) *Builtins {
	if b.last == lastBuiltinType {
		newObjects := make(BuiltinObjectsMap, len(b.Objects))
		newMap := make(map[string]BuiltinType, len(b.Objects))
		for t, o := range b.Objects {
			newObjects[t] = o
		}
		for name, t := range b.Map {
			newMap[name] = t
		}
		b.Objects = newObjects
		b.Map = newMap
	}
	b.last++
	b.Map[name] = b.last
	b.Objects[b.last] = obj
	return b
}

func (b *Builtins) Call(t BuiltinType, c Call) (Object, error) {
	return DoCall(b.Objects[t].(CallerObject), c)
}

func (b *Builtins) Caller(t BuiltinType) CallerObject {
	return b.Objects[t].(CallerObject)
}

func (b *Builtins) Invoker(t BuiltinType, c Call) func() (Object, error) {
	caller := b.Objects[t].(CallerObject)
	return func() (Object, error) {
		return caller.Call(c)
	}
}

func (b *Builtins) ArgsInvoker(t BuiltinType, c Call) func(arg ...Object) (Object, error) {
	caller := b.Objects[t].(CallerObject)
	c.Args = Args{nil}
	return func(arg ...Object) (Object, error) {
		c.Args[0] = arg
		return DoCall(caller, c)
	}
}

func (b *Builtins) Get(t BuiltinType) Object {
	return b.Objects[t]
}

func (b *Builtins) AppendMap(m map[string]Object) {
	for name, o := range m {
		b.Set(name, o)
	}
}

func (b *Builtins) Build() (s *StaticBuiltins) {
	s = &StaticBuiltins{
		builtins: &Builtins{
			Objects: b.Objects.build(),
		},
	}
	s.builtins.Map = make(map[string]BuiltinType, len(b.Map))
	for k, v := range b.Map {
		s.builtins.Map[k] = v
	}
	s.builtins.last = b.last
	return s
}

type BuiltinObjectsMap map[BuiltinType]Object

func (m BuiltinObjectsMap) AddMethod(typ BuiltinType, method ...CallerObjectWithParamTypes) {
	m[typ] = AddMethod(m[typ], method...)
}

func (m BuiltinObjectsMap) build() BuiltinObjectsMap {
	cp := make(BuiltinObjectsMap, len(m))
	for key, value := range m {
		if Callable(value) {
			switch t := value.(type) {
			case *BuiltinFunctionWithMethods:
				if key < BuiltinEnd_ {
					value = t.Copy()
				}
			case *BuiltinObjType:
				t.builtinType = key
				if key < BuiltinEnd_ {
					value = t.Copy().(*BuiltinObjType)
				}
			case *BuiltinFunction:
				if !t.AcceptMethodsDisabled {
					value = AddMethod(value)
				}
			case MethodCaller:
			case *Function:
				f := NewFunc(t.Name(), t.Module)
				if t.Header == nil {
					f.defaul = t
				} else {
					f.AddMethodByTypes(nil, t.ParamTypes(), f, false, nil)
				}
			default:
				if cma, _ := value.(CanCallerObjectMethodsEnabler); cma == nil || !cma.MethodsDisabled() {
					if cwm, _ := value.(*Func); cwm == nil {
						var module *Module
						switch t := value.(type) {
						case interface{ GetModule() *Module }:
							module = t.GetModule()
						}

						f := NewFunc(value.ToString(), module)

						_ = SplitCaller(nil, value, func(co CallerObject, types ParamsTypes) error {
							if types == nil {
								f.defaul = co
							} else {
								_ = f.AddMethodByTypes(nil, types, co, false, nil)
							}
							return nil
						}, func(co CallerObject) error {
							f.defaul = co
							return nil
						})
						value = f
					}
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
		FuncName:              ":makeArray",
		Value:                 funcPiOROe(BuiltinMakeArrayFunc),
		AcceptMethodsDisabled: true,
	},
	BuiltinBinaryOperator: &BuiltinFunction{
		FuncName: "@binaryOperator",
		Value:    BuiltinBinaryOperatorFunc,
	},
	BuiltinSelfAssignOperator: &BuiltinFunction{
		FuncName: "@selfAssignOperator",
		Value:    BuiltinSelfAssignOperatorFunc,
	},
	BuiltinCast: &BuiltinFunction{
		FuncName: "cast",
		Value:    BuiltinCastFunc,
	},
	BuiltinChars: &BuiltinFunction{
		FuncName: "chars",
		Value:    funcPOROe(BuiltinCharsFunc),
	},
	BuiltinAppend: &BuiltinFunction{
		FuncName: "append",
		Value:    BuiltinAppendFunc,
	},
	BuiltinDelete: &BuiltinFunction{
		FuncName: "delete",
		Value:    BuiltinDeleteFunc,
	},
	BuiltinCopy: &BuiltinFunction{
		FuncName: "copy",
		Value:    BuiltinCopyFunc,
	},
	BuiltinDeepCopy: &BuiltinFunction{
		FuncName: "dcopy",
		Value:    BuiltinDeepCopyFunc,
	},
	BuiltinRepeat: &BuiltinFunction{
		FuncName: "repeat",
		Value:    funcPOiROe(BuiltinRepeatFunc),
	},
	BuiltinContains: &BuiltinFunction{
		FuncName: "contains",
		Value:    funcPOOROe(BuiltinContainsFunc),
	},
	BuiltinLen: &BuiltinFunction{
		FuncName: "len",
		Value:    BuiltinLenFunc,
		Header: NewFunctionHeader().
			WithParams(func(np func(name string) *ParamBuilder) {
				np("val").Usage("The value")
			}).
			WithNamedParams(func(np func(name string) *NamedParamBuilder) {
				np("check").
					Type(TFlag).
					Usage("When Yes and value not implements LengthGetter interface return error ErrNotLengther, otherwise return 0.")
			}),
	},
	BuiltinCap: &BuiltinFunction{
		FuncName: "cap",
		Value:    funcPORO(BuiltinCapFunc),
	},
	BuiltinSort: &BuiltinFunction{
		FuncName: "sort",
		Value:    funcPpVM_OCo_less_ROe(BuiltinSortFunc),
	},
	BuiltinSortReverse: &BuiltinFunction{
		FuncName: "sortReverse",
		Value:    funcPpVM_OCo_less_ROe(BuiltinSortReverseFunc),
	},
	BuiltinTypeName: &BuiltinFunction{
		FuncName:              "typeName",
		Value:                 funcPORO(BuiltinTypeNameFunc),
		AcceptMethodsDisabled: true,
	},
	BuiltinPrint: &BuiltinFunction{
		FuncName: "print",
		Value:    BuiltinPrintFunc,
	},
	BuiltinPrintf: &BuiltinFunction{
		FuncName: "printf",
		Value:    BuiltinPrintfFunc,
	},
	BuiltinPrintln: &BuiltinFunction{
		FuncName: "println",
		Value:    BuiltinPrintlnFunc,
	},
	BuiltinSprintf: &BuiltinFunction{
		FuncName:              "sprintf",
		Value:                 BuiltinSprintfFunc,
		AcceptMethodsDisabled: true,
	},
	BuiltinGlobals: &BuiltinFunction{
		FuncName:              "globals",
		Value:                 BuiltinGlobalsFunc,
		AcceptMethodsDisabled: true,
	},
	BuiltinRepr: &BuiltinFunction{
		FuncName: "repr",
		Value:    BuiltinReprFunc,
	},
	BuiltinNamedParamTypeCheck: &BuiltinFunction{
		FuncName:              "namedParamTypeCheck",
		Value:                 BuiltinNamedParamTypeCheckFunc,
		AcceptMethodsDisabled: true,
	},
	BuiltinIs: &BuiltinFunction{
		FuncName:              "is",
		Value:                 BuiltinIsFunc,
		AcceptMethodsDisabled: true,
		Usage:                 ``,
	},
	BuiltinIsError: &BuiltinFunction{
		FuncName:              "isError",
		Value:                 BuiltinIsErrorFunc,
		AcceptMethodsDisabled: true,
	},
	BuiltinIsInt: &BuiltinFunction{
		FuncName:              "isInt",
		Value:                 funcPORO(BuiltinIsIntFunc),
		AcceptMethodsDisabled: true,
	},
	BuiltinIsUint: &BuiltinFunction{
		FuncName:              "isUint",
		Value:                 funcPORO(BuiltinIsUintFunc),
		AcceptMethodsDisabled: true,
	},
	BuiltinIsFloat: &BuiltinFunction{
		FuncName:              "isFloat",
		Value:                 funcPORO(BuiltinIsFloatFunc),
		AcceptMethodsDisabled: true,
	},
	BuiltinIsChar: &BuiltinFunction{
		FuncName:              "isChar",
		Value:                 funcPORO(BuiltinIsCharFunc),
		AcceptMethodsDisabled: true,
	},
	BuiltinIsBool: &BuiltinFunction{
		FuncName:              "isBool",
		Value:                 funcPORO(BuiltinIsBoolFunc),
		AcceptMethodsDisabled: true,
	},
	BuiltinIsStr: &BuiltinFunction{
		FuncName:              "isStr",
		Value:                 funcPORO(BuiltinIsStrFunc),
		AcceptMethodsDisabled: true,
	},
	BuiltinIsRawStr: &BuiltinFunction{
		FuncName:              "isRawStr",
		Value:                 funcPORO(BuiltinIsRawStrFunc),
		AcceptMethodsDisabled: true,
	},
	BuiltinIsBytes: &BuiltinFunction{
		FuncName:              "isBytes",
		Value:                 funcPORO(BuiltinIsBytesFunc),
		AcceptMethodsDisabled: true,
	},
	BuiltinIsDict: &BuiltinFunction{
		FuncName:              "isDict",
		Value:                 funcPORO(BuiltinIsDictFunc),
		AcceptMethodsDisabled: true,
	},
	BuiltinIsSyncDict: &BuiltinFunction{
		FuncName:              "isSyncDict",
		Value:                 funcPORO(BuiltinIsSyncDictFunc),
		AcceptMethodsDisabled: true,
	},
	BuiltinIsArray: &BuiltinFunction{
		FuncName:              "isArray",
		Value:                 funcPORO(BuiltinIsArrayFunc),
		AcceptMethodsDisabled: true,
	},
	BuiltinIsNil: &BuiltinFunction{
		FuncName:              "isNil",
		Value:                 funcPORO(BuiltinIsNilFunc),
		AcceptMethodsDisabled: true,
	},
	BuiltinIsFunction: &BuiltinFunction{
		FuncName:              "isFunction",
		Value:                 funcPORO(BuiltinIsFunctionFunc),
		AcceptMethodsDisabled: true,
	},
	BuiltinIsCallable: &BuiltinFunction{
		FuncName: "isCallable",
		Value:    funcPORO(BuiltinIsCallableFunc),
	},
	BuiltinIsIterable: &BuiltinFunction{
		FuncName: "isIterable",
		Value:    funcPpVM_ORO(BuiltinIsIterableFunc),
	},
	BuiltinIsIterator: &BuiltinFunction{
		FuncName: "isIterator",
		Value:    funcPORO(BuiltinIsIteratorFunc),
	},
	BuiltinStdIO: &BuiltinFunction{
		FuncName: "stdio",
		Value:    BuiltinStdIOFunc,
	},
	BuiltinWrap: &BuiltinFunction{
		FuncName: "wrap",
		Value:    BuiltinWrapFunc,
	},
	BuiltinNewClass: &BuiltinFunction{
		FuncName:              "Class",
		Value:                 NewClassFunc,
		AcceptMethodsDisabled: true,
	},
	BuiltinTypeOf: &BuiltinFunction{
		FuncName:              "typeof",
		Value:                 BuiltinTypeOfFunc,
		AcceptMethodsDisabled: true,
	},
	BuiltinAddMethod: &BuiltinFunction{
		FuncName:              "addMethod",
		Value:                 BuiltinAddMethodFunc,
		AcceptMethodsDisabled: true,
	},
	BuiltinRawCaller: &BuiltinFunction{
		FuncName:              "rawCaller",
		Value:                 BuiltinRawCallerFunc,
		AcceptMethodsDisabled: true,
	},
	BuiltinVMPushWriter: &BuiltinFunction{
		FuncName:              "vmPushWriter",
		Value:                 BuiltinPushWriterFunc,
		AcceptMethodsDisabled: true,
	},
	BuiltinVMPopWriter: &BuiltinFunction{
		FuncName: "vmPopWriter",
		Value:    BuiltinPopWriterFunc,
	},
	BuiltinOBStart: &BuiltinFunction{
		FuncName:              "obstart",
		Value:                 BuiltinOBStartFunc,
		AcceptMethodsDisabled: true,
	},
	BuiltinOBEnd: &BuiltinFunction{
		FuncName:              "obend",
		Value:                 BuiltinOBEndFunc,
		AcceptMethodsDisabled: true,
	},
	BuiltinFlush: &BuiltinFunction{
		FuncName: "flush",
		Value:    BuiltinFlushFunc,
	},
	BuiltinUserData: &BuiltinFunction{
		FuncName: "userData",
		Value:    BuiltinUserDataFunc,
	},
	BuiltinClose: &BuiltinFunction{
		FuncName: "close",
		Value:    BuiltinCloseFunc,
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
	// initialization prevent cycle for BuiltinObjects

	BuiltinObjects[BuiltinRead] = &BuiltinFunction{
		FuncName: "read",
		Value:    BuiltinReadFunc,
	}
	BuiltinObjects[BuiltinWrite] = &BuiltinFunction{
		FuncName: "write",
		Value:    BuiltinWriteFunc,
	}
	BuiltinObjects[BuiltinFilter] = &BuiltinFunction{
		FuncName: "filter",
		Value:    BuiltinFilterFunc,
	}
	BuiltinObjects[BuiltinMap] = &BuiltinFunction{
		FuncName: "map",
		Value:    BuiltinMapFunc,
	}
	BuiltinObjects[BuiltinEach] = &BuiltinFunction{
		FuncName: "each",
		Value:    BuiltinEachFunc,
	}
	BuiltinObjects[BuiltinReduce] = &BuiltinFunction{
		FuncName: "reduce",
		Value:    BuiltinReduceFunc,
	}
	BuiltinObjects[BuiltinEach] = &BuiltinFunction{
		FuncName: "each",
		Value:    BuiltinEachFunc,
	}

	BuiltinObjects[BuiltinIterate] = &BuiltinFunction{
		FuncName: "iterate",
		Value:    BuiltinIterateFunc,
	}
	BuiltinObjects[BuiltinKeys] = &BuiltinFunction{
		FuncName: "keys",
		Value:    BuiltinKeysFunc,
	}
	BuiltinObjects[BuiltinValues] = &BuiltinFunction{
		FuncName: "values",
		Value:    BuiltinValuesFunc,
	}
	BuiltinObjects[BuiltinItems] = &BuiltinFunction{
		FuncName: "items",
		Value:    BuiltinItemsFunc,
	}
	BuiltinObjects[BuiltinCollect] = &BuiltinFunction{
		FuncName: "collect",
		Value:    BuiltinCollectFunc,
	}
	BuiltinObjects[BuiltinEnumerate] = &BuiltinFunction{
		FuncName: "enumerate",
		Value:    BuiltinEnumerateFunc,
	}
	BuiltinObjects[BuiltinIterator] = TIterator
	BuiltinObjects[BuiltinZipIterator] = TZipIterator
	BuiltinObjects[BuiltinIteratorInput] = &BuiltinFunction{
		FuncName: "iteratorInput",
		Value:    funcPORO(BuiltinIteratorInputFunc),
	}
	BuiltinObjects[BuiltinToArray] = &BuiltinFunction{
		FuncName: "toArray",
		Value:    BuiltinToArrayFunc,
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
