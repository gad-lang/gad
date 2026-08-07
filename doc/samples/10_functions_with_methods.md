
# Functions with Methods

A function can hold several **methods** — overloads selected at call time by the
argument types. (Plain functions, closures, variadics, named args and `defer`
live in [03_functions.gad](03_functions.gad); describing/checking signatures with
`<…>`, `meti` and `implements` in
[12_method_interfaces.gad](12_method_interfaces.gad).)

## Typed parameters

A parameter may declare a type; the argument is checked against it.

```gad
/// A parameter may declare a type; the argument is checked against it.
repeat := func(s str, n int) {
    out := ""
    for i := 0; i < n; i++ {
        out += s
    }
    return out
}
repeat("ab", 3)
// => ababab
```

## Functions with methods (overloads)

Instead of a single parameter list, write a brace block whose entries are each
`(params) <ret> { body }` (the return-type list and a `=>` expression body are
optional). The dispatcher picks the method whose parameters match the call; if
none match it raises `ErrNoMethodFound`. A named `func area { … }` binds the
function; the same form is valid as an expression value.

```gad
func area {
    (r float)          => 3.14159 * r * r // circle
    (w float, h float) => w * h           // rectangle
}
[area(2.0), area(2.0, 3.0)]
// => [12.56636, 6]
```

New methods can be added to an existing callable later with `met`.

```gad
/// `met` adds a method to an existing callable afterwards.
met area(side int) => side * side // square (int), extends the `area` above
println("area(4) =", area(4))     // 16
```

## Overriding and `$old`

`met ~name(...)` **overrides** an existing method signature instead of raising a
duplicate-method error — the last definition wins. To wrap the previous
implementation (super / around advice), give the override a special `$old` first
parameter: it captures the method being replaced, is dropped from the real
signature, and is callable inside the body. Overrides chain (each `$old` sees the
layer beneath); when no previous method matches, `$old` is `nil`.

```gad
func step(n int) => n * 10
met ~step($old, n int) => $old(n) + 1 // wrap the previous `step`
step(3)                               // 30 + 1
// => 31
```

Under the hood `$old` is `gad.methodFromArgs(fn, types…)`, a builtin that returns
the method a call would dispatch to — selected by an example value or by a type
name — or `nil` when none matches.

```gad
func size(n int) => n * 2
// the (int) method, chosen by an example value or by a type name:
[gad.methodFromArgs(size, 4)(10), gad.methodFromArgs(size, int)(10)]
// => [20, 20]
```

## Example — `10_functions_with_methods.gad`

```gad
/// A parameter may declare a type; the argument is checked against it.
repeat := func(s str, n int) {
    out := ""
    for i := 0; i < n; i++ {
        out += s
    }
    return out
}
repeat("ab", 3)

func area {
    (r float)          => 3.14159 * r * r // circle
    (w float, h float) => w * h           // rectangle
}
[area(2.0), area(2.0, 3.0)]

/// `met` adds a method to an existing callable afterwards.
met area(side int) => side * side // square (int), extends the `area` above
println("area(4) =", area(4))     // 16

func step(n int) => n * 10
met ~step($old, n int) => $old(n) + 1 // wrap the previous `step`
step(3)                               // 30 + 1

func size(n int) => n * 2
// the (int) method, chosen by an example value or by a type name:
[gad.methodFromArgs(size, 4)(10), gad.methodFromArgs(size, int)(10)]

return area(4)
```
