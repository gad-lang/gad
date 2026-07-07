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

## See also

For runnable examples, see:
- `samples/12_method_interfaces.gad` — method interfaces with `meti` and `implements`
- `samples/24_interfaces.gad` — structural `interface{…}` contracts
- `samples/25_method_resolution.gad` — method resolution order with `met` overloads

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

## Interfaces (`interface`)

An `interface { … }` is a richer structural contract that groups typed fields,
`get`/`set`/`prop` accessors and required methods. Like `meti`, it compiles to a
constant value (`Interface`) whose members are read by indexing. The statement
form binds a const; the expression form is a value.

```go
interface Shape {
    *Base                   // parent interface (spread; no alias), like a class

    id int                  // typed field; a bare field defaults to `any`
    label str

    get area uint           // getter (returns the type)
    set scale               // setter (takes the type)
    prop title              // property = getter + setter

    draw()                  // required method, func-header shape (no `<…>`)
    resize(int|uint) <bool> // a bare positional entry is a type: `(_ int|uint)`

    from {                  // a method with several overload signatures
        (str)               //   (meti-style, without the `meti` keyword)
        (w int, h int)
    }
}
```

Members are read by indexing:

```go
Shape.name              // "Shape"
Shape.fields[0].name    // "id"
Shape.fields[0].types   // [int]
Shape.props[0].name     // "area"
Shape.methods           // [draw, resize, from]
Shape.methods[2].headers // the two `from` signatures
```

An anonymous interface (or one used as an expression) is compiled with an
incremented `ifaces#N` name. See [`samples/24_interfaces.gad`](../samples/24_interfaces.gad).

### Structural satisfaction

A value **satisfies** an interface when it has every required field (with an
assignable type), property and method (whose signatures match), plus any
extended interface. Use the [`::` operator](operators.md#assign-to-type) to check
it, or an interface as a parameter type — a non-satisfier is rejected:

```go
interface Greeter { greet() <str> }
class Person { name = ""; methods { greet() => "hi " + this.name } }

Person(; name = "Ann") :: Greeter       // ok -> the person (has greet())
42 :: Greeter                           // raises ErrIncompatibleAssign

func welcome(g interface{ greet() <str> }) => g.greet()
welcome(Person(; name = "Bo"))          // "hi Bo"
welcome(42)                             // rejected: 42 does not satisfy the interface
```

An inline `interface{…}` (or `met<…>`) parameter type is checked up front at the
call. A satisfying class instance, dict or other member-bearing value is
accepted.

Satisfaction works against any member-bearing value, not just class instances:

- **fields** and **properties** match a class field/getter or a key of any
  indexable value (a `dict`, key-value array, …);
- **methods** match a class method, a callable field/key (a `dict` whose value
  is a function), or — for a value that dispatches methods by name
  (`NameCallerObject`) — are accepted optimistically (duck typing), the call
  resolving at runtime.

```go
interface Greeter { name str; greet() <str> }
{ name: "Ann", greet: func() => "hi" } :: Greeter   // ok — dict satisfies it
{ name: "Ann" } :: Greeter                           // rejected — no greet()
```

### Reflected Go values

A Go value handed to a script through reflection (`gad.NewReflectValue` /
`gad.MustNewReflectValue`) satisfies interfaces by its **Go fields and methods**.
This holds for every `ReflectValuer` kind — a struct, a named slice, a named
array and a named map:

```go
// Go side:
type Person struct{ Name string; Age int }
func (p Person) Greet() string { return "hi " + p.Name }
// … expose Person{…} to the script as the global `p` via NewReflectValue.
```

```go
// Script side:
interface Greeter { Name str; Greet() <str> }
p :: Greeter        // ok — the reflected struct has the Name field and Greet()
p.Greet()           // "hi Ann"
```

Two rules apply to the check:

- **Fields** (and properties) are matched **structurally**: a required field the
  Go value does not expose (an exported struct field, or a string key of a
  reflected map) **rejects** it.
- **Methods** are matched **optimistically** (duck typing): the reflected value
  dispatches methods by name, so a required method is accepted at the interface
  check and only resolved when actually called. Requiring a method the Go type
  does not define therefore passes the `::` check but fails at call time.

A `*gad.ReflectType` is itself a type assigner: `rt.CanAssign(obj)` reports
whether `obj` is assignable to that Go type, and it can be used as a parameter
type. See `ExampleNewReflectValue_structuralContract` and the
`TestReflect*StructuralContract` tests in `objects_reflect_interface_test.go`.
