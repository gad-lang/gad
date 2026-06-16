// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

// Package strings provides the importable `strings` module. Its implementation
// now lives in the root gad package as the builtin `strings` namespace; this
// package re-exports it so import("strings") keeps working.
package strings

import "github.com/gad-lang/gad"

const ModuleName = "strings"

// ModuleInit represents strings module.
var ModuleInit = gad.ModuleInitFunc(func(module *gad.Module, c gad.Call) (err error) {
	module.Data = gad.StringsModule()
	return
})
