package encoder

import (
	"github.com/gad-lang/gad"
)

// EncodeBytecodeTo encodes given bc to w io.Writer.
func EncodeBytecodeTo(bc *gad.Bytecode, w Writer) (ModulesSpec, error) {
	return nil, EncodeObject(w, bc)
}

// DecodeBytecodeFrom decodes *gad.Bytecode from given r io.Reader.
func DecodeBytecodeFrom(ctx *Context, r Reader) (*gad.Bytecode, error) {
	return DecodeT[*gad.Bytecode](r, ctx)
}
