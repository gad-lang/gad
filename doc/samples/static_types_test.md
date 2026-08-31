
# Marker types (`type Name { … }`)

A marker type declared with `type Name { … }` is a first-class type **with no
instances**. Its fields, properties and methods live on the type value itself —
inside a method or accessor `this` is the type — and it may declare a `call(…)`
factory, invoked as `Name(…)`, whose result is arbitrary (a factory, not an
instance). It reuses the class body grammar (fields, `props { … }`,
`methods { … }`), swapping `new` for `call`.

Because it has no instances, `x :: Name` and using `Name` as an instance
parameter type are rejected — to dispatch on the type value, use
[`type<Name>`](meta_types_test.gad). There is also an anonymous expression form,
`const Name = type { … }`.

Run it: `gad test samples/static_types_test.gad`

## Example — `static_types_test.gad`

```gad
/**
Static fields and methods: a field is read as `Name.field`; a method runs with
`this` bound to the type, so it can read the type's own fields.
**/
test "static fields and methods" {
    type Palette {
        primary = "#005"
        methods { label() => "primary is " + this.primary }
    }

    t.equal("#005", Palette.primary)
    t.equal("primary is #005", Palette.label())
}

/**
A `call(…)` factory makes the type callable. It returns whatever it likes — never
an instance of the type — and takes overloads like a constructor.
**/
test "the call factory" {
    type Id {
        call(n int) => "id:" + str(n)
        call(s str) => "id:" + s
    }

    t.equal("id:7", Id(7))
    t.equal("id:x", Id("x"))
}

/**
Properties work too: an accessor runs with `this` bound to the type.
**/
test "static property" {
    type Config {
        version = 3
        props { label => "v" + str(this.version) }
    }

    t.equal("v3", Config.label)
}

/**
The anonymous expression form binds a marker type to a name (or any variable).
**/
test "expression form" {
    const Dir = type { up = 1; down = 2 }

    t.equal(1, Dir.up)
    t.equal(2, Dir.down)
}

/**
A marker type dispatches through `type<Name>`, since it is a first-class type
value.
**/
test "dispatch via type<Name>" {
    type Meter {}
    type Gram {}

    func unit {
        (t type<Meter>) => "m"
        (t type<Gram>)  => "g"
    }

    t.equal("m", unit(Meter))
    t.equal("g", unit(Gram))
}

/**
Because a marker type has no instances, the checked cast `x :: Name` is rejected —
use `type<Name>` to match the type value instead.
**/
test "no instances: :: is rejected" {
    type Tag {}

    ok := false
    try {
        5 :: Tag
    } catch {
        ok = true
    }
    t.equal(true, ok)
}
```
