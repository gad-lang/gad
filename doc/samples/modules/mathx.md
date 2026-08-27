# mathx

mathx.gad — a small source module. Import it with import("./mathx.gad").

A module's body runs once; its exports are cached across imports.
See doc/modules.md for detailed documentation.

## Public API

### pi

### square

```gad
square(x)
```

### cube

```gad
cube(x)
```

## Example — `mathx.gad`

```gad
pi := 3.141592653589793

export pi                       // export an existing binding

export square(x) {
    return x * x
}

export cube(x) => x * x * x

/**
area helpers grouped into a single exported dict (they capture pi).
**/
export {
    circleArea: (r) => pi * r * r,
    rectArea: (w, h) => w * h,
}
```
