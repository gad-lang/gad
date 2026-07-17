# Gad Language Documentation

Gad is a fast, dynamic scripting language designed to be embedded into Go
applications. Source code is compiled to bytecode and run on a stack-based
virtual machine written in native Go.

This documentation is a hands-on, example-driven reference. Every example is
written as runnable Gad code; most can be pasted directly into the REPL or a
`.gad` file.

## Table of Contents

1. [Getting Started](getting-started.md) — install, run scripts, the REPL.
2. [Values and Types](values-and-types.md) — every value type and its literals
   (int, uint, float, decimal, bool, flag, char, str, rawStr, bytes, array,
   dict, nil, function, …).
3. [Variables and Scopes](variables-and-scopes.md) — `param`, `global`, `var`,
   `const`, `iota`, `:=` vs `=`, scoping rules.
4. [Operators](operators.md) — unary, binary, ternary, assignment, nullish
   (`??`, `?.`), precedence, selectors, indexers and slicing.
5. [Control Flow](control-flow.md) — `if`, `for`, `for in`, `try/catch/finally`,
   and the `match` expression/statement.
6. [Functions](functions.md) — closures, variadics, named arguments, spreading,
   `return =`, and `defer` / `deferb` handlers.
7. [Collections](collections.md) — arrays, dicts, comprehensions, spread/merge
   literals, and destructuring.
8. [Classes and Objects](classes.md) — `Class(...)`, fields, methods,
   properties, constructors, inheritance and `met` extensions.
   - [Func Headers and Method Interfaces](method-interfaces.md) — `<…>` header
     values, `meti` interfaces and `implements`.
   - [Enums](enums.md) — the `enum` keyword: ordered named integer constants,
     signs, bit flags and computed values.
9. [Strings, Bytes and Regex](strings-bytes-regex.md) — string forms, raw
   strings, heredocs, interpolated strings, **bytes literals** (`b"…"`, `h"…"`) and
   `/regex/` literals.
10. [Error Handling](error-handling.md) — error values, builtin errors,
    `try/catch/finally`, and the `or` fallback operator.
11. [Modules](modules.md) — `import`, `exports`, module parameters.
12. [Builtin Functions](builtins.md) — overview of the builtin library.
13. [Embedding in Go](embedding.md) — compile and run Gad from Go, pass
    globals and arguments, expose Go functions.
14. [Formatting](formatting.md) — the `gad fmt` source formatter, its flags and
    the `.gad.yaml` config file.
15. [Templates](templates.md) — mixed/template mode (`{% … %}`, `{%= … %}`,
    `begin … end`, whitespace trim markers, `.gadt` files).
16. [Doc Comments](doc-comments.md) — `///`, `/**` and `/***` doc comments,
    what they attach to, and how the formatter reflows them.
17. [Conventions](conventions.md) — how primitive types, constants, modules and
    methods are cased, plus the code layout the formatter produces.

## Reference

- [Special `@` Keywords](special-keywords.md) — `@fn`, `@args`, `@nargs`,
  `@name`, `@file`, `@main`, `@module`, `@g`.
- [Tutorial](tutorial.md) — a guided walk-through of the language.
- Generated standard-library references: [`time`](stdlib-time.md),
  [`fmt`](stdlib-fmt.md), [`strings`](stdlib-strings.md),
  [`json`](stdlib-json.md).
- [Testing](stdlib-test.md) — the `test` module and the `gad test` command:
  `*_test.gad` files, `test*`/`bench*` functions, assertions and benchmarks.

## Giom Templates

Giom is an indentation-based HTML template language that embeds Gad, shipped in
this repository as the [`giom`](../giom) submodule. `.giom` files can be run and
debugged with the `gad` CLI and the `gad ide`/VS Code tooling. Its full
documentation lives in [`giom/docs`](../giom/docs):

- [Getting Started](../giom/docs/getting-started.md) — install and render a first template.
- [Syntax](../giom/docs/syntax.md) — tags, `.class`/`#id`, `[attr]` groups,
  `@`-control keywords, `{= … }` interpolation and `~~` code blocks.
- [Components and Slots](../giom/docs/components-and-slots.md) — `@comp`, `@slot`
  and component calls.
- [Embedding in Go](../giom/docs/embedding.md) and the [API](../giom/docs/api.md).

## A Taste of Gad

```go
param *args

// closures, named args and the `or` fallback operator
greet := func(name; greeting="Hello") {
    return greeting + ", " + name + "!"
}

// comprehensions and spread literals
nums := [1, 2, 3, 4]
doubled := [n * 2 for n in nums if n > 1]   // [4, 6, 8]
all := [0, *doubled, 99]                    // [0, 4, 6, 8, 99]

// match expression
kind := match (len(args)) {
    0: "empty"
    1: "single"
    else: "many"
}

println(greet("Gad"))            // Hello, Gad!
println(greet("Gad"; greeting="Hi"))
println(doubled, all, kind)
```
