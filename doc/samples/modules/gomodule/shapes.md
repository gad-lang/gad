# shapes

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

The area of a circle of radius r, or of a square of side s (by name).

```gad
surface(r number) <number>
```

```gad
surface(; side number) <number>
```

## Example — `shapes.gad`

```gad
const PI = 3.141592653589793

/**
The ratio of a circle's circumference to its diameter.
**/
export PI

/**
A shape, named.
**/
export class Shape {
    name str = "?"
    methods {
        /** what the shape is **/
        describe() => "shape " + this.name
    }
}

/**
The units the areas are given in.
**/
export enum Unit { cm; m }

/**
The area of a rectangle w by h.
**/
export area(w number, h number) <number> => w * h

/**
The area of a circle of radius r, or of a square of side s (by name).
**/
export func surface {
    (r number) <number> => PI * r * r
    (; side number) <number> => side * side
}
```
