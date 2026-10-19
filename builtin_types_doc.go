// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

// This file carries the `gad:doc` documentation for the built-in **object
// types** that are available globally (without an `import`) and are produced by
// language constructs rather than by a module — a func-header value
// `<(…)>`, an `interface { … }`, a `meti { … }`, a `Prop(…)`, and so on.
//
// They are grouped under their own `# types module` header so the generator
// emits them to `samples/types.gad` instead of leaking into whichever stdlib
// module happens to be scanned last. Each type's Go doc comment still lives with
// its declaration; the blocks here are the user-facing reference.
//
// `gaddoc api . samples/types.gad types` renders this as `samples/types.gad`.

// gad:doc
// # types module
//
// Gad's built-in **object types** documented below are available in every script
// without an `import`. Unlike the type constructors in the builtins page, these
// are structural values produced by language constructs (a func-header value, an
// `interface`/`meti` declaration, a property), read with indexing.

// gad:doc
// ## Type FuncHeaderObject
// FuncHeaderObject describes a function signature: a name, positional and named
// parameters (each a `typedIdent`) and a return-type list. It is the value of a
// func-header expression `<(params) <return>>`.
//
// Members are read with indexing:
//   - `h.name` -> str
//   - `h.params` -> array of typedIdent
//   - `h.namedParams` -> array of typedIdent
//   - `h.return` -> array of typedIdent
//
// ```gad
// h := <(a int, b str) <r bool>>
// h.name           // ""
// len(h.params)    // 2
// h.params[0].name // "a"
// ```

// gad:doc
// ## Type Interface
// Interface is the value of an `interface { … }` declaration: a structural
// contract of typed fields, getter/setter properties, required methods and an
// optional `parse { … }` group of signatures. It is compiled to a bytecode
// constant; parameter/field types are stored as symbols and resolved per-VM.
//
// Members are read with indexing:
//   - `i.name`    -> str
//   - `i.fields`  -> array of InterfaceField
//   - `i.props`   -> array of InterfaceProp
//   - `i.methods` -> array of InterfaceMethod

// gad:doc
// ## Type MethodInterface
// MethodInterface is a set of required function headers, produced by a `meti`
// expression: `meti { (), (v) <int> }`. Use `implements(fn, mi)` to test whether
// a callable provides every header, and `mi + mi2` (or `mi ++ [mi2, …]`) to merge.
//
// ```gad
// Stringer := meti { () <str> }
// implements((this) => "x", Stringer)   // true if the func matches a header
// ```

// gad:doc
// ## Type Prop
// Prop is a named, callable value backed by getter and setter methods.
//
//	Prop(name str, *methods) -> Prop
//
// The trailing methods are dispatched by their signature when the property is
// called:
//   - **getter**: takes no parameters and returns a value (`prop() -> value`).
//     At most one getter may be registered.
//   - **setter**: takes one parameter and returns nothing (`prop(v)`). Any
//     number of setters may be registered; the one whose parameter type matches
//     the argument is selected.
//
// A property may be created with no methods, but calling such a property is an
// error because no matching method exists. New methods can be attached later
// with the `met` statement.
//
// Example — getter/setter pair plus a typed setter:
//
// ```gad
// var value
// const p = Prop("x", () => value, (v) => {value = v})
// met p(v int) {
//   value = "int value= " + v
// }
// p()      // nil
// p("a")   // setter: value = "a"
// p()      // "a"
// p(1)     // typed setter selected: value = "int value= 1"
// p()      // "int value= 1"
// ```
//
// Example — read-only (getter-only) property:
//
// ```gad
// const pi = Prop("pi", () => 3.14)
// pi()        // 3.14
// ```

// gad:doc
// ## Type ArrayType
// ArrayType is an array type written where a type goes — `[]int`, `[][2]str`,
// `[2..3]<int|str>` — or as a value, `type […]T`. A value satisfies it by being
// an array nested to its depth, each level of its size, whose leaves are of
// its element types. A level's size is `[N]` (exactly), `[..B]` (at most),
// `[A..]` (at least) or `[A..B]`; see the sized arrays sample.
//
// Members are read with indexing:
//   - `t.@depth` -> int, the number of levels
//   - `t.@elem`  -> array of the element types
//   - `t.@size`  -> array of ArraySize, a level each
//
// ```gad ignore
// const t = type [2..3][2]str
// t.@depth          // 2
// t.@elem[0] == str // true
// t.@size[0]        // [nil, 2, 3]
// ```

// gad:doc
// ## Type ArraySize
// ArraySize is the size of a level of an array type (`t.@size[i]`): an
// `[eq, min, max]`, each an int or nil where unset. It equals the array of the
// three, and indexes as one.
//
// Members are read with indexing:
//   - `s.eq`  -> int|nil, exactly that many items
//   - `s.min` -> int|nil, at least
//   - `s.max` -> int|nil, at most
//
// ```gad ignore
// (type [2]int).@size[0].eq        // 2
// (type [..3]int).@size[0] == [nil, nil, 3]
// ```
