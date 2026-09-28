
# Members given at run time (`** EXPR`)

A class body item `** EXPR` adds the members EXPR gives when the class is
declared, after the members it declares — for what is only known then: the
modules a glob import found, the options a plugin brings. A class may have
several; they are applied in order.

EXPR is a **dict** (or a key-value array) with any of:

| Key | Holds |
| --- | --- |
| `fields` | the fields, as a declared class gives them: a name to its default, or to a **spec** |
| `methods` | the methods: an array of named functions, `name(this, …) => …` |
| `props` | the properties: a dict of name → `func { (this) => … }` (the getter, and setters by type) |

A field spec is a key-value array `(; types=[…], nullable=true, meta=(; …),
default=…)`: the types the field accepts (enforced as a declared field's are),
whether it accepts nil, its `[k=v]` metadata and its default.

Fields given as a **dict** come in the order of their names (a dict has no
order of its own); given as a **key-value array** `(; …)`, in its order. The
declared fields always come first.

## Fields

```gad
defaults := {columns: 3, rows: 2}
class Grid {
    name str = "grid"
    ** {fields: defaults + {gap: 8}}
}
g := Grid(; rows = 4)
[g.name, g.columns, g.rows, g.gap, collect(keys(Grid.@fields)).|sort]
// => ["grid", 3, 4, 8, ["columns", "gap", "name", "rows"]]
```

## Typed fields: the spec

```gad
extra := {fields: (;
    title=(; types=[str], nullable=true, meta=(; label="Title")),
    size=(; types=[int], default=10),
)}
class Box {
    ** extra
}
rejects := func(f) { try { f(); return false } catch { return true } }
[
    Box().title,                          // nullable: nil
    Box().size,                           // its default
    Box(; title = "a", size = 2).size,
    rejects(() => Box(; size = "big")),   // field "size" expects int
]
// => [nil, 10, 2, true]
```

## Methods and properties

```gad
class Counter {
    n int = 0
    ** {methods: [twice(this) => this.n * 2]}
    ** {props: {label: func { (this) => "counter " + str(this.n) }}}
}
c := Counter(; n = 21)
[c.twice(), c.label]
// => [42, "counter 21"]
```

## Built from what a glob import found

```gad
// what `import("./plugins/*")::dict` gives: a module by name
plugins := {pt: {lang: "pt"}, en: {lang: "en"}}
langFields := {}
for name, _ in plugins {
    langFields[name] = (; types=[bool], default=false)
}
class Languages {
    ** {fields: langFields}
}
l := Languages(; pt = true)
[l.en, l.pt, collect(keys(Languages.@fields)).|sort]
// => [false, true, ["en", "pt"]]
```

## Nothing to add

A `nil` spread adds nothing; any other value that is not a dict or a key-value
array is an error.

```gad
class Plain {
    a = 1
    ** nil
}
[Plain().a, collect(keys(Plain.@fields))]
// => [1, ["a"]]
```

## Example — `spread_members.gad`

```gad
defaults := {columns: 3, rows: 2}
class Grid {
    name str = "grid"
    ** {fields: defaults + {gap: 8}}
}
g := Grid(; rows = 4)
[g.name, g.columns, g.rows, g.gap, collect(keys(Grid.@fields)).|sort]

extra := {fields: (;
    title=(; types=[str], nullable=true, meta=(; label="Title")),
    size=(; types=[int], default=10),
)}
class Box {
    ** extra
}
rejects := func(f) { try { f(); return false } catch { return true } }
[
    Box().title,                          // nullable: nil
    Box().size,                           // its default
    Box(; title = "a", size = 2).size,
    rejects(() => Box(; size = "big")),   // field "size" expects int
]

class Counter {
    n int = 0
    ** {methods: [twice(this) => this.n * 2]}
    ** {props: {label: func { (this) => "counter " + str(this.n) }}}
}
c := Counter(; n = 21)
[c.twice(), c.label]

// what `import("./plugins/*")::dict` gives: a module by name
plugins := {pt: {lang: "pt"}, en: {lang: "en"}}
langFields := {}
for name, _ in plugins {
    langFields[name] = (; types=[bool], default=false)
}
class Languages {
    ** {fields: langFields}
}
l := Languages(; pt = true)
[l.en, l.pt, collect(keys(Languages.@fields)).|sort]

class Plain {
    a = 1
    ** nil
}
[Plain().a, collect(keys(Plain.@fields))]
```
