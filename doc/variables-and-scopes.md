# Variables and Scopes

[← Back to index](README.md)

Gad has four declaration keywords — `param`, `global`, `var` and `const` — plus
the short declaration operator `:=`.

Valid identifiers may contain letters, digits, `_` and `$` (not as the first
character if it would form a number):

```go
var (_, _a, $_a, a, A, $b, $, a1, $1, $b1, $$)
```

## `:=` vs `=`

* `:=` declares a **new** local variable and assigns to it.
* `=` assigns to an **existing** variable (or a dict/array element).

```go
a := "foo"   // declare 'a'
a = "bar"    // reassign 'a'
b = 1        // illegal: 'b' is not declared
```

A variable may be reassigned a value of a different type:

```go
a := 123       // int
a = "123"      // str
a = [1, 2, 3]  // array
```

## param

`param` declares the parameters of the main script function. It may appear only
once, at the top level, and initial values are illegal (a variadic `*x`
defaults to `[]`, everything else to `nil`). Use parentheses for multiple
parameters; the positional list and the named list are separated by `;`.

```go
param (arg0, arg1, *rest)          // positional + variadic
param (;x, y=1, z={}, **named)     // named only (with defaults)
param (a, b, *rest; x, y=1, **nx)  // mixed
```

Named arguments and defaults follow the same rules as
[function parameters](functions.md#named-arguments).

## global

`global` declares variables backed by the host-provided globals object. Reads
and writes go through that object, so globals are how an embedding Go program
exchanges data with a script (and how source modules share state). The statement
may appear multiple times.

```go
global foo
global (bar, baz)

foo = 10           // writes back to the globals object
g := @g            // the whole globals object
println(g["foo"])  // 10
```

If the host passes `nil` for globals, a temporary object is used.

### Defaults

A grouped `global (…)` may give a name a default that is applied only when the
host did not already provide a value. There are two flavours, matching the
coalescing operators:

- `name = value` applies the default when the global is **nil or absent**
  (like `name ??= value`).
- `name !?= value` applies it only when the global is **absent**; a value the
  host set to `nil` is kept (like `@g["name"] !?= value`).

```go
global (page = 1, limit = 20)   // page/limit default unless the host set them
global (user !?= "guest")       // only when "user" is not provided at all

// mix plain names and both default forms
global (db, verbose = no, trace !?= no)
```

## var

`var` declares one or more local variables, optionally with initializers.
Uninitialized variables are `nil`. Tuple assignment is not allowed in a `var`
statement (use [destructuring](collections.md#destructuring) with `:=`).

```go
var foo                  // nil
var (bar, baz = 1)       // bar == nil, baz == 1
var (
    x = 1
    y
    z = "z"
)
```

A function value that refers to itself must be declared before it is assigned,
because the right-hand side is compiled before the left:

```go
var f
f = func() {
    return f   // ok: 'f' already declared
}
```

## const

`const` declares read-only local bindings; an initializer is required and
reassignment is a compile error. The binding is read-only, but the value it
refers to may still be mutable.

```go
const (
    a = 1
    b = {foo: "bar"}
)
a = 2          // compile error
b.foo = "baz"  // allowed: the dict itself is mutable
```

### iota

Inside a `const` block, `iota` counts declarations from 0 and may appear in any
expression on the right-hand side.

```go
const (
    x = iota   // 0
    y          // 1
    z          // 2
)

const (
    a = 1 << iota  // 1
    b              // 2
    c              // 4
)

const (
    _ = iota
    kb = "size" + iota  // "size1"
    mb                  // "size2"
)
```

If a variable named `iota` exists before the `const` block, it shadows the
enumerator and no counting happens.

## Scopes and Capturing

Inner functions capture variables from enclosing scopes. Re-declaring a name
with `:=` shadows the outer one.

```go
a := "outer"
func() {
    a = "changed"   // assigns to the outer 'a'
    a := "inner"    // shadows: new local 'a'
}()
```

## Destructuring

See [Collections → Destructuring](collections.md#destructuring) for the full
destructuring reference. A runnable example covering array, named `(;…)` and
TypeScript-style `{…}` destructuring is available in
`samples/27_destructuring.gad`.

Like Go, a loop variable captured by a closure holds its final value unless you
bind a fresh copy inside the loop:

```go
var f
for i := 0; i < 3; i++ {
    i := i           // fresh binding per iteration
    f = func() { return i }
}
println(f())         // 2
```
