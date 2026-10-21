// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

// gaddoc reads a go package, which must be a gad stdlib module, extracts and
// groups package comments to create the gad module documentation.
//
// usage: ./gaddoc <source dir> <output file>
//
//	go run ./cmd/gaddoc ./stdlib/time ./doc/stdlib-time.md
//	go run ./cmd/gaddoc api <source dir> <output file.gad> <module>
//
// The command lives in cmd/internal/gaddoc, shared with `gad gomodule`.
package main

import "github.com/gad-lang/gad/cmd/internal/gaddoc"

func main() { gaddoc.Main() }
