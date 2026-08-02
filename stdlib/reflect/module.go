// Package reflect provides the importable `reflect` module. Its implementation
// lives in the root gad package as the builtin `reflect` namespace; this package
// re-exports it so import("reflect") works alongside the direct `reflect.get` /
// `reflect.set` builtin namespace.
package reflect

import "github.com/gad-lang/gad"

const ModuleName = "reflect"

// ModuleInit initializes the reflect module.
var ModuleInit = gad.ModuleInitFunc(func(module *gad.Module, c gad.Call) (err error) {
	module.Data = gad.ReflectModule()
	return
})
