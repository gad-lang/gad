
# decimal

A **decimal** is an arbitrary-precision base-10 number: `0.1 + 0.2` is
exactly `0.3`. Its literals end in `d` (`1.50d`); its scale is kept, and
printed (`1.50d`).

## Constructors

`decimal(v)` converts a str (its digits, exactly), an int, a uint, a float.

```gad
[decimal("1.10"), decimal(2), typeName(decimal(1.5))]
= [1.10d, 2d, "decimal"]
```

## Methods

`d.trunc(precision)` truncates to `precision` decimal places.

```gad
(3.14159d).trunc(2)
= 3.14d
```

## Operators

The arithmetic (`+ - * / % **`), the unary `-a`/`+a`, the increment and the
comparisons; an int or a float operand is converted. A division keeps 16
decimal places (`10d / 4d` is `2.5000000000000000d`, equal to `2.5d`).

```gad
[0.1d + 0.2d, 0.1d + 0.2d == 0.3d, 10d / 4d == 2.5d, -1.5d]
= [0.3d, true, true, -1.5d]
```

## Type

### decimal

```gad
decimal type
```

#### Constructors

```gad
decimal(v bool)
decimal(v char)
decimal(v decimal)
decimal(v float)
decimal(v int)
decimal(v rawstr)
decimal(v str)
decimal(v uint)
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
Range(from decimal, to decimal; step)
bool(v decimal)
char(v decimal)
float(v decimal)
int(v decimal)
str(v decimal)
uint(v decimal)
```

## Example — `decimal.gad`

```gad
example constructors {
    [decimal("1.10"), decimal(2), typeName(decimal(1.5))]
    = [1.10d, 2d, "decimal"]
}

example methods {
    (3.14159d).trunc(2)
    = 3.14d
}

example operators {
    [0.1d + 0.2d, 0.1d + 0.2d == 0.3d, 10d / 4d == 2.5d, -1.5d]
    = [0.3d, true, true, -1.5d]
}
```
