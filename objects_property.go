package gad

import (
	"fmt"

	"github.com/gad-lang/gad/repr"
)

// gad:doc
// Property a property type.
// Property(name str, *methods) <Property>
//
// the methods are callable:
// - **getter** no have params and return value: `handler() <ret>` (zero or one method)
// - **setter** have one param and not return value: `handler(v)` (any methods)
//
// No methods are required to initialises new property.
// Usage:
// ```gad
// var value
// const prop = Property("x", () => value, (v) => {value = v})
// met prop(v int) => {value = "int value= " + v}
// ```

// TProperty is the builtin `Property` object type.
var TProperty = NewBuiltinObjType("Property").WithNew(NewPropertyFunc)

type Property struct {
	Module   *ModuleSpec
	PropName string
	f        *FuncSpec
}

func NewProperty(module *ModuleSpec, name string) *Property {
	p := &Property{Module: module, PropName: name}
	p.f = NewFuncSpec(p)
	return p
}

func (p *Property) SetModule(m *ModuleSpec) {
	p.Module = m
}

func (p *Property) GetModule() *ModuleSpec {
	return p.Module
}

func (p *Property) FullName() string {
	return p.Module.Name + "." + p.PropName
}

func (p *Property) FuncSpecName() string {
	return "property " + repr.Quote(p.FullName())
}

func (p *Property) ToString() string {
	return p.String()
}

func (p *Property) String() string {
	return string(MustToStr(nil, p))
}

func (p *Property) Print(state *PrinterState) (err error) {
	if !state.IsRepr {
		return state.WriteString(fmt.Sprintf("property %s", repr.Quote(p.FullName())))
	}
	return p.f.PrintFuncWrapper(state, p)
}

// AddMethodByTypes register property methods.
// - **getter** no have params and return value: `handler() <ret>`
// - **setter** have one param and not return value: `handler(v)`
func (p *Property) AddMethodByTypes(_ *VM, argTypes ParamsTypes, handler CallerObject, override bool, onAdd func(method *TypedCallerMethod) error) error {
	switch len(argTypes) {
	case 0:
		// getter
		return p.AddGetter(handler, onAdd)
	case 2:
		// setter
		return p.AddSetter(handler, argTypes[0], override, onAdd)
	default:
		return ErrProperty.NewErrorf("Getter or Setter of property %s requires 0 (getter) or 1 parameter (setter)", p.FullName())
	}
}

func (p *Property) IsFalsy() bool {
	return p.f.IsFalsy()
}

func (p *Property) Type() ObjectType {
	return TProperty
}

func (p *Property) Equal(right Object) bool {
	if ot, ok := right.(*Property); ok && ot == p {
		return true
	}
	return false
}

func (p Property) Clone() *Property {
	cp := &p
	cp.f = cp.f.CopyWithTarget(cp)
	return &p
}

func (p *Property) Name() string {
	return p.PropName
}

func (p *Property) Add(handler CallerObject, argTypes ParamsTypes) (err error) {
	return p.AddMethodByTypes(nil, argTypes, handler, false, nil)
}

func (p *Property) AddGetter(v Object, onAdd func(method *TypedCallerMethod) error) (err error) {
	if IsFunction(v) {
		err = p.f.Methods.Add(nil, NewCallerMethod(p, v.(CallerObject)), false, onAdd)
	} else {
		err = ErrProperty.NewErrorf("Getter of property %s is not a raw caller object", ReprQuote(p.FullName()))
	}
	return
}

func (p *Property) AddSetter(v Object, valueType ParamTypes, override bool, onAdd func(method *TypedCallerMethod) error) (err error) {
	if IsFunction(v) {
		err = p.f.Methods.Add(ParamsTypes{valueType}, NewCallerMethod(p, v.(CallerObject)), override, onAdd)
	} else {
		err = ErrProperty.NewErrorf("Setter of property %s is not a raw caller object", ReprQuote(p.FullName()))
	}
	return
}

func (p *Property) VMAdd(vm *VM, v Object) error {
	return SplitCaller(vm, v, func(co CallerObject, types ParamsTypes) error {
		return p.Add(co, types)
	})
}

// NewPropertyFunc create property instance.
func NewPropertyFunc(c Call) (_ Object, err error) {
	var (
		name = &Arg{Name: "name", TypeAssertion: TypeAssertionFromTypes(TStr, TRawStr)}
		left Array
	)

	if left, err = c.Args.DestructureVar(name); err != nil {
		return
	}

	p := NewProperty(c.VM.CurrentModuleSpec(), name.Value.ToString())

	if err = p.VMAdd(c.VM, left); err != nil {
		return
	}

	return p, nil
}
