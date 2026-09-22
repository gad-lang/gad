package gad

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/gad-lang/gad/repr"
)

type FuncWrapper interface {
	fmt.Stringer
	Object
	Printabler
	Name() string
	FullName() string
	FuncSpecName() string
}

var (
	_ CallerObject = (*Func)(nil)
	_ FuncWrapper  = (*Func)(nil)
	_ MethodCaller = (*Func)(nil)
)

type Func struct {
	*FuncSpec
	module *ModuleSpec
	name   string
}

func NewFunc(name string, module *ModuleSpec) *Func {
	s := &Func{module: module, name: name}
	s.FuncSpec = &FuncSpec{this: s}
	return s
}

func (f *Func) Equal(right Object) bool {
	return f == right
}

func (f *Func) Name() string {
	return f.name
}

func (f *Func) FullName() string {
	if f.module == nil {
		return f.name
	}
	return f.module.Name + "." + f.name
}

func (f *Func) FuncSpecName() string {
	return "func " + ReprQuote(f.FullName())
}

func (f *Func) GetModule() *ModuleSpec {
	return f.module
}

func (f *Func) ToString() string {
	return f.String()
}

func (f *Func) String() string {
	return string(MustToStr(nil, f))
}

func (f *Func) Print(state *PrinterState) (err error) {
	return f.PrintFuncWrapper(state, f)
}

func (f *Func) Type() ObjectType {
	return TFunc
}

// IndexGet implements the function reflection index keys (`@name`, `@methods`,
// `@meta`, `@args`, `@nargs`, `@ret`) via the shared FuncSpec reflection.
func (f *Func) IndexGet(vm *VM, index Object) (Object, error) {
	if v, ok, err := f.reflectIndex(vm, f.name, index); ok {
		return v, err
	}
	return nil, ErrInvalidIndex.NewError(index.ToString())
}

type FuncSpecOption func(spec *FuncSpec)

func FuncSpectWithDefault(co CallerObject) FuncSpecOption {
	return func(spec *FuncSpec) {
		spec.defaul = co
	}
}

type FuncSpec struct {
	defaul  CallerObject
	Methods MethodArgType
	this    FuncWrapper
	// Meta is the function's `[k=v, …]` metadata (or nil); read as `fn.@meta`.
	Meta KeyValueArray
}

func NewFuncSpec(this FuncWrapper, opt ...FuncSpecOption) *FuncSpec {
	f := &FuncSpec{this: this}
	for _, o := range opt {
		o(f)
	}
	return f
}

func (s *FuncSpec) GetFuncSpec() *FuncSpec {
	return s
}

// unwrapCaller peels CallerMethod wrappers off a caller, returning the innermost
// CallerObject (e.g. the raw *CompiledFunction). A property's accessors are
// registered wrapped in a CallerMethod (see Class.AddProperty), so its `@methods`
// must unwrap to reach the callable's own reflection.
func unwrapCaller(c CallerObject) CallerObject {
	for {
		cm, ok := c.(*CallerMethod)
		if !ok || cm.CallerObject == nil {
			return c
		}
		c = cm.CallerObject
	}
}

// callerMethods returns the overload callers of a FuncSpec in declaration order,
// the default (untyped) caller first when present.
func (s *FuncSpec) callerMethods() (out Array) {
	if s.defaul != nil {
		out = append(out, unwrapCaller(s.defaul))
	}
	s.Methods.Walk(func(m *TypedCallerMethod) any {
		out = append(out, unwrapCaller(m.Caller()))
		return nil
	})
	return
}

// reflectIndex answers the function reflection index keys shared by Func,
// ClassMethod and ClassProperty: `@name`, `@methods`, `@meta` and the signature
// keys `@args`/`@nargs`/`@ret`. `@meta` is the spec's OWN metadata; when several
// overloads exist it is NOT any single overload's (use `@methods`), but a single
// overload reports that overload's `@meta`/`@args`/… as a convenience. ok is
// false for a non-reflection index.
func (s *FuncSpec) reflectIndex(vm *VM, name string, index Object) (_ Object, ok bool, err error) {
	delegate := func(key Object) (Object, bool, error) {
		if methods := s.callerMethods(); len(methods) == 1 {
			if ig, isIG := methods[0].(IndexGetter); isIG {
				if v, e := ig.IndexGet(vm, key); e == nil {
					return v, true, nil
				}
			}
		}
		return nil, false, nil
	}
	switch index.ToString() {
	case "@name":
		return Str(name), true, nil
	case "@methods":
		return s.callerMethods(), true, nil
	case "@meta":
		if len(s.Meta) > 0 {
			return s.Meta, true, nil
		}
		if v, got, _ := delegate(index); got {
			return v, true, nil
		}
		return metaObject(s.Meta), true, nil
	case "@args", "@nargs", "@ret":
		if v, got, _ := delegate(index); got {
			return v, true, nil
		}
		return Array{}, true, nil
	}
	return nil, false, nil
}

func (s *FuncSpec) IsFalsy() bool {
	return false
}

func (s FuncSpec) CopyWithTarget(target FuncWrapper) *FuncSpec {
	s.Methods = *s.Methods.Copy()
	s.Methods.Walk(func(m *TypedCallerMethod) any {
		m.target = target
		return nil
	})
	return &s
}

func (s *FuncSpec) HasCallerMethods() bool {
	// hasMethod is maintained by MethodArgType.Add; it is equivalent to
	// !s.Methods.IsZero() but O(1) (no tree walk) on the hot operator-dispatch
	// path. Methods are never removed, so the flag never over-reports.
	return s.Methods.hasMethod
}

func (s *FuncSpec) AddMethodByTypes(_ *VM, argTypes ParamsTypes, handler CallerObject, override bool, onAdd func(tcm *TypedCallerMethod) error) error {
	target := s.this
	return s.Methods.Add(argTypes, NewCallerMethod(target, handler), override, onAdd)
}

// AddMethod Add caller method.
func (s *FuncSpec) AddMethod(vm *VM, handler CallerObject, override bool, onAdd func(tcm *TypedCallerMethod) error) error {
	target := s.this
	return SplitCaller(vm, handler, func(co CallerObject, types ParamsTypes) error {
		return s.Methods.Add(types, NewCallerMethod(target, handler), override, onAdd)
	})
}

func (s *FuncSpec) PrintFuncWrapper(state *PrinterState, fo FuncWrapper) (err error) {
	if !state.IsRepr {
		name := fo.FuncSpecName()
		n := s.Methods.NumMethods()
		if n > 0 {
			name = fmt.Sprintf("%s with %d methods", name, n)
		}
		return state.WriteString(repr.Quote(name))
	} else {
		defer state.WrapRepr(fo)()
	}

	var items Array

	s.Methods.WalkSorted(func(m *TypedCallerMethod) any {
		items = append(items, m)
		return nil
	})

	state.WriteString(repr.QuotePrefix)
	state.WriteString(fo.FuncSpecName())
	state.WriteString(" ")

	if l := len(items); l == 0 {
		state.WriteString("without methods")
	} else {
		fmt.Fprintf(state, "with %d methods: ", len(items))
		defer state.WithValueBackup(typedCallerMethodContextKeyNoTarget, true)()
		err = items.PrintObject(state, nil)
	}

	state.WriteString(repr.QuoteSufix)
	return
}

func (s *FuncSpec) Call(c Call) (Object, error) {
	caller, validate := s.CallerMethodWithValidationCheckOfArgs(c.Args)
	if caller == nil {
		types := c.Args.Types()
		if len(types) == 0 {
			return nil, ErrNoMethodFound.NewErrorf("func %s no have method without params", ReprQuote(s.this.FullName()))
		}
		return nil, ErrNoMethodFound.NewErrorf("func %s no have method with params %s", ReprQuote(s.this.FullName()), types)
	}
	c.SkipValidation = !validate
	return YieldCall(caller, &c), nil
}

func (s *FuncSpec) CallerMethodWithValidationCheckOfArgs(args Args) (CallerObject, bool) {
	return s.CallerMethodWithValidationCheckOfArgsTypes(args.Types())
}

func (s *FuncSpec) CallerMethodOfArgsTypes(types ObjectTypeArray) (co CallerObject) {
	var method *TypedCallerMethod
	if method, co = s.MethodOrDefault(types); co == nil && method != nil {
		// Return the TypedCallerMethod itself (callers such as isIterable assert
		// it back to *TypedCallerMethod to inspect its param types).
		co = method
	}
	return
}

func (s *FuncSpec) CallerMethodDefault() CallerObject {
	return s.defaul
}

func (s *FuncSpec) CallerMethodWithValidationCheckOfArgsTypes(types ObjectTypeArray) (co CallerObject, validate bool) {
	var method *TypedCallerMethod
	if method, co = s.MethodOrDefault(types); co == nil {
		if method != nil {
			co = method.CallerObject
			// A structural param keys as TAny in the tree; the exact match does
			// not prove the constraint, so force value-based validation.
			if method.forceValidate {
				validate = true
			}
		}
	} else {
		validate = true
	}
	return
}

func (s *FuncSpec) MethodOrDefault(types ObjectTypeArray) (method *TypedCallerMethod, defaul CallerObject) {
	if method = s.Methods.GetMethod(types); method == nil && s.defaul != nil {
		defaul = s.defaul
	}
	return
}

func (s *FuncSpec) CallerMethods() *MethodArgType {
	return &s.Methods
}

func AddMethod(target Object, method ...TypedCallerObjectWithParamTypes) Object {
	return AddMethodOverride(false, target, method...)
}

func AddMethodT[T Object](target T, method ...TypedCallerObjectWithParamTypes) T {
	ret := AddMethodOverride(false, target, method...)
	if reflect.TypeFor[T]() != reflect.TypeOf(ret) {
		panic("AddMethodT[T] must be of type T")
	}
	return target
}

func AddMethodOverride(override bool, target Object, method ...TypedCallerObjectWithParamTypes) Object {
	addMethod := func(target MethodAdder, method ...TypedCallerObjectWithParamTypes) {
		for i, m := range method {
			if err := target.AddMethodByTypes(nil, m.ParamTypes(), m, override, nil); err != nil {
				panic(fmt.Errorf("failed to add method %d: %v", i, err))
			}
		}
	}
	switch t := target.(type) {
	case *Func:
		for i, m := range method {
			if err := t.AddMethodByTypes(nil, m.ParamTypes(), m, override, nil); err != nil {
				panic(fmt.Errorf("failed to add method %d: %v", i, err))
			}
		}
		return t
	case *BuiltinObjType:
		addMethod(t, method...)
		return t
	case *BuiltinFunction:
		if t.AcceptMethodsDisabled {
			panic(errors.New(t.ToString() + " not accept methods"))
		}
		f := NewBuiltinFunctionWithMethods(t.FuncName, t.Module)
		if t.Header != nil {
			if err := f.AddMethodByTypes(nil, t.Header.ParamTypes(), t, override, nil); err != nil {
				panic(fmt.Errorf("failed to add method initial: %v", err))
			}
		} else {
			f.defaul = t
		}
		addMethod(f, method...)
		return f
	case *Function:
		f := NewFunc(t.FuncName, t.Module)
		if t.Header != nil {
			if err := f.AddMethodByTypes(nil, t.Header.ParamTypes(), t, override, nil); err != nil {
				panic(fmt.Errorf("failed to add method initial: %v", err))
			}
		} else {
			f.defaul = t
		}
		addMethod(f, method...)
		return f
	case MethodAdder:
		addMethod(t, method...)
		return target
	default:
		panic(fmt.Errorf("unknown type %T", target))
	}
}
