// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import "encoding/base64"

// base64ModuleSpec is the module spec shared by the builtin `base64` namespace
// members and the importable encoding/base64 module.
var base64ModuleSpec = NewModuleSpecFromName("base64")

// base64Module is the `base64` builtin namespace (Go's encoding/base64),
// available to scripts without an import.
var base64Module = Dict{
	"NewEncoding":    MustNewReflectValue(base64.NewEncoding),
	"URLEncoding":    MustNewReflectValue(base64.URLEncoding),
	"RawURLEncoding": MustNewReflectValue(base64.RawURLEncoding),
	"StdEncoding":    MustNewReflectValue(base64.StdEncoding),
	"RawStdEncoding": MustNewReflectValue(base64.RawStdEncoding),
}

// Base64Module returns the `base64` builtin namespace. It is shared (not copied)
// and is also used by the stdlib `encoding/base64` importable module.
func Base64Module() Dict { return base64Module }
