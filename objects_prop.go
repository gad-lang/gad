package gad

import (
	"fmt"

	"github.com/gad-lang/gad/repr"
)

// gad:doc
// ## Type Prop
// Prop is a named, callable value backed by getter and setter methods.
//
//	Prop(name str, *methods) -> Prop
//
// The trailing methods are dispatched by their signature when the property is
// called:
//   - **getter**: takes no parameters and returns a value (`prop() -> value`).
//     At most one getter may be registered.
//   - **setter**: takes one parameter and returns nothing (`prop(v)`). Any
//     number of setters may be registered; the one whose parameter type matches
//     the argument is selected.
//
// A property may be created with no methods, but calling such a property is an
// error because no matching method exists. New methods can be attached later
// with the `met` statement.
//
// Example — getter/setter pair plus a typed setter:
//
// ```gad
// var value
// const p = Prop("x", () => value, (v) => {value = v})
// met p(v int) {
//   value = "int value= " + v
// }
// p()      // nil
// p("a")   // setter: value = "a"
// p()      // "a"
// p(1)     // typed setter selected: value = "int value= 1"
// p()      // "int value= 1"
// ```
//
// Example — read-only (getter-only) property:
//
// ```gad
// const pi = Prop("pi", () => 3.14)
// pi()        // 3.14
// ```

// TProp is the builtin `Prop` object type.
var TProp = RegisterBuiltinType(BuiltinProp, "Prop", Prop{}, NewPropFunc)

var _ CallerObject = (*Prop)(nil)

// Prop is a named, callable Object whose invocations are dispatched to
// getter and setter methods held in its FuncSpec. Calling it with no arguments
// runs the getter; calling it with one argument runs the setter whose parameter
// type matches the argument. It implements CallerObject, so a Prop value can
// be called directly like a function.
type Prop struct {
	Module   *ModuleSpec
	PropName string
	f        *FuncSpec
}

// NewProp returns a method-less Prop named name bound to module. Use
// AddGetter, AddSetter or AddMethodByTypes to attach behaviour.
func NewProp(module *ModuleSpec, name string) *Prop {
	p := &Prop{Module: module, PropName: name}
	p.f = NewFuncSpec(p)
	return p
}

func (p *Prop) SetModule(m *ModuleSpec) {
	p.Module = m
}

func (p *Prop) GetModule() *ModuleSpec {
	return p.Module
}

func (p *Prop) FullName() string {
	return p.Module.Name + "." + p.PropName
}

func (p *Prop) FuncSpecName() string {
	return "prop " + repr.Quote(p.FullName())
}

func (p *Prop) ToString() string {
	return p.String()
}

func (p *Prop) String() string {
	return string(MustToStr(nil, p))
}

func (p *Prop) Print(state *PrinterState) (err error) {
	if !state.IsRepr {
		return state.WriteString(fmt.Sprintf("prop %s", repr.Quote(p.FullName())))
	}
	return p.f.PrintFuncWrapper(state, p)
}

// AddMethodByTypes register prop methods.
// - **getter** no have params and return value: `handler() <ret>`
// - **setter** have one param and not return value: `handler(v)`
func (p *Prop) AddMethodByTypes(_ *VM, argTypes ParamsTypes, handler CallerObject, override bool, onAdd func(method *TypedCallerMethod) error) error {
	switch len(argTypes) {
	case 0:
		// getter
		return p.AddGetter(handler, onAdd)
	case 1:
		// setter
		return p.AddSetter(handler, argTypes[0], override, onAdd)
	default:
		return ErrProp.NewErrorf("Getter or Setter of prop %s requires 0 (getter) or 1 parameter (setter)", p.FullName())
	}
}

func (p *Prop) IsFalsy() bool {
	return p.f.IsFalsy()
}

func (p *Prop) Type() ObjectType {
	return TProp
}

func (p *Prop) Equal(right Object) bool {
	if ot, ok := right.(*Prop); ok && ot == p {
		return true
	}
	return false
}

func (p Prop) Clone() *Prop {
	cp := &p
	cp.f = cp.f.CopyWithTarget(cp)
	return &p
}

func (p *Prop) Name() string {
	return p.PropName
}

// Call dispatches a prop invocation through its FuncSpec methods (like
// *Func): no args invokes the getter, one arg the matching setter.
func (p *Prop) Call(c Call) (Object, error) {
	return p.f.Call(c)
}

func (p *Prop) Add(handler CallerObject, argTypes ParamsTypes) (err error) {
	return p.AddMethodByTypes(nil, argTypes, handler, false, nil)
}

func (p *Prop) AddGetter(v Object, onAdd func(method *TypedCallerMethod) error) (err error) {
	if IsFunction(v) {
		err = p.f.Methods.Add(nil, NewCallerMethod(p, v.(CallerObject)), false, onAdd)
	} else {
		err = ErrProp.NewErrorf("Getter of prop %s is not a raw caller object", ReprQuote(p.FullName()))
	}
	return
}

func (p *Prop) AddSetter(v Object, valueType ParamTypes, override bool, onAdd func(method *TypedCallerMethod) error) (err error) {
	if IsFunction(v) {
		err = p.f.Methods.Add(ParamsTypes{valueType}, NewCallerMethod(p, v.(CallerObject)), override, onAdd)
	} else {
		err = ErrProp.NewErrorf("Setter of prop %s is not a raw caller object", ReprQuote(p.FullName()))
	}
	return
}

func (p *Prop) VMAdd(vm *VM, v Object) error {
	return SplitCaller(vm, v, func(co CallerObject, types ParamsTypes) error {
		return p.Add(co, types)
	})
}

// NewPropFunc create prop instance.
func NewPropFunc(c Call) (_ Object, err error) {
	var (
		name = &Arg{Name: "name", TypeAssertion: TypeAssertionFromTypes(TStr, TRawStr)}
		left Array
	)

	if left, err = c.Args.DestructureVar(name); err != nil {
		return
	}

	p := NewProp(c.VM.CurrentModuleSpec(), name.Value.ToString())

	if err = p.VMAdd(c.VM, left); err != nil {
		return
	}

	return p, nil
}
