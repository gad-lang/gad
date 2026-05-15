package path

import (
	"path"

	"github.com/gad-lang/gad"
)

const ModuleName = "path"

var (
	Module = gad.Dict{
		"ext":   gad.MustNewReflectValue(path.Ext),
		"clean": gad.MustNewReflectValue(path.Clean),
		"join":  gad.MustNewReflectValue(path.Join),
		"base":  gad.MustNewReflectValue(path.Base),
		"dir":   gad.MustNewReflectValue(path.Dir),
		"isAbs": gad.MustNewReflectValue(path.IsAbs),
	}

	// ModuleInit represents init for module path.
	ModuleInit gad.ModuleInitFunc = func(module *gad.Module, c gad.Call) (err error) {
		module.Data = Module
		return
	}
)
