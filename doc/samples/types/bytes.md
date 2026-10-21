
# bytes

A **bytes** is a mutable sequence of bytes. Its literals are `b"…"` (text)
and `h"…"` (hex); a collection prints one as `bytes("…")`.

## Constructors

`bytes(v)` is the bytes of a str, a new bytes of an int length (zeros);
`bytes(b1, b2, …)` of byte values.

```gad
[str(bytes("hi")), len(bytes(3)), bytes(104, 105) == bytes("hi")]
= ["hi", 1, true]
```

## Operators

`+` concatenates; a bytes indexes by byte (an int) and slices.

```gad
b := bytes("ab") + bytes("c")
[str(b), b[0], str(b[1:]), len(b)]
= ["abc", 97, "bc", 3]
```

## See also

- [Strings, bytes and regexps](../strings_bytes_regex.md).

## Type

### bytes

```gad
bytes type
```

#### Constructors

```gad
bytes(v array)
bytes(v bytes)
bytes(v int)
bytes(v rawstr)
bytes(v str)
```

#### Operators

```gad
a + b
a ++ b
a < b
a <= b
a === b
a > b
a >= b
a in b
```

#### Conversions

```gad
str(v bytes)
uuid(v bytes)
```

## Example — `bytes.gad`

```gad
example constructors {
    [str(bytes("hi")), len(bytes(3)), bytes(104, 105) == bytes("hi")]
    = ["hi", 1, true]
}

example operators {
    b := bytes("ab") + bytes("c")
    [str(b), b[0], str(b[1:]), len(b)]
    = ["abc", 97, "bc", 3]
}
```
