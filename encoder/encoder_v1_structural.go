package encoder

import "github.com/gad-lang/gad"

// (De)serialization for the structural type constants a type position compiles
// to: an array type (`[]int`, `[]<int|str>`) and a pointer type (`*int`,
// `*<int|str>`). Their element types are symbols, resolved by the running VM.
func init() {
	writeStrings := func(ctx *WriteContext, names []string) (err error) {
		if err = writeInt(ctx, len(names)); err != nil {
			return
		}
		for _, n := range names {
			if err = writeString(ctx, n); err != nil {
				return
			}
		}
		return
	}
	readStrings := func(ctx *ReadContext) (names []string, err error) {
		var n int
		if n, err = readInt(ctx); err != nil {
			return
		}
		names = make([]string, n)
		for i := range names {
			if names[i], err = readString(ctx); err != nil {
				return
			}
		}
		return
	}

	ArrayTypeV1.Encode = func(ctx *WriteContext, o any) (err error) {
		t := o.(*gad.ArrayType)
		if err = writeInt(ctx, t.Depth); err != nil {
			return
		}
		if err = EncodeArray(ctx, []*gad.SymbolInfo(t.Elem)); err != nil {
			return
		}
		return writeStrings(ctx, t.ElemNames)
	}
	ArrayTypeV1.Decode = func(ctx *ReadContext) (_ any, err error) {
		t := &gad.ArrayType{}
		if t.Depth, err = readInt(ctx); err != nil {
			return
		}
		var elem []*gad.SymbolInfo
		if elem, err = DecodeArray[*gad.SymbolInfo](ctx); err != nil {
			return
		}
		t.Elem = elem
		if t.ElemNames, err = readStrings(ctx); err != nil {
			return
		}
		return t, nil
	}

	// V2: V1, then the sizes of the levels — the levels as written, the
	// symbols of the variables they name, and a level each: Eq, Min, Max.
	writeRef := func(ctx *WriteContext, r gad.ArraySizeRef) (err error) {
		if err = writeBool(ctx, r.Set); err != nil {
			return
		}
		if err = writeInt64(ctx, r.N); err != nil {
			return
		}
		return writeInt(ctx, r.Sym)
	}
	readRef := func(ctx *ReadContext) (r gad.ArraySizeRef, err error) {
		if r.Set, err = readBool(ctx); err != nil {
			return
		}
		if r.N, err = readInt64(ctx); err != nil {
			return
		}
		r.Sym, err = readInt(ctx)
		return
	}
	ArrayTypeV2.Encode = func(ctx *WriteContext, o any) (err error) {
		t := o.(*gad.ArrayType)
		if err = ArrayTypeV1.Encode(ctx, o); err != nil {
			return
		}
		if err = writeString(ctx, t.LevelsText); err != nil {
			return
		}
		if err = EncodeArray(ctx, []*gad.SymbolInfo(t.SizeSyms)); err != nil {
			return
		}
		if err = writeInt(ctx, len(t.Levels)); err != nil {
			return
		}
		for _, l := range t.Levels {
			for _, r := range []gad.ArraySizeRef{l.Eq, l.Min, l.Max} {
				if err = writeRef(ctx, r); err != nil {
					return
				}
			}
		}
		return
	}
	ArrayTypeV2.Decode = func(ctx *ReadContext) (_ any, err error) {
		var v any
		if v, err = ArrayTypeV1.Decode(ctx); err != nil {
			return
		}
		t := v.(*gad.ArrayType)
		if t.LevelsText, err = readString(ctx); err != nil {
			return
		}
		var syms []*gad.SymbolInfo
		if syms, err = DecodeArray[*gad.SymbolInfo](ctx); err != nil {
			return
		}
		if len(syms) > 0 {
			t.SizeSyms = syms
		}
		var n int
		if n, err = readInt(ctx); err != nil {
			return
		}
		if n > 0 {
			t.Levels = make([]gad.ArrayLevel, n)
			for i := range t.Levels {
				l := &t.Levels[i]
				for _, r := range []*gad.ArraySizeRef{&l.Eq, &l.Min, &l.Max} {
					if *r, err = readRef(ctx); err != nil {
						return
					}
				}
			}
		}
		return t, nil
	}

	PtrTypeV1.Encode = func(ctx *WriteContext, o any) (err error) {
		t := o.(*gad.PtrType)
		if err = EncodeArray(ctx, []*gad.SymbolInfo(t.Elem)); err != nil {
			return
		}
		return writeStrings(ctx, t.ElemNames)
	}
	PtrTypeV1.Decode = func(ctx *ReadContext) (_ any, err error) {
		t := &gad.PtrType{}
		var elem []*gad.SymbolInfo
		if elem, err = DecodeArray[*gad.SymbolInfo](ctx); err != nil {
			return
		}
		t.Elem = elem
		if t.ElemNames, err = readStrings(ctx); err != nil {
			return
		}
		return t, nil
	}
}
