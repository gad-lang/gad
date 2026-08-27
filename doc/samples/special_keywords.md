
# Special `@` Keywords

Gad has a small set of built-in keywords, all written with a leading `@`, that
expose information about the running function, the current module, and the host
globals. Each is an expression that yields a value directly (compiled to a
single opcode — no function call), so they are cheap and usable anywhere an
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

Inside a function, `@fn`, `@args` and `@nargs` describe the active call: `@fn` is
the running function (use it for anonymous recursion), `@args` the positional
arguments as an array, and `@nargs` the call's named-argument set (useful for
inspecting or forwarding named arguments generically).

```gad
/**
@fn — the running function; here used for anonymous recursion.
**/
fact := func(n) {
    return n <= 1 ? 1 : n * @fn(n - 1)
}
println("5! =", fact(5)) // 120

/**
@args — the positional arguments as an array.
**/
sum := func(*_) {
    total := 0
    for v in @args {
        total += v
    }
    return total
}
println("sum =", sum(1, 2, 3, 4)) // 10

/**
@nargs — the named-argument set of the current call (extra names beyond the
declared parameters).
**/
forward := func(; **extra) => @nargs
println("nargs type =", typeName(forward(; a = 1)))
```

## Module introspection

`@name`, `@file`, `@main` and `@module` describe the module the code runs in:
`@name` is the module name, `@file` the source path/URL, `@main` is `true` in the
entry module (guard "run only when executed directly" code with it), and
`@module` is the live module object (its exports, params and metadata).

```gad
/**
Module info: @name, @file, @main, @module.
**/
println("name =", @name, "| main =", @main)
```

## Globals: `@g`

`@g` is the host-provided globals object — the channel an embedding Go program
uses to exchange data with a script. It is a short form for the whole globals
container and can be read, indexed and assigned. It replaces the former
`globals()` builtin and pairs naturally with the absent-coalescing operators and
with `global` declaration defaults (`global (user !?= "guest")` lowers to
`@g["user"] !?= "guest"`). See [Variables and Scopes](02_values_and_types.md)
and [Operators](17_unary_operators.md).

```gad
/**
@g — the host globals object (short form of the old `globals()`).
**/
@g["hits"] = (@g["hits"] !? 0) + 1
@g["hits"] = (@g["hits"] !? 0) + 1
println("hits =", @g["hits"])      // 2
println("has hits:", "hits" in @g) // true
```

## Example — `special_keywords.gad`

```gad
/**
@fn — the running function; here used for anonymous recursion.
**/
fact := func(n) {
    return n <= 1 ? 1 : n * @fn(n - 1)
}
println("5! =", fact(5)) // 120

/**
@args — the positional arguments as an array.
**/
sum := func(*_) {
    total := 0
    for v in @args {
        total += v
    }
    return total
}
println("sum =", sum(1, 2, 3, 4)) // 10

/**
@nargs — the named-argument set of the current call (extra names beyond the
declared parameters).
**/
forward := func(; **extra) => @nargs
println("nargs type =", typeName(forward(; a = 1)))

/**
Module info: @name, @file, @main, @module.
**/
println("name =", @name, "| main =", @main)

/**
@g — the host globals object (short form of the old `globals()`).
**/
@g["hits"] = (@g["hits"] !? 0) + 1
@g["hits"] = (@g["hits"] !? 0) + 1
println("hits =", @g["hits"])      // 2
println("has hits:", "hits" in @g) // true

return @g["hits"]
```
