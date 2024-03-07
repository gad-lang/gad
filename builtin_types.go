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
	TFlag = RegisterBuiltinType(BuiltinFlag, "flag", Yes, funcPORO(BuiltinFlagFunc))
	TBool = RegisterBuiltinType(BuiltinBool, "bool", True, funcPORO(BuiltinBoolFunc))
	TInt = RegisterBuiltinType(BuiltinInt, "int", Int(0), funcPi64RO(BuiltinIntFunc))
	TUint = RegisterBuiltinType(BuiltinUint, "uint", Uint(0), funcPu64RO(BuiltinUintFunc))
	TFloat = RegisterBuiltinType(BuiltinFloat, "float", Float(0), funcPf64RO(BuiltinFloatFunc))
	TDecimal = RegisterBuiltinType(BuiltinDecimal, "decimal", Decimal{}, funcPpVM_OROe(BuiltinDecimalFunc))
	TChar = RegisterBuiltinType(BuiltinChar, "char", Char(0), funcPOROe(BuiltinCharFunc))
	TRawStr = RegisterBuiltinType(BuiltinRawStr, "rawstr", RawStr(""), BuiltinRawStrFunc)
	TStr = RegisterBuiltinType(BuiltinStr, "str", Str(""), BuiltinStringFunc)
	TBytes = RegisterBuiltinType(BuiltinBytes, "bytes", Bytes{}, BuiltinBytesFunc)
	TBuffer = RegisterBuiltinType(BuiltinBuffer, "buffer", Buffer{}, BuiltinBufferFunc)
	TArray = RegisterBuiltinType(BuiltinArray, "array", Array{}, func(c Call) (ret Object, err error) {
		return c.Args.Values(), nil
	})
	TDict = RegisterBuiltinType(BuiltinDict, "dict", Dict{}, BuiltinDictFunc)
	TSyncDict = RegisterBuiltinType(BuiltinSyncDic, "syncDict", SyncDict{}, BuiltinSyncDictFunc)
	TKeyValue = RegisterBuiltinType(BuiltinKeyValue, "keyValue", KeyValue{}, BuiltinKeyValueFunc)
	TKeyValueArray = RegisterBuiltinType(BuiltinKeyValueArray, "keyValueArray", KeyValueArray{}, BuiltinKeyValueArrayFunc)
	TError = RegisterBuiltinType(BuiltinError, "error", Error{}, funcPORO(BuiltinErrorFunc))
}
