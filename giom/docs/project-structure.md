# Project Structure

Giom lives in the `giom/` directory of the Gad repository as the `giom`
sub-package.

```text
giom/
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

The package path is `github.com/gad-lang/gad/giom`.

This package holds the Giom **runtime** — the render tree and the `giom`
builtins. Giom **compilation** is part of Gad itself
(`gad.CompileOptions.GiomOptions`), not this package.

Important exported names:

- `AppendBuiltins` — register the `giom` builtins.
- `Render` / `NewRender` — the cached template engine.
- `Element`, `Tag`, `Text` — the render tree types (defined in `element.go`)
  that a compiled template builds and returns; see [API Reference](api.md).

## `node/`

AST node definitions and conversion helpers. The converter turns Giom-specific
nodes into Gad AST nodes when possible.

## `parser/`

Indentation-aware parser and scanner. It parses Giom template source into Giom
AST nodes.

## `token/`

Token definitions for Giom syntax.

## `examples/cms/`

A nested Go module containing a full web application example. It depends on the
`giom` package through:

```go
replace github.com/gad-lang/gad/giom => ../..
```

## Samples

Runnable `.giom` examples live under the repository's top-level
[`samples/giom/`](../../samples/giom) directory.
