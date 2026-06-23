package gad

// addCtorMethods registers typed single-argument constructor overloads on a
// global builtin type (in BuiltinObjects), one per source type. The handler
// runs the conversion for the matched type; the typed `T(v <kind>)` headers
// drive VM dispatch and show up in `repr(T; indent)`. Inputs not matched by an
// overload fall through to the constructor's default handler.
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

// ctorTypeError builds a constructor default handler that rejects any input not
// matched by a typed overload, naming the accepted types. Used by the
// value-type constructors whose accepted inputs are fully enumerated by their
// typed overloads (so the conversion is the AddMethod dispatch, not a catch-all
// coercion).
func ctorTypeError(accepts string) CallableFunc {
	return func(c Call) (Object, error) {
		if err := c.Args.CheckLen(1); err != nil {
			return Nil, err
		}
		return Nil, NewArgumentTypeError("1st", accepts, c.Args.Get(0).Type().Name())
	}
}

// convAccepts is the human-readable accepted-type list for the value-type
// conversion constructors (int/uint/float/char).
const convAccepts = "int|uint|float|decimal|char|bool|str|rawstr"

// registerCtorMethods adds typed constructor overloads to the value-type
// builtins. Called from builtin_types.go's init after all types are registered.
//
// int/uint/float/char are fully replaced: the typed overloads ARE the
// conversion dispatch and the default errors (no catch-all coercion). bool/flag
// (truthiness, accept any) and decimal/str/rawstr/bytes keep their catch-all
// default, so for those the overloads only add typed headers.
func registerCtorMethods() {
	// Built here (not as a package var) because the T* keys are assigned in
	// builtin_types.go's init, which runs before this is called.
	convKinds := []ObjectType{TInt, TUint, TFloat, TDecimal, TChar, TBool, TStr, TRawStr}

	addCtorMethods(BuiltinInt, "int", NewIntFunc, convKinds...)
	addCtorMethods(BuiltinUint, "uint", NewUintFunc, convKinds...)
	addCtorMethods(BuiltinFloat, "float", NewFloatFunc, convKinds...)
	addCtorMethods(BuiltinChar, "char", NewCharFunc, convKinds...)

	addCtorMethods(BuiltinBool, "bool", NewBoolFunc, convKinds...)
	addCtorMethods(BuiltinDecimal, "decimal", NewDecimalFunc,
		TInt, TUint, TFloat, TDecimal, TChar, TBool, TStr, TRawStr)
	addCtorMethods(BuiltinStr, "str", NewStrFunc,
		TInt, TUint, TFloat, TDecimal, TChar, TBool, TStr, TRawStr, TBytes)
	addCtorMethods(BuiltinRawStr, "rawstr", NewRawStrFunc, TStr, TRawStr)
	addCtorMethods(BuiltinBytes, "bytes", NewBytesFunc, TStr, TRawStr, TBytes, TArray, TInt)
}
