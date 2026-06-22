package gad

import (
	"strconv"

	"github.com/gad-lang/gad/token"
)

// RangeType is the builtin `Range` object type. The `Range(from, to; step)`
// constructor and the `..` operator both produce *Range values.
var RangeType = RegisterBuiltinType(BuiltinRange, "Range", (*Range)(nil), rangeCtor)

// Range is an iterable produced by the `..` operator or the Range(from, to;
// step) constructor. It yields values from From toward To (inclusive),
// advancing by Step (default 1) in the direction of To. From and To are both
// Int or both Char; Step is a positive magnitude.
type Range struct {
	From Object // Int or Char
	To   Object
	Step Int // step magnitude; <= 0 means the default (1)
}

func (o *Range) isChar() bool {
	_, ok := o.From.(Char)
	return ok
}

// Type returns the Range builtin type.
func (o *Range) Type() ObjectType { return RangeType }

// stepMag returns the effective (positive) step magnitude.
func (o *Range) stepMag() int64 {
	if o.Step <= 0 {
		return 1
	}
	return int64(o.Step)
}

func rangeInt64(v Object) int64 {
	switch t := v.(type) {
	case Int:
		return int64(t)
	case Char:
		return int64(t)
	}
	return 0
}

func (o *Range) value(v int64) Object {
	if o.isChar() {
		return Char(v)
	}
	return Int(v)
}

func (o *Range) ToString() string {
	s := o.From.ToString() + " .. " + o.To.ToString()
	if o.Step > 0 {
		s += " / " + strconv.FormatInt(int64(o.Step), 10)
	}
	return s
}

func (o *Range) Equal(right Object) bool {
	r, ok := right.(*Range)
	return ok && o.From.Equal(r.From) && o.To.Equal(r.To) && o.stepMag() == r.stepMag()
}

func (o *Range) IsFalsy() bool { return false }

// withStep returns a copy of the range with the given step magnitude.
func (o *Range) withStep(step Int) *Range {
	return &Range{From: o.From, To: o.To, Step: step}
}

// Iterate yields the range values; the entry key is the 0-based position and
// the value is the Int/Char element.
func (o *Range) Iterate(_ *VM, _ *NamedArgs) Iterator {
	var (
		from = rangeInt64(o.From)
		to   = rangeInt64(o.To)
		step = o.stepMag()
		asc  = from <= to
		cur  int64
		idx  int64
	)

	done := func(v int64) bool {
		if asc {
			return v > to
		}
		return v < to
	}

	set := func(state *IteratorState) {
		if done(cur) {
			state.Mode = IteratorStateModeDone
			return
		}
		state.Entry.K = Int(idx)
		state.Entry.V = o.value(cur)
	}

	return NewIterator(
		func(_ *VM) (state *IteratorState, err error) {
			state = &IteratorState{}
			cur, idx = from, 0
			set(state)
			return
		},
		func(_ *VM, state *IteratorState) (err error) {
			if asc {
				cur += step
			} else {
				cur -= step
			}
			idx++
			set(state)
			return
		},
	).SetInput(o).SetItType(o.Type())
}

// IndexGet exposes the `from` and `to` fields and the `step` method. `r.step()`
// returns the current step magnitude; `r.step(n)` returns a new range with step
// n.
func (o *Range) IndexGet(_ *VM, index Object) (Object, error) {
	switch index.ToString() {
	case "from":
		return o.From, nil
	case "to":
		return o.To, nil
	case "step":
		return NewFunction("step", func(c Call) (Object, error) {
			if c.Args.Length() == 0 {
				return Int(o.stepMag()), nil
			}
			n, ok := c.Args.GetOnly(0).(Int)
			if !ok {
				return nil, NewArgumentTypeError("0", "int", c.Args.GetOnly(0).Type().Name())
			}
			return o.withStep(n), nil
		}), nil
	}
	return nil, ErrNotIndexable
}

// BinaryOp handles `range / step`, returning a new range with that step.
func (o *Range) BinaryOp(_ *VM, tok token.Token, right Object) (Object, error) {
	if tok == token.Quo {
		if n, ok := right.(Int); ok {
			return o.withStep(n), nil
		}
	}
	return nil, NewOperandTypeError(tok.String(), o.Type().Name(), right.Type().Name())
}

// rangeCtor is the Range(from, to; step) constructor body shared by the int and
// char typed methods.
func rangeCtor(c Call) (Object, error) {
	from := c.Args.GetOnly(0)
	to := c.Args.GetOnly(1)
	r := &Range{From: from, To: to}
	if step := c.NamedArgs.GetValueOrNil("step"); step != nil && step != Nil {
		n, ok := step.(Int)
		if !ok {
			return nil, NewArgumentTypeError("step", "int", step.Type().Name())
		}
		r.Step = n
	}
	return r, nil
}

func init() {
	// Register the typed `Range(from, to; step)` methods (int and char) on the
	// Range builtin, mirroring `meth Range(from int, to int; step) { … }`.
	method := func(t ObjectType) *Function {
		return NewFunction("Range", rangeCtor,
			FunctionWithParams(func(p func(name string) *ParamBuilder) {
				p("from").Type(t)
				p("to").Type(t)
			}),
			FunctionWithNamedParams(func(np func(name string) *NamedParamBuilder) {
				np("step").Type(TInt)
			}),
		)
	}
	BuiltinObjects.AddMethod(BuiltinRange, method(TInt), method(TChar))
}
