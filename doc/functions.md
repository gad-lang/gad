# Functions

[← Back to index](README.md)

Functions are first-class values. Gad has no named function declarations — every
function is an anonymous value bound to a variable.

```go
sum := func(a, b) {
    return a + b
}

var mul = func(a, b) {
    return a * b
}
```

## Arrow Closures

`(params) => expr` is a shorthand closure whose body is a single expression.

```go
double := (x) => x * 2
add := (a, b) => a + b
println(double(21), add(2, 3))   // 42 5
```

## Closures

Inner functions capture variables from their enclosing scope.

```go
adder := func(base) {
    return (x) => base + x   // captures 'base'
}
add5 := adder(5)
println(add5(4))   // 9
```

## Variadic Parameters

The last positional parameter may be variadic (`*name`); it collects the
remaining positional arguments into an array.

```go
variadic := func(a, b, *c) {
    return [a, b, c]
}
variadic(1, 2, 3, 4)   // [1, 2, [3, 4]]
variadic(1, 2)         // [1, 2, []]
```

Only the **last** positional parameter may be variadic.

## Spreading Arguments

Use `*` to spread an array as positional arguments at the call site:

```go
f := func(a, b, c) { return a + b + c }
f(*[1, 2, 3])   // 6
f(1, *[2, 3])   // 6
```

## Named Arguments

Parameters after a `;` are **named**. They may have defaults, and a trailing
`**name` collects any extra named arguments into a `namedArgs` value. Callers
pass named arguments after a `;` as `name=value`, and may spread a dict with
`**`.

```go
greet := func(name; greeting="Hello", **rest) {
    return greeting + ", " + name
}

greet("Gad")                       // "Hello, Gad"
greet("Gad"; greeting="Hi")        // "Hi, Gad"
greet("Gad"; **{greeting: "Hey"})  // "Hey, Gad"
```

A function can declare both positional and named parameters, in that order:

```go
func(a, b, *pos; x, y=1, **named) { /* ... */ }
```

## Argument Count

A call must supply the right number of positional arguments (variadics aside),
or it raises `WrongNumArgumentsError`:

```go
f := func(a, b) {}
f(1, 2, 3)   // RuntimeError: WrongNumArgumentsError
```

## return = (assign the result)

`return = expr` sets the function's result value without leaving the function;
execution continues. The final result is whatever `return =` last set (any later
bare expression value is ignored). This pairs naturally with
[deferred handlers](#deferred-handlers), which can read and rewrite it via
`$ret`.

```go
f := func(x) {
    return = x   // bind the result slot to x
    x++          // keep running; mutating x updates the result
}
println(f(10))   // 11
```

> Note: `return = x` captures a *reference* to the result slot; mutating `x`
> afterwards updates the result, which is why `f(10)` above yields `11`.

## Deferred Handlers

`defer` registers a handler that runs when the enclosing **function** returns,
in last-in-first-out order. Inside a handler, `$ret` is the (mutable) return
value and `$err` is the error being propagated, if any.

```go
f := func() {
    defer { println("cleanup") }
    println("body")
    return 1
}
f()   // prints: body, then cleanup
```

Three conditional variants run only on the matching exit:

* `defer_ok { … }` — runs only on a normal return (no error).
* `defer_err { … }` — runs only when an error is propagating.
* `defer` — always runs.

A `defer_err` handler can **recover** by clearing `$err` (and optionally setting
`$ret`):

```go
safe := func() {
    defer_err {
        $ret = "recovered: " + str($err)
        $err = nil          // swallow the error
    }
    throw "boom"
}
println(safe())   // recovered: error: boom
```

### Block-scoped: deferb

`deferb` (and `deferb_ok` / `deferb_err`) run when the **enclosing block**
exits, rather than the whole function — useful for scoped cleanup. Block defers
have no `$ret` (a block has no return value).

```go
out := ""
{
    deferb { out += "d1 " }
    deferb { out += "d2 " }
    out += "body "
}
out += "after"
println(out)   // body d2 d1 after   (LIFO at block exit)
```
