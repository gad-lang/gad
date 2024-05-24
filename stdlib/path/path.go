package path

import (
	"path"

	"github.com/gad-lang/gad"
)

var Module = gad.Dict{
	"ext":   gad.MustNewReflectValue(path.Ext),
	"clean": gad.MustNewReflectValue(path.Clean),
	"join":  gad.MustNewReflectValue(path.Join),
	"base":  gad.MustNewReflectValue(path.Base),
	"dir":   gad.MustNewReflectValue(path.Dir),
	"isAbs": gad.MustNewReflectValue(path.IsAbs),
}
