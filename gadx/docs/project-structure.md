# Project Structure

Gadx lives in the `gadx/` directory of the Gad repository as the `gadx`
sub-package.

```text
gadx/
├── builtins.go
├── element.go
├── render.go
├── go.mod
├── node/
├── parser/
├── token/
├── examples/
│   └── cms/
├── docs/
├── LICENSE
└── CLAUDE.md
```

## Root Package

The package path is `github.com/gad-lang/gad/gadx`.

This package holds the Gadx **runtime** — the render tree and the `gadx`
builtins. Gadx **compilation** is part of Gad itself (selected by a `.gadx`
`gad.CompileOptions.ModuleFile`), not this package.

Important exported names:

- `AppendBuiltins` — register the `gadx` builtins.
- `Render` / `NewRender` — the cached template engine.
- `Element`, `Tag`, `Text` — the render tree types (defined in `element.go`)
  that a compiled template builds and returns; see [API Reference](api.md).

## `node/`

AST node definitions and conversion helpers. The converter turns Gadx-specific
nodes into Gad AST nodes when possible.

## `parser/`

Indentation-aware parser and scanner. It parses Gadx template source into Gadx
AST nodes.

## `token/`

Token definitions for Gadx syntax.

## `examples/cms/`

A nested Go module containing a full web application example. It depends on the
`gadx` package through:

```go
replace github.com/gad-lang/gad/gadx => ../..
```

## Samples

Runnable `.gadx` examples live under the repository's top-level
[`samples/gadx/`](../../samples/gadx) directory.
