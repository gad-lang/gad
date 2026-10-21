
# uint

A **uint** is an unsigned 64-bit integer. Its literals end in `u` (`2u`,
`0xFFu`); a collection prints its uints so (`[2u]`), told apart from ints.

## Constructors

`uint(v)` converts an int, a float, a decimal, a char, a bool, a str.

```gad
[uint(3), uint("7"), uint('A'), uint(2.9), typeName(uint(1))]
= [3u, 7u, 65u, 2u, "uint"]
```

## Operators

As an [int](int.md)'s: arithmetic, bitwise, unary, increment, comparisons —
a uint with a uint is a uint.

```gad
[2u + 3u, 10u / 3u, 6u & 3u, 1u << 4u, 5u > 2u]
= [5u, 3u, 2u, 16u, true]
```

## Type

### uint

```gad
uint type
```

#### Constructors

```gad
uint(v bool)
uint(v char)
uint(v decimal)
uint(v float)
uint(v int)
uint(v rawstr)
uint(v str)
uint(v uint)
```

#### Operators

```gad
+a
-a
^a
a % b
a & b
a &^ b
a * b
a ** b
a + b
a - b
a / b
a < b
a << b
a <= b
a === b
a > b
a >= b
a >> b
a ^ b
a | b
a++
a--
```

#### Conversions

```gad
Range(from uint, to uint; step)
bool(v uint)
char(v uint)
decimal(v uint)
float(v uint)
int(v uint)
str(v uint)
```

## Example — `uint.gad`

```gad
example constructors {
    [uint(3), uint("7"), uint('A'), uint(2.9), typeName(uint(1))]
    = [3u, 7u, 65u, 2u, "uint"]
}

example operators {
    [2u + 3u, 10u / 3u, 6u & 3u, 1u << 4u, 5u > 2u]
    = [5u, 3u, 2u, 16u, true]
}
```
