
# Values and Types

In Gad, everything is a value and every value has a type. Use `typeof` (or
`typeName`) to inspect a value's type at runtime.

## Type overview

| Type            | Description                        | Go equivalent       |
|-----------------|-----------------------------------|---------------------|
| `int` / `uint`  | signed / unsigned 64-bit integer  | `int64` / `uint64`  |
| `float`         | 64-bit floating point             | `float64`           |
| `decimal`       | arbitrary-precision decimal (`2d`)| shopspring/decimal  |
| `bool`          | `true` / `false`                  | `bool`              |
| `flag`          | `yes` / `no` (prints `on`/`off`)  | `bool`              |
| `char`          | a single unicode code point (`'A'`)| `rune`             |
| `str` / `rawStr`| unicode / raw (un-escaped) string | `string`            |
| `bytes`         | byte slice (`b"…"`, `h"…"`)        | `[]byte`            |
| `array` / `dict`| ordered list / string-keyed map   | `[]Object` / map    |
| `keyValue` / `keyValueArray` | `k=v` pair / ordered pairs | —              |
| `error` / `nil` | error value / absence of a value  | —                   |

Related chapters: [Collections](collections.gad),
[KeyValue arrays](key_value_array.gad),
[Strings, bytes & regex](strings_bytes_regex.gad),
[Functions](functions.gad), [Properties](properties.gad).

## Type constructors

Every value type is **callable as a constructor** that converts a compatible
value (`int("42")`, `float(3)`, `str(7)`, `char(65)`, `bool(0)`, …). Each is
built from typed methods — one overload per input kind — so the conversion is
chosen by the argument's type. List them with `repr(T; indent)`. New typed
methods can be added with `met` (or from Go with `AddMethod`).

```gad
[int("42"), int("0x1F"), float(-51), str(1984), char(88), int('A')]
// => [42, 31, -51, "1984", 'X', 65]
```

## Numbers, booleans & flags

Numeric literals: `int` (`19`, `0x1F`=31, `017`=15), `uint` (`5u`), `float`
(`1e10`), `decimal` (`2d`). `bool` is `true`/`false`; `flag` is a distinct on/off
type written `yes`/`no` and printed `on`/`off`.

```gad
[typeName(19), typeName(5u), typeName(1e10), typeName(2d), 0x1F, 017]
// => ["int", "uint", "float", "decimal", 31, 15]
```

## Characters

A `char` is a single unicode code point in single quotes. Adding an int shifts
the code point and keeps the `char` type.

```gad
['A' + 1, char(88), int('A'), 'ç' > '9']
// => ['B', 'X', 65, true]
```

## Equality

`==` compares values and coerces between numeric kinds; `===` is strict (same
type and value) and `!==` its negation. For `array`/`dict`, `===` is object
identity — every literal is a fresh object.

```gad
a := [1, 2]
[1 == 1u, 1 === 1u, 1.0 === 1, a === a, a === [1, 2]]
// => [true, false, false, true, false]
```

## Nil

`nil` represents a missing or undefined value: a function with no explicit
`return`, a missing dict key, and some builtins yield `nil`.

```gad
x := func() { y := 4 }() // no explicit return -> nil
[isNil(x), {a: "foo"}["b"] == nil]
// => [true, true]
```

## Copy semantics

Assignment copies values, except the reference types `array`, `dict` and `bytes`,
which share their backing storage (as in Go). Use `copy` for a shallow copy and
`dcopy` for a deep copy.

```gad
orig := [1, 2, 3]
alias := orig  // shares storage
alias[0] = 99
indep := copy(orig) // independent shallow copy
indep[1] = 0
[orig[0], orig[1]]
// => [99, 2]
```

## Example — `values_and_types.gad`

```gad
[int("42"), int("0x1F"), float(-51), str(1984), char(88), int('A')]

[typeName(19), typeName(5u), typeName(1e10), typeName(2d), 0x1F, 017]

['A' + 1, char(88), int('A'), 'ç' > '9']

a := [1, 2]
[1 == 1u, 1 === 1u, 1.0 === 1, a === a, a === [1, 2]]

x := func() { y := 4 }() // no explicit return -> nil
[isNil(x), {a: "foo"}["b"] == nil]

orig := [1, 2, 3]
alias := orig  // shares storage
alias[0] = 99
indep := copy(orig) // independent shallow copy
indep[1] = 0
[orig[0], orig[1]]

return [typeof(42), typeof("s"), typeof([1]), typeof({a: 1})]
```
