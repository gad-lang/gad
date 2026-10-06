package gad

// The element types of a structural type written as a value — `[]Size`,
// `*Item` — are symbols, resolved when a value is checked against it. A local
// or free variable of the function that writes it reads from that function's
// frame, gone once it returns:
//
//	mk := func() {
//		enum Size { S, M }
//		class Form { sizes []Size } // the type []Size: Size, a local of mk
//		return Form
//	}
//
// As an interface's field types are (objects_interface_types.go), they are
// bound where the type is written: TypeCells(type, keys, cell…), a copy whose
// elements read the cells — as a closure captures them.

// TypeCells is the `TypeCells(type, keys, cell…)` builtin a structural type
// (an *ArrayType, a *PtrType) compiles to when its element types name a local
// or free variable: a copy with them bound to the cells — cells[i] by the
// TypeKey keys[i].
func TypeCellsFunc(c Call) (Object, error) {
	if err := c.Args.CheckMinLen(2); err != nil {
		return nil, err
	}
	keys, ok := c.Args.Get(1).(Array)
	if !ok {
		return nil, NewArgumentTypeError("2nd (keys)", "array", c.Args.Get(1).Type().Name())
	}
	if n := c.Args.Length() - 2; n != len(keys) {
		return nil, ErrWrongNumArguments.NewErrorf("want %d cells, got %d", len(keys), n)
	}
	cells := make(map[int64]Object, len(keys))
	for i, k := range keys {
		key, ok := k.(Int)
		if !ok {
			return nil, NewArgumentTypeError("2nd (keys)", "array of int", k.Type().Name())
		}
		cells[int64(key)] = c.Args.Get(i + 2)
	}
	switch t := c.Args.Get(0).(type) {
	case *ArrayType:
		cp := *t
		cp.Bound = bindCells(t.Elem, cells)
		return &cp, nil
	case *PtrType:
		cp := *t
		cp.Bound = bindCells(t.Elem, cells)
		return &cp, nil
	}
	return c.Args.Get(0), nil
}

// bindCells are the cells of the symbols of elem, by their index (nil: not
// a variable's).
func bindCells(elem ParamType, cells map[int64]Object) []Object {
	var bound []Object
	for i, s := range elem {
		if s.Scope != ScopeLocal && s.Scope != ScopeFree {
			continue
		}
		if cell := cells[TypeKey(s)]; cell != nil {
			if bound == nil {
				bound = make([]Object, len(elem))
			}
			bound[i] = cell
		}
	}
	return bound
}

// elemResolver resolves the symbols of elem: what bound has of them (a cell
// read now), else the symbol in the current frame.
func elemResolver(vm *VM, elem ParamType, bound []Object) func(*SymbolInfo) (Object, error) {
	return func(s *SymbolInfo) (Object, error) {
		for i, e := range elem {
			if e != s || i >= len(bound) || bound[i] == nil {
				continue
			}
			if p, ok := bound[i].(*ObjectPtr); ok {
				// a cell taken before its variable was defined is left
				// behind by the definition: the frame's, while it runs
				if *p.Value != nil {
					return *p.Value, nil
				}
				break
			}
			return bound[i], nil
		}
		return vm.GetSymbolValue(s)
	}
}

// structuralCells are the local and free variables the element types of a
// structural type constant name: their keys, their symbols; nil when none.
func structuralCells(obj Object) (keys Array, cells []*SymbolInfo) {
	var elem ParamType
	switch t := obj.(type) {
	case *ArrayType:
		elem = t.Elem
	case *PtrType:
		elem = t.Elem
	default:
		return nil, nil
	}
	seen := map[int64]bool{}
	for _, s := range elem {
		if (s.Scope == ScopeLocal || s.Scope == ScopeFree) && !seen[TypeKey(s)] {
			seen[TypeKey(s)] = true
			keys = append(keys, Int(TypeKey(s)))
			cells = append(cells, s)
		}
	}
	return
}
