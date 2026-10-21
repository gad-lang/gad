package gad

import (
	"reflect"
	"sort"
	"strings"

	"github.com/gad-lang/gad/token"
)

// TypeMember is a member of a type, as its documentation shows it: its group
// (Constructors, Methods, Properties, Operators, Conversions, Overloads), its
// signature and its doc.
type TypeMember struct {
	Group     string
	Signature string
	Doc       string
}

// The groups of a type's members, in the order they are documented.
const (
	TypeMemberConstructors = "Constructors"
	TypeMemberMethods      = "Methods"
	TypeMemberProperties   = "Properties"
	TypeMemberOperators    = "Operators"
	TypeMemberConversions  = "Conversions"
	TypeMemberOverloads    = "Overloads"
)

var typeMemberGroupOrder = map[string]int{
	TypeMemberConstructors: 0,
	TypeMemberMethods:      1,
	TypeMemberProperties:   2,
	TypeMemberOperators:    3,
	TypeMemberConversions:  4,
	TypeMemberOverloads:    5,
}

// DescribeType is what builtins b register of the type t, discovered — its
// members for its documentation:
//
//   - Constructors: the overloads of t itself (`str(v int)`);
//   - Methods, Properties: what t registers (a builtin type's methods,
//     getters and setters);
//   - Operators: the operators its values implement (`a + b`, `-a`, `a += b`),
//     the methods BinOp…/UnOp…/SelfAssignOp… of their Go type;
//   - Conversions: the constructors of other types that take a t (`int(v str)`);
//   - Overloads: the other builtins with an overload that takes a t
//     (`len(v str)`), a method registered on them (`met len(…)`) too.
func DescribeType(b *Builtins, t ObjectType) (members []TypeMember) {
	if bt, ok := t.(*BuiltinObjType); ok {
		ctors := callerMembers(TypeMemberConstructors, t.Name(), bt.FuncSpec, nil)
		if len(ctors) == 0 {
			// no overload: its constructor is the default (`array(*args)`)
			if d, ok := bt.FuncSpec.CallerMethodDefault().(*Function); ok {
				sig := t.Name() + "(…)" // its arguments, not declared
				if d.Header != nil {
					sig = t.Name() + d.Header.String()
				}
				ctors = []TypeMember{{Group: TypeMemberConstructors, Signature: sig, Doc: d.Usage}}
			}
		}
		members = append(members, ctors...)
		members = append(members, typeDictMembers(TypeMemberMethods, bt.Methods())...)
		members = append(members, typeDictMembers(TypeMemberProperties, bt.Getters())...)
		members = append(members, typeDictMembers(TypeMemberProperties, bt.Setters())...)
	}
	members = append(members, operatorMembers(t)...)

	names := make([]string, 0, len(b.NameSet))
	for name := range b.NameSet {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		o := b.Objects[b.NameSet[name]]
		if o == nil || o == Object(t) || strings.Contains(name, ".") {
			continue
		}
		spec := funcSpecOf(o)
		if spec == nil {
			continue
		}
		group := TypeMemberOverloads
		if _, isType := o.(ObjectType); isType {
			group = TypeMemberConversions
		}
		members = append(members, callerMembers(group, name, spec, func(m *TypedCallerMethod) bool {
			for _, pt := range m.Types() {
				if pt != nil && pt.Equal(t) {
					return true
				}
			}
			return false
		})...)
	}
	sort.SliceStable(members, func(i, j int) bool {
		gi, gj := typeMemberGroupOrder[members[i].Group], typeMemberGroupOrder[members[j].Group]
		if gi != gj {
			return gi < gj
		}
		return members[i].Signature < members[j].Signature
	})
	return
}

// funcSpecOf is the FuncSpec of a callable with overloads, nil when it has
// none.
func funcSpecOf(o Object) (spec *FuncSpec) {
	defer func() {
		if recover() != nil {
			spec = nil
		}
	}()
	switch t := o.(type) {
	case *BuiltinObjType:
		return t.FuncSpec
	case *BuiltinFunctionWithMethods:
		return t.FuncSpec
	case *Func:
		return t.FuncSpec
	}
	return nil
}

// callerMembers are the overloads of spec (those keep accepts, all when nil),
// as members of group named name.
func callerMembers(group, name string, spec *FuncSpec, keep func(m *TypedCallerMethod) bool) (out []TypeMember) {
	if spec == nil {
		return
	}
	spec.CallerMethods().Walk(func(m *TypedCallerMethod) any {
		if keep != nil && !keep(m) {
			return nil
		}
		out = append(out, callerMember(group, name, m))
		return nil
	})
	return
}

// callerMember is the overload m of name: its header's signature and its
// usage when it is a documented function.
func callerMember(group, name string, m *TypedCallerMethod) TypeMember {
	var co Object
	if m.CallerMethod != nil {
		co = m.CallerMethod.CallerObject
	}
	switch f := co.(type) {
	case *Function:
		if f.Header != nil {
			return TypeMember{Group: group, Signature: name + f.Header.String(), Doc: f.Usage}
		}
	case *BuiltinFunction:
		if f.Header != nil {
			return TypeMember{Group: group, Signature: name + f.Header.String(), Doc: f.Usage}
		}
	}
	types := make([]string, len(m.Types()))
	for i, t := range m.Types() {
		if t == nil {
			types[i] = "any"
		} else {
			types[i] = t.Name()
		}
	}
	return TypeMember{Group: group, Signature: name + "(" + strings.Join(types, ", ") + ")"}
}

// typeDictMembers are the callables of d (a type's methods, getters, setters).
func typeDictMembers(group string, d Dict) (out []TypeMember) {
	keys := make([]string, 0, len(d))
	for k := range d {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		sig := k
		switch f := d[k].(type) {
		case *Function:
			if f.Header != nil {
				sig += f.Header.String()
			}
			out = append(out, TypeMember{Group: group, Signature: sig, Doc: f.Usage})
			continue
		}
		out = append(out, TypeMember{Group: group, Signature: sig})
	}
	return
}

// operatorNames maps a token's name (`Add`) to its symbol (`+`).
var operatorNames = func() map[string]string {
	m := make(map[string]string, token.NumTokens)
	for tok := token.Token(0); int(tok) < token.NumTokens; tok++ {
		m[tok.Name()] = tok.String()
	}
	return m
}()

// operatorMembers are the operators the values of t implement: the methods
// BinOp{Op}, UnOp{Op}, SelfAssignOp{Op} of their Go type (Types).
func operatorMembers(t ObjectType) (out []TypeMember) {
	var rt reflect.Type
	for goType, ot := range Types {
		if ot == t {
			rt = goType
			break
		}
	}
	if rt == nil {
		return
	}
	seen := map[string]bool{}
	for _, typ := range []reflect.Type{rt, reflect.PointerTo(rt)} {
		for i := 0; i < typ.NumMethod(); i++ {
			name := typ.Method(i).Name
			var sig string
			switch {
			case strings.HasPrefix(name, "BinOp"):
				if op, ok := operatorNames[strings.TrimPrefix(name, "BinOp")]; ok {
					sig = "a " + op + " b"
				}
			case strings.HasPrefix(name, "UnOp"):
				if op, ok := operatorNames[strings.TrimPrefix(name, "UnOp")]; ok {
					if op == "++" || op == "--" {
						sig = "a" + op // `x++` is postfix
					} else {
						sig = op + "a"
					}
				}
			case strings.HasPrefix(name, "SelfAssignOp"):
				if op, ok := operatorNames[strings.TrimPrefix(name, "SelfAssignOp")]; ok {
					sig = "a " + op + "= b"
				}
			}
			if sig != "" && !seen[sig] {
				seen[sig] = true
				out = append(out, TypeMember{Group: TypeMemberOperators, Signature: sig})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Signature < out[j].Signature })
	return
}
