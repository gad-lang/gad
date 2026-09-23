
# Typed arrays (`type NAME []…`)

`type NAME []ELEM` declares a **typed array type**: a named, first-class type
whose values are arrays bound to it — every item is checked against the element
type.

- `type numerics []<int|uint|float|decimal>` — items of a type union
  (`[]int` needs no envelope for one type).
- `type users []{ name; id }` — items satisfying an inline interface; the long
  form is `type users [] interface { name; id }`.
- `type grid [][]int` — deeper nesting: each item is an array of `int`.

Calling the type builds a value — each positional argument is an item
(`numerics(1, 2.5)`; spread an array with `numerics(*arr)`). A typed array
**behaves as an array**: int indexes, slicing, iteration, `len`, `sort`, `in`,
`copy`, `+`, `++`, `+=` and `delete` — but every write is checked. Its
`typeName` is the declared name.

```gad
type numerics []<int|uint|float|decimal>   // items of a type union
type users []{ name; id }                  // items satisfying an interface
type grid [][]int                          // arrays of int

n := numerics(1, 2.5, 3u)
u := users({name: "ana", id: 1})
g := grid([1, 2], [3])
[typeName(n), str(n), u[0].name, str(g)]
// => ["numerics", "numerics[1, 2.5, 3]", "ana", "grid[[1, 2], [3]]"]
```

```gad
type ints []int

xs := ints(3, 1, 2)
xs[0] = 30          // index set (checked)
delete xs[1]        // delete an item
xs += 5             // append (checked)
more := xs ++ [6]   // a new typed array with 6 appended
sum := 0
for v in xs { sum += v }

// every write is checked: a wrong item throws
fails := func(f) { try { f(); return false } catch { return true } }
rejected := [
    fails(func() { xs[0] = "x" }),
    fails(func() { xs += "x" }),
    fails(func() { ints(1, "y") }),
]
[str(xs), str(xs[1:]), str(more), sum, 5 in xs, str(sort(ints(3, 1, 2))), rejected]
// => ["ints[30, 2, 5]", "ints[2, 5]", "ints[30, 2, 5, 6]", 37, true, "ints[1, 2, 3]", [true, true, true]]
```

## Casts

A typed array type is **nominal**, like a class: `v :: numerics` and a
`numerics` parameter accept only a `numerics` value. The transforming cast `:::`
converts, checking every item, in both directions:

- `array ::: numerics` — an array (or another typed array, or a
  [slice interface](interface_arrays_test.gad) value) into a typed array;
- `typed ::: array` — back to a plain array (a copy of the items);
- `typed ::: iface` — into a slice interface value.

```gad
type reals []<int|float>
interface realsIface []<int|float>

sat := func(v, T) { try { v :: T; return true } catch { return false } }
r := [1, 2.5] ::: reals         // array -> typed array (items checked)
plain := r ::: array            // typed array -> array
asIface := r ::: realsIface     // typed array -> slice interface value
back := asIface ::: reals       // and back

avg := func(xs reals) => (xs[0] + xs[1]) / 2
[typeName(r), typeName(plain), typeName(asIface), typeName(back),
 sat([1, 2.5], reals), sat(r, reals), avg(r)]
// => ["reals", "array", "array", "reals", false, true, 1.75]
```

## Members

An optional class-like body after the element declares **fields** (with
defaults, types and metadata), **properties**, **methods** and **constructor
overloads** (`new`) — member registration is optional. On a value an int index
reaches the items (`ta[0]`); any other key reaches the members: a property (its
getter/setter get the value as `this`), a field, or a method bound to the value.
Named arguments set the fields: `numerics(1, 2; label="x")`. A `new(…)`
overload receives the initiator `new` first and builds the value with
`new(*items; field=…)`.

```gad
type measures []<int|float> {
    /// what is measured
    [db=(;column="lbl")]
    label = "none"
    /// a multiplier
    scale int = 1

    /// `n` zero items
    new(n int) {
        items := []
        for i := 0; i < n; i++ { items += 0 }
        return new(*items; label="zeros")
    }

    props {
        /// the scaled sum
        [computed]
        total => this.sum() * this.scale
    }

    methods {
        /// sums the items
        [route="/sum"]
        sum() {
            s := 0
            for v in this { s += v }
            return s
        }
        /// appends an item, returning the value
        add(v int|float) {
            this += v
            return this
        }
    }
}

m := measures(1, 2.5; scale=2)   // items + a named field value
m.label = "cm"                   // a field write (checked against its type)
m.add(3)
z := measures(2)                 // the `new(n int)` overload
[str(m), m.total, m[1], z.label, len(z)]
// => ["measures[1, 2.5, 3]{label: \"cm\", scale: 2}", 13, 2.5, "zeros", 2]
```

## Reflection and metadata

A typed array type — and each of its members — carries [metadata](metadata.gad)
(`[k=v, …]` before the declaration or member) and answers reflection keys:

| Key | Value |
|-----|-------|
| `T.@name` | the declared name |
| `T.@meta` | the declaration's metadata (a key-value array) |
| `T.@elem` / `T.@depth` | the element type / the number of `[]` |
| `T.@fields`, `T.@props`, `T.@methods` | the body members, by name |
| `T.@new` | the constructor overloads (nil without any) |
| `T.NAME` | a body member, e.g. `T.total.@meta` |

```gad
/// sensor readings
[unit="celsius", version=2]
type readings []float {
    [db=(;column="src")]
    source = ""
    methods {
        [route="/max"]
        max() {
            best := this[0]
            for v in this { if v > best { best = v } }
            return best
        }
    }
}

[readings.@name, str(readings.@meta), readings.@depth, readings.@elem == float,
 str(readings.source.@meta), str(readings.max.@meta), readings(20.5, 22.0).max()]
// => ["readings", "(;unit=\"celsius\", version=2)", 1, true, "(;db=(;column=\"src\"))", "(;route=\"/max\")", 22]
```

## Disabling the item check

The item check is on by default. An embedding Go host can trust its data and
skip it on a hot path with the run flag `gad.RunFlagSkipTypedArrayItemCheck`
(`vm.RunOpts(&gad.RunOpts{Flags: gad.RunFlagSkipTypedArrayItemCheck})`), the
same way `RunFlagSkipReceiverTypeCheck` skips the class `this` check.

## Example — `typed_arrays.gad`

```gad
type numerics []<int|uint|float|decimal>   // items of a type union
type users []{ name; id }                  // items satisfying an interface
type grid [][]int                          // arrays of int

n := numerics(1, 2.5, 3u)
u := users({name: "ana", id: 1})
g := grid([1, 2], [3])
[typeName(n), str(n), u[0].name, str(g)]

type ints []int

xs := ints(3, 1, 2)
xs[0] = 30          // index set (checked)
delete xs[1]        // delete an item
xs += 5             // append (checked)
more := xs ++ [6]   // a new typed array with 6 appended
sum := 0
for v in xs { sum += v }

// every write is checked: a wrong item throws
fails := func(f) { try { f(); return false } catch { return true } }
rejected := [
    fails(func() { xs[0] = "x" }),
    fails(func() { xs += "x" }),
    fails(func() { ints(1, "y") }),
]
[str(xs), str(xs[1:]), str(more), sum, 5 in xs, str(sort(ints(3, 1, 2))), rejected]

type reals []<int|float>
interface realsIface []<int|float>

sat := func(v, T) { try { v :: T; return true } catch { return false } }
r := [1, 2.5] ::: reals         // array -> typed array (items checked)
plain := r ::: array            // typed array -> array
asIface := r ::: realsIface     // typed array -> slice interface value
back := asIface ::: reals       // and back

avg := func(xs reals) => (xs[0] + xs[1]) / 2
[typeName(r), typeName(plain), typeName(asIface), typeName(back),
 sat([1, 2.5], reals), sat(r, reals), avg(r)]

type measures []<int|float> {
    /// what is measured
    [db=(;column="lbl")]
    label = "none"
    /// a multiplier
    scale int = 1

    /// `n` zero items
    new(n int) {
        items := []
        for i := 0; i < n; i++ { items += 0 }
        return new(*items; label="zeros")
    }

    props {
        /// the scaled sum
        [computed]
        total => this.sum() * this.scale
    }

    methods {
        /// sums the items
        [route="/sum"]
        sum() {
            s := 0
            for v in this { s += v }
            return s
        }
        /// appends an item, returning the value
        add(v int|float) {
            this += v
            return this
        }
    }
}

m := measures(1, 2.5; scale=2)   // items + a named field value
m.label = "cm"                   // a field write (checked against its type)
m.add(3)
z := measures(2)                 // the `new(n int)` overload
[str(m), m.total, m[1], z.label, len(z)]

/// sensor readings
[unit="celsius", version=2]
type readings []float {
    [db=(;column="src")]
    source = ""
    methods {
        [route="/max"]
        max() {
            best := this[0]
            for v in this { if v > best { best = v } }
            return best
        }
    }
}

[readings.@name, str(readings.@meta), readings.@depth, readings.@elem == float,
 str(readings.source.@meta), str(readings.max.@meta), readings(20.5, 22.0).max()]
```
