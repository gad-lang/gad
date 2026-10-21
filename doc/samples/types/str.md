
# str

A **str** is a string of text: an immutable sequence of UTF-8 bytes, indexed
by byte. Its literals are `"…"` (with escapes: `"a\tb\n"`) and the heredocs
(see [heredocs](../heredocs.md)); a `` `…` `` literal is a [rawstr](rawstr.md).

## Constructors

`str(v)` is the text of any value: a number, a bool, a char, bytes, a
rawstr — a str of an array or a dict is its literal (`[1, "a"]`). It never
fails.

```gad
[str(12), str(1.5), str(true), str('a'), str(bytes("hi")), str([1, "a"]), str(nil)]
= ["12", "1.5", "true", "a", "hi", "[1, \"a\"]", "nil"]
```

## Operators

`+` concatenates (a str to anything: the other side's text); `<`, `<=`, `>`,
`>=` compare byte-wise; `in` tests a substring. A str indexes by byte
(`s[0]`, an int) and slices (`s[1:3]`).

```gad
s := "gad"
["a" + "b", "n=" + 1, "ga" in s, s < "gb", s[0], s[1:3], len(s)]
= ["ab", "n=1", true, true, 103, "ad", 3]
```

## See also

- [Strings, bytes and regexps](../strings_bytes_regex.md), the `strings`
  module ([stdlib/strings](../stdlib/strings.md)).

## Type

### str

```gad
str type
```

#### Constructors

```gad
str(v bool)
str(v bytes)
str(v char)
str(v decimal)
str(v float)
str(v int)
str(v rawstr)
str(v str)
str(v uint)
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
bool(v str)
bytes(v str)
char(v str)
decimal(v str)
float(v str)
int(v str)
rawstr(v str)
uint(v str)
uuid(v str)
```

## Example — `str.gad`

```gad
example constructors {
    [str(12), str(1.5), str(true), str('a'), str(bytes("hi")), str([1, "a"]), str(nil)]
    = ["12", "1.5", "true", "a", "hi", "[1, \"a\"]", "nil"]
}

example operators {
    s := "gad"
    ["a" + "b", "n=" + 1, "ga" in s, s < "gb", s[0], s[1:3], len(s)]
    = ["ab", "n=1", true, true, 103, "ad", 3]
}
```
