# Values and Types

[← Back to index](README.md)

In Gad, everything is a value and every value has a type. Use the `typeName`
builtin to inspect a value's type at runtime.

```go
println(typeName(42))      // int
println(typeName(3.14))    // float
println(typeName("hi"))    // str
println(typeName([1, 2]))  // array
```

## Type Overview

| Type               | Description                          | Go equivalent          |
|--------------------|--------------------------------------|------------------------|
| `int`              | signed 64-bit integer                | `int64`                |
| `uint`             | unsigned 64-bit integer              | `uint64`               |
| `float`            | 64-bit floating point                | `float64`              |
| `decimal`          | arbitrary-precision decimal          | shopspring/decimal     |
| `bool`             | `true` / `false`                     | `bool`                 |
| `flag`             | `yes` / `no` (prints `on` / `off`)   | `bool`                 |
| `char`             | a single unicode code point          | `rune`                 |
| `str`              | unicode string                       | `string`               |
| `rawStr`           | raw (un-escaped) string              | `string`               |
| `bytes`            | byte slice                           | `[]byte`               |
| `array`            | ordered list of values               | `[]Object`             |
| `dict`             | string-keyed map of values           | `map[string]Object`    |
| `keyValue`         | a single `key=value` pair            | —                      |
| `keyValueArray`    | ordered list of `key=value` pairs    | —                      |
| `error`            | error value                          | —                      |
| `nil`              | absence of a value                   | —                      |
| `compiledFunction` | a Gad function                       | —                      |

## Type Constructors and Methods

Every value type is **callable as a constructor** that converts a compatible
value: `int("42")`, `float(3)`, `str(7)`, `char(65)`, `bool(0)`,
`bytes("hi")`, … Each constructor is built from **typed methods** — one
overload per accepted input kind — so the conversion that runs is chosen by the
argument's type.

You can list a type's constructor methods (and any methods added to it) with
`repr(T; indent)`:

```go
repr(int; indent)
// ‹builtin type ‹int› with N methods: [
//   ⨍(bool)    🠆 ‹function int(v bool)›,
//   ⨍(char)    🠆 ‹function int(v char)›,
//   ⨍(float)   🠆 ‹function int(v float)›,
//   ⨍(str)     🠆 ‹function int(v str)›,
//   … ]
```

The value-type constructors (`bool`, `int`, `uint`, `float`, `decimal`, `char`,
`str`, `rawStr`, `bytes`) accept the numeric kinds, `char`, `bool` and string
forms; an input not matched by a typed method falls back to the constructor's
default coercion. The `time` module types (`time`, `calendarDate`,
`calendarTime`, `duration`, `Location`) and the `Range` type expose their typed
constructors the same way.

New typed methods can be added to a type from a script with `met`, or from Go
with `AddMethod` — see [Embedding](embedding.md#typed-methods-with-addmethod).

## Numbers

```go
19 + 84        // int
1u + 5u        // uint
-9.22 + 1e10   // float
2d + 0.5d      // decimal (suffix d)
0x1F           // hex int   == 31
017            // octal int == 15
```

Convert between numeric types with the constructor builtins:

```go
println(int("-999"))   // -999
println(int("0x1F"))   // 31
println(float(-51))    // -51
println(decimal(1))    // 1
println(string(1984))  // "1984"
```

## Booleans and Flags

Gad has two boolean-like types. `bool` is the usual `true`/`false`. `flag` is a
distinct on/off type written `yes`/`no` and printed as `on`/`off`.

```go
println(true || false)      // true
println(yes, no)            // on off
println(typeName(yes))      // flag
println(flag("a"), flag("")) // on off
```

## Characters

A `char` is a single unicode code point written with single quotes. Characters
support arithmetic and comparison; adding an int shifts the code point and keeps
the `char` type.

```go
'ç' > '9'         // true
println('A' + 1)  // B   (still a char)
println(char(88)) // X   (code point 88)
println(int('A')) // 65
```

## Strings, Bytes and Regex

Strings, raw strings, heredocs, template strings, **bytes literals**
(`b"…"`, `h"…"`) and `/regex/` literals each have their own chapter:
[Strings, Bytes and Regex](strings-bytes-regex.md). A quick taste:

```go
"foo" + `bar`    // "foobar"   (str + rawStr)
b"Hello"         // bytes from a string
h"ffccf1c2"      // bytes from hex
/ab+/            // a compiled regexp
```

## Arrays

An array is an ordered list of values of any type, indexed with `[]`.

```go
a := ["foo", 'x', [1, 2, 3], {bar: 2u}, true, nil]
println(a[0])    // "foo"
println(a[2][1]) // 2
println(len(a))  // 6
```

See [Collections](collections.md) for slicing, comprehensions and spreading.

## Dicts

A dict maps string keys to values. Access elements with `[]` or the `.`
selector.

```go
m := {a: 1, "b": false, c: "foo"}
println(m.a)      // 1
println(m["b"])   // false
println(m.x)      // nil  (missing key)
m.x = 10          // add a key
```

`{}` at statement position opens a block, **not** an empty dict. Wrap a dict in
parentheses where a block would otherwise be parsed: `({})`.

## keyValue and keyValueArray

A `key=value` pair (`[k=v]`) is its own value, and a parenthesised `;`-prefixed
list is a `keyValueArray` — an **ordered** list of pairs (duplicate keys allowed)
and the literal form behind named arguments.

```go
println([a=1])              // [a=1]            (keyValue)
println((;a=1, b=2))        // (;a=1, b=2)      (keyValueArray)
println(typeName([a=1]))    // keyValue
println(typeName((;a=1)))   // keyValueArray
```

A keyValue exposes `.k`, `.v` and `.array` (`[k, v]`); within a keyValueArray the
entries are mutable (`kva[i].v = …`).

Keys may be identifiers, strings, numbers, booleans, or a dynamic `[(expr)=v]`.
A bare key is a **flag** (`yes`); `=no` drops the entry. Values are any
expression — including **functions and closures**. A key may also be **typed**
(`name Type`), which records the type as metadata (the same form types named
parameters, e.g. `func(; n int = 0)`):

```go
(; debug, verbose=no, level=3)          // (;debug, level=3)
(; greet() => "hi", add(a, b) => a + b) // closure/func values
(; id int, label str = "none")          // typed keys
```

Useful methods: `.flag(name)`, `.values(names…)` (all values, or only those
named), `.delete(names…)`, and `**` spreads another keyValueArray. Iterate it as
`for k, v in kva`, and `dict(kva)` converts it (last value per key wins). See
[`samples/22_key_value_array.gad`](../samples/22_key_value_array.gad).

## Nil

`nil` represents a missing or undefined value. Functions with no explicit
`return`, missing dict keys and some builtins yield `nil`.

```go
a := func() { b := 4 }()  // a == nil
c := {a: "foo"}["b"]      // c == nil
println(isNil(a), c == nil)
```

## Functions

Functions are first-class values; they can be stored, passed and returned.

```go
add := func(a, b) { return a + b }
mul := (a, b) => a * b      // arrow closure
println(add(2, 3), mul(2, 3))
```

See [Functions](functions.md) for closures, variadics, named arguments and
deferred handlers.

## Properties

A `Prop` is a named, callable value whose getter and setter are dispatched by
the call signature: calling it with no argument runs the getter, calling it with
one argument runs the setter whose parameter type matches.

A property can be built with the `Prop` constructor:

```go
var value
const p = Prop("x", () => value, (v) => { value = v })

// attach a typed setter later with `met`
met p(v int) {
  value = "int value= " + v
}

p()      // nil
p("a")   // setter runs: value = "a"
p()      // "a"
p(1)     // typed (int) setter selected
p()      // "int value= 1"
```

The same property can be declared with the `prop` keyword, which uses the
[func-with-methods](functions.md#functions-with-methods) body syntax. A method
with no parameters is the getter; a method with one parameter is a setter:

```go
var value
prop x {
  ()      => value          // getter
  (v)     { value = v }     // setter
  (v int) { value = "int value= " + v }   // typed setter
}
```

At most one getter may be registered; any number of setters may be registered
and are selected by their parameter type. A property created with no methods is
valid, but calling it is an error because no matching method exists.

```go
const pi = Prop("pi", () => 3.14)   // read-only
pi()        // 3.14
```

## Copy Semantics

Assignment copies values, except for the reference types `array`, `dict` and
`bytes`, which share their backing storage (as in Go). Use `copy` for a shallow
copy and `dcopy` for a deep copy.

```go
a := [1, 2, 3]
b := a          // shares storage
b[0] = 99
println(a[0])   // 99

c := copy(a)    // independent copy
c[0] = 0
println(a[0])   // 99
```

## See also

For a runnable tour of the built-in types, see `samples/02_values_and_types.gad`.
