# Error Handling

[← Back to index](README.md)

## Error Values

An `error` value carries a `name` and a `message`. Create one with the `error`
builtin (its first argument is converted to a string message) and inspect it
with the `.name` and `.message` selectors.

```go
err := error("oops")
println(isError(err))   // true
println(err.name)       // error
println(err.message)    // oops
```

An error value is **falsy**, so it can be tested directly, but note that simply
*holding* an error does not stop execution — only `throw` (or a failing
operation) does.

## throw

`throw` raises any value as an error, unwinding until a `catch` handles it.

```go
func() {
    throw "something went wrong"
}()
```

## try / catch / finally

`try` runs a block; `catch` handles a raised error (optionally binding it);
`finally` always runs. `catch` and `finally` are each optional, but at least one
must be present.

```go
try {
    throw "boom"
} catch err {
    println("caught:", err.message)   // caught: boom
} finally {
    println("always runs")
}
```

A `catch` without a binding ignores the error value:

```go
try {
    risky()
} catch {
    println("failed, continuing")
}
```

## Builtin Errors

Builtin errors have a `name` but no message. Call `.New(message)` to create a
wrapped instance with a message.

```go
e := NotImplementedError.New("todo: parse v2")
println(e.name)      // NotImplementedError
println(e.message)   // todo: parse v2
```

Available builtin errors:

| Identifier                  | Raised when…                              |
|-----------------------------|-------------------------------------------|
| `WrongNumArgumentsError`    | a call has the wrong number of arguments  |
| `InvalidOperatorError`      | an operator is not defined for the types  |
| `IndexOutOfBoundsError`     | an index is outside a sequence            |
| `NotIterableError`          | a value cannot be iterated                |
| `NotIndexableError`         | a value cannot be indexed                 |
| `NotIndexAssignableError`   | an index cannot be assigned               |
| `NotCallableError`          | a non-function value is called            |
| `NotImplementedError`       | a feature is not implemented              |
| `ZeroDivisionError`         | division (or modulo) by zero              |
| `TypeError`                 | an unexpected type is encountered         |

You can `catch` these like any other error, or compare an error's `.name`.

## The `or` Fallback Operator

`expr or fallback` evaluates `expr`; if evaluating it **throws**, the thrown
error is swallowed and `fallback` is used instead. It is a concise alternative
to a `try/catch` for expression-level recovery.

```go
mayThrow := func() { throw "fail" }

z := mayThrow() or 99            // 99
y := 1 + (mayThrow() or 10)      // 11
ok := (2 * 3) or 0               // 6  (no throw → left value)
```

Inside the fallback, the caught error is bound to `$err`, so the fallback can
inspect or reuse it:

```go
v := mayThrow() or ("recovered: " + str($err))   // "recovered: error: fail"
// reuse $err to re-throw selectively
n := compute() or ($err.name == "error" ? -1 : throw $err)
```

`or` triggers only on a *thrown* error, not on a value that merely *is* an
error. If the fallback is itself an error value, it is re-thrown.

## Recovering with `defer_err`

A `defer_err` handler runs when a function exits with an error and can recover
by clearing `$err`, optionally setting the result via `$ret`. See
[Functions → Deferred Handlers](functions.md#deferred-handlers).

```go
safe := func() {
    defer_err {
        $ret = "recovered: " + str($err)
        $err = nil
    }
    throw "boom"
}
println(safe())   // recovered: error: boom
```
