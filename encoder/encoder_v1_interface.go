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
		return EncodeArray(ctx, f.TypesSymbols)
	}
	InterfaceFieldV1.Decode = func(ctx *ReadContext) (_ any, err error) {
		f := new(gad.InterfaceField)
		if f.Name, err = readString(ctx); err != nil {
			return
		}
		if f.TypesSymbols, err = DecodeArray[*gad.SymbolInfo](ctx); err != nil {
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
		return EncodeArray(ctx, i.Methods)
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
