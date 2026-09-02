# exports

exports.gad — the module `export` forms (and a module param).

`export` accepts a declaration directly. Each form behaves like writing the
declaration and then exporting its name (`export class User { … }` ≡
`class User { … }; export User`): it binds a real module-local, usable by the
rest of the module, and exports only that name.

    export class Name { … }        // a class
    export mixin Name { … }        // a mixin
    export interface Name { … }     // an interface
    export type Name { … }         // a marker/static type
    export type Name <A|B>          // a type union (a const)
    export func name(…) { … }       // a function (also `export name(…) { … }`)
    export NAME = EXPR              // a read-only value (a const)

Properties are exported with `export prop`:

    export prop name { … }         // a managed field behind a getter/setter
    export prop name = init         // a live read/write binding over a module var

`param (…)` declares import-time module parameters, passed as named import args
(`import("./exports.gad"; lang="pt")`). A module runs once and is then cached, so
the params apply on its first import.

A doc comment before the `export` documents the exported symbol (its signature,
members or value) as the module's public API.

Imported by main.gad.

## Public API

### Point

```gad
Point class
```

A 2D point.

### Number

```gad
Number type <int|float>
```

Numbers accepted by scale: an int or a float.

### scale

```gad
scale(p, n Number)
```

Scale a point's coordinates by n (a Number).

### origin

```gad
origin = Point()
```

The origin point.

### greet

```gad
greet(name)
```

Greet name in the module's configured language.

### language

```gad
language = lang
```

The language this module was imported with.

### count

```gad
count = prop count {() => value; (n) { value = ((n < 0) ? 0 : n) }; }
```

A managed counter: reads return the current value; writes clamp at 0. Member
access on the imported module (`m.count` / `m.count = n`) delegates to the
getter/setter.

### total

```gad
total = prop total {() => total; ($value) { total = $value }; }
```

A live binding: `total` is a real module var; both the module and importers
read/write the same value.

### inc

```gad
inc()
```

A plain exported function that bumps both, showing the shared state.

## Example — `exports.gad`

```gad
param (;lang="en")

// --- Declaration exports ---------------------------------------------------

/// A 2D point.
export class Point {
    x = 0
    y = 0
    methods {
        /// Manhattan length from the origin.
        len() => this.x + this.y
    }
}

/// Numbers accepted by scale: an int or a float.
export type Number <int|float>

/// Scale a point's coordinates by n (a Number).
export func scale(p, n Number) {
    // `Point` and `Number` are real module locals, usable here.
    return Point(; x: p.x * n, y: p.y * n)
}

/// The origin point.
export origin = Point()

// --- A parameterised greeting ----------------------------------------------

hello := match lang {
    "pt": "Olá"
    "es": "Hola"
    else: "Hello"
}

/// Greet name in the module's configured language.
export greet(name) => #"{hello}, {name}!"

/// The language this module was imported with.
export language = lang

// --- Property exports ------------------------------------------------------

var value = 0

/**
A managed counter: reads return the current value; writes clamp at 0. Member
access on the imported module (`m.count` / `m.count = n`) delegates to the
getter/setter.
**/
export prop count {
    () => value                              // getter
    (n) { value = n < 0 ? 0 : n }            // setter (never goes negative)
}

/**
A live binding: `total` is a real module var; both the module and importers
read/write the same value.
**/
export prop total = 0

/**
A plain exported function that bumps both, showing the shared state.
**/
export inc() {
    value = value + 1
    total = total + 1
}
```
