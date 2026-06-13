# Gad Samples

A tour of the Gad language and standard-library modules. Every file is a small,
self-contained program you can run, format and debug.

## Running

```sh
gad run samples/01_hello.gad        # run a single sample
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

## Modules

The [`modules/`](modules) directory shows source modules and imports. Because
relative imports resolve against the importing file's directory, run the entry
point from inside that directory (the IDE does this automatically):

```sh
cd samples/modules && gad run main.gad
```

| File                | Purpose                                              |
|---------------------|------------------------------------------------------|
| `modules/mathx.gad` | a module that `export`s constants and functions      |
| `modules/greet.gad` | a parameterised module (`param (;lang="en")`)        |
| `modules/main.gad`  | imports both, including a parameterised import       |

## Standard library

The [`stdlib/`](stdlib) directory imports builtin modules:

| File                       | Module    |
|----------------------------|-----------|
| `stdlib/use_strings.gad`   | `strings` |
| `stdlib/use_fmt.gad`       | `fmt`     |
| `stdlib/use_json.gad`      | `json`    |
| `stdlib/use_time.gad`      | `time`    |
