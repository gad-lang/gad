
# Templates (mixed mode)

Gad can run in **mixed / template mode**, where a file is plain text with
embedded Gad code — useful for generating configuration, JSON, HTML or any text
output. This sample is itself a template that generates a JSON users list (see
the Example below); [template_example.gadt](template_example.gadt) is a `.gadt` variant.

## Enabling

Three ways to put a file in template mode: a `# gad: mixed` directive on the
first line of a `.gad` file (as here), the `--template` flag of `gad run`, or a
`.gadt` file extension (automatic).

## Tags

- `{% … %}` — a **code block**: runs statements, emits nothing itself.
- `{%= expr %}` — a **value**: evaluates `expr` and writes it to the output.

Everything outside the tags is literal text, preserved exactly.

## Control flow

Control-flow statements use the `begin … end` block form (the template
equivalent of `{ … }`), the body being the template text between the tags —
`{% for i, x in xs begin %} … {% end %}`, `{% if ok begin %}yes{% end %}`.

## Whitespace trim markers

A `-` or `--` next to a delimiter trims adjacent whitespace: `{%-` / `-%}` trim
blanks but **keep a single newline**; `{%--` / `--%}` trim **all** adjacent
whitespace (newlines included). `{%-` / `{%--` trim the preceding text's trailing
whitespace; `-%}` / `--%}` the following text's leading whitespace.

## Custom delimiters

The `{%` / `%}` delimiters can be changed per file
(`# gad: mixed, delimiter=["<?", "?>"]`), per run
(`--template-start-delimiter` / `--template-end-delimiter`), or per workspace
(`.gad.yaml` `template.start_delimiter` / `end_delimiter`; CLI flags win).

`gad fmt` formats templates: literal text verbatim, the Gad code inside tags
formatted, tags kept inline, the terminator normalized to `{% end %}`.

## Example — `template.gad`

```gad
# gad: mixed
{%
var (
  users = [    "joe", "mary", "george"   ]
  lastIndex = len(users)-1
)
--%}
{
  "users": [
{%--  for i,      name in    users begin %}
    "{%=name    %}"{%= i < lastIndex ? "," : "" %}
{%-- end %}
  ]
}
```
