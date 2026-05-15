package http

import (
	"net/http"

	"github.com/gad-lang/gad"
)

const ModuleName = "http"

var (
	// ModuleInit represents init for module http.
	ModuleInit gad.ModuleInitFunc = func(module *gad.Module, c gad.Call) (err error) {
		spec := module.Spec
		module.Data = gad.Dict{
			"url": &gad.Function{
				Module:   spec,
				FuncName: "url",
				Value:    URL,
			},
			"header": &gad.Function{
				Module:   spec,
				FuncName: "header",
				Value:    Header,
			},
			"request": &gad.Function{
				Module:   spec,
				FuncName: "request",
				Value:    Request,
			},
			"get": &gad.Function{
				Module:   spec,
				FuncName: "get",
				Value:    Get,
			},
			"exec": gad.MustNewReflectValue(http.DefaultClient.Do),
		}
		return
	}
)
