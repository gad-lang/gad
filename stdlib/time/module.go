// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

// Package time provides the importable `time` module. Its implementation now
// lives in the root gad package as the builtin `time` namespace; this package
// re-exports it so import("time") keeps working.
package time

import "github.com/gad-lang/gad"

const ModuleName = "time"

// Time and Location are the gad time/location object types.
type (
	Time     = gad.Time
	Location = gad.Location
)

// Type objects for time and location values.
var (
	TimeType     = gad.TimeType
	LocationType = gad.TimeLocationType
)

func getModule() gad.Dict { return gad.TimeModule() }

// ModuleInit represents time module.
var ModuleInit = gad.ModuleInitFunc(func(module *gad.Module, c gad.Call) (err error) {
	module.Data = gad.TimeModule()
	return
})
