# Gad Samples

A tour of the Gad language and standard-library modules. Every file is a small,
self-contained program you can run, format and debug.

## Running

```sh
gad samples/01_hello.gad        # run a single sample
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
layout settings are stored in [`.gad.yaml`](.gad.yaml).

## Language tour

| File                          | Topics                                            |
|-------------------------------|---------------------------------------------------|
| `01_hello.gad`                | printing, variables, template strings `#"…{x}…"`  |
| `02_values_and_types.gad`     | primitive types and `typeof(v)`                   |
| `03_functions.gad`            | functions, arrow closures, closures, variadics    |
| `04_collections.gad`          | arrays, dicts, spread literals, iteration         |
| `05_comprehensions.gad`       | array and dict comprehensions                     |
| `06_control_flow.gad`         | `if`/`else`, `for`, `match`                        |
| `07_error_handling.gad`       | errors, `try`/`catch`/`finally`, the `or` fallback |
| `08_strings_bytes_regex.gad`  | strings, `b"…"`/`h"…"` bytes, `/regex/` literals  |
| `10_functions_with_methods.gad` | typed params, func-with-methods, `met`, `prop` |
| `11_classes.gad`              | classes: fields, methods, properties, inheritance |
| `12_method_interfaces.gad`   | `<…>` func headers, `meti` interfaces, `implements` |
| `13_ranges.gad`               | the `..` range operator, steps, temporal ranges   |
| `14_user_operators.gad`       | user operators `<<<` `>>>` `%%` via `gad.binOp`  |
| `15_in_operator.gad`          | the `in` membership operator                       |
| `16_doc_comments.gad`         | doc comments and `>>>` examples                    |
| `17_unary_operators.gad`      | unary `!` `-` `+` `^` `++` `--` and `gad.unOp`   |
| `18_with.gad`                 | the `with` context manager (statement + expression) |
| `19_class_syntax.gad`         | the `class` keyword (expression + statement forms) |
| `20_enum.gad`                 | the `enum` keyword: values, signs, bit flags, members |
| `21_heredocs.gad`             | heredocs `"""…"""` / `` ```…``` `` and template heredocs |
| `22_key_value_array.gad`      | `keyValue` / `keyValueArray` (`(;…)`): flags, funcs, typed keys |
| `23_template.gadt`            | `.gadt` template mode: `{% %}`/`{%= %}` tags, trim markers |
| `24_interfaces.gad`           | `interface { … }`: typed fields, get/set/prop, methods, `*Parent` spreads, structural satisfaction (`::`) |
| `25_method_resolution.gad`    | dispatch rules: arity, specificity, subtypes, fallback, unions, variadics, `met`/override, structural `met<…>` params |
| `26_embed.gad`                | `embed(...)`: embed a file/directory at compile time (`.name`, `.size`, `.data`, entries, `sources=`) |

## Modules

The [`modules/`](modules) directory shows source modules and imports. Because
relative imports resolve against the importing file's directory, run the entry
point from inside that directory (the IDE does this automatically):

```sh
cd samples/modules && gad main.gad
```

| File                | Purpose                                              |
|---------------------|------------------------------------------------------|
| `modules/mathx.gad` | a module that `export`s constants and functions      |
| `modules/greet.gad` | a parameterised module (`param (;lang="en")`)        |
| `modules/main.gad`  | imports both, including a parameterised import       |

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
