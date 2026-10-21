
# bool

A **bool** is `true` or `false`. Every value has a truthiness — what `if`,
`!` and `bool(v)` see —: `false`, `nil`, `0`, `""`, an empty array or dict
are falsy, the rest truthy.

## Constructors

`bool(v)` is the truthiness of any value; it never fails.

```gad
[bool(1), bool(0), bool(""), bool("x"), bool([]), bool(nil)]
= [true, false, false, true, false, false]
```

## Operators

`!a` negates; `&&` and `||` short-circuit, giving one of their operands.

```gad
[!true, true && false, false || "x", nil || 0 || "z"]
= [false, false, "x", "z"]
```

## See also

- `flag` (`yes`/`no`), the truth of a named option.

## Type

### bool

```gad
bool type
```

#### Constructors

```gad
bool(v bool)
bool(v char)
bool(v decimal)
bool(v float)
bool(v int)
bool(v rawstr)
bool(v str)
bool(v uint)
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
```

#### Conversions

```gad
char(v bool)
decimal(v bool)
float(v bool)
int(v bool)
str(v bool)
uint(v bool)
```

## Example — `bool.gad`

```gad
example constructors {
    [bool(1), bool(0), bool(""), bool("x"), bool([]), bool(nil)]
    = [true, false, false, true, false, false]
}

example operators {
    [!true, true && false, false || "x", nil || 0 || "z"]
    = [false, false, "x", "z"]
}
```
