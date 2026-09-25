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
