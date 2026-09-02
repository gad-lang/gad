# Gad Samples

A tour of the Gad language and standard-library modules. Every file is a small,
self-contained program you can run, format and debug.

## Running

```sh
gad samples/hello.gad        # run a single sample
```

> The canonical formatter (`gad fmt`, or **Format** in the IDE) rewrites code to
> a normalized layout and does not preserve comments, so it is intentionally not
> run over these annotated samples.

The samples directory is also the default workspace for the bundled web IDE:

```sh
gad ide samples        # or, from the repo: make ide
```

In the IDE you can open files in tabs, **Format**, **Run** and **Debug** them
(set breakpoints, step, inspect the call stack and locals), and configure
per-file run arguments, builtin-module toggles and output capture. Formatter and
layout settings are stored in [`.gad/gad.yaml`](.gad/gad.yaml).

## Language tour

| File                          | Topics                                            |
|-------------------------------|---------------------------------------------------|
| `hello.gad`                | printing, variables, interpolated strings `#"…{x}…"`  |
| `values_and_types.gad`     | primitive types and `typeof(v)`                   |
| `functions.gad`            | functions, arrow closures, closures, variadics    |
| `collections.gad`          | arrays, dicts, spread literals, iteration         |
| `comprehensions.gad`       | array and dict comprehensions                     |
| `control_flow.gad`         | `if`/`else`, `for`, `match`                        |
| `error_handling.gad`       | errors, `try`/`catch`/`finally`, the `or` fallback |
| `strings_bytes_regex.gad`  | strings, `b"…"`/`h"…"` bytes, `/regex/` literals  |
| `functions_with_methods.gad` | typed params, func-with-methods, `met`, `prop` |
| `class/classes.gad`           | classes via the `Class(…)` builtin: fields, methods, properties, inheritance |
| `class/syntax.gad`            | the `class` keyword (expression + statement forms) |
| `class/class_type.gad`        | inspect a class value: `MyClass.@name/@fields/…`, members by name, `MyClass[name]` |
| `class/field_types.gad`       | typed & nullable class fields (`x int`, `x? int|str`) |
| `class/field_defaults_test.gad` | field defaults: literal, `(= expr)` and per-instance `initFields` |
| `mixins.gad`               | `mixin { … }` and `use A, B`: reusable field/prop/method bundles, parents, dedup, the `this` interface |
| `method_interfaces.gad`   | `<…>` func headers, `meti` interfaces, `implements` |
| `ranges.gad`               | the `..` range operator, steps, temporal ranges   |
| `user_operators.gad`       | user operators `<<<` `>>>` `%%` via `gad.binOp`  |
| `in_operator.gad`          | the `in` membership operator                       |
| `doc_comments.gad`         | doc comments and `>>>` examples                    |
| `unary_operators.gad`      | unary `!` `-` `+` `^` `++` `--` and `gad.unOp`   |
| `with.gad`                 | the `with` context manager (statement + expression) |
| `enum.gad`                 | the `enum` keyword: values, signs, bit flags, members |
| `heredocs.gad`             | heredocs `"""…"""` / `` ```…``` `` and template heredocs |
| `key_value_array.gad`      | `keyValue` / `keyValueArray` (`(;…)`): flags, funcs, typed keys |
| `template_example.gadt`            | `.gadt` template mode: `{% %}`/`{%= %}` tags, trim markers |
| `interfaces.gad`           | `interface { … }`: typed fields, get/set/prop, methods, `*Parent` spreads, structural satisfaction (`::`) |
| `method_resolution.gad`    | dispatch rules: arity, specificity, subtypes, fallback, unions, variadics, `met`/override, structural `met<…>` params |
| `embed.gad`                | `embed(...)`: embed a file/directory at compile time (`.name`, `.size`, `.data`, entries, `sources=`) |
| `destructuring.gad`        | destructuring: arrays, named `(; target:key )` and TypeScript `{ key: target, **rest }`, defaults, any named source |

## Modules

The [`modules/`](modules) directory shows source modules and imports. Because
relative imports resolve against the importing file's directory, run the entry
point from inside that directory (the IDE does this automatically):

```sh
cd samples/modules && gad main.gad
```

| File                  | Purpose                                                  |
|-----------------------|----------------------------------------------------------|
| `modules/mathx.gad`   | a module that `export`s constants and functions          |
| `modules/exports.gad` | every `export` form (class/type/func/prop) + a `param`   |
| `modules/main.gad`    | imports both, including a parameterised import           |

## Standard library

The [`stdlib/`](stdlib) directory uses stdlib modules. `strings`, `fmt`, `time`
and `base64` are **builtin namespaces** — available without an `import`; `json`
is imported:

| File                       | Module    | Import? |
|----------------------------|-----------|---------|
| `stdlib/use_strings.gad`   | `strings` | no      |
| `stdlib/use_fmt.gad`       | `fmt`     | no      |
| `stdlib/use_time.gad`      | `time`    | no      |
| `stdlib/use_base64.gad`    | `base64`  | no      |
| `stdlib/use_json.gad`      | `json`    | yes     |
