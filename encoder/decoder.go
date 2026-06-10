package encoder

import (
	"errors"
	"strconv"

	"github.com/gad-lang/gad"
)

func DecodeIterator(ctx *ReadContext, init func(l int), cb func(i int) error) (err error) {
	var l int
	if l, err = readInt(ctx); err != nil {
		return
	}

	if init != nil {
		init(l)
	}

	for i := 0; i < l; i++ {
		if err = cb(i); err != nil {
			return
		}
	}
	return
}

func DecodeItems[V any](ctx *ReadContext, init func(l int), cb func(i int, v V) error) (err error) {
	return DecodeIterator(ctx, init, func(i int) (err error) {
		var v any
		if v, err = Decode(ctx); err != nil {
			return
		}
		return cb(i, v.(V))
	})
}

func DecodeArray[T any](ctx *ReadContext, init ...func(arr []T)) (arr []T, err error) {
	err = DecodeItems(ctx,
		func(l int) {
			arr = make([]T, l)
			for _, f := range init {
				f(arr)
			}
		},
		func(i int, v T) error {
			arr[i] = v
			return nil
		},
	)
	return
}

func DecodeDict(ctx *ReadContext) (d gad.Dict, err error) {
	err = DecodeIterator(ctx,
		func(l int) {
			d = make(gad.Dict, l)
		},
		func(i int) (err error) {
			var k string
			if k, err = readString(ctx); err != nil {
				return
			}
			var v any
			if v, err = Decode(ctx); err != nil {
				return
			}
			d[k] = v.(gad.Object)
			return
		},
	)
	return
}

func DecodeFields(ctx *ReadContext, cb func(field uint8) error) (err error) {
	var field byte
	for {
		if field, err = ctx.ReadByte(); err != nil || field == FieldEOF {
			return
		}

		if err = cb(field); err != nil {
			return
		}
	}
}

func Decode(ctx *ReadContext) (any, error) {
	typeID, err := ctx.ReadByte()
	if err != nil {
		return nil, err
	}
	version, err := ctx.ReadByte()
	if err != nil {
		return nil, err
	}

	versions := Encoders.byTypeVersion[typeID]
	if versions == nil {
		return nil, errors.New(
			"decode error: unknown type: " + strconv.Itoa(int(typeID)),
		)
	}

	ed := versions[version]
	if ed == nil {
		return nil, errors.New(
			"decode error: unknown version " + strconv.Itoa(int(version)) +
				" for type " + strconv.Itoa(int(typeID)),
		)
	}

	return ed.Decode(ctx)
}

func DecodeT[T any](ctx *ReadContext) (o T, err error) {
	var ret any
	if ret, err = Decode(ctx); err == nil {
		o = ret.(T)
	}
	return
}
