package gad

import (
	"strconv"
	"strings"
)

// ArrayType is an array type written where a type goes: `[]int`, `[][]str`,
// `[]<int|str>`. It accepts an ARRAY nested to Depth whose leaf elements are
// each accepted by one of Elem — the same rule a parameter's declared types
// follow, because Elem is exactly that: the element's allowed types, as the
// symbols the compiler resolved them to.
//
// `[]int` is the short form of `[]<int>`: one type needs no envelope.
type ArrayType struct {
	// Depth is how many `[]` were written: 1 for `[]int`, 3 for `[][][]int`.
	Depth int
	// Elem are the types an element may be, as symbols resolved per VM — so an
	// element type may be anything a parameter type may be, a class or an
	// interface included.
	Elem ParamType
	// ElemNames are those types as written, for the type's own name.
	ElemNames []string
	// Bound holds, by the index of Elem, the cells of the variables they
	// name, bound where the type was written (TypeCellsFunc).
	Bound []Object
	// Levels are the sizes of the levels, by level (`[2]`, `[..3]`, `[2..]`,
	// `[2..5]`); nil when no level has one.
	Levels []ArrayLevel
	// LevelsText is the levels as written (`[][2]`, `[n..]`); "" when none
	// has a size.
	LevelsText string
	// SizeSyms are the variables the sizes name, as symbols — resolved when a
	// value is checked, as the element types are —, and SizeBound their cells
	// bound where the type was written (TypeCellsFunc), by index.
	SizeSyms  ParamType
	SizeBound []Object
}

var (
	_ IndexGetter   = (*ArrayType)(nil)
	_ Object        = (*ArrayType)(nil)
	_ TypeAssigner  = (*ArrayType)(nil)
	_ vmCanAssigner = (*ArrayType)(nil)
)

// An array type is STRUCTURAL, not an ObjectType: it answers about the shape of
// a value (an array nested so deep, of such elements), and there is no type
// chain to walk to it.
func (t *ArrayType) Type() ObjectType { return TArray }

func (t *ArrayType) Name() string {
	var b strings.Builder
	if t.LevelsText != "" {
		b.WriteString(t.LevelsText)
	} else {
		for i := 0; i < t.Depth; i++ {
			b.WriteString("[]")
		}
	}
	if len(t.ElemNames) == 1 {
		b.WriteString(t.ElemNames[0])
	} else {
		b.WriteString("<" + strings.Join(t.ElemNames, "|") + ">")
	}
	return b.String()
}

func (t *ArrayType) ToString() string { return t.Name() }
func (t *ArrayType) String() string   { return t.Name() }
func (t *ArrayType) IsFalsy() bool    { return false }

func (t *ArrayType) Equal(right Object) bool {
	r, ok := right.(*ArrayType)
	return ok && r == t
}

// AssignTo is how a structural type is asked "does this value satisfy you":
// the caller passes the receiver itself as `to` (see ifaceFieldTypeOK), so the
// answer is the check, not an identity test.
func (t *ArrayType) AssignTo(vm *VM, obj Object, to TypeAssigner) (Object, error) {
	if to != TypeAssigner(t) {
		return nil, ErrIncompatibleAssign
	}
	ok, err := t.CanAssignVM(vm, obj)
	if err != nil {
		return nil, err
	}
	if !ok {
		if t.sizeMismatch(vm, obj) != "" {
			return nil, incompatibleAssign(vm, obj, t)
		}
		return nil, ErrIncompatibleAssign
	}
	return obj, nil
}

// CanAssign without a VM cannot resolve the element types, so it only answers
// the part it can see: obj must be an array nested to Depth. CanAssignVM is the
// real check, and ParamType.Accept prefers it.
func (t *ArrayType) CanAssign(obj Object) (bool, error) {
	bounds, err := t.bounds(nil)
	if err != nil {
		return false, err
	}
	return t.nested(obj, t.Depth, bounds, nil)
}

// CanAssignVM reports whether obj is an array nested to Depth, each level of
// its size, whose leaves are each accepted by one of the element types.
func (t *ArrayType) CanAssignVM(vm *VM, obj Object) (bool, error) {
	bounds, err := t.bounds(vm)
	if err != nil {
		return false, err
	}
	return t.nested(obj, t.Depth, bounds, vm)
}

// nested walks depth levels of array — the number of items of each checked
// against its level's size first —, checking the leaves when a VM is at hand
// to resolve the element types.
func (t *ArrayType) nested(obj Object, depth int, bounds []arrayBound, vm *VM) (bool, error) {
	if depth == 0 {
		if vm == nil || len(t.Elem) == 0 {
			return true, nil
		}
		return t.Elem.AcceptResolve(vm, obj, elemResolver(vm, t.Elem, t.Bound))
	}

	arr, ok := obj.(Array)
	if !ok {
		return false, nil
	}
	if level := t.Depth - depth; level < len(bounds) && !bounds[level].accepts(len(arr)) {
		return false, nil
	}
	for _, item := range arr {
		ok, err := t.nested(item, depth-1, bounds, vm)
		if err != nil || !ok {
			return ok, err
		}
	}
	return true, nil
}

// bounds are the sizes of the levels resolved: their variables read now (a
// cell bound where the type was written, else the symbol in the frame); nil
// when no level has a size. Without a VM a level naming a variable is not
// checked (its bound is left out).
func (t *ArrayType) bounds(vm *VM) ([]arrayBound, error) {
	if len(t.Levels) == 0 {
		return nil, nil
	}
	out := make([]arrayBound, len(t.Levels))
	for i, l := range t.Levels {
		b := &out[i]
		var err error
		if b.eq, b.hasEq, err = t.sizeOf(vm, l.Eq); err != nil {
			return nil, err
		}
		if b.min, b.hasMin, err = t.sizeOf(vm, l.Min); err != nil {
			return nil, err
		}
		if b.max, b.hasMax, err = t.sizeOf(vm, l.Max); err != nil {
			return nil, err
		}
		if b.hasMin && b.hasMax && b.min > b.max {
			return nil, ErrType.NewErrorf("array type %s: level %d: the minimum %d is greater than the maximum %d",
				t.Name(), i+1, b.min, b.max)
		}
	}
	return out, nil
}

// sizeOf is a bound of a level: its number, or the value of its variable — an
// int, not negative.
func (t *ArrayType) sizeOf(vm *VM, r ArraySizeRef) (int64, bool, error) {
	if !r.Set {
		return 0, false, nil
	}
	if r.Sym < 0 {
		return r.N, true, nil
	}
	if vm == nil || r.Sym >= len(t.SizeSyms) {
		return 0, false, nil
	}
	s := t.SizeSyms[r.Sym]
	v, err := elemResolver(vm, t.SizeSyms, t.SizeBound)(s)
	if err != nil {
		return 0, false, err
	}
	var n int64
	switch x := v.(type) {
	case Int:
		n = int64(x)
	case Uint:
		n = int64(x)
	default:
		return 0, false, ErrType.NewErrorf("array type %s: the size %s is not an int: %s", t.Name(), s.Name, v.Type().Name())
	}
	if n < 0 {
		return 0, false, ErrType.NewErrorf("array type %s: the size %s is negative: %d", t.Name(), s.Name, n)
	}
	return n, true, nil
}

// sizeMismatch says which level of obj is not of its size ("level 1 has 3
// items, wants exactly 2"); "" when the sizes are not why obj is refused.
func (t *ArrayType) sizeMismatch(vm *VM, obj Object) string {
	bounds, err := t.bounds(vm)
	if err != nil || len(bounds) == 0 {
		return ""
	}
	return t.mismatchFrom(bounds, obj, 0)
}

// mismatchFrom is sizeMismatch of o at level (its levels from there on).
func (t *ArrayType) mismatchFrom(bounds []arrayBound, o Object, level int) string {
	arr, ok := o.(Array)
	if !ok || level >= t.Depth {
		return ""
	}
	if level < len(bounds) && !bounds[level].accepts(len(arr)) {
		return "level " + strconv.Itoa(level+1) + " has " + strconv.Itoa(len(arr)) + " items, wants " + bounds[level].want()
	}
	for _, item := range arr {
		if why := t.mismatchFrom(bounds, item, level+1); why != "" {
			return why
		}
	}
	return ""
}

// IndexGet reflects the array type, with the same keys an array-of-types
// interface answers to:
//
//   - `@depth` — how many `[]` were written;
//   - `@elem`  — the types an element may be, resolved in vm (the element types
//     are symbols, so without a VM there is nothing to resolve them to).
func (t *ArrayType) IndexGet(vm *VM, index Object) (Object, error) {
	switch index.ToString() {
	case "@depth":
		return Int(t.Depth), nil
	case "@size":
		return t.sizeArray(vm)
	case "@elem":
		out := make(Array, 0, len(t.Elem))
		for _, s := range t.Elem {
			if vm == nil {
				return nil, ErrInvalidIndex.NewError("@elem (no VM to resolve the element types)")
			}
			v, err := elemResolver(vm, t.Elem, t.Bound)(s)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	}
	return nil, ErrInvalidIndex.NewError(index.ToString())
}

// sizeArray is `@size`: an ArraySize a level ([eq, min, max], nil where none).
func (t *ArrayType) sizeArray(vm *VM) (Object, error) {
	bounds, err := t.bounds(vm)
	if err != nil {
		return nil, err
	}
	out := make(Array, t.Depth)
	for i := range out {
		var b arrayBound
		if i < len(bounds) {
			b = bounds[i]
		}
		out[i] = newArraySize(b)
	}
	return out, nil
}

// transform is `obj ::> t`: obj an array nested to Depth, each level of its
// size, with each leaf converted — kept when an element type accepts it, else
// transformed into the first element type that takes it.
func (t *ArrayType) transform(vm *VM, obj Object) (Object, error) {
	bounds, err := t.bounds(vm)
	if err != nil {
		return nil, err
	}
	resolve := elemResolver(vm, t.Elem, t.Bound)
	var walk func(o Object, level int) (Object, error)
	walk = func(o Object, level int) (Object, error) {
		if level == t.Depth {
			return t.transformLeaf(vm, o, resolve)
		}
		arr, ok := typedArraySource(o)
		if !ok {
			return nil, ErrIncompatibleAssign.NewErrorf("%s is not assignable to %s: level %d is not an array",
				ReprQuote(o.Type().Name()), ReprQuote(t.Name()), level+1)
		}
		if level < len(bounds) && !bounds[level].accepts(len(arr)) {
			return nil, ErrIncompatibleAssign.NewErrorf("%s is not assignable to %s: level %d has %d items, wants %s",
				ReprQuote(obj.Type().Name()), ReprQuote(t.Name()), level+1, len(arr), bounds[level].want())
		}
		out := make(Array, len(arr))
		for i, item := range arr {
			if out[i], err = walk(item, level+1); err != nil {
				return nil, err
			}
		}
		return out, nil
	}
	return walk(obj, 0)
}

// transformLeaf is a leaf of transform: v when an element type accepts it,
// else v transformed into the first that takes it (the error of the first
// when none does).
func (t *ArrayType) transformLeaf(vm *VM, v Object, resolve func(*SymbolInfo) (Object, error)) (Object, error) {
	if len(t.Elem) == 0 {
		return v, nil
	}
	if ok, err := t.Elem.AcceptResolve(vm, v, resolve); err != nil || ok {
		return v, err
	}
	var first error
	for _, s := range t.Elem {
		et, err := resolve(s)
		if err != nil {
			return nil, err
		}
		out, err := AssignToTypeTransform(vm, v, et)
		if err == nil {
			return out, nil
		}
		if first == nil {
			first = err
		}
	}
	return nil, first
}
