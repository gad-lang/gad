
# int

An **int** is a signed 64-bit integer. Its literals are decimal (`42`), hex
(`0x2A`) and octal, with a leading zero (`052`).

## Constructors

`int(v)` converts a str (its digits), a float (truncated), a decimal, a char
(its code point), a bool (`1`/`0`), a uint; it throws on anything else.

```gad
[int("42"), int(4.9), int('A'), int(true), int(3u), int(2.5d)]
= [42, 4, 65, 1, 3, 2]
```

## Operators

The arithmetic (`+ - * / % **`), the bitwise (`& | ^ &^ << >>`), the unary
(`-a`, `+a`, `^a`), the increment `a++`/`a--` and the comparisons. `/` of two
ints is an int (truncated); a float operand makes it a float.

```gad
a := 7
[a + 2, a / 2, a % 4, a ** 2, a & 3, a << 1, -a, 0x2A, 052]
= [9, 3, 3, 49d, 3, 14, -7, 42, 42]
```

## See also

- [Values and types](../values_and_types.md), [ranges](../ranges.md)
  (`1 .. 10`).

## Type

### int

```gad
int type
```

#### Constructors

```gad
int(v bool)
int(v char)
int(v decimal)
int(v float)
int(v int)
int(v rawstr)
int(v str)
```

```gad
int(v time; unit)
```

converts time to Unix time value elapsed since January 1, 1970 UTC

```gad
int(v uint)
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
Range(from int, to int; step)
bool(v int)
bytes(v int)
char(v int)
decimal(v int)
float(v int)
str(v int)
uint(v int)
```

## Example — `int.gad`

```gad
example constructors {
    [int("42"), int(4.9), int('A'), int(true), int(3u), int(2.5d)]
    = [42, 4, 65, 1, 3, 2]
}

example operators {
    a := 7
    [a + 2, a / 2, a % 4, a ** 2, a & 3, a << 1, -a, 0x2A, 052]
    = [9, 3, 3, 49d, 3, 14, -7, 42, 42]
}
```
