package http

import (
	"net/http"

	"github.com/gad-lang/gad"
)

var Module = gad.Dict{
	"url": &gad.Function{
		Name:  "url",
		Value: Url,
	},
	"header": &gad.Function{
		Name:  "header",
		Value: Header,
	},
	"request": &gad.Function{
		Name:  "request",
		Value: Request,
	},
	"get": &gad.Function{
		Name:  "get",
		Value: Get,
	},
	"exec": gad.MustNewReflectValue(http.DefaultClient.Do),
}
