package gad

import (
	"fmt"

	"github.com/gad-lang/gad/repr"
)

var (
	_ Object          = (*Module)(nil)
	_ LengthGetter    = (*Module)(nil)
	_ KeysGetter      = (*Module)(nil)
	_ ValuesGetter    = (*Module)(nil)
	_ ItemsGetter     = (*Module)(nil)
	_ IndexGetSetter  = (*Module)(nil)
	_ Printer         = (*Module)(nil)
	_ DictUpdator     = (*Module)(nil)
	_ ToDictConverter = (*Module)(nil)
	_ CanCallerObject = (*Module)(nil)
)

type Modules []*Module

func (m Modules) Get(name string) *Module {
	for _, module := range m {
		if module.Info.Name == name {
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
}

type ModuleGetter interface {
	GetModule() *Module
}

type ModuleSetter interface {
	SetModule(m *Module)
}

// Module represent the module
type Module struct {
	Info          ModuleInfo
	Data          ModuleData
	Init          CallerObject
	Params        MixedParams
	ConstantIndex int
}

func NewModule(info ModuleInfo, f ...func(m *Module)) *Module {
	m := &Module{Info: info}
	m.Params.Named = make(KeyValueArray, 0)
	m.Params.Positional = make(Array, 0)

	for _, f := range f {
		f(m)
	}
	return m
}

func (m *Module) CanCall() bool {
	return m.Data == nil && m.Init != nil
}

func (m *Module) Call(c Call) (ret Object, err error) {
	if ret, err = DoCall(m.Init, c); err != nil {
		return nil, err
	}

	m.Params = *c.Params()

	switch t := ret.(type) {
	case *Module:
		m.Data = t.Data
	case Dict:
		m.Data = t
	default:
		m.Data = Dict{}
		err = ErrType.NewErrorf("module %q init result (%v) isn't dict value", m.Name(), ret.Type().Name())
	}
	ret = m
	return
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
	if m.Info.File == "" {
		s = m.Info.Name
	} else {
		s = fmt.Sprintf("%s %q", m.Info.Name, m.Info.File)
	}
	return ReprQuoteTyped("module", s)
}

func (m *Module) Name() string {
	return m.Info.Name
}

func (m *Module) File() string {
	return m.Info.File
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
		return Str(m.Info.Name), nil
	case AttrFile:
		return Str(m.Info.File), nil
	case AttrParams:
		return &m.Params, nil
	case "@data":
		return m.Data, nil
	}
	return m.Data.IndexGet(vm, index)
}

func (m *Module) IndexSet(vm *VM, index, value Object) error {
	return m.Data.IndexSet(vm, index, value)
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

func (m *Module) UpdateDict(d Dict) {
	for k, v := range m.Data.ToDict() {
		d[k] = v
	}
}

func (m *Module) ToDict() Dict {
	return m.Data.ToDict()
}
