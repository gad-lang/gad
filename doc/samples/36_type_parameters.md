
# Type Parameters (generics with constraints)

A function signature may declare **type parameters** between the name (or `func`)
and the parameter list: `func mySet[T indexable, K int|uint, V number](…)`. Each
entry is `NAME CONSTRAINT`, where the constraint is written exactly like a
parameter type — a single type, an interface, or a `|` union.

A type parameter is a **named, reusable constraint**: wherever its name appears in
a parameter or return type, it is replaced by the constraint. `[T indexable]` then
`target T` is the same as writing `target indexable`; `[K int|uint]` then `k K`
is the same as `k int|uint`. This lets one constraint be named once and reused
across several parameters and the return type.

## Where they work

Type parameters are accepted in every function-signature form:

- **named** — `func mySet[T indexable](target T) <T> { … }`
- **anonymous** — `const f = func[T number](v T) => v` (a space is optional:
  `func [T number]`)
- **dict method shorthand** — `{ scale[V number](v V, n int) <V>: v * n }`
- **func-header value** — `<[T number](v T) <T>>`
- **method interface** — `meti { [T number](v T) <T> }`

```gad
// T is indexable, K is int|uint, V is a number; the return type reuses T.
func mySet[T indexable, K int|uint, V number](target T, k K, v V) <T> {
    target[k] = v
    return target
}
mySet([10, 20, 30], 1, 99)
// => [10, 99, 30]
```

## Reusing one constraint across parameters

Naming the constraint once keeps related parameters in sync — here both operands
and the result share `T`:

```gad
// both arguments and the result share the single constraint T (str or array).
concat := func[T str|array](a T, b T) <T> => a + b
[concat("ab", "cd"), concat([1, 2], [3])]
// => ["abcd", [1, 2, 3]]
```

## Anonymous functions and dict methods

```gad
double := func[T number](v T) <T> => v * 2       // anonymous, `func[…]`
d := { scale[V number](v V, n int) <V>: v * n }  // dict method shorthand
[double(21), d.scale(5, 3)]
// => [42, 15]
```

## Headers and method interfaces

A generic `meti` describes a generic method; a generic function satisfies it:

```gad
m := meti { [T number](v T) <T> }                // a generic method interface
f := func[T number](v T) <T> => v                // a generic function
implements(f, m)
// => true
```

## Example — `36_type_parameters.gad`

```gad
// T is indexable, K is int|uint, V is a number; the return type reuses T.
func mySet[T indexable, K int|uint, V number](target T, k K, v V) <T> {
    target[k] = v
    return target
}
mySet([10, 20, 30], 1, 99)

// both arguments and the result share the single constraint T (str or array).
concat := func[T str|array](a T, b T) <T> => a + b
[concat("ab", "cd"), concat([1, 2], [3])]

double := func[T number](v T) <T> => v * 2       // anonymous, `func[…]`
d := { scale[V number](v V, n int) <V>: v * n }  // dict method shorthand
[double(21), d.scale(5, 3)]

m := meti { [T number](v T) <T> }                // a generic method interface
f := func[T number](v T) <T> => v                // a generic function
implements(f, m)

return "type parameters"
```
