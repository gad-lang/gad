// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

// This file wires the documented public-API stub generation into
// `go generate ./...`. Each directive runs `gaddoc api` from the repo root
// (the package directory of this file), mirroring the Makefile `generate-api`
// target, and rewrites the corresponding `samples/**/<module>.gad` stub from the
// module's `gad:doc`. Regenerate with `go generate ./...` or `make generate-api`.

//go:generate go run ./cmd/gaddoc api . samples/stdlib/fmt.gad fmt
//go:generate go run ./cmd/gaddoc api . samples/stdlib/strings.gad strings
//go:generate go run ./cmd/gaddoc api . samples/stdlib/time.gad time
//go:generate go run ./cmd/gaddoc api . samples/stdlib/base64.gad base64
//go:generate go run ./cmd/gaddoc api . samples/stdlib/gad.gad gad
//go:generate go run ./cmd/gaddoc api ./stdlib/json samples/stdlib/json.gad json
//go:generate go run ./cmd/gaddoc api . samples/builtins.gad builtins
//go:generate go run ./cmd/gaddoc api . samples/types.gad types
