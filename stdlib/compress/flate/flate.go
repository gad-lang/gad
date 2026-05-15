package flate

import (
	"github.com/gad-lang/gad"
)

const ModuleName = "compress/flate"

// ModuleInit represents init for module flate.
var ModuleInit gad.ModuleInitFunc = func(module *gad.Module, c gad.Call) (err error) {
	spec := module.Spec
	module.Data = gad.Dict{
		"encode": &gad.Function{
			Module:   spec,
			FuncName: "encode",
			Value:    Encode,
		},
		"decode": &gad.Function{
			Module:   spec,
			FuncName: "decode",
			Value:    Decode,
		},
	}
	return
}
