package gad

type BuiltinObjType struct {
	NameValue string
	Value     CallableFunc
	getters   Dict
	setters   Dict
	methods   Dict
}

func (b *BuiltinObjType) Fields() Dict {
	return nil
}

func (b *BuiltinObjType) Getters() Dict {
	return b.getters
}

func (b *BuiltinObjType) Setters() Dict {
	return b.setters
}

func (b *BuiltinObjType) Methods() Dict {
	return b.methods
}

func (b *BuiltinObjType) IsChildOf(t ObjectType) bool {
	return t == TBase
}

func NewBuiltinObjType(name string, init CallableFunc) *BuiltinObjType {
	return &BuiltinObjType{NameValue: name, Value: init}
}

func (b *BuiltinObjType) Name() string {
	return b.NameValue
}

func (b *BuiltinObjType) Type() ObjectType {
	return TBase
}

func (b *BuiltinObjType) ToString() string {
	return ReprQuote("builtinType " + b.NameValue)
}

func (b *BuiltinObjType) IsFalsy() bool {
	return false
}

func (b *BuiltinObjType) Equal(right Object) bool {
	v, ok := right.(*BuiltinObjType)
	if !ok {
		return false
	}
	return v == b
}

func (b *BuiltinObjType) Call(c Call) (Object, error) {
	return b.Value(c)
}

func (b *BuiltinObjType) New(*VM, Dict) (Object, error) {
	return Nil, nil
}

func (b *BuiltinObjType) String() string {
	return ReprQuote("builtinType:" + b.Name())
}

var (
	TNil,
	TFlag,
	TBool,
	TInt,
	TUint,
	TFloat,
	TDecimal,
	TChar,
	TRawStr,
	TStr,
	TBytes,
	TBuffer,
	TArray,
	TDict,
	TSyncDict,
	TKeyValue,
	TKeyValueArray,
	TRegexp,
	TRegexpStrsResult,
	TRegexpStrsSliceResult,
	TRegexpBytesResult,
	TRegexpBytesSliceResult,
	TMixedParams,
	TError ObjectType

	TBuiltinFunction = &BuiltinObjType{
		NameValue: "builtinFunction",
	}
	TCallWrapper = &BuiltinObjType{
		NameValue: "callwrap",
	}
	TCompiledFunction = &BuiltinObjType{
		NameValue: "compiledFunction",
	}
	TFunction = &BuiltinObjType{
		NameValue: "function",
	}
	TKeyValueArrays = &BuiltinObjType{
		NameValue: "keyValueArrays",
	}
	TArgs = &BuiltinObjType{
		NameValue: "args",
	}
	TNamedArgs = &BuiltinObjType{
		NameValue: "namedArgs",
	}
	TObjectPtr = &BuiltinObjType{
		NameValue: "objectPtr",
	}
	TReader = &BuiltinObjType{
		NameValue: "reader",
	}
	TWriter = &BuiltinObjType{
		NameValue: "writer",
	}
	TDiscardWriter = &BuiltinObjType{
		NameValue: "discardWriter",
	}
	TObjectTypeArray = &BuiltinObjType{
		NameValue: "objectTypeArray",
	}
	TReflectMethod = &BuiltinObjType{
		NameValue: "reflectMethod",
	}
	TIndexGetProxy = &BuiltinObjType{
		NameValue: "indexGetProxy",
	}
)

func init() {
	TNil = RegisterBuiltinType(BuiltinNil, "nil", Nil, nil)
	TFlag = RegisterBuiltinType(BuiltinFlag, "flag", Yes, NewFlagFunc)
	TBool = RegisterBuiltinType(BuiltinBool, "bool", True, NewBoolFunc)
	TInt = RegisterBuiltinType(BuiltinInt, "int", Int(0), NewIntFunc)
	TUint = RegisterBuiltinType(BuiltinUint, "uint", Uint(0), NewUintFunc)
	TFloat = RegisterBuiltinType(BuiltinFloat, "float", Float(0), NewFloatFunc)
	TDecimal = RegisterBuiltinType(BuiltinDecimal, "decimal", Decimal{}, NewDecimalFunc)
	TChar = RegisterBuiltinType(BuiltinChar, "char", Char(0), NewCharFunc)
	TRawStr = RegisterBuiltinType(BuiltinRawStr, "rawstr", RawStr(""), NewRawStrFunc)
	TStr = RegisterBuiltinType(BuiltinStr, "str", Str(""), NewStringFunc)
	TBytes = RegisterBuiltinType(BuiltinBytes, "bytes", Bytes{}, NewBytesFunc)
	TBuffer = RegisterBuiltinType(BuiltinBuffer, "buffer", Buffer{}, NewBufferFunc)
	TArray = RegisterBuiltinType(BuiltinArray, "array", Array{}, NewArrayFunc)
	TDict = RegisterBuiltinType(BuiltinDict, "dict", Dict{}, NewDictFunc)
	TSyncDict = RegisterBuiltinType(BuiltinSyncDic, "syncDict", SyncDict{}, NewSyncDictFunc)
	TKeyValue = RegisterBuiltinType(BuiltinKeyValue, "keyValue", KeyValue{}, NewKeyValueFunc)
	TKeyValueArray = RegisterBuiltinType(BuiltinKeyValueArray, "keyValueArray", KeyValueArray{}, NewKeyValueArrayFunc)
	TRegexp = RegisterBuiltinType(BuiltinRegexp, "regexp", Regexp{}, NewRegexpFunc)
	TRegexpStrsResult = RegisterBuiltinType(BuiltinRegexpStrsResult, "regexpStrsResult", RegexpStrsResult{}, nil)
	TRegexpStrsSliceResult = RegisterBuiltinType(BuiltinRegexpStrsSliceResult, "regexpStrsSliceResult", RegexpStrsSliceResult{}, nil)
	TRegexpBytesResult = RegisterBuiltinType(BuiltinRegexpBytesResult, "regexpBytesResult", RegexpBytesResult{}, nil)
	TRegexpBytesSliceResult = RegisterBuiltinType(BuiltinRegexpBytesSliceResult, "regexpBytesSliceResult", RegexpBytesSliceResult{}, nil)
	TMixedParams = RegisterBuiltinType(BuiltinMixedParams, "MixedParams", MixedParams{}, NewMixedParamsFunc)
	TError = RegisterBuiltinType(BuiltinError, "error", Error{}, NewErrorFunc)
}
