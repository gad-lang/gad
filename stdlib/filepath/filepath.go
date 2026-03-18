package filepath

import (
	"path/filepath"

	"github.com/gad-lang/gad"
)

var (
	TWalkIterator = gad.NewType("WalkIterator", gad.TIterator)

	ModuleInit = gad.ModuleInitFunc(func(module *gad.Module, c gad.Call) (data gad.ModuleData, err error) {
		return gad.Dict{
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
				FuncName: "walk",
				Value:    Walk,
			},
			TWalkSkip.Name(): TWalkSkip,
			"glob": &gad.BuiltinFunction{
				FuncName: "glob",
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
				FuncName: "splitList",
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
		}, nil
	})
)
