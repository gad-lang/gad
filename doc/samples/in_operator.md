
# Membership (`in` / `ain`)

`A in B` tests membership and yields a bool — a **value** for arrays/bytes, a
**key** for the dict kinds, a **substring** for strings. `A ain B` ("all in") is
true when **every** value of the left operand is a member of `B`. Both have
comparison precedence and dispatch on the **right** operand (`met gad.binOpIn` /
`gad.binOpAin` in Gad). `in` is also the `for x in y` loop separator —
parenthesize to use it as the operator inside a for header: `for (x in y)`. Part
of the [Operators](user_operators.gad) chapter.

## Containers

```gad
[
    [2 in [1, 2, 3], 9 in [1, 2, 3]],           // array: a value
    ["a" in {a: 1, b: 2}, "z" in {a: 1}],       // dict: a key
    104 in bytes("hi"),                         // bytes: a byte value ('h')
    ['e' in "hello", "ell" in "hello", "xyz" in "hello"], // string: a substring
]
// => [[true, false], [true, false], true, [true, true, false]]
```

## In conditions

`in` has comparison precedence (`1 + 1 in [2]` is `(1 + 1) in [2]`). In a `for`
header, `for x in y` is the loop and `for (x in y)` a for-cond using the
operator.

```gad
nums := [1, 2, 3, 4, 5]
evens := [n for n in nums if n % 2 in [0]]
found := []
i := 0
for (i in [0, 1, 2]) {      // a for-cond, not a loop over the array
    found += i
    i++
}
[evens, found]
// => [[2, 4], [0, 1, 2]]
```

## Custom membership

When neither operand is a built-in container, `in` falls back to the
binary-operator machinery, so a type can define it. `int in int` has no
built-in meaning; here it is given bit-set membership (is bit N set in mask?).

```gad
met gad.binOpIn(bit int, mask int) {
    return mask & (1 << bit) != 0
}
mask := 6                   // 0b110: bit 1 set, bit 0 clear
[1 in mask, 0 in mask]
// => [true, false]
```

## All in (`ain`)

The left operand is an array of values (a scalar is treated as one element); an
empty array is vacuously true. `ain` falls back through `in`, so dict keys and
string substrings work too.

```gad
[
    [1, 2] ain [1, 2, 3],
    [1, 4] ain [1, 2, 3],
    [] ain [1, 2, 3],               // vacuously true
    ["a", "b"] ain {a: 1, b: 2},    // dict keys
    ["ell", "hel"] ain "hello",     // substrings
]
// => [true, false, true, true, true]
```

## Example — `in_operator.gad`

```gad
[
    [2 in [1, 2, 3], 9 in [1, 2, 3]],           // array: a value
    ["a" in {a: 1, b: 2}, "z" in {a: 1}],       // dict: a key
    104 in bytes("hi"),                         // bytes: a byte value ('h')
    ['e' in "hello", "ell" in "hello", "xyz" in "hello"], // string: a substring
]

nums := [1, 2, 3, 4, 5]
evens := [n for n in nums if n % 2 in [0]]
found := []
i := 0
for (i in [0, 1, 2]) {      // a for-cond, not a loop over the array
    found += i
    i++
}
[evens, found]

met gad.binOpIn(bit int, mask int) {
    return mask & (1 << bit) != 0
}
mask := 6                   // 0b110: bit 1 set, bit 0 clear
[1 in mask, 0 in mask]

[
    [1, 2] ain [1, 2, 3],
    [1, 4] ain [1, 2, 3],
    [] ain [1, 2, 3],               // vacuously true
    ["a", "b"] ain {a: 1, b: 2},    // dict keys
    ["ell", "hel"] ain "hello",     // substrings
]
```
