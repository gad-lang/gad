package gad

type BuiltinObjType struct {
	NameValue string
	Value     CallableFunc
	getters   Map
	setters   Map
	methods   Map
}

func (b *BuiltinObjType) Fields() Map {
	return nil
}

func (b *BuiltinObjType) Getters() Map {
	return b.getters
}

func (b *BuiltinObjType) Setters() Map {
	return b.setters
}

func (b *BuiltinObjType) Methods() Map {
	return b.methods
}

func (b *BuiltinObjType) IsChildOf(ObjectType) bool {
	return false
}

func NewBuiltinObjType(name string, init CallableFunc) *BuiltinObjType {
	return &BuiltinObjType{NameValue: name, Value: init}
}

func (b *BuiltinObjType) Name() string {
	return b.NameValue
}

func (b *BuiltinObjType) Type() ObjectType {
	return TNil
}

func (b *BuiltinObjType) ToString() string {
	return "Type::" + b.NameValue
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

func (b *BuiltinObjType) New(*VM, Map) (Object, error) {
	return Nil, nil
}

var (
	TNil,
	TBool,
	TInt,
	TUint,
	TFloat,
	TDecimal,
	TChar,
	TString,
	TBytes,
	TBuffer,
	TArray,
	TMap,
	TSyncMap,
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
)

func init() {
	TNil = RegisterBuiltinType(BuiltinNil, "nil", Nil, func(call Call) (ret Object, err error) {
		return Nil, nil
	})
	TBool = RegisterBuiltinType(BuiltinBool, "bool", True, funcPORO(builtinBoolFunc))
	TInt = RegisterBuiltinType(BuiltinInt, "int", Int(0), funcPi64RO(builtinIntFunc))
	TUint = RegisterBuiltinType(BuiltinUint, "uint", Uint(0), funcPu64RO(builtinUintFunc))
	TFloat = RegisterBuiltinType(BuiltinFloat, "float", Float(0), funcPf64RO(builtinFloatFunc))
	TDecimal = RegisterBuiltinType(BuiltinDecimal, "decimal", Decimal{}, funcPOROe(builtinDecimalFunc))
	TChar = RegisterBuiltinType(BuiltinChar, "char", Char(0), funcPOROe(builtinCharFunc))
	TString = RegisterBuiltinType(BuiltinString, "string", String(""), builtinStringFunc)
	TBytes = RegisterBuiltinType(BuiltinBytes, "bytes", Bytes{}, builtinBytesFunc)
	TBuffer = RegisterBuiltinType(BuiltinBuffer, "buffer", Buffer{}, builtinBufferFunc)
	TArray = RegisterBuiltinType(BuiltinArray, "array", Array{}, func(c Call) (ret Object, err error) {
		return c.Args.Values(), nil
	})
	TMap = RegisterBuiltinType(BuiltinMap, "map", Map{}, func(Call) (ret Object, err error) {
		return Map{}, nil
	})
	TSyncMap = RegisterBuiltinType(BuiltinSyncMap, "syncMap", SyncMap{}, builtinSyncMapFunc)
	TKeyValue = RegisterBuiltinType(BuiltinKeyValue, "keyValue", KeyValue{}, builtinKeyValueFunc)
	TKeyValueArray = RegisterBuiltinType(BuiltinKeyValueArray, "keyValueArray", KeyValueArray{}, builtinKeyValueArrayFunc)
	TError = RegisterBuiltinType(BuiltinError, "error", Error{}, funcPORO(builtinErrorFunc))
}
