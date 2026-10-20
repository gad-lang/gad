
# Built-in Types

Gad's built-in **object types** documented below are available in every script
without an `import`. Unlike the type constructors in the builtins page, these
are structural values produced by language constructs (a func-header value, an
`interface`/`meti` declaration, a property), read with indexing.

## Type FuncHeaderObject
FuncHeaderObject describes a function signature: a name, positional and named
parameters (each a `typedIdent`) and a return-type list. It is the value of a
func-header expression `<(params) <return>>`.

Members are read with indexing:
  - `h.name` -> str
  - `h.params` -> array of typedIdent
  - `h.namedParams` -> array of typedIdent
  - `h.return` -> array of typedIdent

```gad ignore
h := <(a int, b str) <r bool>>
h.name           // ""
len(h.params)    // 2
h.params[0].name // "a"
```

## Type Interface
Interface is the value of an `interface { … }` declaration: a structural
contract of typed fields, getter/setter properties, required methods and an
optional `parse { … }` group of signatures. It is compiled to a bytecode
constant; parameter/field types are stored as symbols and resolved per-VM.

Members are read with indexing:
  - `i.name`    -> str
  - `i.fields`  -> array of InterfaceField
  - `i.props`   -> array of InterfaceProp
  - `i.methods` -> array of InterfaceMethod

## Type MethodInterface
MethodInterface is a set of required function headers, produced by a `meti`
expression: `meti { (), (v) <int> }`. Use `implements(fn, mi)` to test whether
a callable provides every header, and `mi + mi2` (or `mi ++ [mi2, …]`) to merge.

```gad ignore
Stringer := meti { () <str> }
implements((this) => "x", Stringer)   // true if the func matches a header
```

## Type Prop
Prop is a named, callable value backed by getter and setter methods.

    Prop(name str, *methods) -> Prop

The trailing methods are dispatched by their signature when the property is
called:
  - **getter**: takes no parameters and returns a value (`prop() -> value`).
    At most one getter may be registered.
  - **setter**: takes one parameter and returns nothing (`prop(v)`). Any
    number of setters may be registered; the one whose parameter type matches
    the argument is selected.

A property may be created with no methods, but calling such a property is an
error because no matching method exists. New methods can be attached later
with the `met` statement.

Example — getter/setter pair plus a typed setter:

```gad ignore
var value
const p = Prop("x", () => value, (v) => {value = v})
met p(v int) {
  value = "int value= " + v
}
p()      // nil
p("a")   // setter: value = "a"
p()      // "a"
p(1)     // typed setter selected: value = "int value= 1"
p()      // "int value= 1"
```

Example — read-only (getter-only) property:

```gad ignore
const pi = Prop("pi", () => 3.14)
pi()        // 3.14
```

## Type ArrayType
ArrayType is an array type written where a type goes — `[]int`, `[][2]str`,
`[2..3]<int|str>` — or as a value, `type […]T`. A value satisfies it by being
an array nested to its depth, each level of its size, whose leaves are of
its element types. A level's size is `[N]` (exactly), `[..B]` (at most),
`[A..]` (at least) or `[A..B]`; see the sized arrays sample.

Members are read with indexing:
  - `t.@depth` -> int, the number of levels
  - `t.@elem`  -> array of the element types
  - `t.@size`  -> array of ArraySize, a level each

```gad ignore
const t = type [2..3][2]str
t.@depth          // 2
t.@elem[0] == str // true
t.@size[0]        // [nil, 2, 3]
```

An array type is a constructor: from its items, checked as `::` checks
them (`(type [2]str)("a", "b")`), or by its dimensions — `size=[…]` for the
levels with no exact size, `fill` each leaf, a value or a function of the
indexes of the leaf. The value is a plain array. Its type is `arrayType`,
whose constructor builds one at run time: `arrayType(elem; depth, size)`,
`size` a level each (an int, or the `[eq, min, max]` of `@size`). Two array
types of the same depth, sizes and element types are `==`. `type []any` is
the builtin `array` itself (`array === type []any`).

```gad ignore
(type [2][3]int)(; fill=0)               // [[0, 0, 0], [0, 0, 0]]
(type [][]int)(; size=[2, 2], fill=(i, j) => i + j)  // [[0, 1], [1, 2]]
arrayType(int; size=[2]) == type [2]int  // true
typeof([1, 2]) === type []any            // true
```

## Type ArraySize
ArraySize is the size of a level of an array type (`t.@size[i]`): an
`[eq, min, max]`, each an int or nil where unset. It equals the array of the
three, and indexes as one.

Members are read with indexing:
  - `s.eq`  -> int|nil, exactly that many items
  - `s.min` -> int|nil, at least
  - `s.max` -> int|nil, at most

```gad ignore
(type [2]int).@size[0].eq        // 2
(type [..3]int).@size[0] == [nil, nil, 3]
```

## Type DictType
DictType is the generic of the dict types: `DictType[K, V]` the type of a dict
whose keys are of K (a type of strings: str, an enum, a union of them) and
whose values are of V; `DictType[V]` is `DictType[str, V]`, and `dict` is
`DictType[str, any]` itself. It is structural (a plain dict of those keys and
values is of it) and a constructor (named args, `[k=v]`, `[key, value]`
pairs, a dict or a key-value array as entries; one array alone its indexes as
keys), checking each entry; `::>` converts. `DictType(K, V)` makes it from
types held in variables. `type NAME dict[K, V]` declares a nominal typed dict
type, its writes checked (see the typed dicts sample).

Members are read with indexing: `t.@key`, `t.@value`, `t.@name`.

```gad
D := DictType[str, int]
[D(; a=1), D.@value == int, dict === DictType[str, any]]
= [{a: 1}, true, true]
```

## Type KeyValueArrayType
KeyValueArrayType is the generic of the key-value array types:
`KeyValueArrayType[K, V]` the type of a key-value array whose keys are of K
(any type) and whose values are of V, the pairs in order, duplicates kept, a
reset (`key=no`) never checked; `KeyValueArrayType[V]` is
`KeyValueArrayType[any, V]`, and `keyValueArray` is
`KeyValueArrayType[any, any]` itself. Structural and a constructor, as
DictType is; `type NAME keyValueArray[K, V]` declares a nominal one.

```gad
K := KeyValueArrayType[str, int]
[K(; a=1, a=2), keyValueArray === KeyValueArrayType[any, any]]
= [(;a=1, a=2), true]
```
