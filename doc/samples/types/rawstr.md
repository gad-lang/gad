
# rawstr

A **rawstr** is an uninterpreted string: what a `` `…` `` literal writes,
escapes and all, as it is (`` `a\nb` `` is four characters). It behaves as a
[str](str.md) — it concatenates, compares, indexes —, kept apart from it so
that code that builds text (templates, generated source) can tell what must
not be escaped again.

## Constructors

`rawstr(v)` is the text of a value as a rawstr.

```gad
[rawstr("a"), typeName(rawstr(1)), len(`a\nb`), isRawStr(`x`)]
= [`a`, "rawstr", 4, true]
```

## Operators

```gad
[`a` + `b`, typeName(`a` + "b"), `ab` < `b`, "b" in `abc`]
= [`ab`, "rawstr", true, true]
```

## Type

### rawstr

```gad
rawstr type
```

#### Constructors

```gad
rawstr(v rawstr)
rawstr(v str)
```

#### Operators

```gad
a + b
a < b
a <= b
a === b
a > b
a >= b
a in b
```

#### Conversions

```gad
bool(v rawstr)
bytes(v rawstr)
char(v rawstr)
decimal(v rawstr)
float(v rawstr)
int(v rawstr)
str(v rawstr)
uint(v rawstr)
uuid(v rawstr)
```

## Example — `rawstr.gad`

```gad
example constructors {
    [rawstr("a"), typeName(rawstr(1)), len(`a\nb`), isRawStr(`x`)]
    = [`a`, "rawstr", 4, true]
}

example operators {
    [`a` + `b`, typeName(`a` + "b"), `ab` < `b`, "b" in `abc`]
    = [`ab`, "rawstr", true, true]
}
```
