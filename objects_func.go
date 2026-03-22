package gad

import (
	"errors"
	"fmt"

	"github.com/gad-lang/gad/repr"
)

type FuncWrapper interface {
	fmt.Stringer
	Object
	Printer
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
	module *Module
	name   string
}

func NewFunc(name string, module *Module) *Func {
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
	return f.module.Info.Name + "." + f.name
}

func (f *Func) FuncSpecName() string {
	return "func " + ReprQuote(f.FullName())
}

func (f *Func) GetModule() *Module {
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
	return !s.Methods.IsZero()
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
	c.SafeArgs = !validate
	return YieldCall(caller, &c), nil
}

func (s *FuncSpec) CallerMethodWithValidationCheckOfArgs(args Args) (CallerObject, bool) {
	return s.CallerMethodWithValidationCheckOfArgsTypes(args.Types())
}

func (s *FuncSpec) CallerMethodOfArgs(args Args) (co CallerObject) {
	return s.CallerMethodOfArgsTypes(args.Types())
}

func (s *FuncSpec) CallerMethodOfArgsTypes(types ObjectTypeArray) (co CallerObject) {
	if m := s.Methods.GetMethod(types); m != nil {
		return m
	}
	if s.defaul != nil {
		return s.defaul
	}
	return
}

func (s *FuncSpec) CallerMethodDefault() CallerObject {
	return s.defaul
}

func (s *FuncSpec) CallerMethodWithValidationCheckOfArgsTypes(types ObjectTypeArray) (co CallerObject, validate bool) {
	if method := s.Methods.GetMethod(types); method != nil {
		return method.CallerObject, false
	} else if s.defaul != nil {
		return s.defaul, true
	}
	return
}

func (s *FuncSpec) CallerMethods() *MethodArgType {
	return &s.Methods
}

func AddMethod(target Object, method ...CallerObjectWithParamTypes) Object {
	addMethod := func(target MethodAdder, method ...CallerObjectWithParamTypes) {
		for i, m := range method {
			if err := target.AddMethodByTypes(nil, m.ParamTypes(), m, false, nil); err != nil {
				panic(fmt.Errorf("failed to add method %d: %v", i, err))
			}
		}
	}
	switch t := target.(type) {
	case *Func:
		for i, m := range method {
			if err := t.AddMethodByTypes(nil, m.ParamTypes(), m, false, nil); err != nil {
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
			if err := f.AddMethodByTypes(nil, t.Header.ParamTypes(), t, false, nil); err != nil {
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
			if err := f.AddMethodByTypes(nil, t.Header.ParamTypes(), t, false, nil); err != nil {
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
