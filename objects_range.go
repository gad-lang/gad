package gad

import (
	"time"

	"github.com/shopspring/decimal"
)

// RangeType is the builtin `Range` object type. The `Range(from, to; step)`
// constructor and the `..` operator both produce *Range values.
var RangeType = RegisterBuiltinType(BuiltinRange, "Range", (*Range)(nil), NewRangeFunc)

// Range is an iterable produced by the `..` operator or the Range(from, to;
// step) constructor. It yields values from From toward To (inclusive),
// advancing by Step in the direction of To. From and To share an element kind;
// Step is a number for numeric/char ranges and a duration for temporal ranges
// (nil selects the default: 1 for numbers, one day for dates/times).
type Range struct {
	From Object
	To   Object
	Step Object // nil = default
}

// Type returns the Range builtin type.
func (o *Range) Type() ObjectType { return RangeType }

// effectiveStep returns the configured step, or the kind's default when unset.
func (o *Range) effectiveStep() Object {
	if o.Step != nil {
		return o.Step
	}
	switch o.From.(type) {
	case Float:
		return Float(1)
	case Decimal:
		return DecimalFromInt(1)
	case *Time, CalendarDate, CalendarTime:
		return Duration(24 * time.Hour)
	default: // Int, Uint, Char
		return Int(1)
	}
}

// ToString renders the range as `from .. to` (with ` / step` when a step is set).
func (o *Range) ToString() string {
	s := o.From.ToString() + " .. " + o.To.ToString()
	if o.Step != nil {
		s += " / " + o.Step.ToString()
	}
	return s
}

// Equal reports whether right is a range with equal bounds and effective step.
func (o *Range) Equal(right Object) bool {
	r, ok := right.(*Range)
	return ok && o.From.Equal(r.From) && o.To.Equal(r.To) &&
		o.effectiveStep().Equal(r.effectiveStep())
}

// IsFalsy always reports false: a range yields at least its From element.
func (o *Range) IsFalsy() bool { return false }

// withStep returns a copy of the range with the given step.
func (o *Range) withStep(step Object) *Range {
	return &Range{From: o.From, To: o.To, Step: step}
}

// ascending reports whether the range runs from a lower value up to a higher
// one (From <= To).
func (o *Range) ascending() bool { return rangeCmp(o.From, o.To) <= 0 }

// Iterate yields the range values; the entry key is the 0-based position and
// the value is the element.
func (o *Range) Iterate(_ *VM, _ *NamedArgs) Iterator {
	var (
		step = o.effectiveStep()
		asc  = o.ascending()
		cur  Object
		idx  int64
	)

	set := func(state *IteratorState) {
		if rangeReached(cur, o.To, asc) {
			state.Mode = IteratorStateModeDone
			return
		}
		state.Entry.K = Int(idx)
		state.Entry.V = cur
	}

	return NewIterator(
		func(_ *VM) (state *IteratorState, err error) {
			state = &IteratorState{}
			cur, idx = o.From, 0
			set(state)
			return
		},
		func(_ *VM, state *IteratorState) (err error) {
			next := rangeAdvance(cur, step, asc)
			// Guard against a non-advancing step (e.g. a zero or wrong-typed
			// step) so iteration always terminates.
			if rangeCmp(next, cur) == 0 {
				state.Mode = IteratorStateModeDone
				return
			}
			cur = next
			idx++
			set(state)
			return
		},
	).SetInput(o).SetItType(o.Type())
}

// IndexGet exposes the `from` and `to` fields and the `step` method. `r.step()`
// returns the current step; `r.step(n)` returns a new range with step n.
func (o *Range) IndexGet(_ *VM, index Object) (Object, error) {
	switch index.ToString() {
	case "from":
		return o.From, nil
	case "to":
		return o.To, nil
	case "step":
		return NewFunction("step", func(c Call) (Object, error) {
			if c.Args.Length() == 0 {
				return o.effectiveStep(), nil
			}
			return o.withStep(c.Args.GetOnly(0)), nil
		}), nil
	}
	return nil, ErrNotIndexable
}

// BinOpQuo handles `range / step` (ObjectWithQuoBinOperator), returning a new
// range with that step.
func (o *Range) BinOpQuo(_ *VM, right Object) (Object, error) {
	return o.withStep(right), nil
}

// NewRangeFunc is the Range(from, to; step) constructor body shared by the typed
// methods.
func NewRangeFunc(c Call) (Object, error) {
	r := &Range{From: c.Args.GetOnly(0), To: c.Args.GetOnly(1)}
	if step := c.NamedArgs.GetValueOrNil("step"); step != nil && step != Nil {
		r.Step = step
	}
	return r, nil
}

// --- per-kind arithmetic ----------------------------------------------------

func rangeInt64(o Object) int64 {
	switch t := o.(type) {
	case Int:
		return int64(t)
	case Uint:
		return int64(t)
	case Char:
		return int64(t)
	case Float:
		return int64(t)
	case Duration:
		return int64(t)
	}
	return 0
}

func rangeFloat64(o Object) float64 {
	switch t := o.(type) {
	case Float:
		return float64(t)
	case Int:
		return float64(t)
	case Uint:
		return float64(t)
	case Char:
		return float64(t)
	case Decimal:
		f, _ := decimal.Decimal(t).Float64()
		return f
	}
	return 0
}

func rangeDecimal(o Object) decimal.Decimal {
	switch t := o.(type) {
	case Decimal:
		return decimal.Decimal(t)
	case Int:
		return decimal.NewFromInt(int64(t))
	case Uint:
		return decimal.NewFromInt(int64(t))
	case Float:
		return decimal.NewFromFloat(float64(t))
	}
	return decimal.Zero
}

func cmpInt64(a, b int64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func cmpUint64(a, b uint64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// rangeCmp orders two values of the same element kind (-1, 0, 1).
func rangeCmp(a, b Object) int {
	switch ta := a.(type) {
	case Uint:
		if tb, ok := b.(Uint); ok {
			return cmpUint64(uint64(ta), uint64(tb))
		}
	case Float:
		fa, fb := float64(ta), rangeFloat64(b)
		switch {
		case fa < fb:
			return -1
		case fa > fb:
			return 1
		}
		return 0
	case Decimal:
		return decimal.Decimal(ta).Cmp(rangeDecimal(b))
	case *Time:
		if tb, ok := b.(*Time); ok {
			return ta.Value.Compare(tb.Value)
		}
	case CalendarDate:
		if tb, ok := b.(CalendarDate); ok {
			return cmpUint64(uint64(ta), uint64(tb)) // YYYYMMDD is monotonic
		}
	case CalendarTime:
		if tb, ok := b.(CalendarTime); ok {
			return cmpInt64(int64(ta), int64(tb)) // stored as UnixNano
		}
	}
	// Int, Char (and numeric fallbacks)
	return cmpInt64(rangeInt64(a), rangeInt64(b))
}

// rangeReached reports whether v has passed to in the iteration direction.
func rangeReached(v, to Object, asc bool) bool {
	if asc {
		return rangeCmp(v, to) > 0
	}
	return rangeCmp(v, to) < 0
}

// rangeAdvance returns v advanced by step in the iteration direction.
func rangeAdvance(v, step Object, asc bool) Object {
	var sign int64 = 1
	if !asc {
		sign = -1
	}
	switch t := v.(type) {
	case Int:
		return Int(int64(t) + sign*rangeInt64(step))
	case Uint:
		return Uint(uint64(int64(uint64(t)) + sign*rangeInt64(step)))
	case Char:
		return Char(int64(t) + sign*rangeInt64(step))
	case Float:
		return Float(float64(t) + float64(sign)*rangeFloat64(step))
	case Decimal:
		d := rangeDecimal(step)
		if !asc {
			d = d.Neg()
		}
		return Decimal(decimal.Decimal(t).Add(d))
	case *Time:
		return &Time{Value: t.Value.Add(time.Duration(sign) * time.Duration(rangeInt64(step)))}
	case CalendarTime:
		return CalendarTime(int64(t) + sign*rangeInt64(step))
	case CalendarDate:
		tm := t.Time(nil).Add(time.Duration(sign) * time.Duration(rangeInt64(step)))
		return CalendarDateFromTime(tm)
	}
	return v
}

func init() {
	// Register the typed `Range(from, to; step)` methods for each element kind,
	// mirroring `meth Range(from T, to T; step) { … }`. Numeric/char ranges take
	// a numeric step (default 1); temporal ranges take a duration step (default
	// one day).
	numeric := func(t ObjectType) *Function {
		return NewFunction("Range", NewRangeFunc,
			FunctionWithParams(func(p func(name string) *ParamBuilder) {
				p("from").Type(t)
				p("to").Type(t)
			}),
			FunctionWithNamedParams(func(np func(name string) *NamedParamBuilder) {
				np("step")
			}),
		)
	}
	temporal := func(t ObjectType) *Function {
		return NewFunction("Range", NewRangeFunc,
			FunctionWithParams(func(p func(name string) *ParamBuilder) {
				p("from").Type(t)
				p("to").Type(t)
			}),
			FunctionWithNamedParams(func(np func(name string) *NamedParamBuilder) {
				np("step").Type(DurationType)
			}),
		)
	}

	BuiltinObjects.AddMethod(BuiltinRange,
		numeric(TInt), numeric(TUint), numeric(TChar),
		numeric(TFloat), numeric(TDecimal))
	BuiltinObjects.AddMethod(BuiltinRange,
		temporal(TimeType), temporal(CalendarDateType), temporal(CalendarTimeType))
}
