package gad

import "strings"

// gad:doc
// ## Type Interface
// Interface is the value of an `interface { … }` declaration: a structural
// contract of typed fields, getter/setter properties, required methods and an
// optional `parse { … }` group of signatures. It is compiled to a bytecode
// constant; parameter/field types are stored as symbols and resolved per-VM.
//
// Members are read with indexing:
//   - `i.name`    -> str
//   - `i.fields`  -> array of InterfaceField
//   - `i.props`   -> array of InterfaceProp
//   - `i.methods` -> array of InterfaceMethod

// TInterface is the builtin `Interface` object type. It has no constructor.
var TInterface = RegisterBuiltinType(BuiltinInterface, "Interface", Interface{}, nil)

// Object types for the interface members. They are internal representations
// carried inside an Interface constant, not user-constructible.
var (
	TInterfaceField  = NewBuiltinObjType("InterfaceField")
	TInterfaceProp   = NewBuiltinObjType("InterfaceProp")
	TInterfaceMethod = NewBuiltinObjType("InterfaceMethod")
)

var (
	_ IndexGetter = (*Interface)(nil)
	_ IndexGetter = (*InterfaceField)(nil)
	_ IndexGetter = (*InterfaceProp)(nil)
	_ IndexGetter = (*InterfaceMethod)(nil)
)

// Interface is the value of an `interface { … }` (see TInterface).
type Interface struct {
	IName   string
	Module  *ModuleSpec       // module the interface was compiled in (for FullName)
	Extends ParamType         // parent interface symbol refs (from `extends { … }`)
	Fields  []*InterfaceField // typed fields
	Props   []*InterfaceProp  // getter/setter properties
	Methods []*InterfaceMethod
}

// InterfaceField is a typed field of an interface (see gad.Param for the type
// symbol/ObjectType split).
type InterfaceField struct {
	Iface        *Interface
	Name         string
	TypesSymbols ParamType   // compile-time type symbols
	Types        ObjectTypes // resolved types (when built at run time)
}

// InterfaceProp is a getter and/or setter property of an interface.
type InterfaceProp struct {
	Iface   *Interface
	Name    string
	Getter  *FuncHeaderObject   // the getter signature, or nil
	Setters []*FuncHeaderObject // the setter signatures
}

// InterfaceMethod is a required method of an interface: a name and its overload
// signatures (like a MethodInterface).
type InterfaceMethod struct {
	Iface   *Interface
	Name    string
	Headers []*FuncHeaderObject
}

// --- Interface ---

func (i *Interface) Type() ObjectType { return TInterface }

// AssignTo makes *Interface a TypeAssigner: obj is assignable to the interface
// `to` when it structurally satisfies it (see CanAssignVM).
func (i *Interface) AssignTo(vm *VM, obj Object, to TypeAssigner) (Object, error) {
	if ti, _ := to.(*Interface); ti != nil {
		if ok, err := ti.CanAssignVM(vm, obj); err != nil {
			return nil, err
		} else if ok {
			return obj, nil
		}
	}
	return nil, ErrIncompatibleCast
}

// CanAssign reports whether obj structurally satisfies the interface. It has no
// VM, so field-type symbols and parent interfaces that need one are skipped;
// prefer CanAssignVM (used by parameter checking and the `::` operator).
func (i *Interface) CanAssign(obj Object) (bool, error) {
	return i.CanAssignVM(nil, obj)
}

// CanAssignVM reports whether obj structurally satisfies the interface: it has
// every required field (with an assignable type), property and method (whose
// signatures satisfy the required headers), and satisfies every extended
// interface. vm resolves field-type symbols, property/method calls and the
// parent-interface symbols; when nil those VM-dependent checks are relaxed.
func (i *Interface) CanAssignVM(vm *VM, obj Object) (bool, error) {
	if obj == nil || obj == Nil {
		return false, nil
	}

	if vm != nil {
		for _, sym := range i.Extends {
			pv, err := vm.GetSymbolValue(sym)
			if err != nil {
				return false, err
			}
			if parent, _ := pv.(*Interface); parent != nil {
				if ok, err := parent.CanAssignVM(vm, obj); err != nil || !ok {
					return ok, err
				}
			}
		}
	}

	for _, f := range i.Fields {
		v, ok := ifaceGet(vm, obj, f.Name)
		if !ok {
			return false, nil
		}
		if ok, err := ifaceFieldTypeOK(vm, f, v); err != nil || !ok {
			return ok, err
		}
	}

	for _, p := range i.Props {
		if _, ok := ifaceGet(vm, obj, p.Name); !ok {
			return false, nil
		}
	}

	for _, m := range i.Methods {
		if ok, err := ifaceMethodOK(vm, obj, m); err != nil || !ok {
			return ok, err
		}
	}
	return true, nil
}

// ifaceGet resolves a named member value of obj — a field, a property getter, an
// index, or a class method — for interface field/property checks, returning
// ok=false when it is absent. It covers a ClassInstance (fields/getters/methods)
// and any IndexGetter (Dict, KeyValueArray, …). A property that maps to a plain
// field or index therefore satisfies by presence; a field satisfies against
// another field or an index entry.
func ifaceGet(vm *VM, obj Object, name string) (Object, bool) {
	switch t := obj.(type) {
	case *ClassInstance:
		if v, err := t.GetFieldValue(vm, name); err == nil {
			return v, true
		}
		if m := t.GetMethod(name); m != nil {
			return m, true
		}
		return nil, false
	case IndexGetter:
		if v, err := t.IndexGet(vm, Str(name)); err == nil && v != nil && v != Nil {
			return v, true
		}
	}
	return nil, false
}

// ifaceMethodOK reports whether obj provides the interface method m: a callable
// member — a class method or a callable field/index value — whose signatures
// satisfy the required headers, or, when obj has no such member but dispatches
// methods by name (a NameCallerObject with an open method set), the presence of
// that dynamic dispatch. A ClassInstance has a discoverable, finite method set,
// so a missing method there is a genuine miss rather than deferred to CallName.
func ifaceMethodOK(vm *VM, obj Object, m *InterfaceMethod) (bool, error) {
	if v, ok := ifaceGet(vm, obj, m.Name); ok {
		if _, isCaller := v.(CallerObject); isCaller {
			mi := &MethodInterface{MIName: m.Name, Headers: m.Headers}
			return MethodInterfaceImplements(vm, v, mi)
		}
		return false, nil // the member exists but is not callable
	}
	if _, isInst := obj.(*ClassInstance); isInst {
		return false, nil
	}
	if _, ok := obj.(NameCallerObject); ok {
		return true, nil
	}
	return false, nil
}

// ifaceFieldTypeOK reports whether v is assignable to the interface field's
// declared type(s). An untyped field only requires presence.
func ifaceFieldTypeOK(vm *VM, f *InterfaceField, v Object) (bool, error) {
	types := f.Types
	if len(types) == 0 && vm != nil {
		for _, sym := range f.TypesSymbols {
			tv, err := vm.GetSymbolValue(sym)
			if err != nil {
				return false, err
			}
			if ot, _ := tv.(ObjectType); ot != nil {
				types = append(types, ot)
			}
		}
	}
	if len(types) == 0 {
		return true, nil
	}
	vt := v.Type()
	if vm != nil {
		vt = vm.ResolveType(vt)
	}
	for _, t := range types {
		if t == TAny || IsTypeAssignableTo(vt, t) {
			return true, nil
		}
	}
	return false, nil
}

func (i *Interface) Name() string { return i.IName }
func (i *Interface) IsFalsy() bool {
	return len(i.Fields) == 0 && len(i.Props) == 0 && len(i.Methods) == 0
}
func (i *Interface) ToString() string { return i.String() }

// FullName is the interface name qualified by its module, or just the name when
// there is no (or an unnamed) module or the interface is anonymous.
// FullName returns the module-qualified name `MODULE_NAME.NAME` when the module
// name is set; otherwise the bare name (or an empty string when unnamed).
func (i *Interface) FullName() string {
	if i.IName == "" {
		return ""
	}
	if i.Module != nil && i.Module.Name != "" {
		return i.Module.Name + "." + i.IName
	}
	return i.IName
}

func (i *Interface) String() string {
	var b strings.Builder
	b.WriteString("interface ")
	if n := i.FullName(); n != "" {
		b.WriteString(n)
		b.WriteString(" ")
	}
	b.WriteString("{")
	sep := ""
	for _, f := range i.Fields {
		b.WriteString(sep)
		b.WriteString(f.Name)
		sep = "; "
	}
	for _, p := range i.Props {
		b.WriteString(sep)
		if p.Getter != nil {
			b.WriteString("get ")
		} else {
			b.WriteString("set ")
		}
		b.WriteString(p.Name)
		sep = "; "
	}
	for _, m := range i.Methods {
		b.WriteString(sep)
		b.WriteString(m.Name)
		b.WriteString("()")
		sep = "; "
	}
	b.WriteString("}")
	return b.String()
}

func (i *Interface) Equal(right Object) bool {
	o, ok := right.(*Interface)
	if !ok || i.IName != o.IName ||
		len(i.Fields) != len(o.Fields) ||
		len(i.Props) != len(o.Props) ||
		len(i.Methods) != len(o.Methods) {
		return false
	}
	for k := range i.Fields {
		if !i.Fields[k].Equal(o.Fields[k]) {
			return false
		}
	}
	for k := range i.Props {
		if !i.Props[k].Equal(o.Props[k]) {
			return false
		}
	}
	for k := range i.Methods {
		if !i.Methods[k].Equal(o.Methods[k]) {
			return false
		}
	}
	return true
}

func objectArray[T Object](s []T) Array {
	arr := make(Array, len(s))
	for i, v := range s {
		arr[i] = v
	}
	return arr
}

func (i *Interface) IndexGet(_ *VM, index Object) (Object, error) {
	switch index.ToString() {
	case "name":
		return Str(i.IName), nil
	case "fields":
		return objectArray(i.Fields), nil
	case "props":
		return objectArray(i.Props), nil
	case "methods":
		return objectArray(i.Methods), nil
	}
	return nil, ErrInvalidIndex.NewError(index.ToString())
}

// Fluid construction. Each method appends a member and returns the interface so
// calls can be chained. The appended member's Iface back-reference is set.

// WithField appends a typed field.
func (i *Interface) WithField(name string, types ...ObjectType) *Interface {
	i.Fields = append(i.Fields, &InterfaceField{Iface: i, Name: name, Types: types})
	return i
}

// WithGetter appends a getter property (an InterfaceProp with a Getter).
func (i *Interface) WithGetter(name string, getter *FuncHeaderObject) *Interface {
	i.Props = append(i.Props, &InterfaceProp{Iface: i, Name: name, Getter: getter})
	return i
}

// WithSetter appends a setter property (an InterfaceProp with Setters).
func (i *Interface) WithSetter(name string, setters ...*FuncHeaderObject) *Interface {
	i.Props = append(i.Props, &InterfaceProp{Iface: i, Name: name, Setters: setters})
	return i
}

// WithMethod appends a required method with its overload signatures.
func (i *Interface) WithMethod(name string, headers ...*FuncHeaderObject) *Interface {
	i.Methods = append(i.Methods, &InterfaceMethod{Iface: i, Name: name, Headers: headers})
	return i
}

// --- InterfaceField ---

func (f *InterfaceField) Type() ObjectType { return TInterfaceField }
func (f *InterfaceField) IsFalsy() bool    { return f.Name == "" }
func (f *InterfaceField) ToString() string {
	if names := f.typeNames(); len(names) > 0 {
		return f.Name + " " + strings.Join(names, "|")
	}
	return f.Name
}

func (f *InterfaceField) typeNames() []string {
	if len(f.Types) > 0 {
		names := make([]string, len(f.Types))
		for i, t := range f.Types {
			names[i] = t.Name()
		}
		return names
	}
	names := make([]string, len(f.TypesSymbols))
	for i, s := range f.TypesSymbols {
		names[i] = s.Name
	}
	return names
}

func (f *InterfaceField) Equal(right Object) bool {
	o, ok := right.(*InterfaceField)
	if !ok || f.Name != o.Name || len(f.TypesSymbols) != len(o.TypesSymbols) {
		return false
	}
	for i := range f.TypesSymbols {
		if f.TypesSymbols[i].Name != o.TypesSymbols[i].Name {
			return false
		}
	}
	return true
}

func (f *InterfaceField) IndexGet(vm *VM, index Object) (Object, error) {
	switch index.ToString() {
	case "name":
		return Str(f.Name), nil
	case "types":
		if len(f.TypesSymbols) == 0 || vm == nil {
			return objectArray(f.Types), nil
		}
		out := make(Array, len(f.TypesSymbols))
		for i, s := range f.TypesSymbols {
			v, err := vm.GetSymbolValue(s)
			if err != nil {
				return nil, err
			}
			out[i] = v
		}
		return out, nil
	}
	return nil, ErrInvalidIndex.NewError(index.ToString())
}

// --- InterfaceProp ---

func (p *InterfaceProp) Type() ObjectType { return TInterfaceProp }
func (p *InterfaceProp) IsFalsy() bool    { return p.Name == "" }
func (p *InterfaceProp) ToString() string {
	kind := "prop"
	switch {
	case p.Getter != nil && len(p.Setters) == 0:
		kind = "get"
	case p.Getter == nil && len(p.Setters) > 0:
		kind = "set"
	}
	return kind + " " + p.Name
}

func (p *InterfaceProp) Equal(right Object) bool {
	o, ok := right.(*InterfaceProp)
	return ok && p.Name == o.Name &&
		(p.Getter == nil) == (o.Getter == nil) &&
		len(p.Setters) == len(o.Setters)
}

func (p *InterfaceProp) IndexGet(_ *VM, index Object) (Object, error) {
	switch index.ToString() {
	case "name":
		return Str(p.Name), nil
	case "getter":
		if p.Getter == nil {
			return Nil, nil
		}
		return p.Getter, nil
	case "setters":
		return objectArray(p.Setters), nil
	}
	return nil, ErrInvalidIndex.NewError(index.ToString())
}

// --- InterfaceMethod ---

func (m *InterfaceMethod) Type() ObjectType { return TInterfaceMethod }
func (m *InterfaceMethod) IsFalsy() bool    { return m.Name == "" }
func (m *InterfaceMethod) ToString() string { return m.Name + "()" }

func (m *InterfaceMethod) Equal(right Object) bool {
	o, ok := right.(*InterfaceMethod)
	if !ok || m.Name != o.Name || len(m.Headers) != len(o.Headers) {
		return false
	}
	for i := range m.Headers {
		if !m.Headers[i].Equal(o.Headers[i]) {
			return false
		}
	}
	return true
}

func (m *InterfaceMethod) IndexGet(_ *VM, index Object) (Object, error) {
	switch index.ToString() {
	case "name":
		return Str(m.Name), nil
	case "headers":
		return objectArray(m.Headers), nil
	}
	return nil, ErrInvalidIndex.NewError(index.ToString())
}
