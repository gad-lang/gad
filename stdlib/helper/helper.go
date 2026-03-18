package helper

import (
	"github.com/gad-lang/gad"
	goflate "github.com/gad-lang/gad/stdlib/compress/flate"
	gadbase64 "github.com/gad-lang/gad/stdlib/encoding/base64"
	gadfpath "github.com/gad-lang/gad/stdlib/filepath"
	gadfmt "github.com/gad-lang/gad/stdlib/fmt"
	gadhttp "github.com/gad-lang/gad/stdlib/http"
	gadjson "github.com/gad-lang/gad/stdlib/json"
	gados "github.com/gad-lang/gad/stdlib/os"
	gadpath "github.com/gad-lang/gad/stdlib/path"
	gadstrings "github.com/gad-lang/gad/stdlib/strings"
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
	mm.AddBuiltinModuleInit("time", gadtime.ModuleInit).
		AddBuiltinModuleInit("strings", gadstrings.ModuleInit).
		AddBuiltinModuleInit("fmt", gadfmt.ModuleInit).
		AddBuiltinModuleInit("json", gadjson.ModuleInit).
		AddBuiltinModule("path", gadpath.Module).
		AddBuiltinModule("encoding/base64", gadbase64.Module).
		AddBuiltinModule("compress/flate", goflate.Module)

	if !b.Safe {
		if !b.Disabled["http"] {
			mm.AddBuiltinModule("http", gadhttp.Module)
		}
		if !b.Disabled["os"] {
			mm.AddBuiltinModuleInit("os", gados.ModuleInit)
		}
		if !b.Disabled["filepath"] {
			mm.AddBuiltinModuleInit("filepath", gadfpath.ModuleInit)
		}
	}
	return mm
}
