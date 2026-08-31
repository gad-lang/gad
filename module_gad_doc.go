// Copyright (c) 2020-2023 Ozan Hacıbekiroğlu.
// Use of this source code is governed by a MIT License
// that can be found in the LICENSE file.

package gad

// This file carries the `gad:doc` documentation for the `gad` namespace (the
// language's own reflective/meta API, available in every script without an
// `import`). `gaddoc api . samples/stdlib/gad.gad gad` renders it as the
// documented Gad API stub `samples/stdlib/gad.gad`, and `gaddoc . doc/stdlib/gad.md
// gad` renders the same source as Markdown.
//
// The member set is defined by GadModule() in module_gad.go; keep this doc in
// sync when adding or removing a member. Examples inside the `## Functions`
// section use a plain `Example:` lead-in (not a `## Example` sub-heading), because
// a level-2 heading would close the function section early.

// gad:doc
// # gad module
//
// The `gad` namespace exposes the language's own reflective and meta API — parse
// and evaluate Gad source at runtime, quote/unquote string literals, build a fast
// repeatable call, drive the `with` statement's hooks, and reach the operator
// dispatch that user operators extend. It is a builtin namespace: every member is
// available as `gad.<name>` with no `import`.
//
// ## Functions
//
// parse(script str; type, name) <sourceFile>
// Parses Gad `script` and returns its parsed source object (an AST), without
// running it. `type` selects the dialect (see `SourceType`); `name` sets the
// reported file name.
// Example:
// ```gad
// typeName(gad.parse("a := 1"))
// >>> "SourceFileObject"
// ```
//
// parseFile(pth str) <sourceFile>
// Like `parse`, but reads the Gad source from the file at `pth`.
//
// eval(source any; type) <any>
// Compiles and runs `source` — a source str (with an optional `type` dialect), a
// parsed source object, a statements object, or a single statement — and returns
// its result.
// Example:
// ```gad
// gad.eval("40 + 2")
// >>> 42
// gad.eval(gad.parse("40 + 2"))
// >>> 42
// ```
//
// quote(s str; maxCols, fence) <str>
// Returns `s` as a Gad string literal (quoted and escaped). `maxCols` wraps long
// strings and `fence` picks the quoting style.
// Example:
// ```gad
// gad.quote("hi")
// >>> "\"hi\""
// ```
//
// unquote(lit str) <str>
// The inverse of `quote`: parses a Gad string literal `lit` back to its str value.
// Example:
// ```gad
// gad.unquote(gad.quote("hi"))
// >>> "hi"
// ```
//
// invoker(fn any, args array; **nargs) <ret callable>
// Returns a repeatable, pre-resolved call: it resolves `fn`'s overload for the
// parameter types held in `args` (the initial elements are the types) and binds
// it to that same array. Calling the result runs the resolved overload with the
// array's current values, reusing the array and skipping type dispatch and
// validation. Mutate the array between calls to feed new values. `**nargs` are
// captured once and forwarded to every call. See samples/testing/invoker_test.gad.
// Example:
// ```gad
// args := [int]
// inv := gad.invoker(str, args)
// args[0] = 42
// inv()
// >>> "42"
// ```
//
// transform(value any; **paths) <any>
// Rewrites a JSON-like `value` (nested dicts/arrays) bottom-up, mapping yq-style
// path patterns to transformer functions. Each named arg is a path — `.` the root,
// `.key` a dict child, `.*` every dict child, `.**` every child, `.[]`/`.key[]`
// every array index, `.[N]`/`.key[N]` a specific index, `."k e y"` a quoted key —
// and its function receives the matched node and returns its replacement. Children
// are transformed before their container, so a container matcher sees the
// transformed children; the most specific pattern wins at a node, and a matcher's
// typed first param is enforced. The transformed value is returned. See
// samples/transform_mapped_test.gad.
// Example:
// ```gad
// gad.transform([1, 2, 3]; ".[]" = (n int) => n * 10)
// >>> [10, 20, 30]
// ```
//
// ## Operators
//
// Binary, self-assign and unary operators dispatch through the `gad` namespace, so
// user code can both call an operator and extend it for its own types:
//
// - `gad.binOp{Op}(left, right)` — a binary operator (`gad.binOpAdd(2, 3)` -> `5`).
// - `gad.selfAssignOp{Op}(left, right)` — a compound-assignment operator.
// - `gad.unOp{Op}(operand)` — a unary operator.
//
// A type customises an operator by adding a typed method, e.g.
// `met gad.binOpAdd(a Vec, b Vec) { … }`. See the User Operators chapter
// (samples/user_operators.gad).
//
// ## Context managers
//
// The `with` statement runs a resource's hooks through these two functions:
//
// - `gad.enter(resource)` — runs the resource's `enter()` hook, returning its value.
// - `gad.exit(resource, err)` — runs the resource's `exit(err)` cleanup hook.
//
// See the `with` chapter (samples/with.gad).
//
// ## Type objects
//
// The `gad` namespace also exposes the language's non-literal meta and structural
// type objects — the types that have no source literal of their own. Use them for
// reflection and for `::` type checks, e.g. `x :: gad.Class` tests whether `x` is a
// class, and `typeName(x) == "Enum"` matches `x :: gad.Enum`.
//
// - `Class` — the type of a `class` definition (a class value).
// - `Enum` — the type of an `enum` definition (an enum value).
// - `Interface` — the type of an `interface { … }` structural type.
// - `MethodInterface` — the type of a `meti` method interface.
// - `Module` — the type of an imported module.
// - `ModuleSpec` — the type of a module spec (a module's static descriptor).
// - `Symbol` — the type of a compiled symbol.
// - `SourceType` — the dialect enum for `parse` / `eval`: `GAD`, `TEMPLATE`, `GADX`.
// - `SourceFileObject` — the type of a parsed source file (returned by `parse`).
// - `StmtsObject` / `StmtObject` — the statements / single-statement AST types.
// - `Env` — the runtime environment type.
//
// Example:
// ```gad
// class Point { x int; y int }
// bool(Point(; x=1, y=2) :: gad.Class)
// >>> true
// ```
//
// ## Meta
//
// - `methodFromArgs` — a meta builtin that resolves a method from a call's args.
