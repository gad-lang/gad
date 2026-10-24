
# `shapes` module

A module written in gad and shipped in Go: `gad gomodule` compiles it into the
package `shapes/` (its `.gadb` embedded, its docs as `gad:doc` comments, its
API stub `shapes/shapes.gad`), and a Go init of the package (`shapes/init.go`)
adds `GoShape`, a class in Go that extends `Shape`:

```sh
cd samples/modules/gomodule
gad gomodule --out shapes shapes.gad     # or: go generate ./shapes
```

A host registers it: `shapes.AddTo(mm)`; a program imports it by name:
`import("shapes")`.

## Public API

### PI

```gad
PI = 3.141592653589793
```

The ratio of a circle's circumference to its diameter.

### Shape

```gad
Shape class
```

A shape, named.

#### Fields

```gad
name str = "?"
```

#### Methods

```gad
describe()
```

what the shape is

### Unit

```gad
Unit enum
```

The units the areas are given in.

#### Variants

```gad
cm
m
```

### area

```gad
area(w number, h number) <number>
```

The area of a rectangle w by h.

### surface

```gad
surface(r number) <number>
```

The area of a circle of radius r, or of a square of side s (by name).

```gad
surface(; side number) <number>
```

### GoShape

```gad
GoShape class
```

A Shape whose label is its name in upper case (a class in Go: the module's
Go init).

#### Parents

```gad
*Shape
```

→ [Shape](#shape)

#### Methods

```gad
label() <str>
```

the name, in upper case

## Example — `shapes.gad`

```gad
/**
The ratio of a circle's circumference to its diameter.
**/
export const PI = 3.141592653589793

/**
A shape, named.
**/
export class Shape {
    name str = "?"
    methods {
        /** what the shape is **/
        describe() => nil
    }
}

/**
The units the areas are given in.
**/
export enum Unit { cm; m }

/**
The area of a rectangle w by h.
**/
export area(w number, h number) <number> => nil

export func surface {
/// The area of a circle of radius r, or of a square of side s (by name).
(r number) <number> => nil
(; side number) <number> => nil
}

/**
A Shape whose label is its name in upper case (a class in Go: the module's
Go init).
**/
export class GoShape {
    *Shape
    methods {
        /** the name, in upper case **/
        label() <str> => nil
    }
}
```
