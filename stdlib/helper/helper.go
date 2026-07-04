package helper

import (
	"github.com/gad-lang/gad"
	gadflate "github.com/gad-lang/gad/stdlib/compress/flate"
	gadbase64 "github.com/gad-lang/gad/stdlib/encoding/base64"
	gadfpath "github.com/gad-lang/gad/stdlib/filepath"
	gadfmt "github.com/gad-lang/gad/stdlib/fmt"
	gadhttp "github.com/gad-lang/gad/stdlib/http"
	gadjson "github.com/gad-lang/gad/stdlib/json"
	gados "github.com/gad-lang/gad/stdlib/os"
	gadpath "github.com/gad-lang/gad/stdlib/path"
	gadstrings "github.com/gad-lang/gad/stdlib/strings"
	gadtest "github.com/gad-lang/gad/stdlib/test"
	gadtime "github.com/gad-lang/gad/stdlib/time"
)

type ModuleMapBuilder struct {
	Safe     bool
	Disabled map[string]bool
}

func NewModuleMapBuilder() *ModuleMapBuilder {
	return &ModuleMapBuilder{}
}

func NewModuleMap() *gad.ModuleMap {
	return NewModuleMapBuilder().Build()
}

func (b *ModuleMapBuilder) Build() *gad.ModuleMap {
	return b.BuildTo(gad.NewModuleMap())
}

func (b *ModuleMapBuilder) BuildTo(mm *gad.ModuleMap) *gad.ModuleMap {
	mm.AddBuiltinModuleInit(gadtime.ModuleName, gadtime.ModuleInit).
		AddBuiltinModuleInit(gadstrings.ModuleName, gadstrings.ModuleInit).
		AddBuiltinModuleInit(gadfmt.ModuleName, gadfmt.ModuleInit).
		AddBuiltinModuleInit(gadjson.ModuleName, gadjson.ModuleInit).
		AddBuiltinModule(gadpath.ModuleName, gadpath.Module).
		AddBuiltinModuleInit(gadbase64.ModuleName, gadbase64.ModuleInit).
		AddBuiltinModuleInit(gadflate.ModuleName, gadflate.ModuleInit).
		AddBuiltinModuleInit(gadtest.ModuleName, gadtest.ModuleInit)

	if !b.Safe {
		if !b.Disabled[gadhttp.ModuleName] {
			mm.AddBuiltinModuleInit(gadhttp.ModuleName, gadhttp.ModuleInit)
		}
		if !b.Disabled[gados.ModuleName] {
			mm.AddBuiltinModuleInit(gados.ModuleName, gados.ModuleInit)
		}
		if !b.Disabled[gadfpath.ModuleName] {
			mm.AddBuiltinModuleInit(gadfpath.ModuleName, gadfpath.ModuleInit)
		}
	}
	return mm
}
