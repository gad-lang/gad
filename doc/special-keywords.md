# Special `@` Keywords

[← Back to index](README.md)

Gad has a small set of built-in keywords, all written with a leading `@`, that
expose information about the running function, the current module, and the host
globals. Each is an expression that yields a value directly (compiled to a
single opcode — no function call), so they are cheap and can be used anywhere an
expression is allowed.

| Keyword    | Yields | Scope |
|------------|--------|-------|
| `@fn`      | the currently executing function | inside a function |
| `@args`    | the call's positional arguments (an array) | inside a function |
| `@nargs`   | the call's named arguments | inside a function |
| `@name`    | the current module's name (string) | anywhere |
| `@file`    | the current module's file path / URL (string) | anywhere |
| `@main`    | `true` when the current module is the entry module | anywhere |
| `@module`  | the current module object | anywhere |
| `@g`       | the host-provided globals object | anywhere |

## Function introspection

Inside a function, `@fn`, `@args` and `@nargs` describe the active call.

```go
// @fn is the running function — use it for anonymous recursion.
fact := func(n) {
    return n <= 1 ? 1 : n * @fn(n - 1)
}
println(fact(5))          // 120

// @args are the positional arguments as an array.
sum := func(*_) {
    total := 0
    for v in @args { total += v }
    return total
}
println(sum(1, 2, 3))     // 6  (@args == [1, 2, 3], len(@args) == 3)
```

`@nargs` is the call's named-argument set, useful for inspecting or forwarding
named arguments generically.

## Module introspection

`@name`, `@file`, `@main` and `@module` describe the module the code runs in.

```go
if @main {                 // only when run as the entry module
    println("running", @name, "from", @file)
}

m := @module               // the module object itself
```

- `@name` — the module name (the main module has a conventional name).
- `@file` — the source path or URL the module was loaded from.
- `@main` — `true` in the entry module, `false` in imported modules; use it to
  guard "run this only when executed directly" code.
- `@module` — the live module object (its exports, params and metadata).

## Globals: `@g`

`@g` is the host-provided globals object — the channel an embedding Go program
uses to exchange data with a script. It is a short form for the whole globals
container and can be read, indexed and assigned:

```go
@g["count"] = (@g["count"] !? 0) + 1   // read/write host state
println("user" in @g)                  // membership test
for k, v in @g { println(k, v) }       // iterate
```

`@g` replaces the former `globals()` builtin. It pairs naturally with the
absent-coalescing operators and with `global` declaration defaults (`global
(user !?= "guest")` lowers to `@g["user"] !?= "guest"`). See
[Variables and Scopes](variables-and-scopes.md) and [Operators](operators.md).
