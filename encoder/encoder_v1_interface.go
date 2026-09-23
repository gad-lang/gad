package encoder

import "github.com/gad-lang/gad"

// (De)serialization for interface constants: *Interface and its members. Member
// back-references (Iface) are not encoded; they are restored after the parent
// Interface is decoded. The module is stored by name only (enough for FullName;
// type symbols resolve against the running VM).

func init() {
	InterfaceFieldV1.Encode = func(ctx *WriteContext, o any) (err error) {
		f := o.(*gad.InterfaceField)
		if err = writeString(ctx, f.Name); err != nil {
			return
		}
		if err = EncodeArray(ctx, f.TypesSymbols); err != nil {
			return
		}
		if err = writeBool(ctx, f.Nullable); err != nil {
			return
		}
		return encodeMeta(ctx, f.Meta)
	}
	InterfaceFieldV1.Decode = func(ctx *ReadContext) (_ any, err error) {
		f := new(gad.InterfaceField)
		if f.Name, err = readString(ctx); err != nil {
			return
		}
		if f.TypesSymbols, err = DecodeArray[*gad.SymbolInfo](ctx); err != nil {
			return
		}
		if f.Nullable, err = readBool(ctx); err != nil {
			return
		}
		if f.Meta, err = decodeMeta(ctx); err != nil {
			return
		}
		return f, nil
	}

	InterfacePropV1.Encode = func(ctx *WriteContext, o any) (err error) {
		p := o.(*gad.InterfaceProp)
		if err = writeString(ctx, p.Name); err != nil {
			return
		}
		if p.Getter != nil {
			if err = ctx.WriteByte(1); err != nil {
				return
			}
			if err = EncodeObject(ctx, p.Getter); err != nil {
				return
			}
		} else if err = ctx.WriteByte(0); err != nil {
			return
		}
		return EncodeArray(ctx, p.Setters)
	}
	InterfacePropV1.Decode = func(ctx *ReadContext) (_ any, err error) {
		p := new(gad.InterfaceProp)
		if p.Name, err = readString(ctx); err != nil {
			return
		}
		var has byte
		if has, err = ctx.ReadByte(); err != nil {
			return
		}
		if has == 1 {
			var v any
			if v, err = Decode(ctx); err != nil {
				return
			}
			p.Getter = v.(*gad.FuncHeaderObject)
		}
		if p.Setters, err = DecodeArray[*gad.FuncHeaderObject](ctx); err != nil {
			return
		}
		return p, nil
	}

	InterfaceMethodV1.Encode = func(ctx *WriteContext, o any) (err error) {
		m := o.(*gad.InterfaceMethod)
		if err = writeString(ctx, m.Name); err != nil {
			return
		}
		return EncodeArray(ctx, m.Headers)
	}
	InterfaceMethodV1.Decode = func(ctx *ReadContext) (_ any, err error) {
		m := new(gad.InterfaceMethod)
		if m.Name, err = readString(ctx); err != nil {
			return
		}
		if m.Headers, err = DecodeArray[*gad.FuncHeaderObject](ctx); err != nil {
			return
		}
		return m, nil
	}

	InterfaceContextFuncV1.Encode = func(ctx *WriteContext, o any) (err error) {
		cf := o.(*gad.InterfaceContextFunc)
		if err = writeString(ctx, cf.FnName); err != nil {
			return
		}
		// cf.Fn (the captured callable) is runtime-bound (OpInterfaceBind) and is
		// never part of the constant template, so it is not encoded.
		return EncodeArray(ctx, cf.Headers)
	}
	InterfaceContextFuncV1.Decode = func(ctx *ReadContext) (_ any, err error) {
		cf := new(gad.InterfaceContextFunc)
		if cf.FnName, err = readString(ctx); err != nil {
			return
		}
		if cf.Headers, err = DecodeArray[*gad.FuncHeaderObject](ctx); err != nil {
			return
		}
		return cf, nil
	}

	InterfaceV1.Encode = func(ctx *WriteContext, o any) (err error) {
		i := o.(*gad.Interface)
		if err = writeString(ctx, i.IName); err != nil {
			return
		}
		var moduleName string
		if i.Module != nil {
			moduleName = i.Module.Name
		}
		if err = writeString(ctx, moduleName); err != nil {
			return
		}
		if err = EncodeArray(ctx, i.Extends); err != nil {
			return
		}
		if err = EncodeArray(ctx, i.Fields); err != nil {
			return
		}
		if err = EncodeArray(ctx, i.Props); err != nil {
			return
		}
		if err = EncodeArray(ctx, i.Methods); err != nil {
			return
		}
		if err = EncodeArray(ctx, i.ContextFuncs); err != nil {
			return
		}
		// Array interface (depth + optional leaf element types), `**rest` and
		// metadata.
		if err = writeInt(ctx, i.ArrayDepth); err != nil {
			return
		}
		if err = writeBool(ctx, i.Elem != nil); err != nil {
			return
		}
		if i.Elem != nil {
			if err = EncodeObject(ctx, i.Elem); err != nil {
				return
			}
		}
		if err = writeString(ctx, i.Rest); err != nil {
			return
		}
		return encodeMeta(ctx, i.Meta)
	}
	InterfaceV1.Decode = func(ctx *ReadContext) (_ any, err error) {
		i := new(gad.Interface)
		if i.IName, err = readString(ctx); err != nil {
			return
		}
		var moduleName string
		if moduleName, err = readString(ctx); err != nil {
			return
		}
		if moduleName != "" {
			i.Module = gad.NewModuleSpecFromName(moduleName)
		}
		if i.Extends, err = DecodeArray[*gad.SymbolInfo](ctx); err != nil {
			return
		}
		if i.Fields, err = DecodeArray[*gad.InterfaceField](ctx); err != nil {
			return
		}
		if i.Props, err = DecodeArray[*gad.InterfaceProp](ctx); err != nil {
			return
		}
		if i.Methods, err = DecodeArray[*gad.InterfaceMethod](ctx); err != nil {
			return
		}
		if i.ContextFuncs, err = DecodeArray[*gad.InterfaceContextFunc](ctx); err != nil {
			return
		}
		if i.ArrayDepth, err = readInt(ctx); err != nil {
			return
		}
		var hasElem bool
		if hasElem, err = readBool(ctx); err != nil {
			return
		}
		if hasElem {
			if i.Elem, err = DecodeT[*gad.InterfaceField](ctx); err != nil {
				return
			}
			i.Elem.Iface = i
		}
		if i.Rest, err = readString(ctx); err != nil {
			return
		}
		if i.Meta, err = decodeMeta(ctx); err != nil {
			return
		}
		// Restore member back-references.
		for _, f := range i.Fields {
			f.Iface = i
		}
		for _, p := range i.Props {
			p.Iface = i
		}
		for _, m := range i.Methods {
			m.Iface = i
		}
		return i, nil
	}
}

// encodeMeta writes an optional `[k=v, …]` metadata KeyValueArray (a presence
// flag, then the array).
func encodeMeta(ctx *WriteContext, meta gad.KeyValueArray) (err error) {
	if err = writeBool(ctx, len(meta) > 0); err != nil || len(meta) == 0 {
		return
	}
	return EncodeObject(ctx, meta)
}

// decodeMeta reads what encodeMeta wrote (nil when absent).
func decodeMeta(ctx *ReadContext) (meta gad.KeyValueArray, err error) {
	var has bool
	if has, err = readBool(ctx); err != nil || !has {
		return
	}
	return DecodeT[gad.KeyValueArray](ctx)
}
