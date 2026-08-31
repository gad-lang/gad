package gad

// TStaticType is the base type of every marker type declared with `type … { … }`.
var TStaticType = NewType("staticType", TBase)

// StaticType is a MARKER type declared with `type Name { … }` (or the expression
// `type { … }`). Unlike a class it has NO instances: its fields, properties and
// methods live on the type value itself (`this` inside a method or accessor is the
// type), and it may declare a `call(…)` factory — called as `Name(…)` — whose
// result is arbitrary (a factory, not an instance).
//
// It is a first-class ObjectType, so `type<Name>` dispatches on it and it can be
// passed around as a value. Because it has no instances, `x :: Name` and using
// `Name` as an instance parameter type are rejected (use `type<Name>` to match the
// type value).
//
// A StaticType can also be built from Go with NewStaticType and the With… helpers.
type StaticType struct {
	name    string
	module  *ModuleSpec
	fields  Dict                    // static field values
	props   map[string]CallerObject // property accessors (get: this; set: this, value)
	methods map[string]CallerObject // methods (this = the type)
	call    CallerObject            // the `call(…)` factory, or nil
}

var (
	_ Object         = (*StaticType)(nil)
	_ ObjectType     = (*StaticType)(nil)
	_ TypeAssigner   = (*StaticType)(nil)
	_ IndexGetSetter = (*StaticType)(nil)
)

// NewStaticType builds an empty marker type; populate it with the With… helpers.
func NewStaticType(name string) *StaticType {
	return &StaticType{name: name, fields: Dict{}}
}

// NewStaticTypeFunc is the `StaticType(name[, define])` builtin a `type … { … }`
// literal lowers to: name is required; the optional define handler receives the
// in-construction type and a `define` function that populates it (see Define).
func NewStaticTypeFunc(c Call) (ret Object, err error) {
	nameArg := &Arg{Name: "name", TypeAssertion: TypeAssertionFromTypes(TStr)}
	rest, err := c.Args.DestructureRangeVar(1, nameArg)
	if err != nil {
		return
	}

	t := NewStaticType(string(nameArg.Value.(Str)))
	t.module = c.VM.CurrentModuleSpec()

	if len(rest) > 0 {
		handler, ok := rest[0].(CallerObject)
		if !ok {
			return nil, NewArgumentTypeError("2nd (define)", "callable", rest[0].Type().Name())
		}
		_, err = handler.Call(Call{
			Context: c.Context,
			VM:      c.VM,
			Args: Args{{
				t,
				NewFunction("define", func(c Call) (Object, error) {
					return nil, t.Define(c)
				}),
			}},
		})
		return t, err
	}
	return t, nil
}

// WithField sets a static field value; returns the receiver for chaining.
func (t *StaticType) WithField(name string, value Object) *StaticType {
	t.fields[name] = value
	return t
}

// WithMethod adds a static method (its first parameter receives `this` — the type
// — when called as `Name.method(…)`).
func (t *StaticType) WithMethod(name string, fn CallerObject) *StaticType {
	if t.methods == nil {
		t.methods = map[string]CallerObject{}
	}
	t.methods[name] = fn
	return t
}

// WithProperty adds a static property accessor: one callable that returns the
// value when called with `this` (get) and writes it when called with `this, value`
// (set) — an AddMethod-style overloaded function handles both.
func (t *StaticType) WithProperty(name string, accessor CallerObject) *StaticType {
	if t.props == nil {
		t.props = map[string]CallerObject{}
	}
	t.props[name] = accessor
	return t
}

// WithCall sets the `call(…)` factory invoked as `Name(…)`.
func (t *StaticType) WithCall(fn CallerObject) *StaticType {
	t.call = fn
	return t
}

func (t *StaticType) SetModule(m *ModuleSpec) { t.module = m }
func (t *StaticType) GetModule() *ModuleSpec  { return t.module }
func (t *StaticType) GadObjectType()          {}
func (t *StaticType) Type() ObjectType        { return TStaticType }
func (t *StaticType) Name() string            { return t.name }

func (t *StaticType) FullName() string {
	if t.module != nil && t.module.Name != "" {
		return t.module.Name + "." + t.name
	}
	return t.name
}

func (t *StaticType) ToString() string { return ReprQuoteTyped("type", t.FullName()) }
func (t *StaticType) String() string   { return t.ToString() }
func (t *StaticType) IsFalsy() bool    { return false }

func (t *StaticType) Equal(right Object) bool {
	r, ok := right.(*StaticType)
	return ok && t == r
}

// CanAssign always fails: a marker type has no instances, so nothing is
// assignable to it. Match the type VALUE with `type<Name>` instead.
func (t *StaticType) CanAssign(Object) (bool, error) {
	return false, ErrType.NewError(t.name + " is a marker type with no instances; use type<" + t.name + ">")
}

func (t *StaticType) AssignTo(_ *VM, _ Object, _ TypeAssigner) (Object, error) {
	return nil, ErrType.NewError(t.name + " is a marker type with no instances; use type<" + t.name + ">")
}

// Call invokes the `call(…)` factory with `this` (the type) prepended. The result
// is whatever the factory returns — never an instance of the type.
func (t *StaticType) Call(c Call) (Object, error) {
	if t.call == nil {
		return nil, ErrNotCallable.NewError(t.name + " has no call factory")
	}
	c.Args = append(Args{{t}}, c.Args...)
	return YieldCall(t.call, &c), nil
}

// IndexGet resolves a member on the type value: a field value, a property (its
// getter is invoked with `this`), or a method (returned bound to `this`).
func (t *StaticType) IndexGet(vm *VM, index Object) (Object, error) {
	name := index.ToString()
	if p := t.props[name]; p != nil {
		return DoCall(p, Call{VM: vm, Args: Args{{t}}})
	}
	if m := t.methods[name]; m != nil {
		return t.boundMethod(name, m), nil
	}
	if v, ok := t.fields[name]; ok {
		return v, nil
	}
	return Nil, nil
}

// IndexSet writes a member: a property (its accessor invoked with `this, value`)
// or a static field.
func (t *StaticType) IndexSet(vm *VM, index, value Object) error {
	name := index.ToString()
	if p := t.props[name]; p != nil {
		_, err := DoCall(p, Call{VM: vm, Args: Args{{t, value}}})
		return err
	}
	t.fields[name] = value
	return nil
}

// Define populates the marker type from a `define(; fields=…, methods=…,
// properties=…, call=…)` call emitted by the compiler for a `type … { … }`
// literal. fields is a dict of static values; methods and properties are dicts of
// name → callable (each callable takes `this` as its first parameter); call is the
// factory callable.
func (t *StaticType) Define(c Call) (err error) {
	membersInto := func(what string, dst *map[string]CallerObject) func(Object) error {
		return func(v Object) error {
			d, _ := v.(Dict)
			for k, val := range d {
				co, ok := val.(CallerObject)
				if !ok || !Callable(val) {
					return ErrType.NewError(what + " " + ReprQuote(k) + " is not callable")
				}
				if *dst == nil {
					*dst = map[string]CallerObject{}
				}
				(*dst)[k] = co
			}
			return nil
		}
	}
	var (
		fields = &NamedArgVar{
			Name: "fields", TypeAssertion: TypeAssertionFromTypes(TDict),
			Do: func(v Object) error {
				for k, val := range v.(Dict) {
					t.fields[k] = val
				}
				return nil
			},
		}
		methods = &NamedArgVar{
			Name: "methods", TypeAssertion: TypeAssertionFromTypes(TDict),
			Do: membersInto("method", &t.methods),
		}
		properties = &NamedArgVar{
			Name: "props", TypeAssertion: TypeAssertionFromTypes(TDict),
			Do: membersInto("prop", &t.props),
		}
		call = &NamedArgVar{
			Name: "call", TypeAssertion: NewTypeAssertion(TypeAssertions(WithCallable())),
			Do: func(v Object) error {
				t.call, _ = v.(CallerObject)
				return nil
			},
		}
	)
	return c.NamedArgs.GetDo(fields, methods, properties, call)
}

// boundMethod wraps a method so calling it prepends `this` (the type).
func (t *StaticType) boundMethod(name string, m CallerObject) CallerObject {
	return &Function{
		FuncName: name,
		ToStringFunc: func() string {
			return ReprQuote("staticMethod of " + t.FullName() + "#" + name)
		},
		Value: func(c Call) (Object, error) {
			c.Args = append(Args{{t}}, c.Args...)
			return YieldCall(m, &c), nil
		},
	}
}
