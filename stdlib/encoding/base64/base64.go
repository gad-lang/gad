package base64

import (
	"github.com/gad-lang/gad"
)

const ModuleName = "encoding/base64"

var (
	// ModuleData is the base64 module data. The implementation now lives in the
	// root gad package as the builtin `base64` namespace; this re-exports it so
	// `import("encoding/base64")` keeps working.
	ModuleData = gad.Base64Module()

	// ModuleInit represents init for module base64.
	ModuleInit gad.ModuleInitFunc = func(module *gad.Module, c gad.Call) (err error) {
		module.Data = ModuleData
		return
	}
)
