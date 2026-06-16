// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

// Package fmt provides the importable `fmt` module. Its implementation now lives
// in the root gad package as the builtin `fmt` namespace; this package
// re-exports it so import("fmt") keeps working.
package fmt

import "github.com/gad-lang/gad"

const ModuleName = "fmt"

// ScanArg is the interface implemented by the scan-argument objects returned by
// fmt.ScanArg.
type ScanArg = gad.FmtScanArg

// ModuleInit represents fmt module.
var ModuleInit = gad.ModuleInitFunc(func(module *gad.Module, c gad.Call) (err error) {
	module.Data = gad.FmtModule()
	return
})
