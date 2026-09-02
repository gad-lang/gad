# exports

exports.gad — declaration exports.

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

A doc comment before the `export` documents the exported symbol (its signature,
members or value) as the module's public API.

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

## Example — `exports.gad`

```gad
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
```
