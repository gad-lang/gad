package gad

import "strings"

// gad:doc
// ## Type FuncHeaderObject
// FuncHeaderObject describes a function signature: a name, positional and named
// parameters (each a `typedIdent`) and a return-type list. It is the value of a
// func-header expression `<(params) <return>>`.
//
// Members are read with indexing:
//   - `h.name` -> str
//   - `h.params` -> array of typedIdent
//   - `h.namedParams` -> array of typedIdent
//   - `h.return` -> array of typedIdent
//
// ```gad
// h := <(a int, b str) <r bool>>
// h.name           // ""
// len(h.params)    // 2
// h.params[0].name // "a"
// ```

// TFunctionHeader is the builtin `FuncHeaderObject` object type.
var TFunctionHeader = RegisterBuiltinType(BuiltinFunctionHeader, "FunctionHeader", FuncHeaderObject{}, NewFunctionHeaderFunc)

var _ IndexGetter = (*FuncHeaderObject)(nil)

// FuncHeaderObject is a function signature value (see TFunctionHeader).
type FuncHeaderObject struct {
	FuncName    string
	Params      Array // of *TypedIdent
	NamedParams Array // of *TypedIdent
	Return      Array // of *TypedIdent
	// Module is the module the header was compiled in, used to render a
	// module-qualified FullName. Set from *Compiler.module when a func-header
	// value is compiled to a constant; nil for values built at run time.
	Module *ModuleSpec
}

func (h *FuncHeaderObject) Type() ObjectType { return TFunctionHeader }

func (h *FuncHeaderObject) Name() string { return h.FuncName }

// FullName is the header name qualified by its module, e.g. `mod.f`, or just the
// name when there is no (or an unnamed) module. An anonymous header (no name)
// has no FullName.
func (h *FuncHeaderObject) FullName() string {
	if h.FuncName == "" {
		return ""
	}
	if h.Module != nil && h.Module.Name != "" {
		return h.Module.Name + "." + h.FuncName
	}
	return h.FuncName
}

func (h *FuncHeaderObject) IsFalsy() bool { return false }

func (h *FuncHeaderObject) ToString() string { return h.String() }

func (h *FuncHeaderObject) String() string {
	var b strings.Builder
	b.WriteString("<")
	b.WriteString(h.FullName())
	b.WriteString("(")
	writeTypedIdents(&b, h.Params)
	if len(h.NamedParams) > 0 {
		b.WriteString("; ")
		writeTypedIdents(&b, h.NamedParams)
	}
	b.WriteString(")")
	if len(h.Return) > 0 {
		b.WriteString(" <")
		writeTypedIdents(&b, h.Return)
		b.WriteString(">")
	}
	b.WriteString(">")
	return b.String()
}

func writeTypedIdents(b *strings.Builder, arr Array) {
	for i, o := range arr {
		if i > 0 {
			b.WriteString(", ")
		}
		if ti, _ := o.(*TypedIdent); ti != nil {
			b.WriteString(ti.Name)
			if names := ti.typeNames(); len(names) > 0 {
				b.WriteString(" ")
				b.WriteString(strings.Join(names, "|"))
			}
		} else {
			b.WriteString(o.ToString())
		}
	}
}

func (h *FuncHeaderObject) Equal(right Object) bool {
	o, ok := right.(*FuncHeaderObject)
	if !ok {
		return false
	}
	return h.FuncName == o.FuncName &&
		h.Params.Equal(o.Params) &&
		h.NamedParams.Equal(o.NamedParams) &&
		h.Return.Equal(o.Return)
}

// IndexGet exposes name, params, namedParams and return.
func (h *FuncHeaderObject) IndexGet(_ *VM, index Object) (Object, error) {
	switch index.ToString() {
	case "name":
		return Str(h.FuncName), nil
	case "params":
		return h.Params, nil
	case "namedParams":
		return h.NamedParams, nil
	case "return":
		return h.Return, nil
	}
	return nil, ErrInvalidIndex.NewError(index.ToString())
}

// NewFunctionHeaderFunc builds a FuncHeaderObject from
// (name str, params array, namedParams array, return array).
func NewFunctionHeaderFunc(c Call) (_ Object, err error) {
	var (
		name   = &Arg{Name: "name", TypeAssertion: TypeAssertionFromTypes(TStr, TRawStr)}
		params = &Arg{Name: "params", TypeAssertion: TypeAssertionFromTypes(TArray)}
		named  = &Arg{Name: "namedParams", TypeAssertion: TypeAssertionFromTypes(TArray)}
		ret    = &Arg{Name: "return", TypeAssertion: TypeAssertionFromTypes(TArray)}
	)
	if err = c.Args.Destructure(name, params, named, ret); err != nil {
		return
	}
	return &FuncHeaderObject{
		FuncName:    name.Value.ToString(),
		Params:      params.Value.(Array),
		NamedParams: named.Value.(Array),
		Return:      ret.Value.(Array),
	}, nil
}
