# Func Headers and Method Interfaces

[← Back to index](README.md)

Gad can describe function signatures as values and group them into **method
interfaces**, then check whether a callable satisfies an interface — a
lightweight, structural (duck-typed) contract for functions.

## Func-header values

A signature written between angle brackets is a **func-header** value:

```go
<()>                      // no params, no return
<(v int)>                 // one int param
<(int)>                   // one unnamed int param — same as <(_ int)>
<(a int, b str) <r bool>> // two params and a bool return
```

In a header, a **bare positional entry is a type**, not a parameter name: `(int)`
is the unnamed typed param `(_ int)`. Write `name type` (e.g. `(v int)`) for a
named parameter. An untyped parameter defaults to `any`.

It evaluates to a `FunctionHeader` whose parts are read by indexing — `name`,
`params`, `namedParams` and `return` (each parameter is a `typedIdent`). An
anonymous header is compiled with an incremented `fh#N` name:

```go
h := <(a int, b str) <r bool>>
h.name             // "fh#1"
len(h.params)      // 2
h.params[0].name   // "a"
h.params[0].types  // [int]
h.return[0].name   // "r"
```

## Method interfaces (`meti`)

`meti { … }` lists one or more required headers (written without the angle
brackets) and evaluates to a `MethodInterface`. The headers are separated by
commas or newlines:

```go
Stringer := meti { () <str> }
Container := meti {
    (any)          // accept one value (of any type)
    () <int>       // and report a length
}
```

An anonymous `meti { … }` is compiled with an incremented `meti#N` name; the
statement form below names it explicitly.

The statement form `meti Name { … }` binds a const:

```go
meti Adder { (a int, b int) <int> }
```

A `MethodInterface` exposes `name` and `headers`:

```go
Adder.name           // "Adder"
len(Adder.headers)   // 1
```

## Checking conformance with `implements`

`implements(fn, mi, *otherMi)` reports whether `fn` provides **every** header of
all the given interfaces. A header matches one of `fn`'s methods when the
parameter counts are equal and each parameter type is assignable (an untyped
header parameter matches anything):

```go
Stringer := meti { () <str> }
HasAdd   := meti { (a int) }

implements(func() => "x", Stringer)        // true
implements(func(a) => a, Stringer)         // false  (wrong arity)
implements(func(a int) => a, HasAdd)       // true

// a function with several methods can satisfy several interfaces at once
func shape() => "x"
met shape(a int) => a
implements(shape, Stringer, HasAdd)        // true
```

## Composing interfaces

Interfaces merge with `+` or `append`, producing a new interface with all the
headers:

```go
both := Stringer + HasAdd            // or: append(Stringer, HasAdd)
implements(shape, both)              // true
```
