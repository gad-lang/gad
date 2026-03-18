package http

import (
	"net/http"

	"github.com/gad-lang/gad"
)

var Module = gad.Dict{
	"url": &gad.Function{
		FuncName: "url",
		Value:    URL,
	},
	"header": &gad.Function{
		FuncName: "header",
		Value:    Header,
	},
	"request": &gad.Function{
		FuncName: "request",
		Value:    Request,
	},
	"get": &gad.Function{
		FuncName: "get",
		Value:    Get,
	},
	"exec": gad.MustNewReflectValue(http.DefaultClient.Do),
}
