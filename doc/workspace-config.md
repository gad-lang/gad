# Workspace Configuration

[← Back to index](README.md)

A Gad workspace is configured by two files at its root:

- **`.gad.yaml`** — the project configuration read by the `gad` CLI (formatting,
  template mode, transpilation, the `env` section, …).
- **`.gadide.yaml`** — the `gad ide` layout and editor state, kept beside
  `.gad.yaml`. It is split out so IDE state does not clutter the project config.

## Variable expansion

**Every value in the configuration — at any nesting depth — is expanded using
bash-style parameter expansion.** A value is read, its `$VAR` / `${…}` references
are expanded, and the result is converted back to a number or boolean when it
holds one (so `port: "${PORT:-8080}"` yields the integer `8080`).

References come from two sources:

- **Environment variables** — `$NAME` or `${NAME}` reads the process environment.
- **The config itself** — a name that begins with a dot is a path into the
  configuration document, with `.field` and `[index]` navigation:
  `${.ide.panels[0].id}`. Scalar leaves (int, bool, …) are converted to text.

### Supported operators

The full set of bash parameter-expansion operators is supported (semantics match
GNU bash):

| Form | Meaning |
|------|---------|
| `$VAR`, `${VAR}` | the value of `VAR` |
| `${var:-default}` | use `default` if `var` is unset or empty (no assignment) |
| `${var:=default}` | assign `default` to `var` if it is unset or empty |
| `${var:+alternate}` | use `alternate` only if `var` is set and non-empty, else empty |
| `${var:offset:length}` | substring starting at `offset` (`length` optional; negatives count from the end) |
| `${var#pattern}` | remove the shortest matching prefix |
| `${var##pattern}` | remove the longest matching prefix |
| `${var%pattern}` | remove the shortest matching suffix |
| `${var%%pattern}` | remove the longest matching suffix |
| `${var/pattern/string}` | replace the first match of `pattern` |
| `${var//pattern/string}` | replace all matches of `pattern` |
| `${var/#pattern/string}` | replace `pattern` only at the front |
| `${var/%pattern/string}` | replace `pattern` only at the end |

`pattern` is a shell glob (`*`, `?`, `[…]`); unlike filename globbing, `/` is an
ordinary character. Booleans accept `yes`/`no` as well as `true`/`false`
(matching YAML). A backslash escapes the next character (`\$` is a literal `$`).

```yaml
env:
    PATH: "${HOME}/bin:$PATH"        # environment references
    BASE: "${.ide.root:-/srv/app}"   # config self-reference with a default
    KIND: "${FILE##*.}"              # longest-prefix strip → the extension

# Every value is expanded, not just the env section:
title: "${APP_NAME:-Gad} console"
port: "${PORT:-8080}"                # becomes the integer 8080
```

## The `env` section

The `env` section defines environment variables that seed the [`env`
keyword](operators.md) available to scripts run in the workspace. It may be a
mapping or a list of `NAME=value` entries, and its values are expanded as above
(later entries can reference earlier ones). The `env` table **extends** the
process environment: a script sees the host's variables plus these.

```yaml
env:
    APP_HOME: "${HOME}/app"
    PATH: "${APP_HOME}/bin:$PATH"
```

### Portable path lists

A value may be an **array of segments** instead of a string. The segments are
each expanded and then joined with the operating system's path-list separator
(`:` on Unix, `;` on Windows). This keeps `PATH`-like variables portable — you
never hard-code the separator:

```yaml
env:
    GADPATH: ["${HOME}/gadlib", "${.project.shared}", "vendor"]
    # → on Unix:    /home/u/gadlib:/srv/shared:vendor
    # → on Windows: C:\Users\u\gadlib;\srv\shared;vendor
```

## `GADPATH`

`GADPATH` is the module search path, used like Python's `PYTHONPATH`. It is an
OS-list-separated list of directories (`:` on Unix, `;` on Windows), read from
the process environment.

When a script does `import("name")`, the file importer resolves the module in
this order and the **first existing file wins**:

1. relative to the importing file's own directory;
2. then each directory in `GADPATH`, in order.

So `GADPATH` supplies shared libraries that any script can import by a bare name
without a relative path. The active list is also exposed to scripts as the
`SOURCE_PATH` global.

```sh
GADPATH="$HOME/gadlib:/usr/local/share/gad" gad run app.gad
# in app.gad:  lib := import("mylib.gad")   # found in $HOME/gadlib/mylib.gad
```

For the embedding and Go-side API of expansion, see the
[`shellexpand`](https://pkg.go.dev/github.com/gad-lang/gad/shellexpand) package.
