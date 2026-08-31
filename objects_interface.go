package gad

import "strings"

// The user-facing `gad:doc` reference for Interface lives in
// builtin_types_doc.go (the `types` doc module).

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

	// ClassInstanceInterface (`classInstance`) matches an instance of a
	// user-defined class (a value produced by calling a `class`/`Class(…)`).
	ClassInstanceInterface = nativeInterface("classInstance", func(_ *VM, obj Object) bool {
		_, ok := obj.(*ClassInstance)
		return ok
	})

	// ClassTypeInterface (`classType`) matches a user-defined class object (a
	// value produced by `class`/`Class(…)`). Calling a classType yields a
	// `classInstance`.
	ClassTypeInterface = nativeInterface("classType", func(_ *VM, obj Object) bool {
		_, ok := obj.(*Class)
		return ok
	})

	// ReadableInterface (`readable`) matches any value that can be read from — the
	// check `read` uses (ReaderFrom). Named `readable` (not `reader`) because the
	// latter is the narrow builtin reader type; a buffer is readable but is not a
	// `reader`.
	ReadableInterface = nativeInterface("readable", func(_ *VM, obj Object) bool {
		return ReaderFrom(obj) != nil
	})

	// WritableInterface (`writable`) matches any value that can be written to — the
	// check `write` uses (WriterFrom). Named `writable` for the same reason as
	// `readable` (the `writer` builtin type is narrower).
	WritableInterface = nativeInterface("writable", func(_ *VM, obj Object) bool {
		return WriterFrom(obj) != nil
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
	Module  *ModuleSpec // module the interface was compiled in (for FullName)
	Extends ParamType   // parent interface symbol refs (from `extends { … }`)
	// ExtendsIface are parent interfaces held by direct reference (not a symbol),
	// for interfaces assembled at run time — e.g. a mixin's `@interface` extends
	// its `this` interface and each parent mixin's `@interface`. Satisfaction
	// requires the object to satisfy every one of them, like Extends.
	ExtendsIface []*Interface
	Fields       []*InterfaceField // typed fields
	Props        []*InterfaceProp  // getter/setter properties
	Methods      []*InterfaceMethod
	// ContextFuncs are the `funcs { … }` members: required context functions whose
	// captured value (bound at run time, see OpInterfaceBind) must have a
	// signature matching each header, with `@self` standing for this interface.
	ContextFuncs []*InterfaceContextFunc
	// Rest is the `**name` rest-capture field name: on a dict cast (`d :: I`) the
	// keys not named by the interface are gathered into a dict bound to this name
	// in the result. Empty when the interface has no `**` member.
	Rest string
	// ArrayDepth is the number of `[]` after the `interface` keyword
	// (`interface[] P`, `interface[][][] P`): the interface matches an array nested
	// to this depth whose leaf elements each satisfy the members. 0 for a plain
	// interface.
	ArrayDepth int
	// Native, when set, is a builtin interface's satisfaction check (e.g. the
	// `iterable` interface delegates to IsIterable). It replaces the structural
	// member check, so such interfaces can match Go-backed behaviour that is not
	// expressed as Gad members. nil for interfaces compiled from source.
	Native func(vm *VM, obj Object) (bool, error)

	// Cached `@flat` result (see Flatten): the flattened interface, or the
	// collision error, computed once.
	flat      *Interface
	flatErr   error
	flatBuilt bool
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
	// Nullable marks the field as also satisfied by nil (`name? T`, `x? int`).
	Nullable bool
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
	// `interface[] P` (ArrayDepth > 0) matches an array nested to that depth whose
	// leaf elements each satisfy the members.
	if i.ArrayDepth > 0 {
		return i.satisfiesArrayDepth(vm, obj, i.ArrayDepth)
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

// satisfiesArrayDepth reports whether obj is an array nested to depth whose leaf
// elements each satisfy the interface's members (`interface[] P` is depth 1). At
// depth 0 it is the plain member check; deeper, obj must be an Array and every
// element must satisfy depth-1.
func (i *Interface) satisfiesArrayDepth(vm *VM, obj Object, depth int) (bool, error) {
	if depth == 0 {
		return i.canAssignVMUncached(vm, obj)
	}
	arr, ok := obj.(Array)
	if !ok {
		return false, nil
	}
	for _, el := range arr {
		if ok, err := i.satisfiesArrayDepth(vm, el, depth-1); err != nil || !ok {
			return ok, err
		}
	}
	return true, nil
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
	// Directly-held parent interfaces (run-time extends) must all be satisfied.
	for _, parent := range i.ExtendsIface {
		if ok, err := parent.CanAssignVM(vm, obj); err != nil || !ok {
			return ok, err
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
			// indexMember reports a nil (or absent) member as not-present; a
			// nullable field (`name? T`) is satisfied by exactly that.
			if f.Nullable {
				continue
			}
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
	// A nullable field (`name? T`) is also satisfied by nil.
	if (v == Nil || v == nil) && f.Nullable {
		return true, nil
	}
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

// resolveAssigners returns the field's declared types as TypeAssigners — a plain
// ObjectType (a class, int, …) or a structural type such as a nested interface —
// resolving compile-time symbols against vm. Used by the `:::` transforming cast
// to coerce a nested dict into the field's declared shape.
func (f *InterfaceField) resolveAssigners(vm *VM) []TypeAssigner {
	var out []TypeAssigner
	for _, t := range f.Types {
		out = append(out, t)
	}
	if len(out) == 0 && vm != nil {
		for _, sym := range f.TypesSymbols {
			if tv, err := vm.GetSymbolValue(sym); err == nil {
				if ta, _ := tv.(TypeAssigner); ta != nil {
					out = append(out, ta)
				}
			}
		}
	}
	return out
}

// declaredNames is the set of member names the interface declares (fields, props
// and methods) — everything a `**rest` capture excludes.
func (i *Interface) declaredNames() map[string]bool {
	names := make(map[string]bool, len(i.Fields)+len(i.Props)+len(i.Methods))
	for _, f := range i.Fields {
		names[f.Name] = true
	}
	for _, p := range i.Props {
		names[p.Name] = true
	}
	for _, m := range i.Methods {
		names[m.Name] = true
	}
	return names
}

// coerceFieldToInterface converts v to satisfy an interface field's declared
// type: a dict for a class field becomes an instance of that class, a dict for a
// nested-interface field is transformed recursively (`:::`), and any other value
// must already be assignable. Returns the coerced value.
func coerceFieldToInterface(vm *VM, name string, assigners []TypeAssigner, v Object) (Object, error) {
	if len(assigners) == 0 {
		return v, nil // untyped field: keep as-is
	}
	// Already of an accepted type (an instance, a satisfying value, …): keep it.
	var lastErr error
	for _, a := range assigners {
		if _, lastErr = AssignToType(vm, v, a); lastErr == nil {
			return v, nil
		}
	}
	// Not directly assignable — coerce a key/value source into the declared shape:
	// a class field builds an instance, a nested-interface field transforms it.
	if src, ok := asTransformDict(vm, v); ok && len(assigners) == 1 {
		switch t := assigners[0].(type) {
		case *Class:
			clone := make(Dict, len(src))
			for k, val := range src {
				clone[k] = val
			}
			return t.NewInstanceWithFields(vm, clone)
		case *Interface:
			return t.coerceDict(vm, src)
		}
	}
	return nil, ErrType.NewErrorf("field %q: %v", name, lastErr)
}

// coerceDict implements `dict ::: interface`: it returns a NEW dict whose fields
// typed by a class/interface are built from their nested dicts (recursively), and
// whose keys not named by the interface are gathered under the `**name` rest
// field when one is declared. A missing non-nullable field is an error.
// coerceArray implements `array ::: interface[] P`: it returns a NEW array nested
// to depth whose leaf elements are each transformed by coerceDict (so their
// class/interface-typed fields are built). A non-array at a non-zero depth, or a
// leaf that cannot be coerced, is an error.
func (i *Interface) coerceArray(vm *VM, obj Object, depth int) (Object, error) {
	if depth == 0 {
		if d, ok := asTransformDict(vm, obj); ok {
			return i.coerceDict(vm, d)
		}
		if ok, err := i.canAssignVMUncached(vm, obj); err != nil {
			return nil, err
		} else if !ok {
			return nil, ErrType.NewErrorf("%s does not satisfy the interface", obj.Type().Name())
		}
		return obj, nil
	}
	arr, ok := obj.(Array)
	if !ok {
		return nil, ErrType.NewErrorf("expected an array, got %s", obj.Type().Name())
	}
	out := make(Array, len(arr))
	for j, el := range arr {
		c, err := i.coerceArray(vm, el, depth-1)
		if err != nil {
			return nil, err
		}
		out[j] = c
	}
	return out, nil
}

func (i *Interface) coerceDict(vm *VM, d Dict) (Object, error) {
	out := make(Dict, len(d))
	for k, v := range d {
		out[k] = v
	}

	for _, f := range i.Fields {
		v, has := out[f.Name]
		if !has || v == Nil {
			if f.Nullable {
				continue
			}
			return nil, ErrType.NewErrorf("field %q is required", f.Name)
		}
		coerced, err := coerceFieldToInterface(vm, f.Name, f.resolveAssigners(vm), v)
		if err != nil {
			return nil, err
		}
		out[f.Name] = coerced
	}

	// Props and methods must be present (checked, not transformed).
	for _, p := range i.Props {
		if _, has := out[p.Name]; !has {
			return nil, ErrType.NewErrorf("property %q is required", p.Name)
		}
	}
	for _, m := range i.Methods {
		v, has := out[m.Name]
		if !has {
			return nil, ErrType.NewErrorf("method %q is required", m.Name)
		}
		if _, ok := v.(CallerObject); !ok {
			return nil, ErrType.NewErrorf("method %q must be callable", m.Name)
		}
	}

	if i.Rest != "" {
		declared := i.declaredNames()
		rest := Dict{}
		for k, v := range out {
			if !declared[k] {
				rest[k] = v
				delete(out, k)
			}
		}
		out[i.Rest] = rest
	}
	return out, nil
}

func (i *Interface) Name() string { return i.IName }
func (i *Interface) IsFalsy() bool {
	// Falsy only when the interface carries no contract at all. An interface that
	// merely extends others (Extends / ExtendsIface — e.g. a mixin's `@interface`
	// and `@classInterface`), or has context-func requirements, a `**rest` capture
	// or a native predicate, is a real, non-empty interface.
	return len(i.Fields) == 0 && len(i.Props) == 0 && len(i.Methods) == 0 &&
		len(i.Extends) == 0 && len(i.ExtendsIface) == 0 && len(i.ContextFuncs) == 0 &&
		i.Rest == "" && i.Native == nil
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
	// Run-time parent interfaces (a mixin's `@interface` extends its `this` and its
	// parents' `@interface`) render as `*ParentName` spreads, mirroring source.
	for _, p := range i.ExtendsIface {
		b.WriteString(sep)
		b.WriteString("*")
		b.WriteString(p.FullName())
		sep = "; "
	}
	// Each member renders through its own ToString (so a field shows its type,
	// `f int`, and a property its accessor kind, `get p` / `set p` / `prop p`).
	for _, f := range i.Fields {
		b.WriteString(sep)
		b.WriteString(f.ToString())
		sep = "; "
	}
	for _, p := range i.Props {
		b.WriteString(sep)
		b.WriteString(p.ToString())
		sep = "; "
	}
	for _, m := range i.Methods {
		b.WriteString(sep)
		b.WriteString(m.ToString())
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

func (i *Interface) IndexGet(vm *VM, index Object) (Object, error) {
	switch index.ToString() {
	case "name":
		return Str(i.IName), nil
	case "fields":
		return objectArray(i.Fields), nil
	case "props":
		return objectArray(i.Props), nil
	case "methods":
		return objectArray(i.Methods), nil
	case "@flat":
		return i.Flatten(vm)
	}
	return nil, ErrInvalidIndex.NewError(index.ToString())
}

// ErrInterfaceMemberConflict is raised when Flatten (`@flat`) finds one
// member name declared by two different interfaces in an extends graph.
var ErrInterfaceMemberConflict = &Error{Name: "InterfaceMemberConflictError"}

// Flatten returns (and caches) the interface flattened across its whole extends
// graph — both the direct-reference parents (ExtendsIface, e.g. a mixin's
// `@interface`) and the symbol parents (Extends, from a source `interface { *A }`,
// resolved via vm) — into a new Interface with NO extends of its own.
//
// Members with the SAME NAME across interfaces are MERGED, not rejected, as long
// as their signatures are compatible: a getter, its setters (by value type) and a
// method's overloads all combine, and an identical signature seen twice is
// deduplicated silently. A genuine signature CONFLICT returns
// ErrInterfaceMemberConflict:
//   - a name used as two different member kinds (field vs property vs method);
//   - two getters of one name with different return types;
//   - two method overloads with the same parameters but different return types.
//
// Deduplication of a whole interface reached more than once (a diamond) is by
// interface identity, so a shared parent contributes its members once.
func (i *Interface) Flatten(vm *VM) (*Interface, error) {
	if i.flatBuilt {
		return i.flat, i.flatErr
	}
	m := newIfaceMerger(i.IName, i.Module, i.Rest, i.ArrayDepth)
	seen := map[*Interface]bool{}
	var walk func(*Interface) error
	walk = func(x *Interface) error {
		if x == nil || seen[x] {
			return nil
		}
		seen[x] = true
		for _, f := range x.Fields {
			if err := m.addField(f); err != nil {
				return err
			}
		}
		for _, p := range x.Props {
			if err := m.addProp(p); err != nil {
				return err
			}
		}
		for _, meth := range x.Methods {
			if err := m.addMethod(meth); err != nil {
				return err
			}
		}
		m.out.ContextFuncs = append(m.out.ContextFuncs, x.ContextFuncs...)
		for _, p := range x.ExtendsIface {
			if err := walk(p); err != nil {
				return err
			}
		}
		if vm != nil {
			for _, sym := range x.Extends {
				pv, err := vm.GetSymbolValue(sym)
				if err != nil {
					return err
				}
				if parent, _ := pv.(*Interface); parent != nil {
					if err := walk(parent); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}
	i.flatErr = walk(i)
	if i.flatErr == nil {
		i.flat = m.finish()
	}
	i.flatBuilt = true
	return i.flat, i.flatErr
}

// ifaceMerger accumulates the flattened members of an interface graph, merging
// same-name members by signature and rejecting genuine conflicts (see Flatten).
type ifaceMerger struct {
	out   *Interface
	kind  map[string]string            // name -> "field" | "prop" | "method"
	field map[string]*InterfaceField   // name -> the (single) field, for type check
	geter map[string]*InterfaceProp    // name -> the merged property (getter + setters)
	seter map[string]map[string]bool   // name -> set of setter value-type keys
	meth  map[string]map[string]string // name -> paramsKey -> returnKey (overloads)
	mmeth map[string]*InterfaceMethod  // name -> merged method (accumulating headers)
}

func newIfaceMerger(name string, mod *ModuleSpec, rest string, depth int) *ifaceMerger {
	return &ifaceMerger{
		out:   &Interface{IName: name, Module: mod, Rest: rest, ArrayDepth: depth},
		kind:  map[string]string{},
		field: map[string]*InterfaceField{},
		geter: map[string]*InterfaceProp{},
		seter: map[string]map[string]bool{},
		meth:  map[string]map[string]string{},
		mmeth: map[string]*InterfaceMethod{},
	}
}

// claimKind records that name is used as the given member kind, returning a
// conflict error when it is already used as a different kind.
func (m *ifaceMerger) claimKind(name, kind string) error {
	if prev, ok := m.kind[name]; ok && prev != kind {
		return ErrInterfaceMemberConflict.NewErrorf(
			"member %q is declared both as a %s and a %s; a member name must be one consistent kind",
			name, prev, kind)
	}
	m.kind[name] = kind
	return nil
}

func (m *ifaceMerger) addField(f *InterfaceField) error {
	if err := m.claimKind(f.Name, "field"); err != nil {
		return err
	}
	key := ifaceTypesKey(f.Types, f.TypesSymbols)
	if prev, ok := m.field[f.Name]; ok {
		if ifaceTypesKey(prev.Types, prev.TypesSymbols) != key {
			return ErrInterfaceMemberConflict.NewErrorf(
				"field %q is declared with conflicting types across interfaces", f.Name)
		}
		return nil // identical -> dedup
	}
	m.field[f.Name] = f
	m.out.Fields = append(m.out.Fields, f)
	return nil
}

// addProp merges a getter/setter into the single accumulating property of that
// name: a getter (dedup, conflict on a differing return type) plus every distinct
// setter value-type overload. get/set are just short forms of one `prop`.
func (m *ifaceMerger) addProp(p *InterfaceProp) error {
	if err := m.claimKind(p.Name, "prop"); err != nil {
		return err
	}
	merged, ok := m.geter[p.Name]
	if !ok {
		merged = &InterfaceProp{Iface: m.out, Name: p.Name}
		m.geter[p.Name] = merged
		m.seter[p.Name] = map[string]bool{}
		m.out.Props = append(m.out.Props, merged)
	}
	if p.Getter != nil {
		if merged.Getter == nil {
			merged.Getter = p.Getter
		} else if headerReturnKey(merged.Getter) != headerReturnKey(p.Getter) {
			return ErrInterfaceMemberConflict.NewErrorf(
				"getter %q is declared with conflicting return types across interfaces", p.Name)
		}
	}
	set := m.seter[p.Name]
	for _, s := range p.Setters {
		key := headerParamsKey(s)
		if !set[key] { // a new setter value-type overload
			set[key] = true
			merged.Setters = append(merged.Setters, s)
		}
	}
	return nil
}

func (m *ifaceMerger) addMethod(meth *InterfaceMethod) error {
	if err := m.claimKind(meth.Name, "method"); err != nil {
		return err
	}
	overloads := m.meth[meth.Name]
	if overloads == nil {
		overloads = map[string]string{}
		m.meth[meth.Name] = overloads
		merged := &InterfaceMethod{Iface: m.out, Name: meth.Name}
		m.mmeth[meth.Name] = merged
		m.out.Methods = append(m.out.Methods, merged)
	}
	merged := m.mmeth[meth.Name]
	// A method with no explicit headers still asserts the name exists.
	if len(meth.Headers) == 0 {
		if _, ok := overloads[""]; !ok {
			overloads[""] = ""
		}
		return nil
	}
	for _, h := range meth.Headers {
		pk, rk := headerParamsKey(h), headerReturnKey(h)
		if prevRet, ok := overloads[pk]; ok {
			if prevRet != rk {
				return ErrInterfaceMemberConflict.NewErrorf(
					"method %q has two overloads with the same parameters but different return types", meth.Name)
			}
			continue // identical overload -> dedup
		}
		overloads[pk] = rk
		merged.Headers = append(merged.Headers, h)
	}
	return nil
}

func (m *ifaceMerger) finish() *Interface { return m.out }

// ifaceTypesKey renders a field's declared types (resolved ObjectTypes or their
// compile-time symbols) as a stable, order-preserving key for comparison.
func ifaceTypesKey(types ObjectTypes, syms ParamType) string {
	if len(types) > 0 {
		parts := make([]string, len(types))
		for i, t := range types {
			parts[i] = t.Name()
		}
		return strings.Join(parts, "|")
	}
	parts := make([]string, len(syms))
	for i, s := range syms {
		parts[i] = s.Name
	}
	return strings.Join(parts, "|")
}

// headerParamsKey / headerReturnKey render a header's parameter (incl. named) and
// return type lists as stable keys, so two headers are the "same overload" when
// their params match and "conflicting" when params match but returns differ.
func headerParamsKey(h *FuncHeaderObject) string {
	if h == nil {
		return ""
	}
	return typedIdentsKey(h.Params) + ";" + typedIdentsKey(h.NamedParams)
}

func headerReturnKey(h *FuncHeaderObject) string {
	if h == nil {
		return ""
	}
	// A bare return `<int>` stores its type in the ident Name (not Types), so a
	// return slot falls back to Name when it has no explicit types.
	parts := make([]string, 0, len(h.Return))
	for _, o := range h.Return {
		if ti, _ := o.(*TypedIdent); ti != nil {
			if names := ti.typeNames(); len(names) > 0 {
				parts = append(parts, strings.Join(names, "|"))
			} else {
				parts = append(parts, ti.Name)
			}
		}
	}
	return strings.Join(parts, ",")
}

// typedIdentsKey joins the type names of each *TypedIdent in arr; the parameter
// name is ignored — only the types define the signature (an untyped param keys as
// empty, i.e. `any`).
func typedIdentsKey(arr Array) string {
	parts := make([]string, 0, len(arr))
	for _, o := range arr {
		if ti, _ := o.(*TypedIdent); ti != nil {
			parts = append(parts, strings.Join(ti.typeNames(), "|"))
		} else {
			parts = append(parts, "?")
		}
	}
	return strings.Join(parts, ",")
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

// ToString renders a property. A single-accessor property keeps the short form
// `get NAME` / `set NAME`. A property with both a getter and setter(s) — or more
// than one setter — is a combined `prop NAME { get RET; set VAL1|VAL2 }`: the
// getter as `get <returnType>` and the setters as one `set <value types joined by
// |>`. get/set are just short forms of one `prop`.
func (p *InterfaceProp) ToString() string {
	switch {
	case p.Getter != nil && len(p.Setters) == 0:
		return "get " + p.Name
	case p.Getter == nil && len(p.Setters) == 1:
		return "set " + p.Name
	}
	var b strings.Builder
	b.WriteString("prop ")
	b.WriteString(p.Name)
	b.WriteString(" { ")
	sep := ""
	if p.Getter != nil {
		b.WriteString("get")
		if r := accessorTypesList(p.Getter.Return); r != "" {
			b.WriteString(" ")
			b.WriteString(r)
		}
		sep = "; "
	}
	if len(p.Setters) > 0 {
		b.WriteString(sep)
		b.WriteString("set")
		// Combine every setter's value type into one `|`-joined union, deduped.
		var types []string
		seen := map[string]bool{}
		for _, s := range p.Setters {
			for _, tn := range accessorTypesParts(s.Params) {
				if !seen[tn] {
					seen[tn] = true
					types = append(types, tn)
				}
			}
		}
		if len(types) > 0 {
			b.WriteString(" ")
			b.WriteString(strings.Join(types, "|"))
		}
	}
	b.WriteString(" }")
	return b.String()
}

// accessorTypesList renders the types of an accessor's TypedIdent slots as
// `t1, t2` (type-only; the anonymous `_` name is omitted). Empty when none.
func accessorTypesList(arr Array) string {
	return strings.Join(accessorTypesParts(arr), ", ")
}

// accessorTypesParts returns the type name of each typed slot in arr (a bare
// type-only slot keeps its Name; an anonymous `_` slot uses its declared types).
func accessorTypesParts(arr Array) []string {
	parts := make([]string, 0, len(arr))
	for _, o := range arr {
		if ti, _ := o.(*TypedIdent); ti != nil {
			if names := ti.typeNames(); len(names) > 0 {
				parts = append(parts, strings.Join(names, "|"))
			} else if ti.Name != "" && ti.Name != "_" {
				parts = append(parts, ti.Name)
			}
		}
	}
	return parts
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
