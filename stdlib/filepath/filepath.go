package filepath

import (
	"path/filepath"

	"github.com/gad-lang/gad"
)

var (
	TWalkIterator = &gad.Type{TypeName: "WalkIterator", Parent: gad.TIterator}
	Module        = gad.Dict{
		"ext":          gad.MustNewReflectValue(filepath.Ext),
		"clean":        gad.MustNewReflectValue(filepath.Clean),
		"join":         gad.MustNewReflectValue(filepath.Join),
		"base":         gad.MustNewReflectValue(filepath.Base),
		"dir":          gad.MustNewReflectValue(filepath.Dir),
		"isAbs":        gad.MustNewReflectValue(filepath.IsAbs),
		"isLocal":      gad.MustNewReflectValue(filepath.IsLocal),
		"rel":          gad.MustNewReflectValue(filepath.Rel),
		"volumeName":   gad.MustNewReflectValue(filepath.VolumeName),
		"split":        gad.MustNewReflectValue(filepath.Split),
		"match":        gad.MustNewReflectValue(filepath.Match),
		"fromSlash":    gad.MustNewReflectValue(filepath.FromSlash),
		"evalSymlinks": gad.MustNewReflectValue(filepath.EvalSymlinks),
		"walk": &gad.BuiltinFunction{
			Name:  "walk",
			Value: Walk,
		},
		TWalkSkip.TypeName: TWalkSkip,
		"glob": &gad.BuiltinFunction{
			Name: "glob",
			Value: func(c gad.Call) (_ gad.Object, err error) {
				if err = c.Args.CheckLen(1); err != nil {
					return
				}

				var (
					matched []string
					arr     gad.Array
				)

				if matched, err = filepath.Glob(c.Args.GetOnly(0).ToString()); err != nil {
					return
				}

				arr = make(gad.Array, len(matched))

				for i, v := range matched {
					arr[i] = gad.Str(v)
				}

				return arr, nil
			},
		},
		"splitList": &gad.BuiltinFunction{
			Name: "splitList",
			Value: func(c gad.Call) (_ gad.Object, err error) {
				if err = c.Args.CheckLen(1); err != nil {
					return
				}

				var (
					s   = filepath.SplitList(c.Args.GetOnly(0).ToString())
					arr = make(gad.Array, len(s))
				)

				for i, v := range s {
					arr[i] = gad.Str(v)
				}

				return arr, nil
			},
		},
	}
)
