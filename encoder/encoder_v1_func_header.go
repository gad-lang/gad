package encoder

import "github.com/gad-lang/gad"

// (De)serialization for func-header constants: a *FuncHeaderObject and its
// *TypedIdent parameters, whose types are stored as compile-time symbols
// (reusing the SymbolInfo array codec). The header's module is stored by name
// only — enough to render its module-qualified FullName after decode; type
// symbols resolve against the running VM, not the module.

func init() {
	TypedIdentV1.Encode = func(ctx *WriteContext, o any) (err error) {
		t := o.(*gad.TypedIdent)
		if err = writeString(ctx, t.Name); err != nil {
			return
		}
		return EncodeArray(ctx, t.TypesSymbols)
	}

	TypedIdentV1.Decode = func(ctx *ReadContext) (_ any, err error) {
		t := new(gad.TypedIdent)
		if t.Name, err = readString(ctx); err != nil {
			return
		}
		if t.TypesSymbols, err = DecodeArray[*gad.SymbolInfo](ctx); err != nil {
			return
		}
		return t, nil
	}

	FuncHeaderObjectV1.Encode = func(ctx *WriteContext, o any) (err error) {
		h := o.(*gad.FuncHeaderObject)
		if err = writeString(ctx, h.FuncName); err != nil {
			return
		}
		var moduleName string
		if h.Module != nil {
			moduleName = h.Module.Name
		}
		if err = writeString(ctx, moduleName); err != nil {
			return
		}
		if err = EncodeArray(ctx, []gad.Object(h.Params)); err != nil {
			return
		}
		if err = EncodeArray(ctx, []gad.Object(h.NamedParams)); err != nil {
			return
		}
		return EncodeArray(ctx, []gad.Object(h.Return))
	}

	FuncHeaderObjectV1.Decode = func(ctx *ReadContext) (_ any, err error) {
		h := new(gad.FuncHeaderObject)
		if h.FuncName, err = readString(ctx); err != nil {
			return
		}
		var moduleName string
		if moduleName, err = readString(ctx); err != nil {
			return
		}
		if moduleName != "" {
			h.Module = gad.NewModuleSpecFromName(moduleName)
		}
		if h.Params, err = DecodeArray[gad.Object](ctx); err != nil {
			return
		}
		if h.NamedParams, err = DecodeArray[gad.Object](ctx); err != nil {
			return
		}
		if h.Return, err = DecodeArray[gad.Object](ctx); err != nil {
			return
		}
		return h, nil
	}
}
