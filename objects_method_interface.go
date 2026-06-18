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
var TMethodInterface = RegisterBuiltinType(BuiltinMethodInterface, "MethodInterface", MethodInterfaceInstance{}, NewMethodInterfaceFunc)

var (
	_ IndexGetter           = (*MethodInterfaceInstance)(nil)
	_ BinaryOperatorHandler = (*MethodInterfaceInstance)(nil)
)

// MethodInterfaceInstance is the value of a method interface (see
// TMethodInterface): a name and the required function headers.
type MethodInterfaceInstance struct {
	MIName  string
	Headers []*FuncHeaderObject
}

func (m *MethodInterfaceInstance) Type() ObjectType { return TMethodInterface }

func (m *MethodInterfaceInstance) Name() string { return m.MIName }

func (m *MethodInterfaceInstance) IsFalsy() bool { return len(m.Headers) == 0 }

func (m *MethodInterfaceInstance) ToString() string { return m.String() }

func (m *MethodInterfaceInstance) String() string {
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

func (m *MethodInterfaceInstance) Equal(right Object) bool {
	o, ok := right.(*MethodInterfaceInstance)
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

// HeadersArray returns the headers as an Array of FunctionHeader values.
func (m *MethodInterfaceInstance) HeadersArray() Array {
	arr := make(Array, len(m.Headers))
	for i, h := range m.Headers {
		arr[i] = h
	}
	return arr
}

// IndexGet exposes name and headers.
func (m *MethodInterfaceInstance) IndexGet(_ *VM, index Object) (Object, error) {
	switch index.ToString() {
	case "name":
		return Str(m.MIName), nil
	case "headers":
		return m.HeadersArray(), nil
	}
	return nil, ErrInvalidIndex.NewError(index.ToString())
}

// BinaryOp implements `mi + mi2`, merging two interfaces.
func (m *MethodInterfaceInstance) BinaryOp(_ *VM, tok token.Token, right Object) (Object, error) {
	if tok == token.Add {
		if o, ok := right.(*MethodInterfaceInstance); ok {
			return mergeMethodInterfaces(m, o), nil
		}
	}
	return nil, NewOperandTypeError(tok.String(), m.Type().Name(), right.Type().Name())
}

func mergeMethodInterfaces(items ...*MethodInterfaceInstance) *MethodInterfaceInstance {
	out := &MethodInterfaceInstance{}
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

	miv, ok := mi.Value.(*MethodInterfaceInstance)
	if !ok {
		return nil, NewArgumentTypeError("2nd (mi)", "MethodInterface", mi.Value.Type().Name())
	}
	ifaces := []*MethodInterfaceInstance{miv}
	for i, o := range rest {
		m, ok := o.(*MethodInterfaceInstance)
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

// NewMethodInterfaceFunc builds a MethodInterfaceInstance from
// (name str, *headers FunctionHeader).
func NewMethodInterfaceFunc(c Call) (_ Object, err error) {
	name := &Arg{Name: "name", TypeAssertion: TypeAssertionFromTypes(TStr, TRawStr)}
	rest, err := c.Args.DestructureVar(name)
	if err != nil {
		return
	}
	mi := &MethodInterfaceInstance{MIName: name.Value.ToString()}
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
