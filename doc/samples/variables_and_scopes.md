
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

### A const is visible in its whole block

A `const` may be used before it is written: it is visible in the whole block it
is declared in. The named declarations are consts too — `func`, `class`,
`mixin`, `type` (a marker type, a typed array, a type union), `interface`,
`meti` and `enum` —, so they may be written in any order and refer to one
another.

A const is assigned where it is written, unless something before needs its
value: then it is assigned first, right before that (and what it needs, in
turn). A class, a mixin and a marker type exist from the start of the block and
are defined where they are written, so they may refer to each other in a cycle;
extending a class, or using a mixin, needs its definition.

```gad
x := twice(k)          // k and twice are declared below
func twice(n) => n * 2
const k = 21

class Node {
    use Named                   // Named is declared below
    children? Nodes             // Nodes too…
}
type Nodes []Node               // …an array of Node
mixin Named { name = "?" }

n := Node(;name="root", children=Nodes(Node()))
[x, n.name, len(n.children)]
// => [42, "root", 1]
```

A cycle of values (`const a = b; const b = a`) is a compile error, and so is a
const needed before a variable its value uses is declared. A const using `iota`,
repeating the value before it, or destructuring is visible from its declaration
on, as a `var` is.

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

x := twice(k)          // k and twice are declared below
func twice(n) => n * 2
const k = 21

class Node {
    use Named                   // Named is declared below
    children? Nodes             // Nodes too…
}
type Nodes []Node               // …an array of Node
mixin Named { name = "?" }

n := Node(;name="root", children=Nodes(Node()))
[x, n.name, len(n.children)]

var f
for i := 0; i < 3; i++ {
    i := i // fresh binding per iteration
    f = func() => i
}
f()

return "variables"
```
