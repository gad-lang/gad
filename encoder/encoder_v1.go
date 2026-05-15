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
	ErrorV1 EncDec
)

const (
	binNilV1 byte = iota
	binBoolV1
	binFlagV1
	binIntV1
	binUintV1
	binCharV1
	binFloatV1
	binDecimalV1
	binStrV1
	binRawStrV1
	binBytesV1
	binArrayV1
	binDictV1
	binSyncDictV1
	binCompiledFunctionV1
	binBytecodeV1
	binSourceFileSetV1
	binSourceFileV1
	binSymbolInfoV1
	binModuleSpecV1
	binErrorV1
)

func init() {
	Register[gad.NilType](binNilV1, &NilV1)
	Register[gad.Bool](binBoolV1, &BoolV1)
	Register[gad.Flag](binFlagV1, &FlagV1)
	Register[gad.Int](binIntV1, &IntV1)
	Register[gad.Uint](binUintV1, &UintV1)
	Register[gad.Char](binCharV1, &CharV1)
	Register[gad.Float](binFloatV1, &FloatV1)
	Register[gad.Decimal](binDecimalV1, &DecimalV1)
	Register[gad.Str](binStrV1, &StrV1)
	Register[gad.RawStr](binRawStrV1, &RawStrV1)
	Register[gad.Bytes](binBytesV1, &BytesV1)
	Register[gad.Array](binArrayV1, &ArrayV1)
	Register[gad.Dict](binDictV1, &DictV1)
	Register[gad.SyncDict](binSyncDictV1, &SyncDictV1)
	Register[gad.CompiledFunction](binCompiledFunctionV1, &CompiledFunctionV1)
	Register[gad.SymbolInfo](binSymbolInfoV1, &SymbolInfoV1)
	Register[gad.Bytecode](binBytecodeV1, &BytecodeV1)
	Register[source.FileSet](binSourceFileSetV1, &SourceFileSetV1)
	Register[source.File](binSourceFileV1, &SourceFileV1)
	Register[gad.ModuleSpec](binModuleSpecV1, &ModuleSpecV1)
	Register[gad.Error](binErrorV1, &ErrorV1)
}
