package gad

import (
	"strconv"
	"strings"
)

// An array type is a constructor (issue #18): `(type [2]str)("a", "b")` builds
// a value from its items, checked as `::` checks them; `(type [2][3]int)(;
// fill=0)` builds one by its dimensions — `size=[…]` for the levels with no
// exact size, `fill` each leaf (a value, or a function of the indexes of the
// leaf). `array` is `type []any`, and `arrayType(elem; depth, size)` — the
// type of the array types — builds one at run time.

// hasDimsArgs reports whether na has `size` or `fill`: a build by dimensions.
func hasDimsArgs(na *NamedArgs) bool {
	return na.GetPassedValue("size") != nil || na.GetPassedValue("fill") != nil
}

// arrayDimsArgs are the `size` and `fill` of c, no other named argument.
func arrayDimsArgs(c *Call) (size, fill Object, hasFill bool, err error) {
	if err = c.NamedArgs.CheckNames("size", "fill"); err != nil {
		return
	}
	if v := c.NamedArgs.GetPassedValue("size"); v != nil && !IsReset(v) {
		size = v
	}
	if v := c.NamedArgs.GetPassedValue("fill"); v != nil && !IsReset(v) {
		fill, hasFill = v, true
	}
	return
}

// dimsSizes are the sizes of the depth levels of a build: the exact sizes of
// the type (bounds), and size — one int a level, or one a level with no exact
// size —, each in the range of its level.
func dimsSizes(name string, depth int, bounds []arrayBound, size Object) ([]int, error) {
	var given []int
	switch s := size.(type) {
	case nil:
	case Int:
		given = []int{int(s)}
	case Uint:
		given = []int{int(s)}
	case Array:
		for i, v := range s {
			var n int64
			switch x := v.(type) {
			case Int:
				n = int64(x)
			case Uint:
				n = int64(x)
			default:
				return nil, ErrType.NewErrorf("%s: size item %d: %s is not an int", name, i, v.Type().Name())
			}
			if n < 0 {
				return nil, ErrType.NewErrorf("%s: size item %d is negative: %d", name, i, n)
			}
			given = append(given, int(n))
		}
	default:
		return nil, ErrType.NewErrorf("%s: size: an array of ints, not %s", name, size.Type().Name())
	}
	bound := func(l int) arrayBound {
		if l < len(bounds) {
			return bounds[l]
		}
		return arrayBound{}
	}
	sizes := make([]int, depth)
	if len(given) == depth {
		for l, n := range given {
			if b := bound(l); !b.accepts(n) {
				return nil, ErrType.NewErrorf("%s: level %d: size %d, wants %s", name, l+1, n, b.want())
			}
			sizes[l] = n
		}
		return sizes, nil
	}
	var open []int
	for l := range sizes {
		if b := bound(l); b.hasEq {
			sizes[l] = int(b.eq)
		} else {
			open = append(open, l)
		}
	}
	if len(given) != len(open) {
		if len(given) == 0 {
			return nil, ErrType.NewErrorf("%s: level %d has no size: give it with size=[…]", name, open[0]+1)
		}
		if len(open) == depth {
			return nil, ErrType.NewErrorf("%s: %d sizes, wants %d (one a level)", name, len(given), depth)
		}
		return nil, ErrType.NewErrorf("%s: %d sizes, wants %d (one a level) or %d (one a level with no size)",
			name, len(given), depth, len(open))
	}
	for i, l := range open {
		if b := bound(l); !b.accepts(given[i]) {
			return nil, ErrType.NewErrorf("%s: level %d: size %d, wants %s", name, l+1, given[i], b.want())
		}
		sizes[l] = given[i]
	}
	return sizes, nil
}

// buildArrayDims is an array of depth levels of the sizes dimsSizes says, each
// leaf fill — a function called with the indexes of the leaf, else the value —,
// accepted by leaf (nil: any). With no fill a leaf is nil, when leaf takes it.
func buildArrayDims(vm *VM, name string, depth int, bounds []arrayBound, size, fill Object, hasFill bool,
	leaf func(Object) (bool, error)) (Array, error) {
	sizes, err := dimsSizes(name, depth, bounds, size)
	if err != nil {
		return nil, err
	}
	fn := hasFill && IsFunction(fill)
	if !hasFill {
		fill = Nil
	}
	idx := make(Array, depth)
	var walk func(level int) (Object, error)
	walk = func(level int) (Object, error) {
		if level == depth {
			v := fill
			if fn {
				if vm == nil {
					return nil, ErrType.NewErrorf("%s: fill: a function needs a VM", name)
				}
				var err error
				if v, err = vm.Call(fill, Args{append(Array(nil), idx...)}, nil); err != nil {
					return nil, err
				}
			}
			if leaf != nil {
				ok, err := leaf(v)
				if err != nil {
					return nil, err
				}
				if !ok {
					if !hasFill {
						return nil, ErrType.NewErrorf("%s: fill required: the element type does not accept nil", name)
					}
					return nil, ErrType.NewErrorf("%s: fill: %s is not of the element type", name, ReprQuote(v.Type().Name()))
				}
			}
			return v, nil
		}
		out := make(Array, sizes[level])
		for i := range out {
			idx[level] = Int(i)
			var err error
			if out[i], err = walk(level + 1); err != nil {
				return nil, err
			}
		}
		return out, nil
	}
	v, err := walk(0)
	if err != nil {
		return nil, err
	}
	return v.(Array), nil
}

// CanCall reports that an array type is a constructor.
func (t *ArrayType) CanCall() bool { return true }

// Call builds a value of the array type: from its items (`(type [2]str)("a",
// "b")`), checked — not converted: `::>` converts —, or by its dimensions
// (`(type [2][3]int)(; fill=0)`, `(type [][]int)(; size=[2, 3], fill=0)`).
// The value is a plain array: an array type is structural, not bound to it.
func (t *ArrayType) Call(c Call) (Object, error) {
	size, fill, hasFill, err := arrayDimsArgs(&c)
	if err != nil {
		return nil, err
	}
	if c.Args.Length() > 0 || (size == nil && !hasFill) {
		if size != nil || hasFill {
			return nil, ErrType.NewErrorf("%s: items, or size and fill: not both", t.Name())
		}
		items := c.Args.CopyValues()
		if _, err := t.AssignTo(c.VM, items, t); err != nil {
			if why := t.leafMismatch(c.VM, items); why != "" {
				return nil, ErrIncompatibleAssign.NewErrorf("%s: %s", t.Name(), why)
			}
			return nil, err
		}
		return items, nil
	}
	bounds, err := t.bounds(c.VM)
	if err != nil {
		return nil, err
	}
	var leaf func(Object) (bool, error)
	if len(t.Elem) > 0 {
		resolve := elemResolver(c.VM, t.Elem, t.Bound)
		leaf = func(v Object) (bool, error) { return t.Elem.AcceptResolve(c.VM, v, resolve) }
	}
	return buildArrayDims(c.VM, t.Name(), t.Depth, bounds, size, fill, hasFill, leaf)
}

// leafMismatch says which leaf of obj is not of the element types ("item 1.0:
// ‹str› is not of the element type"); "" when none.
func (t *ArrayType) leafMismatch(vm *VM, obj Object) string {
	if vm == nil || len(t.Elem) == 0 {
		return ""
	}
	resolve := elemResolver(vm, t.Elem, t.Bound)
	var walk func(o Object, depth int, path string) string
	walk = func(o Object, depth int, path string) string {
		if depth == 0 {
			if ok, err := t.Elem.AcceptResolve(vm, o, resolve); err == nil && !ok {
				return "item " + path + ": " + ReprQuote(o.Type().Name()) + " is not of the element type"
			}
			return ""
		}
		arr, ok := o.(Array)
		if !ok {
			if path == "" {
				return ""
			}
			return "item " + path + ": " + ReprQuote(o.Type().Name()) + " is not an array"
		}
		for i, item := range arr {
			p := strconv.Itoa(i)
			if path != "" {
				p = path + "." + p
			}
			if why := walk(item, depth-1, p); why != "" {
				return why
			}
		}
		return ""
	}
	return walk(obj, t.Depth, "")
}

// dimsArgs reports whether the named args of a call of t are a build by
// dimensions: `size` or `fill`, not fields of t.
func (t *TypedArrayType) dimsArgs(na *NamedArgs) bool {
	if !hasDimsArgs(na) {
		return false
	}
	if t.members != nil {
		if _, ok := t.members.fieldsMap["size"]; ok {
			return false
		}
		if _, ok := t.members.fieldsMap["fill"]; ok {
			return false
		}
	}
	return true
}

// buildDims is `grid(; fill=0)`: the items of t by its dimensions, each leaf
// of its element type, then built as by its items.
func (t *TypedArrayType) buildDims(c Call) (Object, error) {
	size, fill, hasFill, err := arrayDimsArgs(&c)
	if err != nil {
		return nil, err
	}
	var bounds []arrayBound
	if t.Sizes != nil {
		if bounds, err = t.Sizes.bounds(c.VM); err != nil {
			return nil, err
		}
	}
	leaf := func(v Object) (bool, error) {
		if t.Elem == nil {
			return true, nil
		}
		_, err := AssignToType(c.VM, v, t.Elem)
		return err == nil, nil
	}
	items, err := buildArrayDims(c.VM, t.TypeName, t.Depth, bounds, size, fill, hasFill, leaf)
	if err != nil {
		return nil, err
	}
	return t.build(c.VM, items, nil)
}

// anyArrayReflect are the keys `array` — `type []any` — answers as an array
// type: `@elem` ([any]), `@depth` (1), `@size` (no size).
func anyArrayReflect(index Object) (Object, bool) {
	switch index.ToString() {
	case "@elem":
		return Array{TAny}, true
	case "@depth":
		return Int(1), true
	case "@size":
		return Array{newArraySize(arrayBound{})}, true
	}
	return nil, false
}

// NewArrayTypeFunc is the constructor of `arrayType`: `arrayType(elem; depth,
// size)` the array type of elem — a type, or an array of types (a union) —,
// depth levels deep (1 when not given; the length of size when size is),
// each level of its size: an int (exactly), an [eq, min, max] (as `@size`
// says it, nil where none) or nil. `arrayType(any)` is `array`.
func NewArrayTypeFunc(c Call) (Object, error) {
	if err := c.Args.CheckLen(1); err != nil {
		return nil, err
	}
	if err := c.NamedArgs.CheckNames("depth", "size"); err != nil {
		return nil, err
	}
	var elems Array
	switch e := c.Args.Get(0).(type) {
	case Array:
		elems = e
	default:
		elems = Array{e}
	}
	if len(elems) == 0 {
		return nil, ErrType.NewError("arrayType: no element type")
	}
	t := &ArrayType{Depth: 1}
	for i, e := range elems {
		if _, ok := e.(TypeAssigner); !ok {
			return nil, ErrType.NewErrorf("arrayType: element %d: %s is not a type", i, ReprQuote(e.Type().Name()))
		}
		name := e.ToString()
		if n, ok := e.(interface{ Name() string }); ok {
			name = n.Name()
		}
		t.Elem = append(t.Elem, &SymbolInfo{Name: name, Index: -1, Scope: ScopeLocal})
		t.Bound = append(t.Bound, e)
		t.ElemNames = append(t.ElemNames, name)
	}
	var levels Array
	if v := c.NamedArgs.GetPassedValue("size"); v != nil && !IsReset(v) {
		a, ok := v.(Array)
		if !ok {
			return nil, ErrType.NewErrorf("arrayType: size: an array, not %s", v.Type().Name())
		}
		levels = a
		t.Depth = len(a)
	}
	if v := c.NamedArgs.GetPassedValue("depth"); v != nil && !IsReset(v) {
		d, ok := v.(Int)
		if !ok || d < 1 {
			return nil, ErrType.NewErrorf("arrayType: depth: an int from 1, not %s", v.ToString())
		}
		if levels != nil && int(d) != len(levels) {
			return nil, ErrType.NewErrorf("arrayType: depth %d, but %d sizes", d, len(levels))
		}
		t.Depth = int(d)
	}
	if t.Depth == 0 {
		return nil, ErrType.NewError("arrayType: depth 0")
	}
	if err := t.setLevels(levels); err != nil {
		return nil, err
	}
	// `[]any` is `array`
	if t.Depth == 1 && t.Levels == nil && len(elems) == 1 && sameType(elems[0], TAny) {
		if c.VM != nil {
			return c.VM.Builtins.builtins.Objects[BuiltinArray], nil
		}
		return BuiltinObjects[BuiltinArray], nil
	}
	return t, nil
}

// setLevels sets the sizes of the levels from levels (an item a level).
func (t *ArrayType) setLevels(levels Array) error {
	var any bool
	ls := make([]ArrayLevel, t.Depth)
	ref := func(level int, v Object) (ArraySizeRef, error) {
		switch x := v.(type) {
		case nil, *NilType:
			return ArraySizeRef{}, nil
		case Int:
			if x >= 0 {
				return ArraySizeRef{Set: true, N: int64(x), Sym: -1}, nil
			}
		case Uint:
			return ArraySizeRef{Set: true, N: int64(x), Sym: -1}, nil
		}
		return ArraySizeRef{}, ErrType.NewErrorf("arrayType: level %d: a size is an int, not %s", level+1, v.ToString())
	}
	for i, v := range levels {
		var err error
		l := &ls[i]
		switch x := v.(type) {
		case *ArraySize:
			v = x.Array()
		}
		switch x := v.(type) {
		case Array:
			if len(x) != 3 {
				return ErrType.NewErrorf("arrayType: level %d: [eq, min, max], not %d items", i+1, len(x))
			}
			if l.Eq, err = ref(i, x[0]); err != nil {
				return err
			}
			if l.Min, err = ref(i, x[1]); err != nil {
				return err
			}
			if l.Max, err = ref(i, x[2]); err != nil {
				return err
			}
			if l.Min.Set && l.Max.Set && l.Min.N > l.Max.N {
				return ErrType.NewErrorf("arrayType: level %d: the minimum %d is greater than the maximum %d", i+1, l.Min.N, l.Max.N)
			}
		default:
			if l.Eq, err = ref(i, v); err != nil {
				return err
			}
		}
		any = any || l.Eq.Set || l.Min.Set || l.Max.Set
	}
	if !any {
		return nil
	}
	t.Levels = ls
	var b strings.Builder
	n := func(r ArraySizeRef) string {
		if !r.Set {
			return ""
		}
		return strconv.FormatInt(r.N, 10)
	}
	for _, l := range ls {
		b.WriteByte('[')
		switch {
		case l.Eq.Set:
			b.WriteString(n(l.Eq))
		case l.Min.Set || l.Max.Set:
			b.WriteString(n(l.Min) + ".." + n(l.Max))
		}
		b.WriteByte(']')
	}
	t.LevelsText = b.String()
	return nil
}

// sameLevels reports whether t and r have the same sizes, level by level (a
// size naming a variable: the same variable).
func (t *ArrayType) sameLevels(r *ArrayType) bool {
	if len(t.Levels) == 0 || len(r.Levels) == 0 {
		return len(t.Levels) == len(r.Levels) || t.LevelsText == r.LevelsText
	}
	if len(t.Levels) != len(r.Levels) {
		return false
	}
	same := func(a, b ArraySizeRef) bool {
		if a.Set != b.Set {
			return false
		}
		if !a.Set {
			return true
		}
		if a.Sym >= 0 || b.Sym >= 0 {
			return a.Sym >= 0 && b.Sym >= 0 && a.Sym < len(t.SizeSyms) && b.Sym < len(r.SizeSyms) &&
				sameSymbol(t.SizeSyms[a.Sym], r.SizeSyms[b.Sym])
		}
		return a.N == b.N
	}
	for i, l := range t.Levels {
		o := r.Levels[i]
		if !same(l.Eq, o.Eq) || !same(l.Min, o.Min) || !same(l.Max, o.Max) {
			return false
		}
	}
	return true
}

// sameElems reports whether t and r have the same element types: the same
// objects when both are known without a VM (a builtin, a type given to
// arrayType), else the same symbols, else the same names.
func (t *ArrayType) sameElems(r *ArrayType) bool {
	if len(t.Elem) != len(r.Elem) {
		return false
	}
	for i := range t.Elem {
		a, aok := t.elemObject(i)
		b, bok := r.elemObject(i)
		switch {
		case aok && bok:
			if !sameType(a, b) {
				return false
			}
		case !aok && !bok && sameSymbol(t.Elem[i], r.Elem[i]):
		default:
			if t.ElemNames[i] != r.ElemNames[i] {
				return false
			}
		}
	}
	return true
}

// elemObject is the element type i when it is known without a VM.
func (t *ArrayType) elemObject(i int) (Object, bool) {
	if i < len(t.Bound) && t.Bound[i] != nil {
		if p, ok := t.Bound[i].(*ObjectPtr); ok {
			if *p.Value != nil {
				return *p.Value, true
			}
		} else {
			return t.Bound[i], true
		}
	}
	if s := t.Elem[i]; s.Scope == ScopeBuiltin {
		if o, ok := BuiltinObjects[BuiltinType(s.Index)]; ok && o != nil {
			return o, true
		}
	}
	return nil, false
}

// sameType reports whether a and b are the same type: a builtin type by its
// key (a VM has its own copies of the builtins), else Equal.
func sameType(a, b Object) bool {
	key := func(o Object) (BuiltinType, bool) {
		switch t := o.(type) {
		case *BuiltinObjType:
			return t.builtinType, true
		case BuiltinObjTypeKey:
			return BuiltinType(t), true
		}
		if o == TAny {
			return BuiltinAny, true
		}
		return 0, false
	}
	if ka, ok := key(a); ok {
		kb, ok := key(b)
		return ok && ka == kb
	}
	return a.Equal(b)
}

func sameSymbol(a, b *SymbolInfo) bool {
	return a == b || (a.Scope == b.Scope && a.Index == b.Index && a.Name == b.Name)
}
