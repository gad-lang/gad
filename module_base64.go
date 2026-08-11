// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

import "encoding/base64"

// base64ModuleSpec is the module spec shared by the builtin `base64` namespace
// members and the importable encoding/base64 module.
var base64ModuleSpec = NewModuleSpecFromName("base64")

// gad:doc
// # base64 module
//
// Base64 encodings (Go's encoding/base64), available without an import as the
// `base64` namespace. Each encoding is an `Encoding` value exposing
// `EncodeToString(data bytes) <str>` and `DecodeString(s str) <bytes>`.
//
// ## Example
//
// ```gad
// data := bytes("Gad Lang")
// base64.StdEncoding.EncodeToString(data)
// >>> "R2FkIExhbmc="
// base64.RawStdEncoding.EncodeToString(data)
// >>> "R2FkIExhbmc"
// str(base64.StdEncoding.DecodeString("R2FkIExhbmc="))
// >>> "Gad Lang"
// ```
//
// ## Constants
//
// - `StdEncoding`: the standard padded base64 encoding (RFC 4648).
// - `RawStdEncoding`: the standard base64 encoding without `=` padding.
// - `URLEncoding`: the URL- and filename-safe padded base64 encoding.
// - `RawURLEncoding`: the URL-safe base64 encoding without padding.
// - `NewEncoding`: builds a custom `Encoding` from a 64-character alphabet.

// base64Module is the `base64` builtin namespace (Go's encoding/base64),
// available to scripts without an import.
var base64Module = Dict{
	"NewEncoding":    MustNewReflectValue(base64.NewEncoding),
	"URLEncoding":    MustNewReflectValue(base64.URLEncoding),
	"RawURLEncoding": MustNewReflectValue(base64.RawURLEncoding),
	"StdEncoding":    MustNewReflectValue(base64.StdEncoding),
	"RawStdEncoding": MustNewReflectValue(base64.RawStdEncoding),
}

// Base64Module returns the `base64` builtin namespace as StdModuleData with its
// encodings in the read-only Consts bucket. It is also used by the stdlib
// `encoding/base64` importable module.
func Base64Module() StdModuleData { return StdModuleData{Consts: base64Module} }
