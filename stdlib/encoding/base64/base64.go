package base64

import (
	"encoding/base64"

	"github.com/gad-lang/gad"
)

const ModuleName = "encoding/base64"

var (
	ModuleData = gad.Dict{
		"NewEncoding":    gad.MustNewReflectValue(base64.NewEncoding),
		"URLEncoding":    gad.MustNewReflectValue(base64.URLEncoding),
		"RawURLEncoding": gad.MustNewReflectValue(base64.RawURLEncoding),
		"StdEncoding":    gad.MustNewReflectValue(base64.StdEncoding),
		"RawStdEncoding": gad.MustNewReflectValue(base64.RawStdEncoding),
	}

	// ModuleInit represents init for module base64.
	ModuleInit gad.ModuleInitFunc = func(module *gad.Module, c gad.Call) (err error) {
		module.Data = ModuleData
		return
	}
)
