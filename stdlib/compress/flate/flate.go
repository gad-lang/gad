package flate

import (
	"github.com/gad-lang/gad"
)

var Module = gad.Dict{
	"encode": &gad.Function{
		Name:  "encode",
		Value: Encode,
	},
	"decode": &gad.Function{
		Name:  "decode",
		Value: Decode,
	},
}
