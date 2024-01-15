package helper

import (
	"github.com/gad-lang/gad"
	gadfmt "github.com/gad-lang/gad/stdlib/fmt"
	gadjson "github.com/gad-lang/gad/stdlib/json"
	gadstrings "github.com/gad-lang/gad/stdlib/strings"
	gadtime "github.com/gad-lang/gad/stdlib/time"
)

func NewModuleMap() *gad.ModuleMap {
	return AddMudules(gad.NewModuleMap())
}

func AddMudules(mm *gad.ModuleMap) *gad.ModuleMap {
	return mm.AddBuiltinModule("time", gadtime.Module).
		AddBuiltinModule("strings", gadstrings.Module).
		AddBuiltinModule("fmt", gadfmt.Module).
		AddBuiltinModule("json", gadjson.Module)
}
