package gad

import (
	"reflect"

	"github.com/gad-lang/gad/token"
)

// BinaryOp runs a binary operator on two objects through the per-operator
// ObjectWith{Op}BinOperator dispatch (binOpObject), matching core.binOp.
// Internal callers (sort, value comparisons) and embedders use it.
func BinaryOp(vm *VM, tok token.Token, left, right Object) (Object, error) {
	op := BinaryOperatorType(tok)
	if ret, err, handled := binOpObject(vm, op, left, right); handled {
		return ret, err
	}
	if op == TBinaryOperatorSame {
		return binSameFallback(vm, left, right)
	}
	return nil, NewOperandTypeError(tok.String(), left.Type().Name(), right.Type().Name())
}

// binSameFallback computes `left === right` (strict same-identity) when left
// does not implement ObjectWithSameBinOperator: it tries the right operand's
// implementation, then compares primitive go values by reflect (type + value)
// and any other object by address identity. It never errors.
func binSameFallback(vm *VM, left, right Object) (Object, error) {
	if h, ok := right.(ObjectWithSameBinOperator); ok {
		return h.BinOpSame(vm, left)
	}
	if IsPrimitive(left) && IsPrimitive(right) {
		return Bool(reflect.DeepEqual(left, right)), nil
	}
	return Bool(AddressOf(left) == AddressOf(right)), nil
}

// operatorBinaryMethod is the default handler of core.binOp: it dispatches to
// the left (or, for `in`, the right) operand's per-operator
// ObjectWith{Op}BinOperator implementation via binOpObject. A user-defined
// `met core.binOp(_ TBinaryOperatorX, left T, right U)` is more specific (its
// operator and operand types are typed) and so takes precedence.
func operatorBinaryMethod(c Call) (Object, error) {
	op := c.Args.Get(0).(BinaryOperatorType)
	left, right := c.Args.Get(1), c.Args.Get(2)
	if ret, err, handled := binOpObject(c.VM, op, left, right); handled {
		return ret, err
	}
	if op == TBinaryOperatorSame {
		return binSameFallback(c.VM, left, right)
	}
	return Nil, NewOperandTypeError(op.Token().String(), left.Type().Name(), right.Type().Name())
}

// operatorUnaryMethod is the default handler of core.unOp: it dispatches to the
// operand's per-operator ObjectWith{Op}UnaryOperator implementation via
// unOpObject. The logical NOT (`!`) is universal and falls back to truthiness. A
// user-defined `met core.unOp(_ TUnaryOperatorX, operand T)` is more specific and
// so takes precedence.
func operatorUnaryMethod(c Call) (Object, error) {
	op := c.Args.Get(0).(UnaryOperatorType)
	operand := c.Args.Get(1)
	if ret, err, handled := unOpObject(c.VM, op, operand); handled {
		return ret, err
	}
	if op.Token() == token.Not {
		return Bool(operand.IsFalsy()), nil
	}
	return Nil, ErrType.NewError(
		"invalid type for unary '" + op.Token().String() + "': '" +
			operand.Type().Name() + "'")
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

// unaryOperatorMethod builds one `(op, operand T)` unary operator overload. `op`
// is untyped so the overload matches any unary operator for an operand of type t.
func unaryOperatorMethod(name string, h CallableFunc, t ObjectType) *Function {
	return NewFunction(name, h, FunctionWithParams(func(p func(name string) *ParamBuilder) {
		p("op")
		p("operand").Type(t)
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
			operatorMethod("binOp", operatorBinaryMethod, t))
	}

	// Array is the only builtin type with a SelfAssignOp implementation.
	BuiltinObjects.AddMethod(BuiltinSelfAssignOperator,
		operatorMethod("selfAssignOp", operatorSelfAssignMethod, TArray))

	// Expose the primitive types' UnOp implementations as typed methods of
	// core.unOp. The temporal types (time/duration/calendarDate/calendarTime)
	// have no builtin-type key, so their UnOp implementations are reached through
	// the default handler (unOpObject) instead.
	unaryTypes := []ObjectType{
		TInt, TUint, TFloat, TDecimal, TChar, TBool, TFlag,
	}
	for _, t := range unaryTypes {
		BuiltinObjects.AddMethod(BuiltinUnaryOperator,
			unaryOperatorMethod("unOp", operatorUnaryMethod, t))
	}

	// Expose the operator functions under the global `core` namespace
	// (core.binOp / core.selfAssignOp). Done here, after the methods are
	// registered, so the namespace references the final method-bearing objects.
	registerCoreModule()
}

// coreModuleSpec is the spec for the global `core` namespace.
var coreModuleSpec = NewModuleSpecFromName("core")

// CoreModule returns the `core` builtin namespace (the operator functions).
func CoreModule() Dict {
	return Dict{
		"binOp":        BuiltinObjects[BuiltinBinaryOperator],
		"selfAssignOp": BuiltinObjects[BuiltinSelfAssignOperator],
		"unOp":         BuiltinObjects[BuiltinUnaryOperator],
	}
}

// registerCoreModule registers `core` as a global namespace whose members
// `binOp` / `selfAssignOp` resolve to the existing operator builtins. The
// qualified names map to the same builtin enums used by the VM's operator
// dispatch, so `core.binOp(...)` and `met core.binOp(...)` share identity with
// it.
func registerCoreModule() {
	name := coreModuleSpec.Name
	setOperatorModule(BuiltinObjects[BuiltinBinaryOperator])
	setOperatorModule(BuiltinObjects[BuiltinSelfAssignOperator])
	setOperatorModule(BuiltinObjects[BuiltinUnaryOperator])

	BuiltinsMap[name] = BuiltinModuleCore
	BuiltinObjects[BuiltinModuleCore] = CoreModule()
	BuiltinsMap[name+".binOp"] = BuiltinBinaryOperator
	BuiltinsMap[name+".selfAssignOp"] = BuiltinSelfAssignOperator
	BuiltinsMap[name+".unOp"] = BuiltinUnaryOperator
}

// setOperatorModule ties an operator builtin to the core module spec.
func setOperatorModule(o Object) {
	switch m := o.(type) {
	case *BuiltinFunctionWithMethods:
		m.Module = coreModuleSpec
	case *BuiltinFunction:
		m.Module = coreModuleSpec
	}
}
