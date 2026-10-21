
# char

A **char** is a Unicode code point. Its literals are `'a'`, `'\n'`,
`'é'`; a collection prints them so (`['a']`).

## Constructors

`char(v)` converts an int (a code point) or a single-character str.

```gad
[char(65), char("é"), int('A'), str('z')]
= ['A', 'é', 65, "z"]
```

## Operators

A char compares with a char, and adds an int (the next code points):
`'a' + 1` is `'b'`.

```gad
['a' + 1, 'a' < 'b', 'z' - 1]
= ['b', true, 'y']
```

## Type

### char

```gad
char type
```

#### Constructors

```gad
char(v bool)
char(v char)
char(v decimal)
char(v float)
char(v int)
char(v rawstr)
char(v str)
char(v uint)
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
a++
a--
```

#### Conversions

```gad
Range(from char, to char; step)
bool(v char)
decimal(v char)
float(v char)
int(v char)
str(v char)
uint(v char)
```

## Example — `char.gad`

```gad
example constructors {
    [char(65), char("é"), int('A'), str('z')]
    = ['A', 'é', 65, "z"]
}

example operators {
    ['a' + 1, 'a' < 'b', 'z' - 1]
    = ['b', true, 'y']
}
```
