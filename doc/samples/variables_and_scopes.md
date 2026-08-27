
# Variables and Scopes

Gad has four declaration keywords — `param`, `global`, `var` and `const` — plus
the short declaration operator `:=`. Identifiers may contain letters, digits, `_`
and `$`.

## `:=` vs `=`

`:=` declares a **new** local and assigns to it; `=` assigns to an **existing**
variable (or a dict/array element). A variable may be reassigned a value of a
different type.

```gad
a := 123      // declare 'a' (int)
a = "123"     // reassign (str) — a variable may change type
a = [1, 2, 3] // reassign (array)
a
// => [1, 2, 3]
```

## param

`param` declares the parameters of the main script function. It may appear only
once, at the top level; initial values are illegal (a variadic `*x` defaults to
`[]`, everything else to `nil`). The positional and named lists are separated by
`;`, following the same rules as [function parameters](functions.gad).

```gad ignore
param (arg0, arg1, *rest)         // positional + variadic
param (; x, y = 1, **named)        // named only (with defaults)
param (a, *rest; x, y = 1, **nx)   // mixed
```

A `param` (and a `global`) name may carry a **type** — a single type, a `|` union
or an interface — enforced when a value is bound. `var` and `const` are untyped.

```gad ignore
param (a int, b str)               // typed positionals
param (id int|uint; page int = 1)  // typed union; typed named with default
```

## global

`global` declares variables backed by the host-provided globals object (`@g`) —
how an embedding Go program exchanges data with a script. A grouped `global (…)`
may give defaults: `name = value` applies when the global is nil or absent (like
`??=`), `name !?= value` only when it is absent. A global may also be typed
(`global (page int = 1)`).

```gad
global (page = 1, limit = 20) // default unless the host set them
global (user !?= "guest")     // only when "user" is not provided at all
[page, limit, user]
// => [1, 20, "guest"]
```

## var

`var` declares one or more locals, optionally initialized (uninitialized ⇒
`nil`). A self-referential function value must be declared before it is assigned,
because the right-hand side is compiled first.

```gad
var foo            // nil
var (bar, baz = 1) // bar == nil, baz == 1
[isNil(foo), bar, baz]
// => [true, nil, 1]
```

## const

`const` declares read-only bindings; an initializer is required and reassignment
is a compile error (though the value it refers to may still be mutable). Inside a
`const` block, `iota` counts declarations from 0 and may appear in any
right-hand-side expression.

```gad
const (i0 = iota, i1, i2)         // 0, 1, 2
const (bit1 = 1 << iota, bit2, bit4) // 1, 2, 4 (iota resets per block)
const box = {foo: "bar"}
box.foo = "baz"                   // ok: the binding is read-only, the dict is not
[i0, i1, i2, bit1, bit2, bit4, box.foo]
// => [0, 1, 2, 1, 2, 4, "baz"]
```

## Scopes and capturing

Inner functions capture variables from enclosing scopes; re-declaring a name with
`:=` shadows the outer one. Like Go, a loop variable captured by a closure holds
its final value unless you bind a fresh copy inside the loop.

```gad
var f
for i := 0; i < 3; i++ {
    i := i // fresh binding per iteration
    f = func() => i
}
f()
// => 2
```

## Example — `variables_and_scopes.gad`

```gad
a := 123      // declare 'a' (int)
a = "123"     // reassign (str) — a variable may change type
a = [1, 2, 3] // reassign (array)
a

global (page = 1, limit = 20) // default unless the host set them
global (user !?= "guest")     // only when "user" is not provided at all
[page, limit, user]

var foo            // nil
var (bar, baz = 1) // bar == nil, baz == 1
[isNil(foo), bar, baz]

const (i0 = iota, i1, i2)         // 0, 1, 2
const (bit1 = 1 << iota, bit2, bit4) // 1, 2, 4 (iota resets per block)
const box = {foo: "bar"}
box.foo = "baz"                   // ok: the binding is read-only, the dict is not
[i0, i1, i2, bit1, bit2, bit4, box.foo]

var f
for i := 0; i < 3; i++ {
    i := i // fresh binding per iteration
    f = func() => i
}
f()

return "variables"
```
