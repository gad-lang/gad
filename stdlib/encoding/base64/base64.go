package base64

import (
	"encoding/base64"

	"github.com/gad-lang/gad"
)

var Module = gad.Dict{
	"NewEncoding":    gad.MustNewReflectValue(base64.NewEncoding),
	"URLEncoding":    gad.MustNewReflectValue(base64.URLEncoding),
	"RawURLEncoding": gad.MustNewReflectValue(base64.RawURLEncoding),
	"StdEncoding":    gad.MustNewReflectValue(base64.StdEncoding),
	"RawStdEncoding": gad.MustNewReflectValue(base64.RawStdEncoding),
}
