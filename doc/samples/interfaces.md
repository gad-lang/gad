
# Interfaces (`interface`)

An `interface { … }` is a structural contract grouping typed fields,
`get`/`set`/`prop` accessors, required methods and context-function checks. Like
[`meti`](method_interfaces.gad) it compiles to a value (`Interface`) whose
members are read by indexing; the statement form binds a const, the expression
form is a value (compiled with an `ifaces#N` name).

## Declaring and reflecting

A bare field entry is typed (`id int`); an untyped field defaults to `any`.
`get`/`set`/`prop` declare accessors; a method is `name(params) <return>`, and
its block form `name { (…), … }` groups several overload signatures (like
`meti`, without the keyword). The interface's members are read back by
indexing (`.name`, `.fields`, `.props`, `.methods`); a nested-interface field's
type is itself an `Interface` value, inspectable further.

```gad
// A parent interface, spread into `Shape` with `*Base` below.
interface Base { get kind }

interface Shape {
	*Base

	id int
	label str

	// a nested-interface field (short form): `bounds` is typed by an inline
	// interface with `w int` and `h int`. Long form: `bounds interface { … }`.
	bounds: {
		w int
		h int
	}

	// an ARRAY-interface field (`[]{ … }`): `corners` must be an array whose
	// elements each satisfy `{ x int; y int }`.
	corners: []{
		x int
		y int
	}

	get area uint
	set scale
	prop title

	draw()
	resize(int|uint) <bool>

	// a method with several overload signatures
	from {
		(str)
		(w int, h int)
	}
}

bounds := [f for f in Shape.fields if f.name == "bounds"][0]
[
    Shape.name,
    [f.name for f in Shape.fields],
    Shape.fields[0].types[0] == int,
    [p.name for p in Shape.props],
    [m.name for m in Shape.methods],
    len(Shape.methods[2].headers),                   // from's signatures
    [typeName(bounds.types[0]), [f.name for f in bounds.types[0].fields]],
]
// => ["Shape", ["id", "label", "bounds", "corners"], true, ["area", "scale", "title"], ["draw", "resize", "from"], 2, ["Interface", ["w", "h"]]]
```

## Structural satisfaction

A value **satisfies** an interface when it has every required field (with an
assignable type), property and method (whose signatures match), plus any extended
interface. Check it with the [`::` operator](user_operators.gad) or use an
interface as a parameter type — a non-satisfier is rejected. Satisfaction works
against any member-bearing value: class instances, dicts/key-value arrays (fields
match keys, methods match callable keys) and reflected Go values (fields matched
structurally, methods optimistically / duck-typed). An interface also works as
a parameter type. Because a failed `::` raises an error, the `or` fallback
operator turns it into a value: `obj::Type or fallback` yields the fallback
when obj does not satisfy Type, without a `try`/`catch`.

```gad
interface Greeter { name str; greet() <str> }

class Person {
    name = ""
    methods { greet() => "hi " + this.name }
}
class Anon { label = "?" }                      // no `name`, no greet()
func welcome(g Greeter) => g.greet() + "!"      // an interface parameter

p := Person(; name = "Ada")
d := {name: "Bo", greet: func() => "hi Bo"}     // a dict satisfies it too
bad := {name: "x"}                              // no callable `greet`
[
    (p::Greeter).greet(),
    Anon()::Greeter or "rejected",
    welcome(p),
    (d::Greeter).greet(), welcome(d),
    bad::Greeter or "is bad",                   // `or` turns the error into a value
]
// => ["hi Ada", "rejected", "hi Ada!", "hi Bo", "hi Bo!", "is bad"]
```

A required field marked with a `?` after its name (`x? int`) is **nullable**: a
member that is `nil` — or absent — still satisfies it, so `?` marks an optional
field (see [typed & nullable fields](class/field_types.gad)).

```gad
interface Tagged { name str; tag? int|str }
[
    ({name: "a", tag: 3} :: Tagged).tag,
    ({name: "b", tag: nil} :: Tagged).name,     // a nil tag is fine
    ({name: "c"} :: Tagged).name,               // an absent tag is fine
]
// => [3, "b", "c"]
```

A `**name` member is a **rest capture** used with the transforming cast
[`:::`](transform_cast_test.gad): `d ::: interface { … }` coerces the source's
class/interface-typed fields into their declared shape and gathers the keys not
named by the interface into a dict bound to `name`.

`[]` written after the interface's name makes it a **array interface**, matching
an **array** of satisfying elements: `interface P [] { … }` a flat array,
`interface P [][][] { … }` an array nested three deep, and `interface P []<int|uint>`
an array of the given element types (see [array interfaces](interface_arrays_test.gad);
for a nominal, constructible array type see [typed arrays](typed_arrays.gad)).

## Nested and array fields (`name: { … }`)

A field whose type is a nested interface has a **short form** `name: { … }`,
equal to `name interface { … }`. The colon distinguishes it from the block-method
form `name { … }` (no colon), and it is ONLY for a nested interface — `name: { … }`
must open a brace body (there is no `name: Type`). Nesting may go any depth, and
the value is checked **recursively** on a `::` cast.

A leading `[]` is a **array interface**: `name: []{ … }` == `name interface [] { … }`,
an array whose elements each satisfy the body (`[][]…` nests deeper). `name?: { … }`
(and `name?: []{ … }`) marks the nested field nullable — nil or absent satisfies
it. The formatter always normalizes a nested-interface field to this short form.

```gad
// `sat` reports whether v satisfies interface T (a caught `::` cast).
sat := func(v, T) {
    try { v :: T; return true } catch { return false }
}

// A `::` cast checks a nested `name: { … }` field RECURSIVELY.
Boxed := interface { bounds: { w int, h int } }

// An array field `name: []{ … }` checks every element of the array.
Poly := interface { pts: []{ x int, y int } }

[
    sat({ bounds: { w: 10, h: 20 } }, Boxed),             // nested: both fields present
    sat({ bounds: { w: 10 } }, Boxed),                    // nested: missing h -> false
    sat({ pts: [{ x: 0, y: 0 }, { x: 1, y: 2 }] }, Poly), // array: every element ok
    sat({ pts: [{ x: 0, y: 0 }, { x: 1 }] }, Poly),       // array: 2nd missing y -> false
]
// => [true, false, true, false]
```

## Context-function members (`funcs { … }`)

A `funcs { FnExpr <header>; … }` section requires, per entry, a **free function
in scope** — `FnExpr`, not a method on the object — to *handle* the interface's
object. Each entry is a function followed by its required header (or the block
form `FnExpr { (…); … }` for several signatures). `FnExpr` is captured by value
where the interface is declared; the special positional type **`@self`** marks
where the object is passed (every header must contain at least one `@self`). Such
an interface is a runtime value and can also be built directly in Go (set
`Interface.ContextFuncs`). A function that does not take the object (no
matching `@self` arity) fails the check.

```gad
render := func(indent int, obj) => "<" + str(indent) + ":" + str(obj.name) + ">"
Renderable := interface {
    name str
    funcs {
        render <(indent int, @self)>      // require render(int, <object>) in scope
    }
}

noObj := func(indent int) => indent     // does not take the object
NotRenderable := interface { funcs { noObj <(indent int, @self)> } }

r := {name: "Ada"}
[(r :: Renderable).name, r :: NotRenderable or "no renderer"]
// => ["Ada", "no renderer"]
```

## Caching (embedding)

Interface-satisfaction results are memoized on the **root VM**, keyed by the
interface and the value's type. Hosts can pre-warm/share the cache with
`gad.NewInterfaceSatCache()` + `(*gad.VM).SetInterfaceSatCache`; the Gadx `Render`
engine does this per compiled template.

## Extending (`*Parent`)

An interface extends others with `*` spreads: a value must satisfy every
parent. A spread takes any expression yielding an interface or an **array of
interfaces** (flattened, nesting allowed) — `*A`, `*[A, B]`, or a variable
`*parents` — evaluated **where the interface is declared**, so a local parent
is resolved in its own scope even when the interface is used from a closure. A
non-interface item is an error at the declaration.

```gad
interface HasName { name str }
interface HasAge  { age int }
interface HasTags { tags []str }

// a literal list of parents
interface Human { *[HasName, HasAge] }

// a list held in a variable (nested arrays flatten), plus an own field
personParts := [HasName, [HasAge, HasTags]]
interface Associate { *personParts; id int }

fitsIface := func(v, T) { try { v :: T; return true } catch { return false } }
[
    fitsIface({name: "ann", age: 30}, Human),
    fitsIface({name: "ann"}, Human),              // no age
    fitsIface({name: "b", age: 1, tags: ["x"], id: 7}, Associate),
    len(Human.@flat.fields),      // name, age
    len(Associate.@flat.fields),  // name, age, tags, id
]
// => [true, false, true, 2, 4]
```

## Flattening (`iface.@flat`)

`iface.@flat` collapses an interface's whole extends graph — both the `*A`
spreads and any runtime parents — into a single interface with no extends of
its own, caching the result. Members of the **same name** are MERGED by
signature rather than rejected: a getter, its setters (by value type) and a
method's overloads combine, and an identical signature seen twice is
deduplicated (so a diamond counts a shared parent once).

A combined property renders compactly: a getter with one setter of the same type
is `prop x T` (`prop x` when untyped); differing types or several setter
overloads use the `prop x { get …; set … }` braces form. Only a genuine
signature CONFLICT is rejected — a name used as two different kinds
(field/property/method), a getter with two different return types, or a method
overload with the same parameters but a different return type.

```gad
interface Reader { get pos int; read() <_ str> }
interface Seeker { *Reader; set pos; seek(n int) }
interface HasRun { run() }
flat := Seeker.@flat
[
    [p.name for p in flat.props], [m.name for m in flat.methods],
    str((interface { get x int; set x int }).@flat.props[0]),          // same type
    str((interface { get x int; set x str; set x }).@flat.props[0]),   // mixed types
    interface { *HasRun; get run int }.@flat or "conflict rejected",   // run(): method vs getter
]
// => [["pos"], ["seek", "read"], "prop x int", "prop x { get int; set str|any }", "conflict rejected"]
```

## Example — `interfaces.gad`

````gad
// A parent interface, spread into `Shape` with `*Base` below.
interface Base { get kind }

interface Shape {
	*Base

	id int
	label str

	// a nested-interface field (short form): `bounds` is typed by an inline
	// interface with `w int` and `h int`. Long form: `bounds interface { … }`.
	bounds: {
		w int
		h int
	}

	// an ARRAY-interface field (`[]{ … }`): `corners` must be an array whose
	// elements each satisfy `{ x int; y int }`.
	corners: []{
		x int
		y int
	}

	get area uint
	set scale
	prop title

	draw()
	resize(int|uint) <bool>

	// a method with several overload signatures
	from {
		(str)
		(w int, h int)
	}
}

bounds := [f for f in Shape.fields if f.name == "bounds"][0]
[
    Shape.name,
    [f.name for f in Shape.fields],
    Shape.fields[0].types[0] == int,
    [p.name for p in Shape.props],
    [m.name for m in Shape.methods],
    len(Shape.methods[2].headers),                   // from's signatures
    [typeName(bounds.types[0]), [f.name for f in bounds.types[0].fields]],
]

// `sat` reports whether v satisfies interface T (a caught `::` cast).
sat := func(v, T) {
    try { v :: T; return true } catch { return false }
}

// A `::` cast checks a nested `name: { … }` field RECURSIVELY.
Boxed := interface { bounds: { w int, h int } }

// An array field `name: []{ … }` checks every element of the array.
Poly := interface { pts: []{ x int, y int } }

[
    sat({ bounds: { w: 10, h: 20 } }, Boxed),             // nested: both fields present
    sat({ bounds: { w: 10 } }, Boxed),                    // nested: missing h -> false
    sat({ pts: [{ x: 0, y: 0 }, { x: 1, y: 2 }] }, Poly), // array: every element ok
    sat({ pts: [{ x: 0, y: 0 }, { x: 1 }] }, Poly),       // array: 2nd missing y -> false
]

/**
## An enum as a field's type

A field's type may be an `enum` declared RIGHT THERE, instead of one declared
beside the interface:

```gad
interface { perm enum { Read, Write } }
```

Inline it is anonymous — a name would have nothing to name but the field it is
already on — and it reads back as the enum it is, values and all. A `?` after
the field name works as it does for any other type.
**/
enum Perm { Read, Write }

Named := interface { perm Perm }                    // declared beside, named
Acl := interface { perm enum { Read, Write } }      // declared inline, anonymous
OptAcl := interface { perm? enum { Read, Write } }  // nullable: nil or absent

inline := Acl.fields[0].types[0]
[
    ({perm: Perm.Read} :: Named).perm == Perm.Read,
    [Acl.fields[0].name, typeName(inline), collect(keys(inline))],
    {} :: OptAcl,
]

/**
## An array of a type (`[]T`)

A field's type may be a **array of a type**: `[]int` is an array whose every
element is an `int`, and each further `[]` nests one array deeper
(`[][][]int`). This is the VALUE form — the element is a plain type — next to
the `name: []{ … }` form above, whose element is an interface body.

### The envelope rule

With **several element types, everything is enveloped** in `<…>` and separated
by `|`; an element satisfies the array type when it matches any of them:

```gad
interface { xs []<int|str> }
```

With a **single element type the envelope is dropped**: `[]int` is just the
short form of `[]<int>`, and that is how one type always reads back. The one
exception is a **function header**, which must be enveloped even alone —
its own syntax `<(x int) <ret any>>` IS the envelope:

```gad
interface { fs []<(x int) <ret any>> }   // an array of callables
interface { fs []<<(x int)>|str> }       // among several types, enveloped like any other
```

An empty array satisfies any array type (there is no element to reject), and
`?` after the field name makes the field itself nullable as usual.

> **Where they are read:** wherever a type is written — an interface field, a
> function parameter (a method header's included, positional or named), a class
> field, a `param` declaration. `[` still opens an index and `<` still is a
> comparison everywhere else: a type is read only where the shape cannot be an
> expression (`xs []int`, since an empty index is not one) or where the header's
> closing `>` is followed by an item separator, so `f(a < (b) > c)` stays the
> comparison it looks like.
**/
Ints := interface { xs []int }
Matrix := interface { xs [][]int }                 // nested: an array OF arrays of int
Mixed := interface { xs []<int|str> }              // several element types, enveloped
[
    {xs: [1, 2, 3]} :: Ints or "rejected",
    {xs: [1, "a"]} :: Ints or "rejected",
    {xs: 1} :: Ints or "rejected",                  // not an array
    {xs: []} :: Ints or "rejected",                 // empty satisfies any array type
    str(interface { xs []<int> }.fields[0].types[0]), // `[]<int>` is `[]int`, written long
    {xs: [[1], [2, 3]]} :: Matrix or "rejected",
    {xs: [1, 2]} :: Matrix or "rejected",
    {xs: [1, "a"]} :: Mixed or "rejected",
    {xs: [1, true]} :: Mixed or "rejected",
]

/**
A **function header is a type** wherever a type goes — a field of one requires a
CALLABLE whose signature the header matches — and it is the single type that
keeps its envelope inside an array type (among several element types it is
enveloped like the others).
**/
Fn := interface { f <(x int) <ret any>> }
Fns := interface { fs []<(x int) <ret any>> }
FnOrStr := interface { fs []<<(x int)>|str> }
fits := func(v, T) => (v :: T or nil) != nil
[
    fits({f: func(x int) => x}, Fn), fits({f: 1}, Fn),
    fits({fs: [func(x int) => x]}, Fns), fits({fs: [1]}, Fns),
    fits({fs: [func(x int) => x, "a"]}, FnOrStr),
]

/**
Both types are read wherever a type is written, not only on an interface field.
**/
func total(xs []int) {                       // a function parameter
    sum := 0
    for _, v in xs { sum += v }
    return sum
}
class Basket { items []str = [] }            // a class field
func apply(cb <(x int)>) => cb(2)            // a function-header parameter
[
    total([1, 2, 3]), total(["a"]) or "rejected",
    Basket(; items = ["pão"]).items, Basket(; items = [1]) or "rejected",
    apply(func(x int) => x * 3),
]

// The anonymous expression form is a value like any other.
Point := interface { x int; y int; get norm float }
println("anon:      ", typeName(Point), len(Point.fields), Point.fields[1].name)

interface Greeter { name str; greet() <str> }

class Person {
    name = ""
    methods { greet() => "hi " + this.name }
}
class Anon { label = "?" }                      // no `name`, no greet()
func welcome(g Greeter) => g.greet() + "!"      // an interface parameter

p := Person(; name = "Ada")
d := {name: "Bo", greet: func() => "hi Bo"}     // a dict satisfies it too
bad := {name: "x"}                              // no callable `greet`
[
    (p::Greeter).greet(),
    Anon()::Greeter or "rejected",
    welcome(p),
    (d::Greeter).greet(), welcome(d),
    bad::Greeter or "is bad",                   // `or` turns the error into a value
]

render := func(indent int, obj) => "<" + str(indent) + ":" + str(obj.name) + ">"
Renderable := interface {
    name str
    funcs {
        render <(indent int, @self)>      // require render(int, <object>) in scope
    }
}

noObj := func(indent int) => indent     // does not take the object
NotRenderable := interface { funcs { noObj <(indent int, @self)> } }

r := {name: "Ada"}
[(r :: Renderable).name, r :: NotRenderable or "no renderer"]

interface Tagged { name str; tag? int|str }
[
    ({name: "a", tag: 3} :: Tagged).tag,
    ({name: "b", tag: nil} :: Tagged).name,     // a nil tag is fine
    ({name: "c"} :: Tagged).name,               // an absent tag is fine
]

interface Reader { get pos int; read() <_ str> }
interface Seeker { *Reader; set pos; seek(n int) }
interface HasRun { run() }
flat := Seeker.@flat
[
    [p.name for p in flat.props], [m.name for m in flat.methods],
    str((interface { get x int; set x int }).@flat.props[0]),          // same type
    str((interface { get x int; set x str; set x }).@flat.props[0]),   // mixed types
    interface { *HasRun; get run int }.@flat or "conflict rejected",   // run(): method vs getter
]

interface HasName { name str }
interface HasAge  { age int }
interface HasTags { tags []str }

// a literal list of parents
interface Human { *[HasName, HasAge] }

// a list held in a variable (nested arrays flatten), plus an own field
personParts := [HasName, [HasAge, HasTags]]
interface Associate { *personParts; id int }

fitsIface := func(v, T) { try { v :: T; return true } catch { return false } }
[
    fitsIface({name: "ann", age: 30}, Human),
    fitsIface({name: "ann"}, Human),              // no age
    fitsIface({name: "b", age: 1, tags: ["x"], id: 7}, Associate),
    len(Human.@flat.fields),      // name, age
    len(Associate.@flat.fields),  // name, age, tags, id
]

return Shape.name
````
