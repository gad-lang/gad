# Gad Language Documentation

Gad is a fast, dynamic scripting language designed to be embedded into Go
applications. Source code is compiled to bytecode and run on a stack-based
virtual machine written in native Go.

The **language-feature documentation lives inside the samples**: each
`samples/NN_topic.{gad,gadt,gadx}` carries its chapter as a doc comment and a
runnable example, and `gad doc` generates the rendered pages under
[`doc/samples/`](samples/) (index: [doc/samples/README.md](samples/README.md)).
Regenerate them with `make samples-doc`.

## Language chapters (generated from the samples)

| Chapter | Rendered doc | Sample source |
|---------|--------------|---------------|
| Getting Started | [getting-started.md](getting-started.md) | — |
| Values and Types | [02](samples/02_values_and_types.md) | [`samples/02_values_and_types.gad`](../samples/02_values_and_types.gad) |
| Variables and Scopes | [33](samples/33_variables_and_scopes.md) | [`samples/33_variables_and_scopes.gad`](../samples/33_variables_and_scopes.gad) |
| Operators | [14](samples/14_user_operators.md) | [`samples/14_user_operators.gad`](../samples/14_user_operators.gad) (+ [13](samples/13_ranges.md)/[15](samples/15_in_operator.md)/[17](samples/17_unary_operators.md)/[28](samples/28_absent_coalescing.md)) |
| Control Flow | [06](samples/06_control_flow.md) | [`samples/06_control_flow.gad`](../samples/06_control_flow.gad) (+ [18 with](samples/18_with.md)) |
| Functions | [03](samples/03_functions.md) | [`samples/03_functions.gad`](../samples/03_functions.gad) (+ [10 methods](samples/10_functions_with_methods.md)) |
| Properties | [31](samples/31_properties.md) | [`samples/31_properties.gad`](../samples/31_properties.gad) |
| Collections | [04](samples/04_collections.md) | [`samples/04_collections.gad`](../samples/04_collections.gad) (+ [05](samples/05_comprehensions.md)/[22](samples/22_key_value_array.md)/[27](samples/27_destructuring.md)) |
| Classes | [11](samples/11_classes.md) | [`samples/11_classes.gad`](../samples/11_classes.gad) (+ [19 class syntax](samples/19_class_syntax.md)) |
| Method Interfaces | [12](samples/12_method_interfaces.md) | [`samples/12_method_interfaces.gad`](../samples/12_method_interfaces.gad) |
| Interfaces | [24](samples/24_interfaces.md) | [`samples/24_interfaces.gad`](../samples/24_interfaces.gad) |
| Enums | [20](samples/20_enum.md) | [`samples/20_enum.gad`](../samples/20_enum.gad) |
| Strings, Bytes & Regex | [08](samples/08_strings_bytes_regex.md) | [`samples/08_strings_bytes_regex.gad`](../samples/08_strings_bytes_regex.gad) (+ [21 heredocs](samples/21_heredocs.md)) |
| Error Handling | [07](samples/07_error_handling.md) | [`samples/07_error_handling.gad`](../samples/07_error_handling.gad) |
| Modules and Embedding | [26](samples/26_embed.md) | [`samples/26_embed.gad`](../samples/26_embed.gad) |
| Templates (mixed mode) | [09](samples/09_template.md) | [`samples/09_template.gad`](../samples/09_template.gad) (+ [23 .gadt](samples/23_template.md)) |
| Doc Comments | [16](samples/16_doc_comments.md) | [`samples/16_doc_comments.gad`](../samples/16_doc_comments.gad) |
| Special `@` Keywords | [29](samples/29_special_keywords.md) | [`samples/29_special_keywords.gad`](../samples/29_special_keywords.gad) |
| Metaprogramming | [34](samples/34_metaprogramming.md) | [`samples/34_metaprogramming.gad`](../samples/34_metaprogramming.gad) |

## Reference & tooling

- [Getting Started](getting-started.md) — install, run scripts, the REPL.
- [Embedding in Go](embedding.md) — compile and run Gad from Go, pass globals and
  arguments, expose Go functions and typed methods.
- [Formatting](formatting.md) — the `gad fmt` source formatter and the `.gad.yaml`
  config file.
- [Workspace Configuration](workspace-config.md) — `.gad/gad.yaml` / `ide.yaml`,
  the `env` section, `GADPATH`, and the `doc-templates` directory.
- [Conventions](conventions.md) — how primitive types, constants, modules and
  methods are cased, plus the layout the formatter produces.
- [Builtin Functions](builtins.md) — overview of the builtin library.
- [`reflect`](reflect.md) — raw, delegation-free index access.
- [Testing](stdlib-test.md) — the `test` module and the `gad test` command.
- [Tutorial](tutorial.md) — a guided walk-through of the language.
- Generated standard-library references: [`time`](stdlib-time.md),
  [`fmt`](stdlib-fmt.md), [`strings`](stdlib-strings.md), [`json`](stdlib-json.md).

## Gadx Templates

Gadx is an indentation-based HTML template language that embeds Gad, shipped in
this repository as the [`gadx`](../gadx) submodule. `.gadx` files run and debug
with the `gad` CLI and the IDE tooling. Its documentation lives in
[`gadx/docs`](../gadx/docs).
