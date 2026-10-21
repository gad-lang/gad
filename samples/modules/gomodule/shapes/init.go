package shapes

// The Go init of the module (written by hand: `gad gomodule` never overwrites
// it). It runs after the gad part (shapes.gad), in the VM that imports the
// module, and adds GoShape — a class in Go extending the gad class Shape —,
// whose method label uses the module strings, imported through that VM.

//gad:import strings

import "github.com/gad-lang/gad"

// gad:doc
// # shapes module
//
// ## Declarations
//
// ```gad
// /**
// A Shape whose label is its name in upper case (a class in Go: the module's
// Go init).
// **/
// export class GoShape {
//     *Shape
//     methods {
//         /** the name, in upper case **/
//         label() <str> => nil
//     }
// }
// ```

func init() {
	Init = func(ctx *gad.ModuleInitContext) (gad.Dict, error) {
		base, err := ctx.Get("Shape") // the class of the gad part
		if err != nil {
			return nil, err
		}
		strs, err := ctx.Import("strings") // declared: //gad:import strings
		if err != nil {
			return nil, err
		}
		cls := gad.NewClass("GoShape", ctx.Spec())
		cls.Extends(base.(*gad.Class), "")
		err = cls.AddGoMethod("label", func(c gad.Call) (gad.Object, error) {
			name, err := c.Args.Get(0).(gad.IndexGetter).IndexGet(c.VM, gad.Str("name"))
			if err != nil {
				return nil, err
			}
			return strs.Call(c.VM, "toUpper", name)
		})
		if err != nil {
			return nil, err
		}
		return gad.Dict{"GoShape": cls}, nil
	}
}
