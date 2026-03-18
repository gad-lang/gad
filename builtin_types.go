package gad

type BuiltinObjTypeKey BuiltinType

func (BuiltinObjTypeKey) GadObjectType() {}

func (b BuiltinObjTypeKey) BType() BuiltinType {
	return BuiltinType(b)
}

func (b BuiltinObjTypeKey) IsFalsy() bool {
	return false
}

func (b BuiltinObjTypeKey) Type() ObjectType {
	return TAny
}

func (b BuiltinObjTypeKey) ToString() string {
	return "BuiltinObjTypeKey " + ReprQuote(b.BType().String())
}

func (b BuiltinObjTypeKey) Equal(right Object) bool {
	if r, ok := right.(BuiltinObjTypeKey); ok {
		return r == b
	}
	if r, ok := right.(*BuiltinObjType); ok {
		return BuiltinObjTypeKey(r.builtinType) == b
	}
	return false
}

func (b BuiltinObjTypeKey) Call(c Call) (Object, error) {
	return nil, ErrNotCallable.NewError("is " + b.ToString())
}

func (b BuiltinObjTypeKey) Name() string {
	return b.BType().String()
}

func (b BuiltinObjTypeKey) String() string {
	return ReprQuoteTyped("builtinType", b.FullName())
}

func (b BuiltinObjTypeKey) FullName() string {
	return b.Name()
}

type BuiltinObjType struct {
	getters     Dict
	setters     Dict
	methods     Dict
	builtinType BuiltinType
	name        string
	*FuncSpec
}

func NewBuiltinObjType(name string) *BuiltinObjType {
	t := &BuiltinObjType{name: name}
	t.FuncSpec = NewFuncSpec(t)
	return t
}

func (t *BuiltinObjType) GadObjectType() {}

func (t *BuiltinObjType) GetModule() *Module {
	return nil
}

func (t *BuiltinObjType) FuncSpecName() string {
	return "builtin type " + ReprQuote(t.FullName())
}

func (t *BuiltinObjType) ToString() string {
	return t.String()
}

func (t *BuiltinObjType) TypeKey() BuiltinObjTypeKey {
	return BuiltinObjTypeKey(t.builtinType)
}

func (t *BuiltinObjType) WithNew(f CallableFunc) *BuiltinObjType {
	t.FuncSpec.defaul = &Function{
		FuncName: "init",
		Value:    f,
		ToStringFunc: func() string {
			return t.ToString()
		},
	}
	return t
}

func (t *BuiltinObjType) BuiltinType() BuiltinType {
	return t.builtinType
}

func (t *BuiltinObjType) Fields() Dict {
	return nil
}

func (t *BuiltinObjType) Getters() Dict {
	return t.getters
}

func (t *BuiltinObjType) Setters() Dict {
	return t.setters
}

func (t *BuiltinObjType) Methods() Dict {
	return t.methods
}

func (t *BuiltinObjType) Name() string {
	return t.name
}

func (t *BuiltinObjType) FullName() string {
	return t.name
}

func (t *BuiltinObjType) Type() ObjectType {
	return TBase
}

func (t *BuiltinObjType) IsFalsy() bool {
	return false
}

func (t *BuiltinObjType) Equal(right Object) bool {
	switch r := right.(type) {
	case *BuiltinObjType:
		return t == r
	case BuiltinObjTypeKey:
		return t.TypeKey() == r
	default:
		return false
	}
}

func (t *BuiltinObjType) String() string {
	return ReprQuoteTyped("builtinType", t.FullName())
}

func (t *BuiltinObjType) ReprTypeName() string {
	return "builtinType " + ReprQuote(t.FullName())
}

func (t *BuiltinObjType) Print(state *PrinterState) (err error) {
	if ok, _ := state.options.TypesAsFullNames(); ok {
		return state.WriteString(t.FullName())
	}
	return t.PrintFuncWrapper(state, t)
}

func (t BuiltinObjType) Copy() Object {
	cp := &t
	cp.FuncSpec = cp.FuncSpec.CopyWithTarget(cp)
	return cp
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
	TTypedIdent,
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
	TError,
	TPrinterState,
	TFunc,
	TComputedValue,
	TBuiltinFunction,
	TCallWrapper,
	TCompiledFunction,
	TFunction,
	TKeyValueArrays,
	TArgs,
	TNamedArgs,
	TObjectPtr,
	TReader,
	TWriter,
	TDiscardWriter,
	TObjectTypeArray,
	TReflectMethod,
	TIndexGetProxy BuiltinObjTypeKey
)

func init() {
	BuiltinObjects[BuiltinAny] = TAny
	BuiltinsMap[TAny.Name()] = BuiltinAny

	TNil = RegisterBuiltinType(BuiltinNil, "nil", Nil, nil).TypeKey()
	TFlag = RegisterBuiltinType(BuiltinFlag, "flag", Yes, NewFlagFunc).TypeKey()
	TBool = RegisterBuiltinType(BuiltinBool, "bool", True, NewBoolFunc).TypeKey()
	TInt = RegisterBuiltinType(BuiltinInt, "int", Int(0), NewIntFunc).TypeKey()
	TUint = RegisterBuiltinType(BuiltinUint, "uint", Uint(0), NewUintFunc).TypeKey()
	TFloat = RegisterBuiltinType(BuiltinFloat, "float", Float(0), NewFloatFunc).TypeKey()
	TDecimal = RegisterBuiltinType(BuiltinDecimal, "decimal", Decimal{}, NewDecimalFunc).TypeKey()
	TChar = RegisterBuiltinType(BuiltinChar, "char", Char(0), NewCharFunc).TypeKey()
	TRawStr = RegisterBuiltinType(BuiltinRawStr, "rawstr", RawStr(""), NewRawStrFunc).TypeKey()
	TStr = RegisterBuiltinType(BuiltinStr, "str", Str(""), NewStrFunc).TypeKey()
	TTypedIdent = RegisterBuiltinType(BuiltinTypedIdent, "typedIdent", TypedIdent{}, NewTypedIdentFunc).TypeKey()
	TBytes = RegisterBuiltinType(BuiltinBytes, "bytes", Bytes{}, NewBytesFunc).TypeKey()
	TBuffer = RegisterBuiltinType(BuiltinBuffer, "buffer", Buffer{}, NewBufferFunc).TypeKey()
	TArray = RegisterBuiltinType(BuiltinArray, "array", Array{}, NewArrayFunc).TypeKey()
	TDict = RegisterBuiltinType(BuiltinDict, "dict", Dict{}, NewDictFunc).TypeKey()
	TSyncDict = RegisterBuiltinType(BuiltinSyncDic, "syncDict", SyncDict{}, NewSyncDictFunc).TypeKey()
	TKeyValue = RegisterBuiltinType(BuiltinKeyValue, "keyValue", KeyValue{}, NewKeyValueFunc).TypeKey()
	TKeyValueArray = RegisterBuiltinType(BuiltinKeyValueArray, "keyValueArray", KeyValueArray{}, NewKeyValueArrayFunc).TypeKey()
	TRegexp = RegisterBuiltinType(BuiltinRegexp, "regexp", Regexp{}, NewRegexpFunc).TypeKey()
	TRegexpStrsResult = RegisterBuiltinType(BuiltinRegexpStrsResult, "regexpStrsResult", RegexpStrsResult{}, nil).TypeKey()
	TRegexpStrsSliceResult = RegisterBuiltinType(BuiltinRegexpStrsSliceResult, "regexpStrsSliceResult", RegexpStrsSliceResult{}, nil).TypeKey()
	TRegexpBytesResult = RegisterBuiltinType(BuiltinRegexpBytesResult, "regexpBytesResult", RegexpBytesResult{}, nil).TypeKey()
	TRegexpBytesSliceResult = RegisterBuiltinType(BuiltinRegexpBytesSliceResult, "regexpBytesSliceResult", RegexpBytesSliceResult{}, nil).TypeKey()
	TMixedParams = RegisterBuiltinType(BuiltinMixedParams, "MixedParams", MixedParams{}, NewMixedParamsFunc).TypeKey()
	TError = RegisterBuiltinType(BuiltinError, "error", Error{}, NewErrorFunc).TypeKey()
	TPrinterState = RegisterBuiltinType(BuiltinPrinterState, "PrinterState", PrinterState{}, NewPrinterStateFunc).TypeKey()
	TFunc = RegisterBuiltinType(BuiltinFunc, "Func", FuncSpec{}, NewFuncFunc).TypeKey()
	TComputedValue = RegisterBuiltinType(BuiltinComputedValue, "ComputedValue", ComputedValue{}, NewComputedValue).TypeKey()

	TBuiltinFunction = RegisterBuiltinType(BuiltinStaticTypeBuiltinFunction, "builtinFunction", BuiltinFunction{}, nil).TypeKey()
	TCallWrapper = RegisterBuiltinType(BuiltinStaticTypeCallWrapper, "callwrap", CallWrapper{}, nil).TypeKey()
	TCompiledFunction = RegisterBuiltinType(BuiltinStaticTypeCompiledFunction, "compiledFunction", CompiledFunction{}, nil).TypeKey()
	TFunction = RegisterBuiltinType(BuiltinStaticTypeFunction, "function", Function{}, nil).TypeKey()
	TKeyValueArrays = RegisterBuiltinType(BuiltinStaticTypeKeyValueArrays, "keyValueArrays", KeyValueArrays{}, nil).TypeKey()
	TArgs = RegisterBuiltinType(BuiltinStaticTypeArgs, "args", Args{}, nil).TypeKey()
	TNamedArgs = RegisterBuiltinType(BuiltinStaticTypeNamedArgs, "namedArgs", NamedArgs{}, nil).TypeKey()
	TObjectPtr = RegisterBuiltinType(BuiltinStaticTypeObjectPtr, "objectPtr", ObjectPtr{}, nil).TypeKey()
	TReader = RegisterBuiltinType(BuiltinStaticTypeReader, "reader", reader{}, nil).TypeKey()
	TWriter = RegisterBuiltinType(BuiltinStaticTypeWriter, "writer", writer{}, nil).TypeKey()
	TDiscardWriter = RegisterBuiltinType(BuiltinStaticTypeDiscardWriter, "discardWriter", discardWriter{}, nil).TypeKey()
	TObjectTypeArray = RegisterBuiltinType(BuiltinStaticTypeObjectTypeArray, "objectTypeArray", ObjectTypeArray{}, nil).TypeKey()
	TReflectMethod = RegisterBuiltinType(BuiltinStaticTypeReflectMethod, "reflectMethod", ReflectMethod{}, nil).TypeKey()
	TIndexGetProxy = RegisterBuiltinType(BuiltinStaticTypeIndexGetProxy, "indexGetProxy", IndexGetProxy{}, nil).TypeKey()
}
