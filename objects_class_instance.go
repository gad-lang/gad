package gad

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/gad-lang/gad/utils"
	"github.com/gad-lang/gad/zeroer"
)

const ObjectMethodsGetterFieldName = "__methods__"

var (
	_ Object           = (*ClassInstance)(nil)
	_ Copier           = (*ClassInstance)(nil)
	_ KeysGetter       = (*ClassInstance)(nil)
	_ ValuesGetter     = (*ClassInstance)(nil)
	_ ItemsGetter      = (*ClassInstance)(nil)
	_ IndexGetter      = (*ClassInstance)(nil)
	_ IndexSetter      = (*ClassInstance)(nil)
	_ Printer          = (*ClassInstance)(nil)
	_ NameCallerObject = (*ClassInstance)(nil)
	_ CallerObject     = (*ClassInstance)(nil)
)

// ClassInstance represents map of objects and implements Object interface.
type ClassInstance struct {
	fields       Dict
	parents      map[string]*ClassInstance
	class        *Class
	newCallStack []CallerObject
}

func (o *ClassInstance) Call(c Call) (_ Object, err error) {
	if o.fields == nil {
		c.Args = append(Args{Array{o}}, c.Args...)
		caller, validate := o.class.new.f.CallerMethodWithValidationCheckOfArgs(c.Args)

		if caller == nil {
			return nil, ErrConstructorMethodFound.NewErrorf("no constructor found for params types %s", c.Args.Types())
		}

		for i := len(o.newCallStack) - 1; i >= 0; i-- {
			if o.newCallStack[i] == caller {
				if types, _ := ParamTypesOfRawCaller(c.VM, caller); len(types) == 1 {
					// if default is overrided
					if caller != o.class.new.f.defaul {
						caller = o.class.new.f.defaul
						break
					}
				}
				return nil, ErrConstructorRecursiveCall.NewError(caller.ToString())
			}
		}

		c.SafeArgs = !validate

		o.newCallStack = append(o.newCallStack, caller)
		defer func() {
			o.newCallStack = o.newCallStack[:len(o.newCallStack)-1]
		}()

		if _, err = DoCall(caller, c); err != nil {
			return nil, err
		}
		return o, nil
	} else {
		return nil, ErrClassInstanceInitialized
	}
}

func (o *ClassInstance) Name() string {
	return o.ReprTypeName()
}

func (o *ClassInstance) Init(vm *VM, fields Dict) (err error) {
	o.fields = make(Dict, len(o.class.fieldsMap))
	o.parents = make(map[string]*ClassInstance, len(o.class.parents))

	parentsFields, _ := fields["@parents"].(Dict)

	if parentsFields != nil {
		delete(fields, "@parents")
	}

	for i, fd := range o.class.fieldDefaults {
		if _, err = fd.Call(Call{
			SafeArgs: true,
			VM:       vm,
			Args:     Args{Array{o.fields}},
		}); err != nil {
			return ErrNewClassInstance.NewErrorf("initialize field defaults[%d]: %v", i, err)
		}
	}

	for name, v := range fields {
		o.fields[name] = v
	}

	for name, field := range o.class.fieldsMap {
		if _, ok := o.fields[name]; !ok {
			if field.Value == nil {
				o.fields[name] = Nil
			} else {
				switch t := field.Value.(type) {
				case *ComputedValue:
					o.fields[name], err = DoCall(t.CallerObject, Call{VM: vm})
				default:
					o.fields[name] = field.Value
				}
			}
		}
	}

	for _, parent := range o.class.parents {
		fields, _ := parentsFields[parent.Alias].(Dict)

		if o.parents[parent.Alias], err = parent.Type.NewInstanceWithFields(vm, fields); err != nil {
			return
		}
	}

	return
}

func (o *ClassInstance) ReprTypeName() string {
	return "class instance of " + ReprQuote(o.class.FullName())
}

func (o *ClassInstance) Instances() func(yield func(instance *ClassInstance) bool) {
	return func(yield func(*ClassInstance) bool) {
		if !yield(o) {
			return
		}
		for i := len(o.class.parents) - 1; i >= 0; i-- {
			if !yield(o.parents[o.class.parents[i].Alias]) {
				return
			}
		}
	}
}

func (o *ClassInstance) WalkInstances(cb func(path []*ClassInstance, instance *ClassInstance) (mode utils.WalkMode)) {
	utils.Walk(o,
		func(e *ClassInstance, cb func(*ClassInstance) utils.WalkMode) bool {
			var mode utils.WalkMode
			for i := len(o.class.parents) - 1; i >= 0; i-- {
				mode = cb(o.parents[o.class.parents[i].Alias])
				switch mode {
				case utils.WalkModeBreak:
					return false
				case utils.WalkModeSkipSiblings:
					return true
				}
			}
			return true
		}, cb)
}

func (o *ClassInstance) Type() ObjectType {
	return o.class
}

func (o *ClassInstance) Fields() Dict {
	return o.fields
}

func (o *ClassInstance) Parents() (d Dict) {
	d = make(Dict, len(o.parents))
	for name, instance := range o.parents {
		d[name] = instance
	}
	return d
}

func (o *ClassInstance) ToString() string {
	return o.class.Name() + o.fields.Filter(func(k string, v Object) bool {
		return !zeroer.IsZero(v)
	}).ToString()
}

// CopyInstance copy this instance.
func (o ClassInstance) CopyInstance() *ClassInstance {
	o.fields = Copy(o.fields)
	o.parents = make(map[string]*ClassInstance, len(o.parents))
	for k, v := range o.parents {
		o.parents[k] = v.CopyInstance()
	}
	return &o
}

// Copy implements Copier interface.
func (o *ClassInstance) Copy() Object {
	return o.CopyInstance()
}

// DeepCopyInstance deep copy this instance.
func (o ClassInstance) DeepCopyInstance(vm *VM) (_ *ClassInstance, err error) {
	if o.fields, err = DeepCopy(vm, o.fields); err != nil {
		return
	}

	o.parents = make(map[string]*ClassInstance, len(o.parents))

	for k, v := range o.parents {
		if o.parents[k], err = v.DeepCopyInstance(vm); err != nil {
			return
		}
	}

	return &o, nil
}

// DeepCopy implements DeepCopier interface.
func (o *ClassInstance) DeepCopy(vm *VM) (r Object, err error) {
	return o.DeepCopyInstance(vm)
}

func (o *ClassInstance) ResolveProperty(name string) (inst *ClassInstance, p *ClassProperty) {
	for inst = range o.Instances() {
		if p = inst.class.propertiesMap[name]; p != nil {
			return
		}
	}
	inst = nil
	return
}

func (o *ClassInstance) WalkProperty(name string, f func(inst *ClassInstance, p *ClassProperty) (next bool)) {
	for inst := range o.Instances() {
		if p := inst.class.propertiesMap[name]; p != nil {
			if !f(inst, p) {
				return
			}
		}
	}
}

func (o *ClassInstance) ResolveField(name string) (inst *ClassInstance) {
	for inst = range o.Instances() {
		if _, ok := inst.fields[name]; ok {
			return inst
		}
	}
	return nil
}

func (o *ClassInstance) GetFieldValue(vm *VM, name string) (Object, error) {
	if p, wo := o.GetPropertyGetter(name); p != nil {
		return YieldCall(p, &Call{VM: vm}), nil
	} else if wo {
		return nil, NewStructPropertyInstanceError(name, "write only")
	}

	if inst := o.ResolveField(name); inst != nil {
		return inst.fields[name], nil
	}

	return nil, ErrInvalidIndex.NewError(name)
}

func (o *ClassInstance) SetFieldValue(vm *VM, name string, value Object) error {
	if p, valid := o.GetPropertySetter(name, value.Type()); p != nil {
		_, err := Val(p.Call(Call{VM: vm, Args: Args{Array{value}}}))
		return err
	} else if valid {
		return NewStructPropertyInstanceError(name, "no has setter")
	}

	if inst := o.ResolveField(name); inst != nil {
		inst.fields[name] = value
	} else {
		o.fields[name] = value
	}

	return nil
}

func (o *ClassInstance) Methods() *IndexGetProxy {
	names := func() []string {
		var (
			s = make([]string, len(o.class.methodsMap))
			i int
		)
		for name := range o.class.methodsMap {
			s[i] = name
			i++
		}

		sort.Strings(s)

		return s
	}

	toArray := func() Array {
		arr := make(Array, len(o.class.methodsMap))

		for i, name := range names() {
			arr[i] = o.class.methodsMap[name]
		}
		return arr
	}

	return &IndexGetProxy{
		ToStrFunc: func() string {
			return toArray().ToString()
		},
		PrintFunc: func(s *PrinterState) error {
			return toArray().Print(s)
		},
		IterateFunc: func(vm *VM, na *NamedArgs) Iterator {
			return toArray().Iterate(vm, na)
		},
		GetIndexFunc: func(vm *VM, index Object) (value Object, err error) {
			name := index.ToString()
			if m := o.GetMethod(name); m != nil {
				return m, nil
			}
			return nil, ErrInvalidIndex.NewError(name)
		},
		CallNameFunc: func(name string, c Call) (Object, error) {
			if m := o.GetMethod(name); m != nil {
				return YieldCall(m, &c), nil
			}
			return nil, ErrInvalidIndex.NewError(name)
		},
	}
}

func (o *ClassInstance) Parent(name string) *ClassInstance {
	return o.parents[name]
}

// IndexSet implements Object interface.
func (o *ClassInstance) IndexSet(vm *VM, index, value Object) (err error) {
	return o.SetFieldValue(vm, index.ToString(), value)
}

// IndexGet implements Object interface.
func (o *ClassInstance) IndexGet(vm *VM, index Object) (Object, error) {
	name := index.ToString()
	switch name {
	case "@parents":
		return o.Parents(), nil
	}
	return o.GetFieldValue(vm, name)
}

// Equal implements Object interface.
func (o *ClassInstance) Equal(right Object) bool {
	v, ok := right.(*ClassInstance)
	if !ok {
		return false
	}
	return o.class.Equal(v.class) && o.fields.Equal(v.fields)
}

// IsFalsy implements Object interface.
func (o *ClassInstance) IsFalsy() bool {
	return len(o.fields) == 0
}

func (o *ClassInstance) Items(vm *VM, cb ItemsGetterCallback) (err error) {
	return o.fields.Items(vm, cb)
}

func (o *ClassInstance) Keys() Array {
	return o.fields.Keys()
}

func (o *ClassInstance) Values() Array {
	return o.fields.Values()
}

func (o *ClassInstance) ResolveMethod(name string) (inst *ClassInstance, m *ClassMethod) {
	for inst = range o.Instances() {
		if m = inst.class.methodsMap[name]; m != nil {
			return
		}
	}
	inst = nil
	return
}

func (o *ClassInstance) GetMethod(name string) CallerObject {
	if inst, m := o.ResolveMethod(name); m != nil {
		return &Function{
			FuncName: name,
			ToStringFunc: func() string {
				return ReprQuote("structInstanceMethod of " + o.class.ToString() + "#" + name)
			},
			Value: func(c Call) (Object, error) {
				c.Args = append([]Array{{inst}}, c.Args...)
				return YieldCall(m, &c), nil
			},
		}
	}
	return nil
}

func (o *ClassInstance) GetPropertySetter(name string, typ ObjectType) (handler CallerObject, valid bool) {
	o.WalkProperty(name, func(inst *ClassInstance, p *ClassProperty) bool {
		valid = true
		if f, _ := p.f.CallerMethodWithValidationCheckOfArgsTypes(ObjectTypeArray{o.class, typ}); f != nil {
			handler = &Function{
				FuncName: name,
				ToStringFunc: func() string {
					return ReprQuote("structInstancePropertySetter of " + o.class.ToString() + "#" + name + " as " + f.ToString())
				},
				Value: func(c Call) (Object, error) {
					c.Args = append([]Array{{inst}}, c.Args...)
					return YieldCall(f, &c), nil
				},
			}
			return false
		}
		return true
	})
	return
}

func (o *ClassInstance) GetPropertyGetter(name string) (handler CallerObject, valid bool) {
	o.WalkProperty(name, func(inst *ClassInstance, p *ClassProperty) bool {
		valid = true
		if f, _ := p.f.CallerMethodWithValidationCheckOfArgsTypes(ObjectTypeArray{o.class}); f != nil {
			handler = &Function{
				FuncName: name,
				ToStringFunc: func() string {
					return ReprQuote("structInstancePropertyGetter of " + o.class.ToString() + "#" + name + " as " + f.ToString())
				},
				Value: func(c Call) (Object, error) {
					c.Args = append([]Array{{inst}}, c.Args...)
					return YieldCall(f, &c), nil
				},
			}
			return false
		}
		return true
	})
	return
}

func (o *ClassInstance) CallName(name string, c Call) (_ Object, err error) {
	switch name {
	case "@print":
		return Nil, o.CallPrint(c)
	case "@new":
		c.Args = append(Args{Array{o}}, c.Args...)
		return o.class.Construct(c)
	}

	if m := o.GetMethod(name); m != nil {
		return YieldCall(m, &c), nil
	}
	var v Object
	if v, err = o.IndexGet(c.VM, Str(name)); err != nil {
		return
	}
	if Callable(v) {
		return YieldCall(v.(CallerObject), &c), nil
	}
	return nil, ErrNotCallable.NewError("func " + strconv.Quote(name) + " of type " + v.Type().Name())
}

func (o *ClassInstance) Cast(t ObjectType) *ClassInstance {
	for inst := range o.Instances() {
		if inst.class == t {
			return inst
		}
	}
	return nil
}

func (o *ClassInstance) CastTo(_ *VM, t ObjectType) (Object, error) {
	if inst := o.Cast(t); inst != nil {
		return inst, nil
	}
	return nil, ErrIncompatibleCast
}

func (o *ClassInstance) ToDict() (d Dict) {
	d = Copy(o.fields)
	o.WalkInstances(func(path []*ClassInstance, instance *ClassInstance) (mode utils.WalkMode) {
		for name, value := range instance.fields {
			if d[name] == nil {
				d[name] = value
			}
		}
		return
	})
	return
}

func (o *ClassInstance) Print(state *PrinterState) error {
	if !state.IsRepr {
		defer state.WrapRepr(o)()
	}
	return o.ToDict().PrintObject(state, o)
}

func (o *ClassInstance) CallPrint(c Call) (err error) {
	var (
		state = &Arg{
			Name:          "printerState",
			TypeAssertion: TypeAssertionFromTypes(TPrinterState),
		}
	)

	if err = c.Args.Destructure(state); err != nil {
		return
	}

	return o.ToDict().PrintObject(state.Value.(*PrinterState), o)
}

type ClassInstanceMethod struct {
	this   *ClassInstance
	method *ClassMethod
}

func (m *ClassInstanceMethod) Call(c Call) (Object, error) {
	c.Args = append([]Array{{m.this}}, c.Args...)
	return m.method.f.Call(c)
}

func (m *ClassInstanceMethod) Equal(right Object) bool {
	if r, _ := right.(*ClassInstanceMethod); r == m {
		return true
	}
	return false
}

func (m *ClassInstanceMethod) IsFalsy() bool {
	return false
}

func (m *ClassInstanceMethod) Type() ObjectType {
	return TClassInstanceMethod
}

func (m *ClassInstanceMethod) Name() string {
	return m.method.name
}

func (m *ClassInstanceMethod) FullName() string {
	return m.method.FullName()
}

func (m *ClassInstanceMethod) FuncSpecName() string {
	return "class instance method " + ReprQuote(m.FullName())
}

func (m *ClassInstanceMethod) ToString() string {
	return m.String()
}

func (m *ClassInstanceMethod) String() string {
	return string(MustToStr(nil, m))
}

func (m *ClassInstanceMethod) Print(state *PrinterState) (err error) {
	return m.method.f.PrintFuncWrapper(state, m)
}

var (
	_ Object       = (*ClassInstancePropertyGetter)(nil)
	_ CallerObject = (*ClassInstancePropertyGetter)(nil)
	_ IndexGetter  = (*ClassInstancePropertyGetter)(nil)
)

type ClassInstancePropertyGetter struct {
	this *ClassInstance
	p    *ClassProperty
	h    *TypedCallerMethod
}

func (m *ClassInstancePropertyGetter) Call(c Call) (Object, error) {
	c.Args = Args{Array{m.this}}
	return m.h.Call(c)
}

func (m *ClassInstancePropertyGetter) Name() string {
	return m.p.name
}

func (m *ClassInstancePropertyGetter) Equal(right Object) bool {
	if r, _ := right.(*ClassInstancePropertyGetter); r != nil {
		return r == m || (m.this.Equal(r.this) && m.p.Equal(r.p))
	}
	return false
}

func (m *ClassInstancePropertyGetter) IsFalsy() bool {
	return false
}

func (m *ClassInstancePropertyGetter) Type() ObjectType {
	return TClassInstancePropertyGetter
}

func (m *ClassInstancePropertyGetter) String() string {
	return string(MustToStr(nil, m))
}

func (m *ClassInstancePropertyGetter) ToString() string {
	return string(MustToStr(nil, m))
}

func (m *ClassInstancePropertyGetter) Print(state *PrinterState) (err error) {
	return state.WriteString(fmt.Sprintf("ClassInstancePropertyGetter %s of %v", ReprQuote(m.Name()), ReprQuote(fmt.Sprintf("%s %p", m.this.class.FullName(), m.this))))
}

func (m *ClassInstancePropertyGetter) IndexGet(vm *VM, index Object) (value Object, err error) {
	key := index.ToString()
	switch key {
	case "this":
		return m.this, nil
	case "property":
		return m.p, nil
	case "caller":
		return m.h, nil
	default:
		return nil, ErrInvalidIndex.NewError(index.ToString())
	}
}

var (
	_ Object      = (*ClassInstancePropertySetter)(nil)
	_ IndexGetter = (*ClassInstancePropertySetter)(nil)
)

type ClassInstancePropertySetter struct {
	p    *ClassProperty
	this *ClassInstance
	vm   *VM
}

func (m *ClassInstancePropertySetter) Set(v Object) (err error) {
	return m.this.SetFieldValue(m.vm, m.p.name, v)
}

func (m *ClassInstancePropertySetter) Name() string {
	return m.p.name
}

func (m *ClassInstancePropertySetter) Equal(right Object) bool {
	if r, _ := right.(*ClassInstancePropertySetter); r != nil {
		return r == m || (m.this.Equal(r.this) && m.p.name == r.p.name)
	}
	return false
}

func (m *ClassInstancePropertySetter) IsFalsy() bool {
	return false
}

func (m *ClassInstancePropertySetter) Type() ObjectType {
	return TClassInstancePropertySetter
}

func (m *ClassInstancePropertySetter) String() string {
	return string(MustToStr(nil, m))
}

func (m *ClassInstancePropertySetter) ToString() string {
	return string(MustToStr(nil, m))
}

func (m *ClassInstancePropertySetter) Print(state *PrinterState) (err error) {
	return state.WriteString(fmt.Sprintf("ClassInstancePropertySetter %s of %v", ReprQuote(m.p.name), ReprQuote(fmt.Sprintf("%s %p", m.this.class.FullName(), m.this))))
}

func (m *ClassInstancePropertySetter) IndexGet(vm *VM, index Object) (value Object, err error) {
	key := index.ToString()
	switch key {
	case "name":
		return Str(m.p.name), err
	case "this":
		return m.this, nil
	default:
		return nil, ErrInvalidIndex.NewError(index.ToString())
	}
}
