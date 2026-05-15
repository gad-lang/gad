package encoder

import (
	"errors"
	"io"
	"strconv"

	"github.com/gad-lang/gad"
)

func DecodeIterator(r Reader, init func(l int), cb func(i int) error) (err error) {
	var l int
	if l, err = readInt(r); err != nil {
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

func DecodeItems[V any](r Reader, ctx *Context, init func(l int), cb func(i int, v V) error) (err error) {
	return DecodeIterator(r, init, func(i int) (err error) {
		var v any
		if v, err = Decode(r, ctx); err != nil {
			return
		}
		return cb(i, v.(V))
	})
}

func DecodeArray[T any](r Reader, ctx *Context, init ...func(arr []T)) (arr []T, err error) {
	err = DecodeItems(r, ctx,
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

func DecodeDict(r Reader, ctx *Context) (d gad.Dict, err error) {
	err = DecodeIterator(r,
		func(l int) {
			d = make(gad.Dict, l)
		},
		func(i int) (err error) {
			var k string
			if k, err = readString(r); err != nil {
				return
			}
			var v any
			if v, err = Decode(r, ctx); err != nil {
				return
			}
			d[k] = v.(gad.Object)
			return
		},
	)
	return
}

func DecodeFields(r Reader, cb func(field uint8) error) (err error) {
	var field byte
	for {
		if field, err = r.ReadByte(); err != nil || field == FieldEOF {
			return
		}

		if err = cb(field); err != nil {
			return
		}
	}
}

func readByteFrom(r io.Reader) (byte, error) {
	if br, ok := r.(io.ByteReader); ok {
		return br.ReadByte()
	}

	var one = []byte{0}
	n, err := r.Read(one)
	if err != nil {
		if err == io.EOF {
			if n == 1 {
				return one[0], nil
			}
		}
		return 0, err
	}

	if n == 1 {
		return one[0], nil
	}
	return 0, errors.New("byte read error")
}

func Decode(r Reader, ctx *Context) (any, error) {
	version, err := readByteFrom(r)

	if err != nil {
		return nil, err
	}

	ed := Encoders.byVersion[version]

	if ed == nil {
		return nil, errors.New(
			"decode error: unknown encoding type: " + strconv.Itoa(int(version)),
		)
	}

	return ed.Decode(r, ctx)
}

func DecodeT[T any](r Reader, ctx *Context) (o T, err error) {
	var ret any
	if ret, err = Decode(r, ctx); err == nil {
		o = ret.(T)
	}
	return
}
