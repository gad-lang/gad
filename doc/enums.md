# Enums

[← Back to index](README.md)

An `enum` is an ordered set of named integer constants, computed at compile
time. It is indexable by member name and iterable in declaration order, and each
member carries its name, value and index.

## Defining an enum

```go
// statement form: `enum Name { … }` defines a constant Name
enum Perm {
    Read        // 1
    Write       // 2
    Exec = 10   // explicit; later fields resume from here
    Delete      // 11
}

// expression form: an anonymous enum value
Color := enum { Red, Green, Blue }   // 1, 2, 3

// export form
export enum Perm { Read, Write }
```

Fields are separated by newlines or commas. The enum and its fields accept doc
comments (`///`, `/** … **/`).

## Values

A field without an explicit `= value` takes the previous magnitude **+ 1** (or
**1** for the first field). An explicit value may be an `int` or `uint` literal,
and it may reference earlier fields with integer operators:

```go
enum Perm {
    Read
    Write
    All = Read | Write    // 3
}
```

Whether a member is `int` or `uint` propagates left to right: the default is
`uint`; an explicit value's type carries to later defaulted fields.

### Signs

A `+` or `-` prefix makes a field a signed `int` and sets a **running sign** that
propagates to later defaulted fields; `+` flips it back to positive:

```go
enum Signed {
    -Low      // -1
    Lower     // -2  (sign propagates)
    +High     //  3  (flipped positive)
    Higher    //  4
}
```

### Bit flags

`bit` activates bitwise mode for that field and the ones after it: each defaulted
field is `1 << n`.

```go
enum Flags {
    bit List    // 1 << 0 = 1
    Detail      // 1 << 1 = 2
    Create      // 1 << 2 = 4
    Read = List | Detail   // 3
}
```

### The `_` placeholder

A field named `_` advances the running value but is **not** added to the enum.
Use it to skip values:

```go
enum E { _, Read, Write }        // Read = 2, Write = 3
enum E { Read, _ = 6, Write }    // Read = 1, Write = 7
```

## Using an enum

```go
enum Perm { Read, Write, Exec = 10 }

Perm.Exec.value     // 10   — the underlying int/uint
Perm.Exec.name      // "Exec"
Perm.Exec.index     // 2    — declaration order
Perm.Exec.enum      // the Perm enum

Perm["Write"]       // index by name (errors if unknown)

for name, member in Perm {     // iterate in declaration order
    println(name, member.value)
}
```

See [`samples/20_enum.gad`](../samples/20_enum.gad) for a runnable tour.
