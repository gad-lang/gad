package gad

import (
	"strconv"
	"strings"

	"github.com/gad-lang/gad/token"
)

// gad:doc
// ## Type MethodInterface
// MethodInterface is a set of required function headers, produced by a `meti`
// expression: `meti { (), (v) <int> }`. Use `implements(fn, mi)` to test whether
// a callable provides every header, `append(mi, mi2, …)` or `mi + mi2` to merge.
//
// ```gad
// Stringer := meti { () <str> }
// implements((this) => "x", Stringer)   // true if the func matches a header
// ```

// TMethodInterface is the builtin `MethodInterface` object type.
var TMethodInterface = RegisterBuiltinType(BuiltinMethodInterface, "MethodInterface", MethodInterface{}, NewMethodInterfaceFunc)

var (
	_ IndexGetter              = (*MethodInterface)(nil)
	_ ObjectWithAddBinOperator = (*MethodInterface)(nil)
)

// MethodInterface is the value of a method interface (see
// TMethodInterface): a name and the required function headers.
type MethodInterface struct {
	MIName  string
	Headers []*FuncHeaderObject
}

func (m *MethodInterface) Type() ObjectType { return TMethodInterface }

// AssignTo makes *MethodInterface a TypeAssigner. As a target it accepts a value
// that structurally implements it; as a source it matches only an equal
// interface.
func (m *MethodInterface) AssignTo(vm *VM, obj Object, to TypeAssigner) (Object, error) {
	if ok, err := MethodInterfaceImplements(vm, obj, m); err != nil {
		return nil, err
	} else if ok {
		return obj, nil
	}
	return nil, ErrIncompatibleAssign
}

func (m *MethodInterface) CanAssign(obj Object) (bool, error) {
	switch t := obj.(type) {
	case MethodCaller:
		return MethodInterfaceImplements(nil, t, m)
	}
	return false, nil
}

func (m *MethodInterface) Name() string { return m.MIName }

func (m *MethodInterface) IsFalsy() bool { return len(m.Headers) == 0 }

func (m *MethodInterface) ToString() string { return m.String() }

func (m *MethodInterface) String() string {
	var b strings.Builder
	b.WriteString("meti ")
	b.WriteString(m.MIName)
	b.WriteString("{")
	for i, h := range m.Headers {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(h.headerString())
	}
	b.WriteString("}")
	return b.String()
}

func (m *MethodInterface) Equal(right Object) bool {
	o, ok := right.(*MethodInterface)
	if !ok || m.MIName != o.MIName || len(m.Headers) != len(o.Headers) {
		return false
	}
	for i := range m.Headers {
		if !m.Headers[i].Equal(o.Headers[i]) {
			return false
		}
	}
	return true
}

// BinOpIn implements the `in` operator (ObjectWithInBinOperator): reports
// whether v is one of the interface's function headers (`header in meti`).
func (m *MethodInterface) BinOpIn(_ *VM, v Object) (Object, error) {
	for _, h := range m.Headers {
		if h.Equal(v) {
			return True, nil
		}
	}
	return False, nil
}

// HeadersArray returns the headers as an Array of FunctionHeader values.
func (m *MethodInterface) HeadersArray() Array {
	arr := make(Array, len(m.Headers))
	for i, h := range m.Headers {
		arr[i] = h
	}
	return arr
}

// IndexGet exposes name and headers.
func (m *MethodInterface) IndexGet(_ *VM, index Object) (Object, error) {
	switch index.ToString() {
	case "name":
		return Str(m.MIName), nil
	case "headers":
		return m.HeadersArray(), nil
	}
	return nil, ErrInvalidIndex.NewError(index.ToString())
}

// BinOpAdd implements `mi + mi2` (ObjectWithAddBinOperator), merging two
// interfaces.
func (m *MethodInterface) BinOpAdd(_ *VM, right Object) (Object, error) {
	if o, ok := right.(*MethodInterface); ok {
		return mergeMethodInterfaces(m, o), nil
	}
	return nil, NewOperandTypeError(token.Add.String(), m.Type().Name(), right.Type().Name())
}

func mergeMethodInterfaces(items ...*MethodInterface) *MethodInterface {
	out := &MethodInterface{}
	for _, mi := range items {
		if out.MIName == "" {
			out.MIName = mi.MIName
		}
		out.Headers = append(out.Headers, mi.Headers...)
	}
	return out
}

// headerString renders a FuncHeaderObject as `name(params) <return>` (no angle
// brackets), for use inside a MethodInterface.
func (h *FuncHeaderObject) headerString() string {
	s := h.String()
	// FuncHeaderObject.String wraps in `<…>`; strip the brackets here.
	if len(s) >= 2 && s[0] == '<' && s[len(s)-1] == '>' {
		s = s[1 : len(s)-1]
	}
	return s
}

// BuiltinImplementsFunc reports whether fn provides every header of the given
// method interfaces: implements(fn CALLABLE, mi MethodInterface, *otherMi) <bool>.
// Matching is by parameter arity and assignable parameter types.
func BuiltinImplementsFunc(c Call) (_ Object, err error) {
	// note: validate the interface args by type assertion (not via
	// TypeAssertionFromTypes(TMethodInterface)) to avoid an init cycle
	// between BuiltinObjects, this func and TMethodInterface.
	fn := &Arg{Name: "fn", TypeAssertion: NewTypeAssertion(TypeAssertions(WithCallable()))}
	mi := &Arg{Name: "mi"}
	rest, err := c.Args.DestructureVar(fn, mi)
	if err != nil {
		return
	}

	miv, ok := mi.Value.(*MethodInterface)
	if !ok {
		return nil, NewArgumentTypeError("2nd (mi)", "MethodInterface", mi.Value.Type().Name())
	}
	ifaces := []*MethodInterface{miv}
	for i, o := range rest {
		m, ok := o.(*MethodInterface)
		if !ok {
			return nil, NewArgumentTypeError(
				strconv.Itoa(i+3)+"th (otherMi)", "MethodInterface", o.Type().Name())
		}
		ifaces = append(ifaces, m)
	}

	// collect the callable's per-method parameter-type sets
	var sigs []ParamsTypes
	if err = SplitCaller(c.VM, fn.Value,
		func(_ CallerObject, types ParamsTypes) error {
			sigs = append(sigs, types)
			return nil
		},
		func(_ CallerObject) error {
			sigs = append(sigs, nil) // no declared types: matches anything
			return nil
		}); err != nil {
		return
	}

	for _, iface := range ifaces {
		for _, h := range iface.Headers {
			if !headerMatchesAny(h, sigs) {
				return False, nil
			}
		}
	}
	return True, nil
}

// MethodInterfaceImplements reports whether value (a callable) structurally
// satisfies every header of the given method interface(s) — the same match used
// by the `implements` builtin. A non-callable value never implements.
func MethodInterfaceImplements(vm *VM, value Object, ifaces ...*MethodInterface) (bool, error) {
	if _, ok := value.(CallerObject); !ok {
		return false, nil
	}
	var sigs []ParamsTypes
	if err := SplitCaller(vm, value,
		func(_ CallerObject, types ParamsTypes) error { sigs = append(sigs, types); return nil },
		func(_ CallerObject) error { sigs = append(sigs, nil); return nil },
	); err != nil {
		return false, nil
	}
	for _, iface := range ifaces {
		for _, h := range iface.Headers {
			if !headerMatchesAny(h, sigs) {
				return false, nil
			}
		}
	}
	return true, nil
}

func headerMatchesAny(h *FuncHeaderObject, sigs []ParamsTypes) bool {
	for _, sig := range sigs {
		if sig == nil || headerMatchesSig(h, sig) {
			return true
		}
	}
	return false
}

func headerMatchesSig(h *FuncHeaderObject, sig ParamsTypes) bool {
	if len(h.Params) != len(sig) {
		return false
	}
	for i, p := range h.Params {
		ti, _ := p.(*TypedIdent)
		if ti == nil || !paramMatches(ti.Types, sig[i].Items()) {
			return false
		}
	}
	return true
}

func paramMatches(headerTypes Array, methodTypes ObjectTypes) bool {
	if len(headerTypes) == 0 || len(methodTypes) == 0 {
		return true // either side accepts anything
	}
	for _, ht := range headerTypes {
		htt, ok := ht.(ObjectType)
		if !ok {
			continue
		}
		for _, mt := range methodTypes {
			if IsAssignableTo(htt, mt) {
				return true
			}
		}
	}
	return false
}

// NewMethodInterfaceFunc builds a MethodInterface from
// (name str, *headers FunctionHeader).
func NewMethodInterfaceFunc(c Call) (_ Object, err error) {
	name := &Arg{Name: "name", TypeAssertion: TypeAssertionFromTypes(TStr, TRawStr)}
	rest, err := c.Args.DestructureVar(name)
	if err != nil {
		return
	}
	mi := &MethodInterface{MIName: name.Value.ToString()}
	for i, h := range rest {
		fh, ok := h.(*FuncHeaderObject)
		if !ok {
			return nil, NewArgumentTypeError(
				strconv.Itoa(i+1)+"th (header)", "FunctionHeader", h.Type().Name())
		}
		mi.Headers = append(mi.Headers, fh)
	}
	return mi, nil
}
