package gad

// operatorBinaryMethod is the shared handler for the per-type `@binaryOperator`
// overloads: it runs the left operand's BinaryOp. The overloads differ only in
// the typed `left` parameter, which is what exposes each type's operator support
// as a method of `@binaryOperator` (visible in repr and dispatched by type). A
// user-defined `met @binaryOperator(_ TBinaryOperatorX, left T, right U)` is
// more specific (its operator and right types are typed) and so takes
// precedence.
func operatorBinaryMethod(c Call) (Object, error) {
	op := c.Args.Get(0).(BinaryOperatorType)
	if h, ok := c.Args.Get(1).(BinaryOperatorHandler); ok {
		return h.BinaryOp(c.VM, op.Token(), c.Args.Get(2))
	}
	return Nil, ErrInvalidOperator.NewError(op.Name())
}

// operatorSelfAssignMethod is the shared handler for `@selfAssignOperator`
// overloads: it runs the left operand's SelfAssignOp and, when the operator is
// not handled, falls back to the binary operator (mirroring the default).
func operatorSelfAssignMethod(c Call) (Object, error) {
	op := c.Args.Get(0).(SelfAssignOperatorType)
	left, right := c.Args.Get(1), c.Args.Get(2)
	if h, ok := left.(SelfAssignOperatorHandler); ok {
		if ret, handled, err := h.SelfAssignOp(c.VM, op.Token(), right); err != nil || handled {
			return ret, err
		}
	}
	return c.VM.Builtins.Call(BuiltinBinaryOperator,
		Call{VM: c.VM, Args: Args{{BinaryOperatorType(op), left, right}}})
}

// operatorMethod builds one `(op, left T, right)` operator overload. `op` and
// `right` are untyped so the overload matches any operator/right operand for a
// left operand of type t.
func operatorMethod(name string, h CallableFunc, t ObjectType) *Function {
	return NewFunction(name, h, FunctionWithParams(func(p func(name string) *ParamBuilder) {
		p("op")
		p("left").Type(t)
		p("right")
	}))
}

// registerOperatorMethods exposes the builtin types' BinaryOp / SelfAssignOp
// implementations as typed methods of `@binaryOperator` / `@selfAssignOperator`.
// The operator builtins keep their default handler (delegating to BinaryOp /
// SelfAssignOp for any handler type), so types not listed here, class instances
// and custom Go objects still work. Called from builtin_types.go's init.
func registerOperatorMethods() {
	// Only types with a registered builtin-type key can be used as distinct
	// method param types. The time-module types (time/duration/calendarDate/
	// calendarTime) are NewBuiltinObjType values without such a key, so they are
	// left to the operator builtins' default handler.
	binaryTypes := []ObjectType{
		TInt, TUint, TFloat, TDecimal, TChar, TBool, TFlag,
		TStr, TRawStr, TBytes, TArray, TDict, TSyncDict,
		TKeyValue, TKeyValueArray, RangeType,
	}
	for _, t := range binaryTypes {
		BuiltinObjects.AddMethod(BuiltinBinaryOperator,
			operatorMethod("@binaryOperator", operatorBinaryMethod, t))
	}

	// Array is the only builtin type with a SelfAssignOp implementation.
	BuiltinObjects.AddMethod(BuiltinSelfAssignOperator,
		operatorMethod("@selfAssignOperator", operatorSelfAssignMethod, TArray))
}
