package gad

import (
	"reflect"

	"github.com/gad-lang/gad/token"
)

// BinaryOp runs a binary operator on two objects through the per-operator
// ObjectWith{Op}BinOperator dispatch (binOpObject), matching gad.binOp.
// Internal callers (sort, value comparisons) and embedders use it.
func BinaryOp(vm *VM, tok token.Token, left, right Object) (Object, error) {
	if ret, err, handled := binOpObject(vm, BinaryOperatorType(tok), left, right); handled {
		return ret, err
	}
	switch tok {
	case token.Same:
		return binSameFallback(vm, left, right)
	case token.Ain:
		return binAinFallback(vm, left, right)
	}
	return nil, NewOperandTypeError(tok.String(), left.Type().Name(), right.Type().Name())
}

// binAinFallback computes `left ain right` (every value of the left operand is a
// member of right) when right does not implement ObjectWithAinBinOperator: it
// tests each value of left with the `in` membership operator, routed through
// gad.binOpIn so it resolves both Go containers (ObjectWithInBinOperator) and Gad
// types that define `met gad.binOpIn(…)`. A non-array left is treated as a single
// value, so `x ain B` matches `x in B`; an empty left array yields true.
func binAinFallback(vm *VM, left, right Object) (Object, error) {
	values, ok := left.(Array)
	if !ok {
		values = Array{left}
	}
	for _, v := range values {
		r, err := vm.callBinaryOp(token.In, v, right)
		if err != nil {
			return nil, err
		}
		if r.IsFalsy() {
			return False, nil
		}
	}
	return True, nil
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

// binaryOpDispatch runs a binary operator on two operands: it dispatches to the
// left (or, for `in`, the right) operand's per-operator ObjectWith{Op}BinOperator
// implementation via binOpObject, with the `===` (Same) and `ain` fallbacks. It
// backs both the generic gad.binOp default and every per-operator gad.binOp{Op}
// default. A user-defined `met gad.binOp{Op}(left T, right U)` is more specific
// (its operand types are typed) and so takes precedence.
func binaryOpDispatch(vm *VM, op BinaryOperatorType, left, right Object) (Object, error) {
	if ret, err, handled := binOpObject(vm, op, left, right); handled {
		return ret, err
	}
	switch op.Token() {
	case token.Same:
		return binSameFallback(vm, left, right)
	case token.Ain:
		return binAinFallback(vm, left, right)
	}
	return Nil, NewOperandTypeError(op.Token().String(), left.Type().Name(), right.Type().Name())
}

// operatorBinaryMethod is the generic gad.binOp(op, left, right) default handler.
func operatorBinaryMethod(c Call) (Object, error) {
	return binaryOpDispatch(c.VM, c.Args.Get(0).(BinaryOperatorType), c.Args.Get(1), c.Args.Get(2))
}

// binaryOpHandler builds the op-bound default handler for a per-operator
// gad.binOp{Op}(left, right) builtin.
func binaryOpHandler(op BinaryOperatorType) func(Call) (Object, error) {
	return func(c Call) (Object, error) {
		return binaryOpDispatch(c.VM, op, c.Args.Get(0), c.Args.Get(1))
	}
}

// unaryOpDispatch runs a unary operator on one operand: it dispatches to the
// operand's per-operator ObjectWith{Op}UnaryOperator implementation via
// unOpObject; the logical NOT (`!`) is universal and falls back to truthiness.
func unaryOpDispatch(vm *VM, op UnaryOperatorType, operand Object) (Object, error) {
	if ret, err, handled := unOpObject(vm, op, operand); handled {
		return ret, err
	}
	if op.Token() == token.Not {
		return Bool(operand.IsFalsy()), nil
	}
	return Nil, ErrType.NewError(
		"invalid type for unary '" + op.Token().String() + "': '" +
			operand.Type().Name() + "'")
}

// operatorUnaryMethod is the generic gad.unOp(op, operand) default handler.
func operatorUnaryMethod(c Call) (Object, error) {
	return unaryOpDispatch(c.VM, c.Args.Get(0).(UnaryOperatorType), c.Args.Get(1))
}

// unaryOpHandler builds the op-bound default handler for a per-operator
// gad.unOp{Op}(operand) builtin.
func unaryOpHandler(op UnaryOperatorType) func(Call) (Object, error) {
	return func(c Call) (Object, error) {
		return unaryOpDispatch(c.VM, op, c.Args.Get(0))
	}
}

// selfAssignOpDispatch runs a self-assign operator (`x op= y`): it dispatches to
// the left operand's ObjectWith{Op}SelfAssignOperator implementation and, when
// unhandled, falls back to the binary operator (so `x op= y` runs as `x = x op y`).
func selfAssignOpDispatch(vm *VM, op SelfAssignOperatorType, left, right Object) (Object, error) {
	if ret, err, handled := selfAssignOpObject(vm, op, left, right); handled {
		return ret, err
	}
	// Fall back to the binary operator through its gad.binOp{Op} builtin so a
	// user-defined `met gad.binOp{Op}(…)` overload (which lives on that builtin's
	// method table, not reachable by binOpObject) also backs `x op= y`.
	return vm.callBinaryOp(op.Token(), left, right)
}

// operatorSelfAssignMethod is the generic gad.selfAssignOp(op, left, right)
// default handler.
func operatorSelfAssignMethod(c Call) (Object, error) {
	return selfAssignOpDispatch(c.VM, c.Args.Get(0).(SelfAssignOperatorType), c.Args.Get(1), c.Args.Get(2))
}

// selfAssignOpHandler builds the op-bound default handler for a per-operator
// gad.selfAssignOp{Op}(left, right) builtin.
func selfAssignOpHandler(op SelfAssignOperatorType) func(Call) (Object, error) {
	return func(c Call) (Object, error) {
		return selfAssignOpDispatch(c.VM, op, c.Args.Get(0), c.Args.Get(1))
	}
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
	// gad.unOp. The temporal types (time/duration/calendarDate/calendarTime)
	// have no builtin-type key, so their UnOp implementations are reached through
	// the default handler (unOpObject) instead.
	unaryTypes := []ObjectType{
		TInt, TUint, TFloat, TDecimal, TChar, TBool, TFlag,
	}
	for _, t := range unaryTypes {
		BuiltinObjects.AddMethod(BuiltinUnaryOperator,
			unaryOperatorMethod("unOp", operatorUnaryMethod, t))
	}

	// Expose the operator functions under the global `gad` namespace
	// (gad.binOp / gad.selfAssignOp). Done here, after the methods are
	// registered, so the namespace references the final method-bearing objects.
	registerGadModule()
}
