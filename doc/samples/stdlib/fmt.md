
# `fmt` module

## Scan Examples

A scan function returns the number of items scanned and **throws** on a
scan error, so handle failures with `try`/`catch`. Every arg scanned
before the error is still populated (a partial scan).

```gad ignore
arg1 := fmt.scanArg("str")
arg2 := fmt.scanArg("int")
n := fmt.sscanf("abc123", "%3s%d", arg1, arg2)
fmt.println(n)              // 2, number of scanned items
fmt.println(arg1.Value)     // abc
fmt.println(bool(arg1))     // true, reports whether arg1 is scanned
fmt.println(arg2.Value)     // 123
fmt.println(bool(arg2))     // true, reports whether arg2 is scanned
```

```gad ignore
arg1 := fmt.scanArg("str")
arg2 := fmt.scanArg("int")
arg3 := fmt.scanArg("float")
try {
  fmt.sscanf("abc 123", "%s%d%f", arg1, arg2, arg3)  // throws: EOF
} catch err {
  fmt.println(err)            // the scan error
}
// the partial scan is still readable on the args
fmt.println(arg1.Value)  // abc
fmt.println(bool(arg1))  // true
fmt.println(arg2.Value)  // 123
fmt.println(bool(arg2))  // true
fmt.println(arg3.Value)  // nil
fmt.println(bool(arg3))  // false, not scanned
```

## Public API

### print

```gad
print(*any) <int>
```

Formats using the default formats for its operands and writes to standard
output. Spaces are added between operands when neither is a str.
It returns the number of bytes written and any encountered write error
throws a runtime error.

## Example

```gad
// print writes to stdout and returns the byte count.
fmt.print("a", 1)
>>> 2
```

### printf

```gad
printf(format str, *any) <int>
```

Formats according to a format specifier and writes to standard output.
It returns the number of bytes written and any encountered write error
throws a runtime error.

## Example

```gad
fmt.printf("%d!", 5)
>>> 2
```

### println

```gad
println(*any) <int>
```

Formats using the default formats for its operands and writes to standard
output. Spaces are always added between operands and a newline
is appended. It returns the number of bytes written and any encountered
write error throws a runtime error.

## Example

```gad
fmt.println("hi", 1)   // writes "hi 1\n"
>>> 5
```

### sprint

```gad
sprint(*any) <str>
```

Formats using the default formats for its operands and returns the
resulting str. Spaces are added between operands when neither is a
str.

## Example

```gad
fmt.sprint("x=", 5)
>>> "x=5"
```

### sprintf

```gad
sprintf(format str, *any) <str>
```

Formats according to a format specifier and returns the resulting str.

## Example

```gad
fmt.sprintf("%s=%d", "n", 42)
>>> "n=42"
```

### sprintln

```gad
sprintln(*any) <str>
```

Formats using the default formats for its operands and returns the
resulting str. Spaces are always added between operands and a newline
is appended.

## Example

```gad
fmt.sprintln("a", 1)
>>> "a 1\n"
```

### sscan

```gad
sscan(str str, *scanArg) <int>
```

Scans the argument str, storing successive space-separated values into
successive scanArg arguments. Newlines count as space. If no error is
encountered, it returns the number of items successfully scanned. If that
is less than the number of arguments, a scan error is thrown.

## Example

```gad
fmt.sscan("7", fmt.scanArg("int"))
>>> 1
```

### sscanf

```gad
sscanf(str str, format str, *scanArg) <int>
```

Scans the argument str, storing successive space-separated values into
successive scanArg arguments as determined by the format. It returns the
number of items successfully parsed, and throws on a scan error.
Newlines in the input must match newlines in the format.

## Example

```gad
// scanArg holders receive the parsed fields; sscanf returns the count.
fmt.sscanf("42 hi", "%d %s", fmt.scanArg("int"), fmt.scanArg("str"))
>>> 2
```

### scanArg

```gad
scanArg(typeName str) <scanArg>
```

Returns a `scanArg` object to scan a value of given type name in scan
functions.
Supported type names are `"str", "int", "uint", "float", "char",
"bool", "bytes"`.
It throws a runtime error if type name is not supported.
Alternatively, `str, int, uint, float, char, bool, bytes` builtin
functions can be provided to get the type name from the BuiltinFunction's
Literal field.

## Example

```gad
// scanArg builds a typed holder passed to sscan/sscanf.
typeName(fmt.scanArg("int"))
>>> "scanArg"
```

## Example — `fmt.gad`

````gad
/**
Formats using the default formats for its operands and writes to standard
output. Spaces are added between operands when neither is a str.
It returns the number of bytes written and any encountered write error
throws a runtime error.

## Example

```gad
// print writes to stdout and returns the byte count.
fmt.print("a", 1)
>>> 2
```
**/
export print(*any) <int> => nil

/**
Formats according to a format specifier and writes to standard output.
It returns the number of bytes written and any encountered write error
throws a runtime error.

## Example

```gad
fmt.printf("%d!", 5)
>>> 2
```
**/
export printf(format str, *any) <int> => nil

/**
Formats using the default formats for its operands and writes to standard
output. Spaces are always added between operands and a newline
is appended. It returns the number of bytes written and any encountered
write error throws a runtime error.

## Example

```gad
fmt.println("hi", 1)   // writes "hi 1\n"
>>> 5
```
**/
export println(*any) <int> => nil

/**
Formats using the default formats for its operands and returns the
resulting str. Spaces are added between operands when neither is a
str.

## Example

```gad
fmt.sprint("x=", 5)
>>> "x=5"
```
**/
export sprint(*any) <str> => nil

/**
Formats according to a format specifier and returns the resulting str.

## Example

```gad
fmt.sprintf("%s=%d", "n", 42)
>>> "n=42"
```
**/
export sprintf(format str, *any) <str> => nil

/**
Formats using the default formats for its operands and returns the
resulting str. Spaces are always added between operands and a newline
is appended.

## Example

```gad
fmt.sprintln("a", 1)
>>> "a 1\n"
```
**/
export sprintln(*any) <str> => nil

/**
Scans the argument str, storing successive space-separated values into
successive scanArg arguments. Newlines count as space. If no error is
encountered, it returns the number of items successfully scanned. If that
is less than the number of arguments, a scan error is thrown.

## Example

```gad
fmt.sscan("7", fmt.scanArg("int"))
>>> 1
```
**/
export sscan(str str, *scanArg) <int> => nil

/**
Scans the argument str, storing successive space-separated values into
successive scanArg arguments as determined by the format. It returns the
number of items successfully parsed, and throws on a scan error.
Newlines in the input must match newlines in the format.

## Example

```gad
// scanArg holders receive the parsed fields; sscanf returns the count.
fmt.sscanf("42 hi", "%d %s", fmt.scanArg("int"), fmt.scanArg("str"))
>>> 2
```
**/
export sscanf(str str, format str, *scanArg) <int> => nil

/**
Returns a `scanArg` object to scan a value of given type name in scan
functions.
Supported type names are `"str", "int", "uint", "float", "char",
"bool", "bytes"`.
It throws a runtime error if type name is not supported.
Alternatively, `str, int, uint, float, char, bool, bytes` builtin
functions can be provided to get the type name from the BuiltinFunction's
Literal field.

## Example

```gad
// scanArg builds a typed holder passed to sscan/sscanf.
typeName(fmt.scanArg("int"))
>>> "scanArg"
```
**/
export scanArg(typeName str) <scanArg> => nil
````
