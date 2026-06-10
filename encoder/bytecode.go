package encoder

import (
	"github.com/gad-lang/gad"
)

// EncodeBytecodeTo encodes given bc to w io.Writer.
func EncodeBytecodeTo(ctx *WriteContext, bc *gad.Bytecode) (ModulesSpec, error) {
	return nil, EncodeObject(ctx, bc)
}

// DecodeBytecodeFrom decodes *gad.Bytecode from given r io.Reader.
func DecodeBytecodeFrom(ctx *ReadContext) (*gad.Bytecode, error) {
	return DecodeT[*gad.Bytecode](ctx)
}
