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

// nativeInterface builds a builtin interface whose satisfaction is a native
// predicate over the object (see Interface.Native), rather than a structural
// member check. Such interfaces name a behaviour a Go type provides — being
// iterable, callable, indexable, … — and are usable as parameter/`::` types.
func nativeInterface(name string, pred func(vm *VM, obj Object) bool) *Interface {
	return &Interface{
		IName:  name,
		Native: func(vm *VM, obj Object) (bool, error) { return pred(vm, obj), nil },
	}
}

// Builtin behavioural interfaces. Each matches the values that provide the named
// behaviour (the corresponding Go interface / predicate), so they can type the
// arguments of the builtins that require it, e.g. `filter(it iterable, …)`.
var (
	// IterableInterface (`iterable`) matches values that can be iterated: an
	// iterator, a Go Iterabler, or a type with an `iterator` method (IsIterable).
	IterableInterface = nativeInterface("iterable", func(vm *VM, obj Object) bool {
		if vm != nil {
			return Iterable(vm, obj)
		}
		switch obj.(type) { // no VM: the Go-level check only
		case Iterator, Iterabler:
			return true
		}
		return false
	})

	// CallableInterface (`callable`) matches values that can be called.
	CallableInterface = nativeInterface("callable", func(_ *VM, obj Object) bool {
		return Callable(obj)
	})

	// LengtherInterface (`lengther`) matches values that have a length.
	LengtherInterface = nativeInterface("lengther", func(_ *VM, obj Object) bool {
		_, ok := obj.(LengthGetter)
		return ok
	})

	// IndexableInterface (`indexable`) matches values that support `obj[i]`.
	IndexableInterface = nativeInterface("indexable", func(_ *VM, obj Object) bool {
		_, ok := obj.(IndexGetter)
		return ok
	})

	// IndexAssignableInterface (`indexAssignable`) matches values that support
	// `obj[i] = v`.
	IndexAssignableInterface = nativeInterface("indexAssignable", func(_ *VM, obj Object) bool {
		_, ok := obj.(IndexSetter)
		return ok
	})

	// IndexDeletableInterface (`indexDeletable`) matches values that support
	// deleting an index.
	IndexDeletableInterface = nativeInterface("indexDeletable", func(_ *VM, obj Object) bool {
		_, ok := obj.(IndexDeleter)
		return ok
	})
)

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
	// ContextFuncs are the `funcs { … }` members: required context functions whose
	// captured value (bound at run time, see OpInterfaceBind) must have a
	// signature matching each header, with `@self` standing for this interface.
	ContextFuncs []*InterfaceContextFunc
	// Native, when set, is a builtin interface's satisfaction check (e.g. the
	// `iterable` interface delegates to IsIterable). It replaces the structural
	// member check, so such interfaces can match Go-backed behaviour that is not
	// expressed as Gad members. nil for interfaces compiled from source.
	Native func(vm *VM, obj Object) (bool, error)
}

// InterfaceContextFunc is a context-function member of an interface: the source
// text of the function expression (for messages), the required signature
// headers, and the callable value captured where the interface was declared
// (nil in the un-bound constant template; bound at run time by OpInterfaceBind).
type InterfaceContextFunc struct {
	FnName  string
	Headers []*FuncHeaderObject
	Fn      CallerObject
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

// BindContextFuncs returns a shallow copy of the interface with each
// ContextFuncs entry's Fn set from fns (in order): the runtime binding of the
// captured context-function values (see OpInterfaceBind). len(fns) must equal
// len(i.ContextFuncs).
func (i *Interface) BindContextFuncs(fns []Object) *Interface {
	cp := *i
	cp.ContextFuncs = make([]*InterfaceContextFunc, len(i.ContextFuncs))
	for idx, cf := range i.ContextFuncs {
		bound := *cf
		if idx < len(fns) {
			bound.Fn, _ = fns[idx].(CallerObject)
		}
		cp.ContextFuncs[idx] = &bound
	}
	return &cp
}

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
// CanAssignVM reports whether obj structurally satisfies the interface, memoizing
// the result on the root VM keyed by (interface, obj's ObjectType) when that type
// fully determines the object's members (a class instance or a reflected Go
// value). Dicts and other keys-vary-per-value objects are checked every time.
func (i *Interface) CanAssignVM(vm *VM, obj Object) (bool, error) {
	if obj == nil || obj == Nil {
		return false, nil
	}
	if vm != nil {
		if typ, ok := ifaceCacheableType(obj); ok {
			key := ifaceSatKey{iface: i, typ: typ}
			if v, hit := vm.ifaceSatGet(key); hit {
				return v, nil
			}
			res, err := i.canAssignVMUncached(vm, obj)
			if err == nil {
				vm.ifaceSatPut(key, res)
			}
			return res, err
		}
	}
	return i.canAssignVMUncached(vm, obj)
}

// ifaceCacheableType returns obj's ObjectType and true when that type fully
// determines obj's interface-relevant members, so a satisfaction result is safe
// to cache: a class instance (its class declares the members) or a reflected Go
// value (its Go type does). Other values vary per instance (a dict's keys) and
// are not cacheable.
func ifaceCacheableType(obj Object) (ObjectType, bool) {
	switch obj.(type) {
	case *ClassInstance, ReflectValuer:
		return obj.Type(), true
	}
	return nil, false
}

func (i *Interface) canAssignVMUncached(vm *VM, obj Object) (bool, error) {
	// A builtin interface with a native predicate (e.g. `iterable`) is satisfied
	// by that predicate rather than by structural member probing.
	if i.Native != nil {
		return i.Native(vm, obj)
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

	// Context-function members validate captured free functions (independent of
	// obj): each must have a signature matching every header, with `@self`
	// standing for this interface. Needs a VM to read the resolved header types;
	// without one the check is relaxed (like Extends).
	if vm != nil {
		for _, cf := range i.ContextFuncs {
			if ok, err := i.contextFuncOK(vm, cf); err != nil || !ok {
				return ok, err
			}
		}
	}

	// A class instance satisfies an interface through its class's declared members
	// (fields, property accessors, methods). Other member-bearing values — a
	// dict, a key-value array, a NameCaller — use generic member probing.
	if inst, ok := obj.(*ClassInstance); ok {
		return i.classInstanceSatisfies(vm, inst)
	}
	return i.genericSatisfies(vm, obj)
}

// contextFuncOK reports whether the context function cf.Fn (captured at the
// interface's declaration) has a signature matching every one of cf.Headers. The
// `@self`-typed param stands for this interface: it matches a signature slot that
// is untyped (accepts anything) or typed with this interface. A nil/absent Fn or
// a non-callable value fails.
func (i *Interface) contextFuncOK(vm *VM, cf *InterfaceContextFunc) (bool, error) {
	if cf.Fn == nil {
		return false, nil
	}
	var sigs []ParamsTypes
	if err := SplitCaller(vm, cf.Fn,
		func(_ CallerObject, types ParamsTypes) error { sigs = append(sigs, types); return nil },
		func(_ CallerObject) error { sigs = append(sigs, nil); return nil },
	); err != nil {
		return false, nil
	}
	for _, h := range cf.Headers {
		matched := false
		for _, sig := range sigs {
			if sig == nil || i.headerMatchesCtxSig(vm, h, sig) {
				matched = true
				break
			}
		}
		if !matched {
			return false, nil
		}
	}
	return true, nil
}

// headerMatchesCtxSig reports whether a context-func header matches a signature,
// like headerMatchesSig but a `@self` param matches when the signature slot
// accepts this interface (untyped, or typed with the interface).
func (i *Interface) headerMatchesCtxSig(vm *VM, h *FuncHeaderObject, sig ParamsTypes) bool {
	if len(h.Params) != len(sig) {
		return false
	}
	for idx, p := range h.Params {
		ti, _ := p.(*TypedIdent)
		if ti == nil {
			return false
		}
		if ti.Self {
			if !selfParamMatches(sig[idx].Items()) {
				return false
			}
			continue
		}
		hts, _ := ti.resolveTypes(vm)
		if !paramMatches(hts, sig[idx].Items()) {
			return false
		}
	}
	return true
}

// selfParamMatches reports whether a signature param slot accepts the interface
// (the `@self` type). Since a structural interface is not a concrete ObjectType,
// it is representable in a signature only as an untyped (Any) slot; so `@self`
// matches an untyped slot — the function must accept the interface's objects
// there.
func selfParamMatches(sigTypes ObjectTypes) bool {
	if len(sigTypes) == 0 {
		return true
	}
	for _, t := range sigTypes {
		if t == nil || t == TAny {
			return true
		}
	}
	return false
}

// classInstanceSatisfies is the ClassInstance-specific satisfaction check: each
// interface field is a class field or getter (value type-checked), each property
// is a property accessor or plain field (detected without invoking the getter),
// and each method is a class method whose overloads satisfy the headers. A class
// has a finite, discoverable member set, so a missing member is a genuine miss.
func (i *Interface) classInstanceSatisfies(vm *VM, o *ClassInstance) (bool, error) {
	for _, f := range i.Fields {
		v, err := o.GetFieldValue(vm, f.Name)
		if err != nil {
			return false, nil
		}
		if ok, err := ifaceFieldTypeOK(vm, f, v); err != nil || !ok {
			return ok, err
		}
	}
	for _, p := range i.Props {
		if _, valid := o.GetPropertyGetter(p.Name); valid {
			continue
		}
		if o.ResolveField(p.Name) == nil {
			return false, nil
		}
	}
	for _, m := range i.Methods {
		mm := o.GetMethod(m.Name)
		if mm == nil {
			return false, nil
		}
		mi := &MethodInterface{MIName: m.Name, Headers: m.Headers}
		if ok, err := MethodInterfaceImplements(vm, mm, mi); err != nil || !ok {
			return ok, err
		}
	}
	return true, nil
}

// genericSatisfies is the satisfaction check for a non-class member-bearing
// value: fields and properties match a key of any IndexGetter (a dict, key-value
// array, …), methods match a callable key whose overloads satisfy the headers,
// and a value that dispatches methods by name (a NameCallerObject with an open
// method set) satisfies method requirements optimistically (duck typing).
func (i *Interface) genericSatisfies(vm *VM, obj Object) (bool, error) {
	for _, f := range i.Fields {
		v, ok := indexMember(vm, obj, f.Name)
		if !ok {
			return false, nil
		}
		if ok, err := ifaceFieldTypeOK(vm, f, v); err != nil || !ok {
			return ok, err
		}
	}
	for _, p := range i.Props {
		if _, ok := indexMember(vm, obj, p.Name); !ok {
			return false, nil
		}
	}
	for _, m := range i.Methods {
		if v, ok := indexMember(vm, obj, m.Name); ok {
			if _, isCaller := v.(CallerObject); !isCaller {
				return false, nil // a member exists but is not callable
			}
			mi := &MethodInterface{MIName: m.Name, Headers: m.Headers}
			if ok, err := MethodInterfaceImplements(vm, v, mi); err != nil || !ok {
				return ok, err
			}
			continue
		}
		if _, ok := obj.(NameCallerObject); !ok {
			return false, nil
		}
	}
	return true, nil
}

// indexMember reads a named key from any IndexGetter (dict, key-value array, …),
// returning ok=false when it is absent (missing keys read as Nil).
func indexMember(vm *VM, obj Object, name string) (Object, bool) {
	if ig, ok := obj.(IndexGetter); ok {
		if v, err := ig.IndexGet(vm, Str(name)); err == nil && v != nil && v != Nil {
			return v, true
		}
	}
	return nil, false
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
		len(i.Methods) != len(o.Methods) ||
		len(i.ContextFuncs) != len(o.ContextFuncs) {
		return false
	}
	for k := range i.ContextFuncs {
		if i.ContextFuncs[k].FnName != o.ContextFuncs[k].FnName ||
			len(i.ContextFuncs[k].Headers) != len(o.ContextFuncs[k].Headers) {
			return false
		}
		for h := range i.ContextFuncs[k].Headers {
			if !i.ContextFuncs[k].Headers[h].Equal(o.ContextFuncs[k].Headers[h]) {
				return false
			}
		}
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
