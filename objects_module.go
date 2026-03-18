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
		if module.info.Name == name {
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
	info          ModuleInfo
	data          ModuleData
	init          CallerObject
	params        MixedParams
	constantIndex int
}

func NewModule(info ModuleInfo, dict Dict, init CallerObject) *Module {
	return &Module{info: info, data: dict, init: init}
}

func (m *Module) ConstantIndex() int {
	return m.constantIndex
}

func (m *Module) SetConstantIndex(i int) {
	m.constantIndex = i
}

func (m *Module) CanCall() bool {
	return m.data == nil && m.init != nil
}

func (m *Module) Call(c Call) (ret Object, err error) {
	if ret, err = DoCall(m.init, c); err != nil {
		return nil, err
	}

	m.params = *c.Params()

	switch t := ret.(type) {
	case *Module:
		m.data = t.data
	case Dict:
		m.data = t
	default:
		m.data = Dict{}
		err = ErrType.NewErrorf("module %q init result (%v) isn't dict value", m.Name(), ret.Type().Name())
	}
	ret = m
	return
}

func (m *Module) Dict() Dict {
	return m.data.ToDict()
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
	if m.info.File == "" {
		s = m.info.Name
	} else {
		s = fmt.Sprintf("%s %q", m.info.Name, m.info.File)
	}
	return ReprQuoteTyped("module", s)
}

func (m *Module) Name() string {
	return m.info.Name
}

func (m *Module) File() string {
	return m.info.File
}

func (m *Module) Equal(right Object) bool {
	if r, ok := right.(*Module); ok {
		return m == r
	}
	return false
}

func (m *Module) Length() int {
	return m.data.Length()
}

func (m *Module) Keys() (arr Array) {
	return m.data.Keys()
}

func (m *Module) Values() (arr Array) {
	return m.data.Values()
}

func (m *Module) Items(vm *VM, cb ItemsGetterCallback) (err error) {
	return m.data.Items(vm, cb)
}

func (m *Module) Iterate(vm *VM, na *NamedArgs) Iterator {
	return m.data.Iterate(vm, na)
}

func (m *Module) IndexGet(vm *VM, index Object) (value Object, err error) {
	switch index.ToString() {
	case AttrName:
		return Str(m.info.Name), nil
	case AttrFile:
		return Str(m.info.File), nil
	case AttrParams:
		return &m.params, nil
	case "@data":
		return m.data, nil
	}
	return m.data.IndexGet(vm, index)
}

func (m *Module) IndexSet(vm *VM, index, value Object) error {
	return m.data.IndexSet(vm, index, value)
}

func (m *Module) Main() CallerObject {
	return m.init
}

func (m *Module) Print(state *PrinterState) (err error) {
	fmt.Fprintf(state, "%smodule %q", repr.QuotePrefix, m.Name())
	if file := m.File(); len(file) > 0 {
		fmt.Fprintf(state, " at %q", file)
	}
	d := Dict{}
	if m.data != nil && !m.data.IsFalsy() {
		d["@data"] = m.data
	}
	if !m.params.IsFalsy() {
		d["@params"] = Copy(&m.params)
	}
	if !d.IsFalsy() {
		state.WriteByte(' ')
		err = d.Print(state)
	}
	state.WriteString(repr.QuoteSufix)
	return
}

func (m *Module) UpdateDict(d Dict) {
	for k, v := range m.data.ToDict() {
		d[k] = v
	}
}

func (m *Module) ToDict() Dict {
	return m.data.ToDict()
}
