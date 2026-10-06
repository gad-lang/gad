package gad

// The types of an interface's fields are compile-time symbols, read when a
// value is checked against it. A global, a builtin, a constant reads the same
// anywhere; but a local of the function declaring the interface — or a free
// variable of it — reads from that function's frame, gone once it returns:
//
//	mk := func() {
//		interface Icon { icon str|Icon; width? int }
//		return Icon
//	}
//	{icon: {icon: "x"}} :: mk() // Icon, a local of mk: no frame to read it
//
// As a class has itself in its define callback, a declared interface has
// itself: a field type naming it is bound to the interface; a field type
// naming a local or free variable is bound to its cell, as a closure captures
// it — InterfaceTypes(iface, keys, cell…), where it is declared. The
// interfaces its field types write (a group `{ … }`, an inline `interface
// { … }`), constants, are bound too: their types name the same frame.

// TypeKey is the key a field type symbol's cell is passed by to
// InterfaceTypes: its scope and its index.
func TypeKey(s *SymbolInfo) int64 { return int64(s.Scope)<<32 | int64(s.Index) }

// InterfaceTypesFunc is the `InterfaceTypes(iface, keys, cell…)` builtin a
// declared interface compiles to when a type of its fields names itself or a
// local or free variable: a copy of iface — and of the interfaces its fields
// write — whose fields have those types bound (InterfaceField.Bound): to the
// copy, by its name; to cells[i], by the TypeKey keys[i].
func InterfaceTypesFunc(c Call) (Object, error) {
	if err := c.Args.CheckMinLen(2); err != nil {
		return nil, err
	}
	src, ok := c.Args.Get(0).(*Interface)
	if !ok {
		return nil, NewArgumentTypeError("1st (interface)", "interface", c.Args.Get(0).Type().Name())
	}
	keys, ok := c.Args.Get(1).(Array)
	if !ok {
		return nil, NewArgumentTypeError("2nd (keys)", "array", c.Args.Get(1).Type().Name())
	}
	if n := c.Args.Length() - 2; n != len(keys) {
		return nil, ErrWrongNumArguments.NewErrorf("want %d cells, got %d", len(keys), n)
	}
	b := &ifaceTypesBinder{
		vm:    c.VM,
		self:  src.IName,
		cells: make(map[int64]Object, len(keys)),
		done:  map[*Interface]*Interface{},
	}
	for i, k := range keys {
		key, ok := k.(Int)
		if !ok {
			return nil, NewArgumentTypeError("2nd (keys)", "array of int", k.Type().Name())
		}
		b.cells[int64(key)] = c.Args.Get(i + 2)
	}
	return b.bind(src), nil
}

type ifaceTypesBinder struct {
	vm *VM
	// self is the name of the interface declared; root, its copy
	self  string
	root  *Interface
	cells map[int64]Object
	done  map[*Interface]*Interface
}

func (b *ifaceTypesBinder) bind(i *Interface) *Interface {
	if cp := b.done[i]; cp != nil {
		return cp
	}
	cp := *i
	cp.flat, cp.flatErr, cp.flatBuilt = nil, nil, false
	b.done[i] = &cp
	if b.root == nil {
		b.root = &cp
	}
	cp.Fields = make([]*InterfaceField, len(i.Fields))
	for k, f := range i.Fields {
		cp.Fields[k] = b.field(&cp, f)
	}
	if i.Elem != nil {
		cp.Elem = b.field(&cp, i.Elem)
	}
	return &cp
}

func (b *ifaceTypesBinder) field(iface *Interface, f *InterfaceField) *InterfaceField {
	cp := *f
	cp.Iface = iface
	for k, s := range f.TypesSymbols {
		var v Object
		switch s.Scope {
		case ScopeLocal, ScopeFree, ScopeGlobal:
			if s.Name == b.self {
				v = b.root
			} else if cell := b.cells[TypeKey(s)]; cell != nil {
				v = cell
			}
		case ScopeConstant:
			if b.vm != nil {
				if nested, _ := b.vm.constants[s.Index].(*Interface); nested != nil {
					v = b.bind(nested)
				}
			}
		}
		if v != nil {
			if cp.Bound == nil {
				cp.Bound = make([]Object, len(f.TypesSymbols))
			}
			cp.Bound[k] = v
		}
	}
	return &cp
}

// typeValue is the value of the field's k-th type symbol: what it is bound
// to, where the interface was declared (a cell read now); else the symbol
// read in the current frame.
func (f *InterfaceField) typeValue(vm *VM, k int) (Object, error) {
	if k < len(f.Bound) {
		switch v := f.Bound[k].(type) {
		case nil:
		case *ObjectPtr:
			// a cell taken before its variable was defined (a type declared
			// after the interface) is left behind by the definition: the
			// frame's, while it runs
			if *v.Value != nil {
				return *v.Value, nil
			}
		default:
			return v, nil
		}
	}
	return vm.GetSymbolValue(f.TypesSymbols[k])
}
