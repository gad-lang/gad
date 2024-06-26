
[//]: <> (Generated by gaddoc. DO NOT EDIT.)

# `fmt` Module

## Scan Examples

```go
arg1 := fmt.ScanArg("string")
arg2 := fmt.ScanArg("int")
ret := fmt.Sscanf("abc123", "%3s%d", arg1, arg2)
if isError(ret) {
  // handle error
  fmt.Println(err)
} else {
  fmt.Println(ret)            // 2, number of scanned items
  fmt.Println(arg1.Value)     // abc
  fmt.Println(bool(arg1))     // true, reports whether arg1 is scanned
  fmt.Println(arg2.Value)     // 123
  fmt.Println(bool(arg2))     // true, reports whether arg2 is scanned
}
```

```go
arg1 = fmt.ScanArg("string")
arg2 = fmt.ScanArg("int")
arg3 = fmt.ScanArg("float")
ret = fmt.Sscanf("abc 123", "%s%d%f", arg1, arg2, arg3)
fmt.Println(ret)         // error: EOF
fmt.Println(arg1.Value)  // abc
fmt.Println(bool(arg1))  // true
fmt.Println(arg2.Value)  // 123
fmt.Println(bool(arg2))  // true
fmt.Println(arg3.Value)  // nil
fmt.Println(bool(arg2))  // false, not scanned

// Use if statement or a ternary expression to get the scanned value or a default value.
v := arg1 ? arg1.Value : "default value"
```

## Functions

`Print(...any) -> int`

Formats using the default formats for its operands and writes to standard
output. Spaces are added between operands when neither is a string.
It returns the number of bytes written and any encountered write error
throws a runtime error.

---

`Printf(format string, ...any) -> int`

Formats according to a format specifier and writes to standard output.
It returns the number of bytes written and any encountered write error
throws a runtime error.

---

`Println(...any) -> int`

Formats using the default formats for its operands and writes to standard
output. Spaces are always added between operands and a newline
is appended. It returns the number of bytes written and any encountered
write error throws a runtime error.

---

`Sprint(...any) -> string`

Formats using the default formats for its operands and returns the
resulting string. Spaces are added between operands when neither is a
string.

---

`Sprintf(format string, ...any) -> string`

Formats according to a format specifier and returns the resulting string.

---

`Sprintln(...any) -> string`

Formats using the default formats for its operands and returns the
resulting string. Spaces are always added between operands and a newline
is appended.

---

`Sscan(str string, ScanArg[, ...ScanArg]) -> int | error`

Scans the argument string, storing successive space-separated values into
successive ScanArg arguments. Newlines count as space. If no error is
encountered, it returns the number of items successfully scanned. If that
is less than the number of arguments, error will report why.

---

`Sscanf(str string, format string, ScanArg[, ...ScanArg]) -> int | error`

Scans the argument string, storing successive space-separated values into
successive ScanArg arguments as determined by the format. It returns the
number of items successfully parsed or an error.
Newlines in the input must match newlines in the format.

---

`ScanArg(typeName string) -> scanArg`

Returns a `scanArg` object to scan a value of given type name in scan
functions.
Supported type names are `"string", "int", "uint", "float", "char",
"bool", "bytes"`.
It throws a runtime error if type name is not supported.
Alternatively, `string, int, uint, float, char, bool, bytes` builtin
functions can be provided to get the type name from the BuiltinFunction's
Name field.
