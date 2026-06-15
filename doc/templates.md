# Templates (mixed mode)

Gad can run in **mixed / template mode**, where a file is plain text with
embedded Gad code. This is useful for generating configuration, JSON, HTML or
any other text output.

## Enabling template mode

There are three ways to put a file in template mode:

1. A `# gad: mixed` directive on the first line of a normal `.gad` file.
2. The `--template` flag of `gad run` (no directive needed).
3. A `.gadt` file extension (run as a template automatically).

```sh
gad run page.gadt              # .gadt is template by convention
gad run --template page.gad    # force template mode
```

## Tags

Inside a template, two tag forms embed Gad:

| Tag             | Purpose                                                       |
|-----------------|---------------------------------------------------------------|
| `{% … %}`       | A **code block** — runs Gad statements, emits nothing itself. |
| `{%= expr %}`   | A **value** — evaluates `expr` and writes it into the output. |

```
# gad: mixed
{% name := "Gad" --%}
Hello, {%= name %}!
```

```
Hello, Gad!
```

(`--%}` strips the newline after the code block; see
[Whitespace trim markers](#whitespace-trim-markers) below.)

Everything outside the tags is literal text and is preserved exactly.

## Control flow

Control-flow statements use the `begin … end` block form (the template
equivalent of `{ … }`), with the body being the template text between the tags:

```
# gad: mixed
{% for i, name in ["joe", "mary"] begin --%}
{%= i+1 %}. {%= name %}
{% end %}
```

```
1. joe
2. mary
```

`if` works the same way:

```
{% if ok begin %}yes{% end %}
```

## Whitespace trim markers

A `-` or `--` next to a delimiter trims the whitespace of the adjacent text:

| Marker            | Effect                                                       |
|-------------------|--------------------------------------------------------------|
| `{%-` / `-%}`     | Trim the adjacent blanks but **keep a single newline**.      |
| `{%--` / `--%}`   | Trim **all** adjacent whitespace (newlines included).        |

- `{%-` / `{%--` trim the **trailing** whitespace of the **preceding** text.
- `-%}` / `--%}` trim the **leading** whitespace of the **following** text.

```
# gad: mixed
A
{%-- = 1 --%}
B
```

```
A1B
```

With single dashes the boundary newline is kept:

```
# gad: mixed
A
{%- = 1 -%}
B
```

```
A
1
B
```

## Custom delimiters

The code-block delimiters default to `{%` / `%}`. There are three ways to
change them.

Per file, in the `# gad: mixed` directive, with a `delimiter = [START, END]`
array (string or raw-string values):

```
# gad: mixed, delimiter=["<?", "?>"]
Hi <?= 6*7 ?>!
```

Per run, with flags:

```sh
gad run --template-start-delimiter '<?' --template-end-delimiter '?>' page
```

Per workspace, in `.gad.yaml`:

```yaml
template:
  start_delimiter: "<?"
  end_delimiter: "?>"
```

CLI flags win over the `.gad.yaml` config. Either delimiter flag implies
`--template`.

## Formatting

`gad fmt` formats templates: the literal text is preserved verbatim, the Gad
code inside tags is formatted, and the tags are kept inline. The block
terminator is normalized to `{% end %}` and trim markers are preserved.
