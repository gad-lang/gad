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
		return "[" + strings.Join(s, ", ") + "]"
	}
}

func (t ParamType) Accept(vm *VM, ot ObjectType) (ok bool, err error) {
	if len(t) == 0 {
		ok = true
		return
	}

	var st Object

	for _, symbol := range t {
		if st, err = vm.GetSymbolValue(symbol); err != nil {
			return
		} else {
			if cwm, _ := st.(*CallerObjectWithMethods); cwm != nil {
				st = cwm.CallerObject
			}
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
		p.Type = t
	}
}
func ParamWithTypeO(t ...ObjectType) ParamOption {
	return func(p *Param) {
		p.TypeO = t
	}
}

type Param struct {
	Name  string
	Type  ParamType
	TypeO ObjectTypes
	Var   bool
	Usage string
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
	if len(p.Type) > 0 {
		b.WriteByte(' ')
		b.WriteString(p.Type.String())
	} else if len(p.TypeO) > 0 {
		b.WriteByte(' ')
		b.WriteString(p.TypeO.String())
	}
	return b.String()
}

type Params []*Param

func (p Params) Var() bool {
	if l := len(p); l > 0 && p[l-1].Var {
		return true
	}
	return false
}

func (p Params) Typed() bool {
	for _, param := range p {
		if len(param.Type) > 0 {
			return true
		}
	}
	return false
}

func (p Params) Empty() bool {
	return len(p) == 0
}

func (p Params) String() string {
	var s = make([]string, len(p))
	for i, p := range p {
		s[i] = p.String()
	}
	return strings.Join(s, ", ")
}

func (p Params) RequiredCount() (n int) {
	n = len(p)
	if p.Var() {
		n--
	}
	return
}

type NamedParam struct {
	Name string
	// Value is a script of default value
	Value string
	Usage string
	Index int
	Type  []*SymbolInfo
	Var   bool
}

func (p *NamedParam) String() string {
	var b strings.Builder
	if p.Var {
		b.WriteString("**")
	}
	b.WriteString(p.Name)
	if l := len(p.Type); l > 0 {
		b.WriteByte(' ')
		if l == 1 {
			b.WriteString(p.Type[0].Name)
		} else {
			s := make([]string, l)
			for i, info := range p.Type {
				s[i] = info.Name
			}
			b.WriteString("[" + strings.Join(s, ", ") + "]")
		}
	}
	return b.String()
}

func NewNamedParam(name string, value string) *NamedParam {
	return &NamedParam{Name: name, Value: value}
}

type NamedParams struct {
	Params   []*NamedParam
	len      int
	variadic bool
	byName   map[string]int
}

func NewNamedParams(params ...*NamedParam) (np *NamedParams) {
	for i, param := range params {
		param.Index = i
	}
	np = &NamedParams{Params: params}
	np.len = len(params)
	np.Params = params

	if np.len > 0 {
		np.byName = make(map[string]int, np.len)
		for i, p := range params {
			np.byName[p.Name] = i
		}
		np.variadic = params[len(params)-1].Value == ""
	}
	return
}

func (n *NamedParams) Names() (names []string) {
	names = make([]string, n.len)
	for i, param := range n.Params {
		names[i] = param.Name
	}
	return
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
	for _, param := range n.Params {
		np[param.Name] = param
	}
	return np
}

func (n *NamedParams) String() string {
	var s = make([]string, n.len)
	for i, param := range n.Params {
		if param.Value != "" {
			s[i] = param.Name + "=" + param.Value
		} else {
			s[i] = "**" + param.Name
		}
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
}

func (h *FunctionHeader) String() string {
	var s []string
	for _, param := range h.Params {
		s = append(s, param.String())
	}
	for _, param := range h.NamedParams.Params {
		s = append(s, param.String())
	}
	return "(" + strings.Join(s, ", ") + ")"
}

func (h *FunctionHeader) ParamTypes() (t MultipleObjectTypes) {
	t = make(MultipleObjectTypes, len(h.Params))
	for i, p := range h.Params {
		t[i] = p.TypeO
	}
	return
}

type CallerOption func(c *Caller)

func WithParams(p ...*Param) CallerOption {
	return func(c *Caller) {
		c.Header.Params = p
	}
}

func WithNamedParams(p ...*NamedParam) CallerOption {
	return func(c *Caller) {
		c.Header.NamedParams = *NewNamedParams(p...)
	}
}

type Caller struct {
	Header FunctionHeader
	CallerObject
}

func NewCaller(co CallerObject, opt ...CallerOption) *Caller {
	c := &Caller{CallerObject: co}
	for _, o := range opt {
		o(c)
	}
	return c
}

func (c *Caller) GetHeader() *FunctionHeader {
	return &c.Header
}
