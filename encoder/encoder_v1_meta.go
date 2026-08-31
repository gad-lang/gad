package encoder

import "github.com/gad-lang/gad"

// (De)serialization for a MetaType constant (`type<X>`). Only the compiled form is
// serialized: it carries the target's symbol (X resolves against the running VM);
// the resolved Target is never part of the bytecode.
func init() {
	MetaTypeV1.Encode = func(ctx *WriteContext, o any) (err error) {
		m := o.(gad.MetaType)
		if err = writeBool(ctx, m.TargetSym != nil); err != nil {
			return
		}
		if m.TargetSym != nil {
			return EncodeObject(ctx, m.TargetSym)
		}
		return
	}

	MetaTypeV1.Decode = func(ctx *ReadContext) (_ any, err error) {
		var has bool
		if has, err = readBool(ctx); err != nil {
			return
		}
		m := gad.MetaType{}
		if has {
			var o any
			if o, err = Decode(ctx); err != nil {
				return
			}
			m.TargetSym, _ = o.(*gad.SymbolInfo)
		}
		return m, nil
	}
}
