# Functions

[← Back to index](README.md)

Functions are first-class values. A function literal is `func(params) { … }`,
usually bound to a variable:

```go
sum := func(a, b) {
    return a + b
}

var mul = func(a, b) {
    return a * b
}
```

A `func` with a name is a declaration that binds that name (a `const`); there is
also a shorthand `name(params) { … }` / `name(params) => expr`:

```go
func area(r) {
    return 3.14159 * r * r
}
double(x) => x * 2          // shorthand
println(area(2), double(21))
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

## Functions with methods

A function can hold several **methods** — overloads selected at call time by the
argument types. Instead of a single parameter list and body, write a brace block
whose entries are each `(params) <ret> {body}` (the return-type list and a `=>`
expression body are optional, exactly as for a plain function):

```go
func area {
  (r float)          => 3.14159 * r * r        // circle
  (w float, h float) => w * h                   // rectangle
}

area(2.0)        // 12.56636  — the one-parameter method
area(2.0, 3.0)   // 6.0       — the two-parameter method
```

The dispatcher picks the method whose parameters match the call; if none match
it raises `ErrNoMethodFound`. A named declaration (`func area { ... }`) binds the
function to `area`; the same form is also valid as an expression value.

New methods can be added to an existing callable later with the `met` statement:

```go
met area(s str) => "n/a"     // add a string overload
area("x")                    // "n/a"
```

## Properties (prop)

A **property** is a named, callable value defined with the func-with-methods body
syntax but introduced by the `prop` keyword. A method with no parameters is the
*getter*; a method with one parameter is a *setter*, chosen by the argument type:

```go
var value
prop x {
  ()      => value          // getter:  x()
  (v)     { value = v }     // setter:  x(v)
  (v int) { value = "int= " + v }   // typed setter
}

x("a")   // setter runs
x()      // "a"
x(1)     // typed (int) setter
x()      // "int= 1"
```

A single-accessor property may drop the braces: `prop pi() => 3.14`. Properties
are also available through the [`Prop`](values-and-types.md#properties)
constructor for building them programmatically.

## Computed values

`(= expr)` — or `(= stmt; …; result)` for several statements — creates a
**computed value**: a lazy callable that runs its body and yields the result
each time it is called.

```go
v := 10
c := (= v * 2)
typeName(c)   // "ComputedValue"
c()           // 20
v = 100
c()           // 200  — the body is re-evaluated on every call
```

Computed values shine as class field defaults, where each instance gets its own
freshly-evaluated value (see [Classes → Fields](classes.md#fields)):

```go
n := 0
C := Class("C", (cls, define) => define(; fields = (; id = (= n++))))
[C().id, C().id]   // [1, 2]
```

> See also [Func Headers and Method Interfaces](method-interfaces.md) for
> describing and checking function signatures (`<…>`, `meti`, `implements`).

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

### Shortcut form

Besides the block form, a handler may be a single statement written **without
braces** — `defer Stmt`. It accepts a call (with `$ret` / `$err` passed as
arguments), an assignment, or an increment/decrement:

```go
f := func() {
    defer cleanup($ret, $err)   // call, receives the result and error
    defer_ok log($ret)          // call, only on success
    defer $ret += 1             // assignment to the result
    return 1
}
```

The same shortcut applies to the `deferb*` variants (`deferb out += "x"`,
`deferb i++`, `deferb_err report($err)`).

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
