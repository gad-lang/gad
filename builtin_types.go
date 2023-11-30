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

func (b *BuiltinObjType) New(*VM, Dict) (Object, error) {
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
	TText,
	TString,
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
)

func init() {
	TNil = RegisterBuiltinType(BuiltinNil, "nil", Nil, func(call Call) (ret Object, err error) {
		return Nil, nil
	})
	TBool = RegisterBuiltinType(BuiltinBool, "bool", True, funcPORO(BuiltinBoolFunc))
	TInt = RegisterBuiltinType(BuiltinInt, "int", Int(0), funcPi64RO(BuiltinIntFunc))
	TUint = RegisterBuiltinType(BuiltinUint, "uint", Uint(0), funcPu64RO(BuiltinUintFunc))
	TFloat = RegisterBuiltinType(BuiltinFloat, "float", Float(0), funcPf64RO(BuiltinFloatFunc))
	TDecimal = RegisterBuiltinType(BuiltinDecimal, "decimal", Decimal{}, funcPOROe(BuiltinDecimalFunc))
	TChar = RegisterBuiltinType(BuiltinChar, "char", Char(0), funcPOROe(BuiltinCharFunc))
	TText = RegisterBuiltinType(BuiltinText, "text", Text(""), BuiltinTextFunc)
	TString = RegisterBuiltinType(BuiltinString, "string", String(""), BuiltinStringFunc)
	TBytes = RegisterBuiltinType(BuiltinBytes, "bytes", Bytes{}, BuiltinBytesFunc)
	TBuffer = RegisterBuiltinType(BuiltinBuffer, "buffer", Buffer{}, BuiltinBufferFunc)
	TArray = RegisterBuiltinType(BuiltinArray, "array", Array{}, func(c Call) (ret Object, err error) {
		return c.Args.Values(), nil
	})
	TDict = RegisterBuiltinType(BuiltinDict, "dict", Dict{}, func(Call) (ret Object, err error) {
		return Dict{}, nil
	})
	TSyncDict = RegisterBuiltinType(BuiltinSyncDic, "syncDict", SyncMap{}, BuiltinSyncMapFunc)
	TKeyValue = RegisterBuiltinType(BuiltinKeyValue, "keyValue", KeyValue{}, BuiltinKeyValueFunc)
	TKeyValueArray = RegisterBuiltinType(BuiltinKeyValueArray, "keyValueArray", KeyValueArray{}, BuiltinKeyValueArrayFunc)
	TError = RegisterBuiltinType(BuiltinError, "error", Error{}, funcPORO(BuiltinErrorFunc))
}
