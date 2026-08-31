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

// Accept reports whether obj satisfies the parameter type list. An ObjectType
// entry is matched by the (resolved) type of obj (assignability); a structural
// entry (a meti/interface, i.e. a TypeAssigner that is not an ObjectType) is
// matched by its CanAssign check. An empty list accepts anything.
func (t ParamType) Accept(vm *VM, obj Object) (ok bool, err error) {
	return t.AcceptResolve(vm, obj, vm.GetSymbolValue)
}

// AcceptResolve is Accept with a custom symbol resolver. A compiled function
// resolves its own free-var type symbols against its closure (see
// CompiledFunction.paramTypeSymbolValue) rather than the current frame, because
// argument validation runs before the callee's frame (and its free vars) exist —
// resolving against vm.curFrame would read the caller's slots.
func (t ParamType) AcceptResolve(vm *VM, obj Object, resolve func(*SymbolInfo) (Object, error)) (ok bool, err error) {
	if len(t) == 0 {
		ok = true
		return
	}

	// ot (the resolved type of obj) is only needed to compare against a concrete
	// ObjectType entry; resolve it lazily so the common `TAny` / structural cases
	// avoid the ResolveType map lookup on the hot per-call path.
	var (
		st     Object
		ot     ObjectType
		otDone bool
	)
	for _, symbol := range t {
		if st, err = resolve(symbol); err != nil {
			return
		}
		if st == TAny {
			return true, nil
		}
		if mt, isMeta := st.(MetaType); isMeta {
			// A `type<X>` parameter matches on the argument VALUE (the argument must
			// itself be the type X), not on the argument's type — MetaType is an
			// ObjectType, so it must be checked before the type-level branch below.
			if ok, err = mt.CanAssign(obj); err != nil || ok {
				return
			}
		} else if stot, isOT := st.(ObjectType); isOT {
			if !otDone {
				ot, otDone = vm.ResolveType(obj.Type()), true
			}
			if ot == stot || IsTypeAssignableTo(stot, ot) {
				return true, nil
			}
		} else if vta, isVTA := st.(vmCanAssigner); isVTA {
			// Structural types (meti/interface) may need the VM to resolve
			// candidate signatures (SplitCaller -> ParamTypes(vm)).
			if ok, err = vta.CanAssignVM(vm, obj); err != nil || ok {
				return
			}
		} else if ta, isTA := st.(TypeAssigner); isTA {
			if ok, err = ta.CanAssign(obj); err != nil || ok {
				return
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

type ReturnVar struct {
	Name         string
	TypesSymbols ParamType
	Types        ObjectTypes
	// Assigners carries structural return types (a builtin interface such as
	// `callable`) that are not ObjectTypes and are not resolved through a compiled
	// symbol. When set it takes precedence over Types/TypesSymbols for display.
	Assigners TypeAssignerArray
}

type ReturnVars []*ReturnVar

// String renders a function return-type list as " <T1, T2, ...>".
// It returns an empty string when there are no return types.
func (v ReturnVars) String() string {
	if len(v) == 0 {
		return ""
	}
	s := make([]string, len(v))
	for i, t := range v {
		s[i] = t.String()
	}
	return "<" + strings.Join(s, ", ") + ">"
}

func (v ReturnVars) Types() (t []ObjectTypes) {
	t = make([]ObjectTypes, len(v))
	for i, rv := range v {
		t[i] = rv.Types
	}
	return
}

func (v ReturnVars) VMTypes(vm *VM) (t []ObjectTypes, err error) {
	t = make([]ObjectTypes, len(v))

	for i, p := range v {
		var ts ObjectTypes
		if len(p.TypesSymbols) > 0 {
			ts = make(ObjectTypes, len(p.TypesSymbols))
			for i2, symbol := range p.TypesSymbols {
				if typ, err := vm.GetSymbolValue(symbol); err != nil {
					return nil, err
				} else {
					ts[i2] = typ.(ObjectType)
				}
			}
		} else {
			ts = ObjectTypes{TAny}
		}
		t[i] = ts
	}

	return
}

// String renders a return type as "type" (anonymous) or "name type" (named),
// where multiple types are joined by "|".
func (v *ReturnVar) String() string {
	var b strings.Builder
	if v.Name != "" {
		b.WriteString(v.Name)
		b.WriteByte(' ')
	}
	switch {
	case len(v.Assigners) > 0:
		names := make([]string, len(v.Assigners))
		for i, a := range v.Assigners {
			names[i] = TypeAssignerName(a)
		}
		b.WriteString(strings.Join(names, "|"))
	case len(v.TypesSymbols) > 0:
		b.WriteString(v.TypesSymbols.String())
	case len(v.Types) > 0:
		b.WriteString(v.Types.String())
	default:
		b.WriteString(ObjectTypes{TAny}.String())
	}
	return b.String()
}

func FormatReturnVars(vars ReturnVars) string {
	if len(vars) == 0 {
		return ""
	}
	return " " + vars.String()
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

func NewNamedParam(name, value string) *NamedParam {
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
	ReturnVars  ReturnVars
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
	return "(" + strings.Join(s, "") + ")" + FormatReturnVars(h.ReturnVars)
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

// WithReturnVars declares the function's return types. A type may be any
// TypeAssigner: an ObjectType (a concrete type) or a structural builtin interface
// such as `callable`. It is intended for Go-registered builtins, so the types are
// held directly (no compiled symbol).
func (h *FunctionHeader) WithReturnVars(builder func(ret func(name string, typ ...TypeAssigner))) *FunctionHeader {
	builder(func(name string, typ ...TypeAssigner) {
		rv := &ReturnVar{Name: name}
		if len(typ) == 0 {
			rv.Assigners = TypeAssignerArray{TAny}
		} else {
			rv.Assigners = append(rv.Assigners, typ...)
		}
		h.ReturnVars = append(h.ReturnVars, rv)
	})
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
