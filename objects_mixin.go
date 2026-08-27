package gad

import (
	"sort"
	"strconv"
)

// sortedKeys returns the keys of a string-keyed map in ascending order, for a
// deterministic iteration over an otherwise unordered map.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

var (
	_ Object      = (*Mixin)(nil)
	_ IndexGetter = (*Mixin)(nil)
)

// Mixin is a value created with the `mixin … { … }` literal (the `Mixin(name;
// …)` builtin). A mixin has the basic structure of a class — parent mixins,
// fields, getter/setter properties and methods — plus an optional `this { … }`
// interface declaring what the `this` of its properties/methods must satisfy. It
// is not itself instantiable: a class pulls a mixin's members in with the `use A,
// B` clause, and the mixin's fields/properties/methods become the class's own.
//
// The declared members are kept both as their compiled runtime values (in the
// internal class, so `@fields`/`@props`/`@methods`/`@interface` can reflect them)
// and as the raw define inputs (rawFields/rawProps/rawMethods/initFields), which
// are replayed onto a using class to register fresh, class-bound members.
type Mixin struct {
	name    string
	module  *ModuleSpec
	class   *Class   // internal container for the mixin's own members (reflection)
	parents []*Mixin // parent mixins (from `*A` spreads); may contain duplicates

	// Raw define inputs, replayed onto a using class (see Class.useMixins).
	rawFields  KeyValueArray
	rawProps   Dict
	rawMethods Object
	initFields CallerObject

	iface *Interface // cached @interface (built lazily)
}

// NewMixinFunc is the `Mixin(name[, define])` builtin: it builds a Mixin, then —
// when a define handler is given — invokes it as `define(mixin, defineFn)` so the
// `fields`/`initFields`/`properties`/`methods`/`extends` members are registered
// through the defineFn call, mirroring the class `Class(name, (Type, define) =>
// …)` protocol.
func NewMixinFunc(c Call) (ret Object, err error) {
	nameArg := &Arg{
		Name:          "name",
		TypeAssertion: TypeAssertionFromTypes(TStr),
	}
	rest, err := c.Args.DestructureRangeVar(1, nameArg)
	if err != nil {
		return
	}

	m := &Mixin{name: string(nameArg.Value.(Str)), module: c.VM.CurrentModuleSpec()}
	m.class = NewClass(m.name, m.module)

	if len(rest) > 0 {
		handler, ok := rest[0].(CallerObject)
		if !ok {
			return nil, NewArgumentTypeError("2nd (define)", "callable", rest[0].Type().Name())
		}
		_, err = handler.Call(Call{
			Context: c.Context,
			VM:      c.VM,
			Args: Args{{
				m,
				NewFunction("define", func(c Call) (Object, error) {
					return nil, m.define(c)
				}),
			}},
		})
		return m, err
	}

	return m, m.define(c)
}

// define registers the mixin's members from the call's named args (`fields`,
// `initFields`, `properties`, `methods`, `extends`). Each recognised arg is both
// captured raw (for replay onto a using class) and applied to the internal class
// (for reflection via `@fields`/`@props`/`@methods`).
func (m *Mixin) define(c Call) (err error) {
	var (
		fields = &NamedArgVar{
			Name:          "fields",
			TypeAssertion: TypeAssertionFromTypes(TKeyValueArray),
			Do: func(value Object) error {
				m.rawFields = value.(KeyValueArray)
				return m.class.CallAddFields(Call{VM: c.VM, Args: Args{Array{value}}})
			},
		}

		initFields = &NamedArgVar{
			Name:          "initFields",
			TypeAssertion: NewTypeAssertion(TypeAssertions(WithCallable())),
			Do: func(value Object) error {
				m.initFields, _ = value.(CallerObject)
				m.class.initFields = m.initFields
				return nil
			},
		}

		properties = &NamedArgVar{
			Name:          "properties",
			TypeAssertion: TypeAssertionFromTypes(TDict),
			Do: func(value Object) error {
				m.rawProps = value.(Dict)
				return m.class.CallAddProperties(Call{VM: c.VM, Args: Args{Array{value}}})
			},
		}

		methods = &NamedArgVar{
			Name:          "methods",
			TypeAssertion: TypeAssertionFromTypes(TDict, TKeyValueArray, TArray),
			Do: func(value Object) error {
				m.rawMethods = value
				return m.class.CallAddMethods(Call{VM: c.VM, Args: Args{Array{value}}})
			},
		}

		extends = &NamedArgVar{
			Name:          "extends",
			TypeAssertion: TypeAssertionFromTypes(TArray),
			// The parent mixins (`*A` spreads). Stored as-is (duplicates allowed);
			// deduplication happens where a class uses the mixin, not here.
			Do: func(value Object) error {
				for i, v := range value.(Array) {
					parent, ok := v.(*Mixin)
					if !ok {
						return NewArgumentTypeError(strconv.Itoa(i)+"st (extends)", "Mixin", v.Type().Name())
					}
					m.parents = append(m.parents, parent)
				}
				return nil
			},
		}
	)

	return c.NamedArgs.GetDo(extends, fields, initFields, properties, methods)
}

// lineage appends this mixin's parents (depth-first, parents before self) and
// then the mixin itself into out, skipping any mixin already in seen so a mixin
// that appears more than once across a hierarchy is registered only once (first
// occurrence wins). Deduplication is a using-class concern; a mixin itself keeps
// its raw (possibly duplicated) parents.
func (m *Mixin) lineage(seen map[*Mixin]bool, out *[]*Mixin) {
	for _, p := range m.parents {
		p.lineage(seen, out)
	}
	if seen[m] {
		return
	}
	seen[m] = true
	*out = append(*out, m)
}

func (m *Mixin) Type() ObjectType { return TMixin }

func (m *Mixin) Name() string { return m.name }

func (m *Mixin) FullName() string {
	if m.module == nil {
		return m.name
	}
	return m.module.Name + "." + m.name
}

func (m *Mixin) ToString() string { return string(MustToStr(nil, m)) }

func (m *Mixin) String() string { return TypeToString("mixin " + m.name) }

func (m *Mixin) Repr() string { return "mixin " + m.FullName() }

func (Mixin) IsFalsy() bool { return false }

func (m *Mixin) Equal(right Object) bool {
	r, _ := right.(*Mixin)
	return r == m
}

// Parents returns the mixin's parent mixins as an array (its `@parents`).
func (m *Mixin) Parents() (r Array) {
	r = make(Array, len(m.parents))
	for i, p := range m.parents {
		r[i] = p
	}
	return
}

// Interface builds (and caches) the `@interface`: a new Interface named
// `Name$interface` in the mixin's module that mirrors the mixin's declared
// members — its fields, properties and methods — so a class instance that pulls
// the mixin in structurally satisfies it.
func (m *Mixin) Interface() *Interface {
	if m.iface != nil {
		return m.iface
	}
	i := &Interface{IName: m.name + "$interface", Module: m.module}
	// Fields keep their declaration order and types (`f int` -> a typed member).
	for _, f := range m.class.RawFields() {
		fld := &InterfaceField{Iface: i, Name: f.Name, Nullable: f.Nullable}
		for _, ta := range f.Types {
			if ot, ok := ta.(ObjectType); ok {
				fld.Types = append(fld.Types, ot)
			}
		}
		i.Fields = append(i.Fields, fld)
	}
	// Properties mirror the accessor kind: a getter-only becomes `get p`, a
	// setter-only `set p`, both `prop p`. Sorted by name for a stable interface.
	for _, name := range sortedKeys(m.class.propertiesMap) {
		hasGetter, hasSetter := m.class.propertiesMap[name].accessors()
		prop := &InterfaceProp{Iface: i, Name: name}
		if hasGetter {
			prop.Getter = &FuncHeaderObject{FuncName: name}
		}
		if hasSetter {
			prop.Setters = []*FuncHeaderObject{{FuncName: name}}
		}
		i.Props = append(i.Props, prop)
	}
	for _, name := range sortedKeys(m.class.methodsMap) {
		i.Methods = append(i.Methods, &InterfaceMethod{Iface: i, Name: name})
	}
	m.iface = i
	return i
}

// IndexGet exposes the mixin's reflection attributes, mirroring Class: `@fields`,
// `@props`, `@methods`, `@parents`, `@name`, `@module` and `@interface`.
func (m *Mixin) IndexGet(vm *VM, index Object) (value Object, err error) {
	switch index.ToString() {
	case "@fields":
		return m.class.Fields(), nil
	case "@props":
		return m.class.Properties(), nil
	case "@methods":
		return m.class.Methods(), nil
	case "@parents":
		return m.Parents(), nil
	case "@name":
		return Str(m.name), nil
	case "@module":
		return vm.ModuleFromIndex(m.module.Index), nil
	case "@interface":
		return m.Interface(), nil
	default:
		return nil, ErrInvalidIndex.NewError(index.ToString())
	}
}
