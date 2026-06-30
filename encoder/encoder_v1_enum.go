package encoder

import "github.com/gad-lang/gad"

// Enum is serialized as its module index (resolved against the bytecode's
// modules on decode, like CompiledFunction), its name, and its values in
// declaration order — each a name plus the underlying int/uint object.
func init() {
	EnumV1.Encode = func(ctx *WriteContext, o any) (err error) {
		e := o.(*gad.Enum)

		if m := e.GetModule(); m != nil {
			if err = ctx.WriteByte(0); err != nil {
				return
			}
			if err = writeInt(ctx, m.Index); err != nil {
				return
			}
		}

		if err = ctx.WriteByte(1); err != nil {
			return
		}
		if err = writeString(ctx, e.Name()); err != nil {
			return
		}

		values := e.ToArray()
		if err = ctx.WriteByte(2); err != nil {
			return
		}
		if err = writeInt(ctx, len(values)); err != nil {
			return
		}
		for _, v := range values {
			ev := v.(*gad.EnumValue)
			if err = writeString(ctx, ev.Name); err != nil {
				return
			}
			if err = EncodeObject(ctx, ev.Value); err != nil {
				return
			}
		}

		return ctx.WriteByte(FieldEOF)
	}

	EnumV1.Decode = func(ctx *ReadContext) (_ any, err error) {
		var (
			module *gad.ModuleSpec
			name   string
			names  []string
			vals   []gad.Object
		)

		err = DecodeFields(ctx, func(field uint8) (err error) {
			switch field {
			case 0:
				var idx int
				if idx, err = readInt(ctx); err != nil {
					return
				}
				module = ctx.Modules[idx]
			case 1:
				name, err = readString(ctx)
			case 2:
				var n int
				if n, err = readInt(ctx); err != nil {
					return
				}
				names = make([]string, n)
				vals = make([]gad.Object, n)
				for i := 0; i < n; i++ {
					if names[i], err = readString(ctx); err != nil {
						return
					}
					var v any
					if v, err = Decode(ctx); err != nil {
						return
					}
					vals[i] = v.(gad.Object)
				}
			}
			return
		})
		if err != nil {
			return
		}

		e := gad.NewEnum(name, module)
		for i, vn := range names {
			e.AddValue(vn, vals[i])
		}
		return e, nil
	}
}
