package flate

import (
	"github.com/gad-lang/gad"
)

var Module = gad.Dict{
	"encode": &gad.Function{
		FuncName: "encode",
		Value:    Encode,
	},
	"decode": &gad.Function{
		FuncName: "decode",
		Value:    Decode,
	},
}
