
# The class value — introspection & member access

A `class` value (the thing `class Name { … }` binds, or the `Class(...)` builtin
returns) is a first-class object you can inspect at runtime. Access is by
**attribute** — `MyClass.<NAME>` — or, equivalently, by **item** —
`MyClass[<NAME>]` (the index / "get item" form), which is what you use when the
name is dynamic (a variable or computed string).

## Introspection attributes (`@`-prefixed)

Each returns a live view of the class:

- `MyClass.@name`    — the class name (a `str`).
- `MyClass.@fields`  — a dict of `name -> classField`: its own fields (with
  its mixins' and its spreads'), not its parents'.
- `MyClass.@flat`    — every field, its parents' too, flat: a key-value array
  of `name -> classField` in the order the body declares them — a parent's
  (`*Parent`), a mixin's (`use M`), a spread's (`**E`) where it is written;
  a name once, the nearest class's.
- `MyClass.@props`   — a dict of `name -> classProperty`.
- `MyClass.@methods` — a dict of `name -> classMethod`.
- `MyClass.@parents` — the parent classes (from `*Parent` spreads / `extends`).
- `MyClass.@module`  — the module the class was defined in.

## Members by name

A bare name resolves the property or method itself: `MyClass.area` /
`MyClass["scaled"]` returns the `classProperty` / `classMethod` object (not a
value — it is unbound; call an instance's `inst.area` to evaluate it). An unknown
name is an index error.

## `.name` vs `[name]` (attribute vs item)

`MyClass.foo` and `MyClass["foo"]` are the same lookup (the class's `IndexGet`);
the bracket form just lets the key be an expression — `MyClass[dynamicName]`.

The classes the examples use:

```gad
/**
Shape is a small base class, spread into Circle below.
**/
class Shape {
    /**
    what kind of shape
    **/
    kind str = "?"
}

/**
Circle has a typed field, a property and a method — enough to show every
introspection attribute.
**/
class Circle {
    *Shape
    /**
    the radius
    **/
    r int
    props {
        /**
        the area (a getter)
        **/
        area => 3 * this.r * this.r
    }
    methods {
        /**
        the radius scaled by k
        **/
        scaled(k) => this.r * k
    }
}
```

### Introspection

Each `@`-attribute is a dict/array; iterate `k, v` to read the member names
(`kind` lives on the parent, so it is not among Circle's own fields).

```gad
[
    Circle.@name,
    [k for k, v in Circle.@fields],
    [k for k, v in Circle.@flat],
    [k for k, v in Circle.@props],
    [k for k, v in Circle.@methods],
    len(Circle.@parents),
    typeName(Circle.@module),
]
= ["Circle", ["r"], ["kind", "r"], ["area"], ["scaled"], 1, "Module"]
```

### Members by name, attribute vs item

```gad
attr := "@name"
c := Circle(; r = 2, kind = "circle")   // bound values live on an instance
[
    typeName(Circle.area),               // the unbound property object
    typeName(Circle["scaled"]),          // the unbound method object
    Circle[attr],                        // item form: a dynamic key
    Circle.@name == Circle["@name"],     // the same lookup
    c.area, c.scaled(5),
]
= ["ClassProperty", "ClassMethod", "Circle", true, 12, 10]
```

## Example — `class_type.gad`

```gad
example class_type_defs {
    /**
    Shape is a small base class, spread into Circle below.
    **/
    class Shape {
        /**
        what kind of shape
        **/
        kind str = "?"
    }

    /**
    Circle has a typed field, a property and a method — enough to show every
    introspection attribute.
    **/
    class Circle {
        *Shape
        /**
        the radius
        **/
        r int
        props {
            /**
            the area (a getter)
            **/
            area => 3 * this.r * this.r
        }
        methods {
            /**
            the radius scaled by k
            **/
            scaled(k) => this.r * k
        }
    }

}

example class_type_introspect {
    [
        Circle.@name,
        [k for k, v in Circle.@fields],
        [k for k, v in Circle.@flat],
        [k for k, v in Circle.@props],
        [k for k, v in Circle.@methods],
        len(Circle.@parents),
        typeName(Circle.@module),
    ]
    = ["Circle", ["r"], ["kind", "r"], ["area"], ["scaled"], 1, "Module"]
}

example class_type_members {
    attr := "@name"
    c := Circle(; r = 2, kind = "circle")   // bound values live on an instance
    [
        typeName(Circle.area),               // the unbound property object
        typeName(Circle["scaled"]),          // the unbound method object
        Circle[attr],                        // item form: a dynamic key
        Circle.@name == Circle["@name"],     // the same lookup
        c.area, c.scaled(5),
    ]
    = ["ClassProperty", "ClassMethod", "Circle", true, 12, 10]
}

return Circle.@name
```
