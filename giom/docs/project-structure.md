# Project Structure

The current repository root is the Giom implementation.

```text
.
├── builtins.go
├── element.go
├── compiler.go
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

The root package is `github.com/gad-lang/giom`.

Important exported functions:

- `Compile`
- `CompileFile`
- `AppendBuiltins`

`element.go` defines the render tree types (`Element`, `Tag`, `Text`) that a
compiled template builds and returns; see [API Reference](api.md) for details.

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
root module through:

```go
replace github.com/gad-lang/giom => ../..
```

## Removed Legacy Areas

The old root implementation, old command examples, old samples, and test data
were removed as part of the root migration.
