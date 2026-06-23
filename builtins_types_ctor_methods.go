package gad

// addCtorMethods registers typed single-argument constructor overloads on a
// global builtin type (in BuiltinObjects), one per source type. Each overload
// delegates to the type's existing constructor, so the conversion semantics are
// unchanged while the typed `T(v <kind>)` headers become available for VM
// dispatch and show up in `repr(T; indent)`. Inputs not matched by an overload
// fall through to the constructor's default handler.
func addCtorMethods(bt BuiltinType, name string, ctor CallableFunc, paramTypes ...ObjectType) {
	methods := make([]TypedCallerObjectWithParamTypes, len(paramTypes))
	for i, pt := range paramTypes {
		pt := pt
		methods[i] = NewFunction(name, ctor,
			FunctionWithParams(func(p func(name string) *ParamBuilder) {
				p("v").Type(pt)
			}))
	}
	BuiltinObjects.AddMethod(bt, methods...)
}

// registerCtorMethods adds typed constructor overloads to the value-type
// builtins. Called from builtin_types.go's init after all types are registered.
func registerCtorMethods() {
	num := []ObjectType{TInt, TUint, TFloat, TDecimal, TChar, TBool, TStr}

	addCtorMethods(BuiltinBool, "bool", NewBoolFunc, num...)
	addCtorMethods(BuiltinInt, "int", NewIntFunc, num...)
	addCtorMethods(BuiltinUint, "uint", NewUintFunc, num...)
	addCtorMethods(BuiltinFloat, "float", NewFloatFunc, num...)
	addCtorMethods(BuiltinDecimal, "decimal", NewDecimalFunc, num...)
	addCtorMethods(BuiltinChar, "char", NewCharFunc, TInt, TUint, TFloat, TBool, TStr, TChar)
	addCtorMethods(BuiltinStr, "str", NewStrFunc,
		TInt, TUint, TFloat, TDecimal, TChar, TBool, TStr, TRawStr, TBytes)
	addCtorMethods(BuiltinRawStr, "rawstr", NewRawStrFunc, TStr, TRawStr)
	addCtorMethods(BuiltinBytes, "bytes", NewBytesFunc, TStr, TRawStr, TBytes, TArray, TInt)
}
