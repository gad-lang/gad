package gad

// Per-type unary operator implementations (op_api.go's
// ObjectWith{Op}UnaryOperator interfaces). They are reached through core.unOp:
// the VM's OpUnary calls it and it dispatches here via unOpObject. Logical NOT
// (`!`) is universal (truthiness) and handled by core.unOp's default, so only
// Flag overrides it (to stay a Flag).

// --- Int ---

func (o Int) UnOpSub(*VM) (Object, error) { return -o, nil }
func (o Int) UnOpXor(*VM) (Object, error) { return ^o, nil }
func (o Int) UnOpAdd(*VM) (Object, error) { return o, nil }
func (o Int) UnOpInc(*VM) (Object, error) { return o + 1, nil }
func (o Int) UnOpDec(*VM) (Object, error) { return o - 1, nil }

// --- Uint ---

func (o Uint) UnOpSub(*VM) (Object, error) { return -o, nil }
func (o Uint) UnOpXor(*VM) (Object, error) { return ^o, nil }
func (o Uint) UnOpAdd(*VM) (Object, error) { return o, nil }
func (o Uint) UnOpInc(*VM) (Object, error) { return o + 1, nil }
func (o Uint) UnOpDec(*VM) (Object, error) { return o - 1, nil }

// --- Float ---

func (o Float) UnOpSub(*VM) (Object, error) { return -o, nil }
func (o Float) UnOpAdd(*VM) (Object, error) { return o, nil }
func (o Float) UnOpInc(*VM) (Object, error) { return o + 1, nil }
func (o Float) UnOpDec(*VM) (Object, error) { return o - 1, nil }

// --- Char ---

func (o Char) UnOpSub(*VM) (Object, error) { return Int(-o), nil }
func (o Char) UnOpXor(*VM) (Object, error) { return ^Int(o), nil }
func (o Char) UnOpAdd(*VM) (Object, error) { return o, nil }
func (o Char) UnOpInc(*VM) (Object, error) { return o + 1, nil }
func (o Char) UnOpDec(*VM) (Object, error) { return o - 1, nil }

// --- Decimal ---

func (o Decimal) UnOpInc(*VM) (Object, error) {
	return Decimal(o.ToGo().Add(DecimalFromInt(1).ToGo())), nil
}

func (o Decimal) UnOpDec(*VM) (Object, error) {
	return Decimal(o.ToGo().Sub(DecimalFromInt(1).ToGo())), nil
}

// --- Bool ---

func (o Bool) UnOpSub(*VM) (Object, error) {
	if o {
		return Int(-1), nil
	}
	return Int(0), nil
}

func (o Bool) UnOpXor(*VM) (Object, error) {
	if o {
		return ^Int(1), nil
	}
	return ^Int(0), nil
}

func (o Bool) UnOpAdd(*VM) (Object, error) {
	if o {
		return Int(1), nil
	}
	return Int(0), nil
}

// --- Flag ---

// UnOpNot keeps logical NOT of a Flag a Flag (the universal default would
// return a Bool).
func (o Flag) UnOpNot(*VM) (Object, error) { return Flag(o == No), nil }

func (o Flag) UnOpXor(*VM) (Object, error) {
	if o {
		return ^Int(1), nil
	}
	return ^Int(0), nil
}

func (o Flag) UnOpAdd(*VM) (Object, error) {
	if o {
		return Int(1), nil
	}
	return Int(0), nil
}
