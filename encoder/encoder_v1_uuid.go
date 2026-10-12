package encoder

import (
	"fmt"

	"github.com/gad-lang/gad"
)

// The encoder/decoder of a UUID — a constant `uuid("…")` compiles to —: its
// 16 bytes. The tag is registered in encoder_v1.go.
func init() {
	UUIDV1.Encode = func(ctx *WriteContext, o any) (err error) {
		u := o.(gad.UUID)
		return writeChunk(ctx, u[:])
	}
	UUIDV1.Decode = func(ctx *ReadContext) (_ any, err error) {
		var buf []byte
		if buf, err = readChunk(ctx); err != nil {
			return
		}
		if len(buf) != len(gad.UUID{}) {
			return nil, fmt.Errorf("uuid: %d bytes, want %d", len(buf), len(gad.UUID{}))
		}
		var u gad.UUID
		copy(u[:], buf)
		return u, nil
	}
}
