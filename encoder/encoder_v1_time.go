// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package encoder

import "github.com/gad-lang/gad"

// Encoders/decoders for the time module value types (time, duration, date).
// The tags are registered in encoder_v1.go.
func init() {
	TimeV1.Encode = func(ctx *WriteContext, o any) (err error) {
		var data []byte
		if data, err = o.(*gad.Time).Value.MarshalBinary(); err != nil {
			return
		}
		return writeChunk(ctx, data)
	}
	TimeV1.Decode = func(ctx *ReadContext) (_ any, err error) {
		var buf []byte
		if buf, err = readChunk(ctx); err != nil {
			return
		}
		t := &gad.Time{}
		if err = t.Value.UnmarshalBinary(buf); err != nil {
			return
		}
		return t, nil
	}

	DurationV1.Encode = func(ctx *WriteContext, o any) (err error) {
		return writeInt64(ctx, int64(o.(gad.Duration)))
	}
	DurationV1.Decode = func(ctx *ReadContext) (_ any, err error) {
		var i int64
		i, err = readInt64(ctx)
		return gad.Duration(i), err
	}

	DateV1.Encode = func(ctx *WriteContext, o any) (err error) {
		return writeUint64(ctx, uint64(o.(gad.CalendarDate)))
	}
	DateV1.Decode = func(ctx *ReadContext) (_ any, err error) {
		var i uint64
		i, err = readUint64(ctx)
		return gad.CalendarDate(i), err
	}
}
