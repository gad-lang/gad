
# float

A **float** is a 64-bit IEEE-754 floating-point number. Its literals have a
point or an exponent (`1.5`, `2.0`, `1e20`); a collection prints a whole
float with its point (`[2.0]`), told apart from an int.

## Constructors

`float(v)` converts a str, an int, a uint, a decimal.

```gad
[float("1.5"), float(2), float(3u), float(1.25d)]
= [1.5, 2.0, 3.0, 1.25]
```

## Operators

The arithmetic (`+ - * / % **`), the unary `-a`/`+a`, the increment and the
comparisons; an int operand is converted.

```gad
[1.5 + 1, 7.0 / 2, 2.0 ** 3, -1.5, 0.1 + 0.2 > 0.3]
= [2.5, 3.5, 8.0, -1.5, true]
```

## See also

- [decimal](decimal.md), for exact (base-10) arithmetic.

## Type

### float

```gad
float type
```

#### Constructors

```gad
float(v bool)
float(v char)
float(v decimal)
float(v float)
float(v int)
float(v rawstr)
float(v str)
float(v uint)
```

#### Operators

```gad
+a
-a
a * b
a ** b
a + b
a - b
a / b
a < b
a <= b
a === b
a > b
a >= b
a++
a--
```

#### Conversions

```gad
Range(from float, to float; step)
bool(v float)
char(v float)
decimal(v float)
int(v float)
str(v float)
uint(v float)
```

## Example — `float.gad`

```gad
example constructors {
    [float("1.5"), float(2), float(3u), float(1.25d)]
    = [1.5, 2.0, 3.0, 1.25]
}

example operators {
    [1.5 + 1, 7.0 / 2, 2.0 ** 3, -1.5, 0.1 + 0.2 > 0.3]
    = [2.5, 3.5, 8.0, -1.5, true]
}
```
