package gad

import (
	"fmt"
	"strconv"

	"github.com/gad-lang/gad/repr"
	"github.com/gad-lang/gad/token"
)

type ClassParent struct {
	Alias string
	Type  *Class
}

type ClassField struct {
	class *Class
	Name  string
	Types ObjectTypes
	index int
	Value Object
}

func (f *ClassField) IsFalsy() bool {
	return false
}

func (f *ClassField) Type() ObjectType {
	return TClassField
}

func (f *ClassField) String() string {
	return string(MustToStr(nil, f))
}

func (f *ClassField) ToString() string {
	return f.String()
}

func (f *ClassField) ReprTypeName() string {
	return "class field " + ReprQuote(f.Name) + " of " + ReprQuote(f.class.FullName())
}

func (f *ClassField) Print(state *PrinterState) (err error) {
	defer state.WrapRepr(f)()

	if len(f.Types) > 0 {
		state.WriteString(f.Types.String())
	} else {
		state.WriteString(TAny.name)
	}
	if f.Value != nil {
		state.WriteString(" = ")
		return state.Print(f.Value)
	}
	return
}

func (f *ClassField) Equal(right Object) bool {
	if r, _ := right.(*ClassField); r != nil {
		return r.class == f.class && r.Name == f.Name
	}
	return false
}

type ClasPropertySetter struct {
	class      *Class
	Handler    CallerObject
	ValueTypes ObjectTypes
}

func (s *ClasPropertySetter) ToParamTypes() (t ParamsTypes) {
	return ParamsTypes{
		ObjectTypes{s.class},
		s.ValueTypes,
	}
}

type ClassGoMethodHandler struct {
	Handler    CallerObject
	ParamTypes ParamsTypes
}

type ClassGoMethod struct {
	Name     string
	Handlers []*ClassGoMethodHandler
}

var (
	_ Object      = (*ClassProperty)(nil)
	_ MethodAdder = (*ClassProperty)(nil)
)

type ClassProperty struct {
	class *Class
	name  string
	f     *FuncSpec
}

func NewClassProperty(class *Class, name string) *ClassProperty {
	p := &ClassProperty{class: class, name: name}
	p.f = NewFuncSpec(p)
	return p
}

func (p *ClassProperty) FullName() string {
	return p.class.FullName() + "#" + p.name
}

func (p *ClassProperty) GetModule() *Module {
	return nil
}

func (p *ClassProperty) FuncSpecName() string {
	return "class property " + repr.Quote(p.FullName())
}

func (p *ClassProperty) ToString() string {
	return p.String()
}

func (p *ClassProperty) String() string {
	return string(MustToStr(nil, p))
}

func (p *ClassProperty) Print(state *PrinterState) (err error) {
	if !state.IsRepr {
		return state.WriteString(fmt.Sprintf("class property %s of %s", repr.Quote(p.FullName()), p.class.FullName()))
	}
	return p.f.PrintFuncWrapper(state, p)
}

func (p *ClassProperty) AddMethodByTypes(vm *VM, argTypes ParamsTypes, handler CallerObject, override bool, onAdd func(method *TypedCallerMethod) error) error {
	switch len(argTypes) {
	case 1:
		return p.AddGetter(handler, argTypes[0], override, onAdd)
	case 2:
		return p.AddSetter(handler, argTypes[0], argTypes[1], override, onAdd)
	default:
		return ErrClassPropertyChange.NewErrorf("Getter or Setter of property %s requires 1 and 2 parameters", p.FullName())
	}
}

func (p *ClassProperty) IsFalsy() bool {
	return p.f.IsFalsy()
}

func (p *ClassProperty) Type() ObjectType {
	return TClassProperty
}

func (p *ClassProperty) Equal(right Object) bool {
	if ot, ok := right.(*ClassProperty); ok && ot == p {
		return true
	}
	return false
}

func (p ClassProperty) Clone() *ClassProperty {
	cp := &p
	cp.f = cp.f.CopyWithTarget(cp)
	return &p
}

func (p *ClassProperty) Name() string {
	return p.name
}

func (p *ClassProperty) Add(handler CallerObject, argTypes ParamsTypes) (err error) {
	return p.AddMethodByTypes(nil, argTypes, handler, false, nil)
}

func (p *ClassProperty) AddGetter(v Object, thisType ParamTypes, override bool, onAdd func(method *TypedCallerMethod) error) (err error) {
	if IsFunction(v) {
		v := v.(CallerObject)
		err = p.f.Methods.Add(ParamsTypes{thisType}, NewCallerMethod(p.class, v), override, onAdd)
	} else {
		err = ErrClassPropertyChange.NewErrorf("Getter of property %s is not a raw caller object", ReprQuote(p.FullName()))
	}
	return
}

func (p *ClassProperty) AddSetter(v Object, thisType, valueType ParamTypes, override bool, onAdd func(method *TypedCallerMethod) error) (err error) {
	if IsFunction(v) {
		v := v.(CallerObject)
		err = p.f.Methods.Add(ParamsTypes{thisType, valueType}, NewCallerMethod(p.class, v), override, onAdd)
	} else {
		err = ErrClassPropertyChange.NewErrorf("Setter of property %s is not a raw caller object", ReprQuote(p.FullName()))
	}
	return
}

func (p *ClassProperty) VMAdd(vm *VM, v Object) error {
	return SplitCaller(vm, v, func(co CallerObject, types ParamsTypes) error {
		return p.Add(co, types)
	})
}

func (p *ClassProperty) ReprTypeName() string {
	return "class property " + ReprQuote(p.Name()) + " of " + ReprQuote(p.class.FullName())
}

func (p *ClassProperty) NewGetter(this *ClassInstance) *ClassInstancePropertyGetter {
	co := p.f.Methods.GetMethod(ObjectTypeArray{this.class})
	if co == nil {
		return nil
	}
	return &ClassInstancePropertyGetter{
		this: this,
		p:    p,
		h:    co,
	}
}

func (p *ClassProperty) NewSetter(vm *VM, this *ClassInstance) *ClassInstancePropertySetter {
	s := &ClassInstancePropertySetter{
		p:    p,
		this: this,
		vm:   vm,
	}
	return s
}

var (
	_ Object       = (*ClassConstructor)(nil)
	_ MethodAdder  = (*ClassConstructor)(nil)
	_ CallerObject = (*ClassConstructor)(nil)
)

type ClassConstructor struct {
	class *Class
	f     *FuncSpec
}

func (c *ClassConstructor) Call(c2 Call) (Object, error) {
	return c.f.Call(c2)
}

func (c *ClassConstructor) Name() string {
	return "classConstructor"
}

func (c *ClassConstructor) FullName() string {
	return c.class.FullName() + "#" + c.Name()
}

func (c *ClassConstructor) AddMethodByTypes(vm *VM, argTypes ParamsTypes, handler CallerObject, override bool, onAdd func(method *TypedCallerMethod) error) error {
	if len(argTypes) < 1 {
		return ErrClassMethodRegister.NewErrorf("Method %s for %s requires minimum 1 parameter for Class value", ReprQuote(handler.Name()), ReprQuote(c.FullName()))
	}
	return c.f.AddMethodByTypes(vm, argTypes, handler, override, onAdd)
}

func (c *ClassConstructor) IsFalsy() bool {
	return c.f.IsFalsy()
}

func (c *ClassConstructor) Type() ObjectType {
	return TClassConstructor
}

func (c *ClassConstructor) Print(state *PrinterState) (err error) {
	return c.f.PrintFuncWrapper(state, c)
}

func (c *ClassConstructor) GetModule() *Module {
	return nil
}

func (c *ClassConstructor) FuncSpecName() string {
	return "class constructor of " + ReprQuote(c.class.FullName())
}

func (c *ClassConstructor) ToString() string {
	return c.String()
}

func (c *ClassConstructor) String() string {
	return string(MustToStr(nil, c))
}

func (c *ClassConstructor) Equal(right Object) bool {
	return right == c
}

var (
	_ Object       = (*ClassMethod)(nil)
	_ MethodAdder  = (*ClassMethod)(nil)
	_ CallerObject = (*ClassMethod)(nil)
)

type ClassMethod struct {
	class *Class
	f     *FuncSpec
	name  string
}

func (m *ClassMethod) IsFalsy() bool {
	return false
}

func (m *ClassMethod) ReprTypeName() string {
	return "class method " + ReprQuote(m.Name()) + " of " + ReprQuote(m.class.FullName())
}

func NewClassMethod(name string, class *Class) *ClassMethod {
	m := &ClassMethod{class: class, name: name}
	m.f = NewFuncSpec(m)
	return m
}

func (m *ClassMethod) NewInstance(this *ClassInstance) *ClassInstanceMethod {
	return &ClassInstanceMethod{
		this:   this,
		method: m,
	}
}

func (m *ClassMethod) Call(c Call) (Object, error) {
	return m.f.Call(c)
}

func (m *ClassMethod) Name() string {
	return m.name
}

func (m *ClassMethod) FullName() string {
	return m.class.FullName() + "#" + m.Name()
}

func (m *ClassMethod) Equal(right Object) bool {
	if r, _ := right.(*ClassMethod); r == m {
		return true
	}
	return false
}

func (m *ClassMethod) AddMethodByTypes(vm *VM, argTypes ParamsTypes, handler CallerObject, override bool, onAdd func(method *TypedCallerMethod) error) error {
	if len(argTypes) < 1 {
		return ErrClassMethodRegister.NewErrorf("Method %s for %s requires minimum 1 parameter for instance value", handler.ToString(), ReprQuote(m.FullName()))
	}
	return m.f.AddMethodByTypes(vm, argTypes, handler, override, onAdd)
}

func (m *ClassMethod) Type() ObjectType {
	return TClassMethod
}

func (m *ClassMethod) FuncSpecName() string {
	return "class method " + ReprQuote(m.FullName())
}

func (m *ClassMethod) ToString() string {
	return m.String()
}

func (m *ClassMethod) String() string {
	return string(MustToStr(nil, m))
}

func (m *ClassMethod) Print(state *PrinterState) (err error) {
	return m.f.PrintFuncWrapper(state, m)
}

var (
	_ Object           = (*Class)(nil)
	_ IndexGetter      = (*Class)(nil)
	_ CallerObject     = (*Class)(nil)
	_ NameCallerObject = (*Class)(nil)
	_ Printer          = (*Class)(nil)
	_ MethodAdder      = (*Class)(nil)
)

// Class represents type objects and implements Object interface.
type Class struct {
	new           *ClassConstructor
	name          string
	module        *Module
	parents       []*ClassParent
	fieldsMap     map[string]*ClassField
	propertiesMap map[string]*ClassProperty
	methodsMap    map[string]*ClassMethod
	fieldDefaults []CallerObject
}

func NewClass(name string, module *Module) (t *Class) {
	t = &Class{
		module:        module,
		name:          name,
		fieldsMap:     make(map[string]*ClassField),
		propertiesMap: make(map[string]*ClassProperty),
		methodsMap:    make(map[string]*ClassMethod),
		new:           &ClassConstructor{},
	}

	t.new.class = t
	t.new.f = NewFuncSpec(t.new)
	t.new.f.defaul = &Function{
		FuncName: ReprQuote("new"),
		Value:    t.Construct,
		Header: NewFunctionHeader().WithParams(func(newParam func(name string) *ParamBuilder) {
			newParam("this")
		}),
	}
	return
}

func (Class) GadObjectType() {}

func (t *Class) Name() string {
	return t.name
}

func (t *Class) FullName() string {
	if t.module == nil {
		return t.name
	}
	return t.module.info.Name + "." + t.name
}

func (t *Class) Type() ObjectType {
	return TClass
}

func (t *Class) Module() *Module {
	return t.module
}

func (t *Class) String() string {
	return TypeToString("class " + t.name)
}

func (t *Class) Extends(parent *Class, alias string) *Class {
	if alias == "" {
		alias = parent.name
	}
	t.parents = append(t.parents, &ClassParent{Type: parent, Alias: alias})
	return t
}

func (t *Class) AddMethod(name string) (_ *ClassMethod, err error) {
	if _, ok := t.methodsMap[name]; ok {
		return nil, ErrDefineClass.NewError(fmt.Sprintf("Duplicate method %q", name))
	}

	m := NewClassMethod(name, t)
	t.methodsMap[name] = m
	return m, nil
}

func (t *Class) Constructor() *ClassConstructor {
	return t.new
}

func (t *Class) AddField(field ...*ClassField) error {
	for _, field := range field {
		if _, ok := t.fieldsMap[field.Name]; ok {
			return ErrDefineClass.NewErrorf("duplicate field %q", field.Name)
		}

		t.fieldsMap[field.Name] = &ClassField{
			class: t,
			Name:  field.Name,
			Types: field.Types,
			Value: field.Value,
			index: len(t.fieldsMap),
		}
	}
	return nil
}

func (t *Class) AddProperty(name string, f *Func) (err error) {
	if _, ok := t.propertiesMap[name]; ok {
		return ErrClassPropertyRegister.NewErrorf("property %s already exists", name)
	}

	p := NewClassProperty(t, name)

	if err, _ = f.Methods.Walk(func(m *TypedCallerMethod) any {
		if err := p.f.AddMethodByTypes(nil, ObjectTypes(m.types).Multi(), m.CallerMethod, false, nil); err != nil {
			return err
		}
		return nil
	}).(error); err != nil {
		return
	}

	t.propertiesMap[name] = p
	return
}

func (t *Class) AddCallerMethod(_ *VM, types ParamsTypes, handler CallerObject, override bool, onAdd func(tcm *TypedCallerMethod) error) (err error) {
	if len(types) == 0 {
		// overrides default constructor. uses Type.new to instantiate.
		override = true
	}
	err = t.new.f.Methods.Add(types, NewCallerMethod(t, handler), override, onAdd)
	return
}

func (t *Class) AddMethodByTypes(vm *VM, argTypes ParamsTypes, handler CallerObject, override bool, onAdd func(method *TypedCallerMethod) error) error {
	return t.new.AddMethodByTypes(vm, argTypes, handler, override, onAdd)
}

func (t *Class) New(vm *VM, fields Dict) (Object, error) {
	return t.NewInstanceWithFields(vm, fields)
}

func (t *Class) NewInstanceWithFields(vm *VM, fields Dict) (*ClassInstance, error) {
	instance := t.NewInstance()
	return instance, instance.Init(vm, fields)
}

func (t *Class) NewInstance() (o *ClassInstance) {
	return &ClassInstance{class: t}
}

func (t *Class) Construct(c Call) (o Object, err error) {
	arg := &Arg{
		Name:          "this",
		TypeAssertion: TypeAssertionFromTypes(t),
	}
	if err = c.Args.Destructure(arg); err != nil {
		return
	}

	this := arg.Value.(*ClassInstance)
	return this, this.Init(c.VM, c.NamedArgs.Dict())
}

func (t *Class) Call(c Call) (_ Object, err error) {
	return t.NewInstance().Call(c)
}

func (t *Class) IsChildOf(p ObjectType) bool {
	if st, _ := p.(*Class); st != nil {
		for _, p := range t.parents {
			if st == p.Type || p.Type.IsChildOf(st) {
				return true
			}
		}
	}
	return false
}

func (t *Class) RawParents() []*ClassParent {
	return t.parents
}

func (t *Class) Parents() (r Array) {
	r = make(Array, len(t.parents))
	for i, p := range t.parents {
		r[i] = Dict{"alias": Str(p.Alias), "type": p.Type}
	}
	return r
}

func (t *Class) RawFields() (r []*ClassField) {
	r = make([]*ClassField, len(t.fieldsMap))
	for _, field := range t.fieldsMap {
		r[field.index] = field
	}
	return
}

func (t *Class) Fields() (d Dict) {
	d = make(Dict, len(t.fieldsMap))
	for _, field := range t.fieldsMap {
		d[field.Name] = field
	}
	return d
}

func (t *Class) CallFieldsOf(c Call) (ret Object, err error) {
	var obj = &Arg{
		Name:          "obj",
		TypeAssertion: TypeAssertionFromTypes(t),
	}
	if err = c.Args.Destructure(obj); err != nil {
		return
	}
	return obj.Value.(*ClassInstance).fields, nil
}

func (t *Class) CallAddProperties(call Call) (err error) {
	var (
		items = &Arg{
			Name:          "items",
			TypeAssertion: TypeAssertionFromTypes(TDict),
		}
	)

	if err = call.Args.Destructure(items); err != nil {
		return
	}

	for name, value := range items.Value.(Dict) {
		switch v := value.(type) {
		case *Func:
			if _, err = t.CallAddProperty(Call{VM: call.VM, Args: Args{{Str(name), v}}}); err != nil {
				return
			}
		default:
			return ErrClassMethodRegister.NewErrorf("method %v: unexpected value type: %T", name, v.Type().Name())
		}
	}

	return
}

func (t *Class) CallAddProperty(c Call) (ret Object, err error) {
	var (
		name = &Arg{
			Name:          "name",
			TypeAssertion: TypeAssertionFromTypes(TStr),
		}

		f = &Arg{
			Name:          "func",
			TypeAssertion: TypeAssertionFromTypes(TFunc),
		}
	)

	if err = c.Args.Destructure(name, f); err != nil {
		return
	}

	return Nil, t.AddProperty(name.Value.ToString(), f.Value.(*Func))
}

func (t *Class) CallGetProperty(c Call) (ret Object, err error) {
	name := &Arg{
		Name:          "name",
		TypeAssertion: TypeAssertionFromTypes(TStr),
	}

	if err = c.Args.Destructure(name); err != nil {
		return
	}

	p := t.propertiesMap[name.Value.ToString()]

	if p == nil {
		return Nil, nil
	}

	return p, nil
}

func (t *Class) GetIndexMethod(_ *VM, index Object) (ret Object, err error) {
	var ok bool
	if ret, ok = t.methodsMap[index.ToString()]; !ok {
		err = ErrInvalidIndex.NewError(index.ToString())
	}
	return
}

func (t *Class) AddMethodIndex(c Call) (ret Object, err error) {
	var (
		pName = &Arg{
			Name:          "name",
			TypeAssertion: TypeAssertionFromTypes(TStr),
		}
		pHandlers = &Arg{
			Name:          "handlers",
			TypeAssertion: TypeAssertionFromTypes(TArray).Options(WithRawCallable()),
		}
		pOverride = &NamedArgVar{
			Name:          "override",
			TypeAssertion: NewTypeAssertion(TypeAssertions(WithFlag())),
		}
	)

	if err = c.Args.Destructure(pName, pHandlers); err != nil {
		return
	}

	if err = c.NamedArgs.Get(pOverride); err != nil {
		return
	}

	var (
		name     = pName.Value.ToString()
		handlers Array
		override = !pOverride.IsFalsy()
		tm       = t.methodsMap[name]
		i        int

		h = func(h CallerObject, cb func(co CallerObject, types ParamsTypes) error) {
			if !IsFunction(h) {
				err = ErrClassMethodRegister.NewError(
					fmt.Sprintf("set[%d]: is not a raw caller object", i),
				)
				return
			}

			if err = SplitCaller(c.VM, h, cb); err != nil {
				return
			}

			i++
		}
	)

	switch pHandlers.Value.(type) {
	case Array:
		handlers = pHandlers.Value.(Array)
	case CallerObject:
		handlers = Array{pHandlers.Value}
	}

	if len(handlers) == 0 {
		return nil, ErrClassMethodRegister.NewError("no handlers")
	}

	if tm == nil {
		tm = NewClassMethod(name, t)

		defer func() {
			if err == nil {
				t.methodsMap[name] = tm
			}
		}()
	}

	for _, handler := range handlers {
		h(handler.(CallerObject), func(co CallerObject, types ParamsTypes) (err error) {
			m := NewCallerMethod(t, co)
			if err = tm.f.Methods.Add(types, m, override, nil); err != nil {
				return ErrClassMethodRegister.NewError(
					fmt.Sprintf("set[%d]: %v", i, err.Error()),
				)
			}
			return
		})
	}

	return tm, nil
}

func (t *Class) CallAddMethods(call Call) (err error) {
	var (
		elements KeyValueArray
		items    = &Arg{
			Name:          "items",
			TypeAssertion: TypeAssertionFromTypes(TDict, TKeyValueArray, TArray),
		}
	)

	if err = call.Args.Destructure(items); err != nil {
		return
	}

	switch t := items.Value.(type) {
	case KeyValueArray:
		elements = t
	case Dict:
		elements = t.ToKeyValueArray()
	case Array:
		for i, v := range t {
			switch t := v.(type) {
			case *KeyValue:
				elements = append(elements, t)
			case *Func:
				if t.Name() == "" {
					return ErrClassMethodRegister.NewErrorf("method %d (%s): is anonymous", i, t.ToString())
				}
				elements = append(elements, &KeyValue{K: Str(t.name), V: t})
			case *CompiledFunction:
				if t.FuncName == "" {
					return ErrClassMethodRegister.NewErrorf("method %d (%s): is anonymous", i, t.ToString())
				}
				elements = append(elements, &KeyValue{K: Str(t.FuncName), V: t})
			default:
				return ErrClassMethodRegister.NewErrorf("method %d: is not a *KeyValue or named function", i)
			}
		}
	}

	for _, value := range elements {
		name := value.K.ToString()
		switch v := value.V.(type) {
		case Dict:
			handlers := v["handlers"]
			override := Flag(ObjectOrNil(v["override"]).IsFalsy())
			if IsNil(handlers) {
				return ErrClassMethodRegister.NewErrorf("method %v: no handlers found", name)
			}
			if _, err = t.AddMethodIndex(Call{VM: call.VM, Args: Args{{value.K, handlers}}, NamedArgs: *Dict{"override": override}.ToNamedArgs()}); err != nil {
				return
			}
		case Array:
			if _, err = t.AddMethodIndex(Call{VM: call.VM, Args: Args{{value.K, v}}}); err != nil {
				return
			}
		case CallerObject:
			if _, err = t.AddMethodIndex(Call{VM: call.VM, Args: Args{{value.K, v}}}); err != nil {
				return
			}
		default:
			return ErrClassMethodRegister.NewErrorf("method %v: unexpected value type: %T", name, v.Type().Name())
		}
	}

	return
}

func (t *Class) CallAddFields(call Call) (err error) {
	var (
		items = &Arg{
			Name:          "items",
			TypeAssertion: TypeAssertionFromTypes(TKeyValueArray),
		}

		defaults = &NamedArgVar{
			Name:          "defaults",
			TypeAssertion: NewTypeAssertion(TypeAssertions(WithRawCallable())),
		}
	)

	if err = call.Args.Destructure(items); err != nil {
		return
	}

	for _, value := range items.Value.(KeyValueArray) {
		f := &ClassField{class: t}
		switch tk := value.K.(type) {
		case *TypedIdent:
			types := make(ObjectTypes, len(tk.Types))
			for i, t := range tk.Types {
				types[i] = t.(ObjectType)
			}
			f.Name = tk.Name
			f.Types = types
		default:
			f.Name = value.K.ToString()
		}

		switch t := value.V.(type) {
		case *NilType, Flag:
		default:
			f.Value = t
		}
		if err = t.AddField(f); err != nil {
			return
		}
	}

	if defaults.Value != nil {
		t.fieldDefaults = append(t.fieldDefaults, defaults.Value.(CallerObject))
	}
	return nil
}

func (t *Class) CallExtends(c Call) (err error) {
	if err = c.Args.CheckMinLen(1); err != nil {
		return
	}

	return c.Args.WalkE(func(i int, arg Object) (err error) {
		switch v := arg.(type) {
		case *Class:
			t.Extends(v, "")
		case Array:
			if len(v) != 2 || v[0].Type() != TClass || v[1].Type() != TStr {
				err = NewArgumentTypeError(
					strconv.Itoa(i)+"st",
					"class|array[parent class, alias string]",
					arg.Type().Name(),
				)
			} else {
				t.Extends(v[0].(*Class), v[1].ToString())
			}
			return
		}
		if parent, ok := arg.(*Class); !ok {
			err = NewArgumentTypeError(
				strconv.Itoa(i)+"st",
				"Class",
				arg.Type().Name(),
			)
		} else {
			t.Extends(parent, "")
		}
		return
	})
}

func (t *Class) CallAddNewHandlers(c Call) (err error) {
	if err = c.Args.CheckMinLen(1); err != nil {
		return
	}

	return c.Args.WalkE(func(i int, arg Object) (err error) {
		return SplitCaller(c.VM, arg, func(co CallerObject, types ParamsTypes) (err error) {
			err = t.AddCallerMethod(c.VM, types, co, false, nil)
			return
		})
	})
}

func (t *Class) CallName(name string, c Call) (ret Object, err error) {
	switch name {
	case "new":
		return t.Construct(c)
	case "fields":
		return t.Fields(), nil
	case "fieldsOf":
		return t.CallFieldsOf(c)
	case "addProperty":
		return t.CallAddProperty(c)
	case "addProperties":
		return t, t.CallAddProperties(c)
	case "getProperty":
		return t.CallGetProperty(c)
	case "addFields":
		return t, t.CallAddFields(c)
	case "addMethod":
		return t.AddMethodIndex(c)
	case "addMethods":
		return t, t.CallAddMethods(c)
	case "extends":
		return t, t.CallExtends(c)
	case "addNewHandleres":
		return t, t.CallAddNewHandlers(c)
	}
	return nil, ErrInvalidIndex.NewError(name)
}

func (t *Class) GetProperty(name string) *ClassProperty {
	return t.propertiesMap[name]
}

func (t *Class) Properties() (d Dict) {
	d = make(Dict, len(t.propertiesMap))

	for name, p := range t.propertiesMap {
		d[name] = p
	}

	return d
}

func (t *Class) Methods() (d Dict) {
	d = make(Dict, len(t.methodsMap))
	for name, m := range t.methodsMap {
		d[name] = m
	}
	return d
}

func (t *Class) Repr() string {
	return "class " + t.FullName()
}

func (t *Class) Print(state *PrinterState) error {
	if state.options.IsTypesAsFullNames() {
		return state.WriteString(t.Repr())
	}

	d := Dict{
		"fields":     t.Fields(),
		"properties": t.Properties(),
		"methods":    t.Methods(),
		"new":        t.new,
	}

	defer state.options.Backup(PrintStateOptionSortKeys)()
	state.options.SetSortKeys(PrintStateOptionSortTypeAscending)
	defer state.WrapReprString("class " + t.FullName())()
	return d.PrintObject(state, nil)
}

func (t *Class) IndexGet(vm *VM, index Object) (value Object, err error) {
	key := index.ToString()
	switch key {
	case "@fields":
		return t.Fields(), nil
	case "@properties":
		return t.Properties(), nil
	case "@methods":
		return t.Methods(), nil
	case "@parents":
		return t.Parents(), nil
	case "@name":
		return Str(t.name), nil
	case "@module":
		return t.module, nil
	default:
		if v := t.propertiesMap[key]; v != nil {
			return v, nil
		}

		if v := t.methodsMap[key]; v != nil {
			return v, nil
		}
		return nil, ErrInvalidIndex.NewError(index.ToString())
	}
}

var (
	_ Object       = &Class{}
	_ CallerObject = &Class{}
)

func (t *Class) ToString() string {
	return string(MustToStr(nil, t))
}

// Equal implements Object interface.
func (t *Class) Equal(right Object) bool {
	v, ok := right.(*Class)
	if !ok {
		return false
	}
	return v == t
}

func (Class) IsFalsy() bool { return false }

func (t *Class) BinaryOp(_ *VM, tok token.Token, right Object) (Object, error) {
	if right == Nil {
		switch tok {
		case token.Less, token.LessEq:
			return False, nil
		case token.Greater, token.GreaterEq:
			return True, nil
		}
	}

	return nil, NewOperandTypeError(
		tok.String(),
		t.Type().Name(),
		right.Type().Name())
}
