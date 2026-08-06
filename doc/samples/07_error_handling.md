
# Error Handling

Gad reports failures with **error values** and the `throw` / `try` / `catch` /
`finally` control flow, plus the concise `or` fallback operator.

## Error values

An `error` value carries a `name` and a `message`. Create one with the `error`
builtin (its first argument is converted to a string message) and inspect it
with `.name` and `.message`. An error value is **falsy**, so it can be tested
directly — but merely *holding* an error does not stop execution; only `throw`
(or a failing operation) does.

```gad
err := error("oops")
[isError(err), err.name, err.message]
```

```gad
[true, "error", "oops"]
```

## throw

`throw` raises any value as an error, unwinding until a `catch` handles it.

## try / catch / finally

`try` runs a block; `catch` handles a raised error (optionally binding it);
`finally` always runs. `catch` and `finally` are each optional, but at least one
must be present. A `catch` without a binding ignores the error value.

```gad
safeDiv := func(a, b) {
    try {
        if b == 0 {
            throw "division by zero"
        }
        return a / b
    } catch e {
        return "caught: " + str(e)
    } finally {
        // always runs (cleanup, logging, …)
    }
}
[safeDiv(10, 2), safeDiv(10, 0)]
```

```gad
[5, "caught: error: division by zero"]
```

## Builtin errors

Builtin errors have a `name` but no message. Call `.New(message)` to create a
wrapped instance with a message. `catch` them like any other error, or compare
an error's `.name`.

```gad
e := NotImplementedError.New("todo: parse v2")
[e.name, e.message]
```

```gad
["NotImplementedError", "todo: parse v2"]
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

## The `or` fallback operator

`expr or fallback` evaluates `expr`; if evaluating it **throws**, the thrown
error is swallowed and `fallback` is used instead — a concise alternative to a
`try/catch` for expression-level recovery. Inside the fallback the caught error
is bound to `$err`, so it can be inspected or re-thrown. `or` triggers only on a
*thrown* error, not on a value that merely *is* an error.

```gad
mayThrow := func() { throw "fail" }
z := mayThrow() or 99             // swallow the throw, use the fallback
ok := (2 * 3) or 0                // no throw -> the left value
recovered := mayThrow() or ("recovered: " + str($err)) // $err is the caught error
[z, ok, recovered]
```

```gad
[99, 6, "recovered: error: fail"]
```

## Recovering with `defer_err`

A `defer_err` handler runs when a function exits with an error and can recover by
clearing `$err`, optionally setting the result via `$ret` (see
[Functions](03_functions.md) → deferred handlers).

```gad
safe := func() {
    defer_err {
        $ret = "recovered: " + str($err)
        $err = nil
    }
    throw "boom"
}
safe()
```

```gad
recovered: error: boom
```

## Example — `07_error_handling.gad`

```gad
err := error("oops")
[isError(err), err.name, err.message]

safeDiv := func(a, b) {
    try {
        if b == 0 {
            throw "division by zero"
        }
        return a / b
    } catch e {
        return "caught: " + str(e)
    } finally {
        // always runs (cleanup, logging, …)
    }
}
[safeDiv(10, 2), safeDiv(10, 0)]

e := NotImplementedError.New("todo: parse v2")
[e.name, e.message]

mayThrow := func() { throw "fail" }
z := mayThrow() or 99             // swallow the throw, use the fallback
ok := (2 * 3) or 0                // no throw -> the left value
recovered := mayThrow() or ("recovered: " + str($err)) // $err is the caught error
[z, ok, recovered]

safe := func() {
    defer_err {
        $ret = "recovered: " + str($err)
        $err = nil
    }
    throw "boom"
}
safe()

return safeDiv(20, 4)
```
