package gad

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gad-lang/gad/repr"
)

var (
	_ Object             = (*Module)(nil)
	_ LengthGetter       = (*Module)(nil)
	_ KeysGetter         = (*Module)(nil)
	_ ValuesGetter       = (*Module)(nil)
	_ ItemsGetter        = (*Module)(nil)
	_ IndexGetSetter     = (*Module)(nil)
	_ Printabler         = (*Module)(nil)
	_ IndexSetterUpdater = (*Module)(nil)
	_ ToDictConverter    = (*Module)(nil)
	_ StringIndexSetter  = (*Module)(nil)
)

type Modules []*Module

func (m Modules) Get(name string) *Module {
	for _, module := range m {
		if module.Spec.Name == name {
			return module
		}
	}
	return nil
}

type ModuleData interface {
	Object
	IndexGetSetter
	ToDictConverter
	LengthGetter
	KeysGetter
	ValuesGetter
	ItemsGetter
	Iterabler
	StringIndexSetter
}

type ModuleGetter interface {
	GetModule() *ModuleSpec
}

type ModuleSetter interface {
	SetModule(m *ModuleSpec)
}

// Module represent the module
type Module struct {
	Spec   *ModuleSpec
	Data   ModuleData
	Params MixedParams
}

func NewModule(static *ModuleSpec, f ...func(m *Module)) *Module {
	m := &Module{Spec: static}
	m.Params.Named = make(KeyValueArray, 0)
	m.Params.Positional = make(Array, 0)

	for _, f := range f {
		f(m)
	}

	if m.Data == nil {
		// StdModuleData keeps variables, constants and functions apart, so a
		// compiled module's `export const` is const-enforced and its exported
		// functions land in the Funcs bucket (see OpExtendModule/OpExtendModuleConst).
		m.Data = NewStdModuleData()
	}

	return m
}

func (m *Module) CanCall() bool {
	return m.Data == nil && m.Spec.InitGoFunc != nil && m.Spec.InitCompiledFunc != nil
}

func (m *Module) IsFalsy() bool {
	return false
}

func (m *Module) Type() ObjectType {
	return TModule
}

func (m *Module) String() string {
	return m.ToString()
}

func (m *Module) ToString() string {
	var s string
	if m.Spec.URL == "" {
		s = m.Spec.Name
	} else {
		s = fmt.Sprintf("%s %q", m.Spec.Name, m.Spec.URL)
	}
	return ReprQuoteTyped("module", s)
}

func (m *Module) Name() string {
	return m.Spec.Name
}

func (m *Module) File() string {
	return m.Spec.URL
}

func (m *Module) Equal(right Object) bool {
	if r, ok := right.(*Module); ok {
		return m == r
	}
	return false
}

func (m *Module) Length() int {
	if m.Data == nil {
		return 0
	}
	return m.Data.Length()
}

func (m *Module) Keys() (arr Array) {
	if m.Data == nil {
		return
	}
	return m.Data.Keys()
}

func (m *Module) Values() (arr Array) {
	if m.Data == nil {
		return
	}
	return m.Data.Values()
}

func (m *Module) Items(vm *VM, cb ItemsGetterCallback) (err error) {
	if m.Data == nil {
		return
	}
	return m.Data.Items(vm, cb)
}

func (m *Module) CanIterate() bool {
	return m.Data != nil
}

func (m *Module) Iterate(vm *VM, na *NamedArgs) Iterator {
	return m.Data.Iterate(vm, na)
}

func (m *Module) IndexGet(vm *VM, index Object) (value Object, err error) {
	switch index.ToString() {
	case AttrName:
		return Str(m.Spec.Name), nil
	case AttrFile:
		return Str(m.Spec.URL), nil
	case AttrParams:
		return &m.Params, nil
	case "@main":
		return Bool(m.Spec.IsMain()), nil
	case "@data":
		return m.Data, nil
	}
	return m.Data.IndexGet(vm, index)
}

func (m *Module) IndexSet(vm *VM, index, value Object) error {
	return m.Data.IndexSet(vm, index, value)
}

func (m *Module) Set(key string, value Object) {
	m.Data.Set(key, m.moduleScoped(key, value))
}

// ConstStringIndexSetter is a StringIndexSetter whose members can also be
// declared as read-only constants. Module.SetConst routes `export const` here;
// see StdModuleData.
type ConstStringIndexSetter interface {
	StringIndexSetter
	SetConst(key string, value Object)
}

// SetConst declares a read-only module constant, applying the same module
// scoping as Set. When the module Data cannot distinguish constants it degrades
// to a plain Set (the value is present but not const-enforced).
func (m *Module) SetConst(key string, value Object) {
	value = m.moduleScoped(key, value)
	if cs, ok := m.Data.(ConstStringIndexSetter); ok {
		cs.SetConst(key, value)
		return
	}
	m.Data.Set(key, value)
}

// moduleScoped qualifies a function/type value with this module's spec so its
// name reports as `mod.name`; other values pass through unchanged.
func (m *Module) moduleScoped(_ string, value Object) Object {
	switch t := value.(type) {
	case *Function:
		t = Copy(t)
		t.SetModule(m.Spec)
		return t
	case *Type:
		t = Copy(t)
		t.Module = m.Spec
		return t
	case ModuleSetter:
		t.SetModule(m.Spec)
	}
	return value
}

func (m *Module) Print(state *PrinterState) (err error) {
	fmt.Fprintf(state, "%smodule %q", repr.QuotePrefix, m.Name())
	if file := m.File(); len(file) > 0 {
		fmt.Fprintf(state, " at %q", file)
	}
	d := Dict{}
	if m.Data != nil && !m.Data.IsFalsy() {
		d["@data"] = m.Data
	}
	if !m.Params.IsFalsy() {
		d["@params"] = Copy(&m.Params)
	}
	if !d.IsFalsy() {
		state.WriteByte(' ')
		err = d.Print(state)
	}
	state.WriteString(repr.QuoteSufix)
	return
}

func (m *Module) UpdateIndexSetter(out StringIndexSetter) {
	for k, v := range m.Data.ToDict() {
		out.Set(k, v)
	}
}

func (m *Module) MergeData(d Dict) {
	for k, v := range d {
		m.Set(k, v)
	}
}

func (m *Module) ToDict() Dict {
	return m.Data.ToDict()
}

type ModuleInfo struct {
	Name string
	URL  string
}

// ModuleFlags is a bitmask of module attributes on a ModuleSpec.
type ModuleFlags uint

const (
	// ModuleMain marks the entrypoint (main) module.
	ModuleMain ModuleFlags = 1 << iota
	// ModuleRawArgv marks a module declared with a lone variadic `param (*argv)`
	// (a single variadic positional param, no named params). Its arguments are
	// passed straight through (no `--name` parsing) and argv[0] is the module
	// path used to invoke it (e.g. "a/b/script.gad" for `gad a/b/script.gad`).
	// Independent of ModuleMain.
	ModuleRawArgv
)

// Has reports whether all bits of x are set in f.
func (f ModuleFlags) Has(x ModuleFlags) bool { return f&x == x }

type ModuleSpec struct {
	ModuleInfo
	Index            int
	Path             []int
	Flags            ModuleFlags
	InitCompiledFunc *CompiledFunction
	InitGoFunc       func(module *Module) CallerObject
}

// IsMain reports whether this is the entrypoint (main) module.
func (i *ModuleSpec) IsMain() bool { return i.Flags.Has(ModuleMain) }

// IsRawArgv reports whether this main module was declared with a lone variadic
// `param (*argv)` (raw argument passthrough; see ModuleRawArgv).
func (i *ModuleSpec) IsRawArgv() bool { return i.Flags.Has(ModuleRawArgv) }

func NewModuleSpecFromName(name string, opt ...func(s *ModuleSpec)) *ModuleSpec {
	s := &ModuleSpec{ModuleInfo: ModuleInfo{Name: name}}
	for _, f := range opt {
		f(s)
	}
	return s
}

func (i *ModuleSpec) InitFunc(module *Module) CallerObject {
	if i.InitGoFunc != nil {
		return i.InitGoFunc(module)
	}
	return i.InitCompiledFunc
}

func (i *ModuleSpec) String() string {
	var entries []string

	if i.IsMain() {
		entries = append(entries, "main")
	}
	if i.IsRawArgv() {
		entries = append(entries, "rawArgv")
	}

	if len(i.URL) > 0 {
		entries = append(entries, "file="+strconv.Quote(i.URL))
	}

	if i.InitGoFunc != nil {
		entries = append(entries, "init")
	} else if i.InitCompiledFunc != nil {
		entries = append(entries, fmt.Sprintf("init%s", i.InitCompiledFunc.HeaderString()))
	}

	var s string

	if len(entries) > 0 {
		s = " [" + strings.Join(entries, ", ") + "]"
	}

	return ReprQuoteTyped("static module", i.Name+s)
}
