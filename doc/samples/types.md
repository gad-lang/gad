
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
