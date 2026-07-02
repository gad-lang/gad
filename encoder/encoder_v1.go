package encoder

import (
	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/parser/source"
)

var (
	CompiledFunctionV1,
	NilV1,
	StrV1,
	RawStrV1,
	BytesV1,
	ArrayV1,
	DictV1,
	SyncDictV1,
	IntV1,
	UintV1,
	CharV1,
	FloatV1,
	DecimalV1,
	BoolV1,
	FlagV1,
	BytecodeV1,
	SourceFileSetV1,
	SourceFileV1,
	SymbolInfoV1,
	ModuleSpecV1,
	ErrorV1,
	EmbeddedV1,
	RegexpV1,
	TimeV1,
	DurationV1,
	CalendarDateV1,
	CalendarTimeV1,
	EnumV1,
	TypedIdentV1,
	FuncHeaderObjectV1 EncDec
)

const (
	typeNil byte = iota
	typeBool
	typeFlag
	typeInt
	typeUint
	typeChar
	typeFloat
	typeDecimal
	typeStr
	typeRawStr
	typeBytes
	typeArray
	typeDict
	typeSyncDict
	typeCompiledFunction
	typeBytecode
	typeSourceFileSet
	typeSourceFile
	typeSymbolInfo
	typeModuleSpec
	typeError
	typeEmbedded
	typeRegexp
	typeTime
	typeDuration
	typeCalendarDate
	typeCalendarTime
	typeEnum
	typeTypedIdent
	typeFuncHeaderObject
)

const versionV1 byte = 1

func init() {
	Register[gad.NilType](typeNil, versionV1, &NilV1)
	Register[gad.Bool](typeBool, versionV1, &BoolV1)
	Register[gad.Flag](typeFlag, versionV1, &FlagV1)
	Register[gad.Int](typeInt, versionV1, &IntV1)
	Register[gad.Uint](typeUint, versionV1, &UintV1)
	Register[gad.Char](typeChar, versionV1, &CharV1)
	Register[gad.Float](typeFloat, versionV1, &FloatV1)
	Register[gad.Decimal](typeDecimal, versionV1, &DecimalV1)
	Register[gad.Str](typeStr, versionV1, &StrV1)
	Register[gad.RawStr](typeRawStr, versionV1, &RawStrV1)
	Register[gad.Bytes](typeBytes, versionV1, &BytesV1)
	Register[gad.Array](typeArray, versionV1, &ArrayV1)
	Register[gad.Dict](typeDict, versionV1, &DictV1)
	Register[gad.SyncDict](typeSyncDict, versionV1, &SyncDictV1)
	Register[gad.CompiledFunction](typeCompiledFunction, versionV1, &CompiledFunctionV1)
	Register[gad.SymbolInfo](typeSymbolInfo, versionV1, &SymbolInfoV1)
	Register[gad.Bytecode](typeBytecode, versionV1, &BytecodeV1)
	Register[source.FileSet](typeSourceFileSet, versionV1, &SourceFileSetV1)
	Register[source.File](typeSourceFile, versionV1, &SourceFileV1)
	Register[gad.ModuleSpec](typeModuleSpec, versionV1, &ModuleSpecV1)
	Register[gad.Error](typeError, versionV1, &ErrorV1)
	Register[gad.Embedded](typeEmbedded, versionV1, &EmbeddedV1)
	Register[gad.Regexp](typeRegexp, versionV1, &RegexpV1)
	Register[gad.Time](typeTime, versionV1, &TimeV1)
	Register[gad.Duration](typeDuration, versionV1, &DurationV1)
	Register[gad.CalendarDate](typeCalendarDate, versionV1, &CalendarDateV1)
	Register[gad.CalendarTime](typeCalendarTime, versionV1, &CalendarTimeV1)
	Register[gad.Enum](typeEnum, versionV1, &EnumV1)
	Register[gad.TypedIdent](typeTypedIdent, versionV1, &TypedIdentV1)
	Register[gad.FuncHeaderObject](typeFuncHeaderObject, versionV1, &FuncHeaderObjectV1)
}
