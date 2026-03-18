package gad

import "strings"

type ParamType []*SymbolInfo

func (t ParamType) String() string {
	l := len(t)
	switch l {
	case 0:
		return ""
	case 1:
		return t[0].Name
	default:
		var s = make([]string, len(t))
		for i, symbol := range t {
			s[i] = symbol.Name
		}
		return strings.Join(s, "|")
	}
}

func (t ParamType) Accept(vm *VM, ot ObjectType) (ok bool, err error) {
	if len(t) == 0 {
		ok = true
		return
	}

	ot = vm.ResolveType(ot)
	var st Object

	for _, symbol := range t {
		if st, err = vm.GetSymbolValue(symbol); err != nil {
			return
		} else if st == TAny {
			return true, nil
		} else {
			if ot == st {
				ok = true
				return
			} else if stot, _ := st.(ObjectType); stot != nil {
				if ok = IsTypeAssignableTo(stot, ot); ok {
					return
				}
			}
		}
	}
	return
}

type ParamOption func(p *Param)

func ParamWithType(t ...*SymbolInfo) ParamOption {
	return func(p *Param) {
		p.TypesSymbols = t
	}
}
func ParamWithTypeO(t ...ObjectType) ParamOption {
	return func(p *Param) {
		p.Types = t
	}
}

type Param struct {
	Name         string
	TypesSymbols ParamType
	Types        ObjectTypes
	Var          bool
	Symbol       *SymbolInfo
	Usage        string
	Index        int
}

func NewParam(name string, opt ...ParamOption) *Param {
	p := &Param{Name: name}
	for _, opt := range opt {
		opt(p)
	}
	return p
}

func (p *Param) String() string {
	var b strings.Builder
	if p.Var {
		b.WriteByte('*')
	}
	b.WriteString(p.Name)
	b.WriteByte(' ')
	if len(p.TypesSymbols) > 0 {
		b.WriteString(p.TypesSymbols.String())
	} else if len(p.Types) > 0 {
		b.WriteString(p.Types.String())
	} else {
		b.WriteString(ObjectTypes{TAny}.String())
	}
	return b.String()
}

type Params struct {
	Items    []*Param
	len      int
	variadic bool
	byName   map[string]int
}

func NewParams(params ...*Param) (np *Params) {
	for i, param := range params {
		param.Index = i
	}

	np = &Params{Items: params}
	np.len = len(params)
	np.Items = params

	if np.len > 0 {
		np.byName = make(map[string]int, np.len)
		for i, p := range params {
			np.byName[p.Name] = i
		}
		np.variadic = params[len(params)-1].Var
	}
	return
}

func (p *Params) BuildTypes() (t ParamsTypes) {
	t = make(ParamsTypes, len(p.Items))

	for i, p := range p.Items {
		pt := p.Types
		if len(pt) == 0 {
			pt = ObjectTypes{TAny}
		}
		if p.Var {
			t[i] = VarParamTypes(pt)
		} else {
			t[i] = pt
		}
	}

	return
}

func (p *Params) Names() (names []string) {
	names = make([]string, p.len)
	for i, param := range p.Items {
		names[i] = param.Name
	}
	return
}

func (p *Params) Len() int {
	return p.len
}

func (p *Params) PosLen() int {
	l := p.len
	if p.variadic {
		l--
	}
	return l
}

func (p *Params) Variadic() bool {
	return p.variadic
}

func (p *Params) ByName() map[string]int {
	return p.byName
}

func (p *Params) ToMap() (np map[string]*Param) {
	np = make(map[string]*Param, p.len)
	for _, param := range p.Items {
		np[param.Name] = param
	}
	return np
}

func (p *Params) String() string {
	var s = make([]string, p.len)
	for i, param := range p.Items {
		s[i] = param.String()
	}
	return strings.Join(s, ", ")
}

func (p *Params) Var() bool {
	return p.variadic
}

func (p *Params) Typed() bool {
	for _, param := range p.Items {
		if len(param.TypesSymbols) > 0 {
			return true
		}
	}
	return false
}

func (p *Params) Empty() bool {
	return p.len == 0
}

func (p Params) RequiredCount() (n int) {
	n = len(p.Items)
	if p.variadic {
		n--
	}
	return
}

type NamedParam struct {
	Name string
	// Value is a script of default value
	Value        string
	Usage        string
	Index        int
	TypesSymbols ParamType
	Types        ObjectTypes
	Symbol       *SymbolInfo
	Var          bool
}

func (p *NamedParam) String() string {
	var b strings.Builder
	if p.Var {
		b.WriteString("**")
	}
	b.WriteString(p.Name)
	if l := len(p.TypesSymbols); l > 0 {
		b.WriteByte(' ')
		if l == 1 {
			b.WriteString(p.TypesSymbols[0].Name)
		} else {
			s := make([]string, l)
			for i, info := range p.TypesSymbols {
				s[i] = info.Name
			}
			b.WriteString(strings.Join(s, "|"))
		}
	}

	if len(p.Value) > 0 {
		b.WriteString("=" + p.Value)
	}
	return b.String()
}

func NewNamedParam(name string, value string) *NamedParam {
	return &NamedParam{Name: name, Value: value}
}

func NewVarNamedParam(name string) *NamedParam {
	return &NamedParam{Name: name, Var: true}
}

type NamedParams struct {
	Items    []*NamedParam
	len      int
	variadic bool
	byName   map[string]int
}

func NewNamedParams(params ...*NamedParam) (np *NamedParams) {
	for i, param := range params {
		param.Index = i
	}
	np = &NamedParams{Items: params}
	np.len = len(params)
	np.Items = params

	if np.len > 0 {
		np.byName = make(map[string]int, np.len)
		for i, p := range params {
			if _, ok := np.byName[p.Name]; ok {
				panic("duplicated named param: " + p.Name)
			}
			np.byName[p.Name] = i
		}
		np.variadic = params[len(params)-1].Var
	}
	return
}

func (n *NamedParams) Names() (names []string) {
	names = make([]string, n.len)
	for i, param := range n.Items {
		names[i] = param.Name
	}
	return
}

func (n *NamedParams) EachNonVar(cb func(i int, p *NamedParam)) {
	items := n.Items
	if n.variadic {
		items = items[:len(n.Items)-1]
	}

	for i, item := range items {
		cb(i, item)
	}
}

func (n *NamedParams) Len() int {
	return n.len
}

func (n *NamedParams) Variadic() bool {
	return n.variadic
}

func (n *NamedParams) ByName() map[string]int {
	return n.byName
}

func (n *NamedParams) ToMap() (np map[string]*NamedParam) {
	np = make(map[string]*NamedParam, n.len)
	for _, param := range n.Items {
		np[param.Name] = param
	}
	return np
}

func (n *NamedParams) String() string {
	var s = make([]string, n.len)
	for i, param := range n.Items {
		s[i] = param.String()
	}
	return strings.Join(s, ", ")
}

type FunctionHeaderParam struct {
	Name  string
	Types []ObjectType
	Value string
}

func (p *FunctionHeaderParam) String() string {
	var (
		s = p.Name
		l = len(p.Types)
	)
	switch l {
	case 0:
	case 1:
		s += " " + p.Types[0].Name()
	default:
		var s2 = make([]string, l)
		for i, t2 := range p.Types {
			s2[i] = t2.Name()
		}
		s += " [" + strings.Join(s2, ", ") + "]"
	}
	if p.Value != "" {
		s += "=" + p.Value
	}
	return s
}

type FunctionHeader struct {
	Params      Params
	NamedParams NamedParams
	pt          ParamsTypes
}

func NewFunctionHeader() *FunctionHeader {
	return &FunctionHeader{}
}

func (h *FunctionHeader) String() string {
	var s []string
	if h.Params.len > 0 {
		s = append(s, h.Params.String())
	}
	if h.NamedParams.len > 0 {
		s = append(s, "; ", h.NamedParams.String())
	}
	return "(" + strings.Join(s, "") + ")"
}

func (h *FunctionHeader) ParamTypes() ParamsTypes {
	if h.pt != nil {
		return h.pt
	}

	h.pt = h.Params.BuildTypes()
	return h.pt
}

type ParamBuilder struct {
	name     string
	types    ObjectTypes
	variadic bool
	usage    string
}

func (b *ParamBuilder) Var() *ParamBuilder {
	b.variadic = true
	return b
}

func (b *ParamBuilder) Type(typ ...ObjectType) *ParamBuilder {
	b.types = append(b.types, typ...)
	return b
}

func (b *ParamBuilder) Usage(v string) *ParamBuilder {
	b.usage = v
	return b
}

func (h *FunctionHeader) WithParams(builder func(newParam func(name string) *ParamBuilder)) *FunctionHeader {
	var params []*ParamBuilder
	builder(func(name string) *ParamBuilder {
		p := &ParamBuilder{name: name}
		params = append(params, p)
		return p
	})
	for _, p := range params {
		h.Params.Items = append(h.Params.Items, &Param{
			Name:  p.name,
			Types: p.types,
			Var:   p.variadic,
			Usage: p.usage,
		})
	}
	h.Params = *NewParams(h.Params.Items...)
	return h
}

type NamedParamBuilder struct {
	name     string
	types    ObjectTypes
	variadic bool
	usage    string
}

func (b *NamedParamBuilder) Var() *NamedParamBuilder {
	b.variadic = true
	return b
}

func (b *NamedParamBuilder) Type(typ ...ObjectType) *NamedParamBuilder {
	b.types = append(b.types, typ...)
	return b
}

func (b *NamedParamBuilder) Usage(v string) *NamedParamBuilder {
	b.usage = v
	return b
}

func (h *FunctionHeader) WithNamedParams(builder func(newParam func(name string) *NamedParamBuilder)) *FunctionHeader {
	var params []*NamedParamBuilder
	builder(func(name string) *NamedParamBuilder {
		p := &NamedParamBuilder{name: name}
		params = append(params, p)
		return p
	})
	for _, p := range params {
		h.NamedParams.Items = append(h.NamedParams.Items, &NamedParam{Name: p.name, Types: p.types, Var: p.variadic, Usage: p.usage})
	}
	h.NamedParams = *NewNamedParams(h.NamedParams.Items...)
	return h
}
