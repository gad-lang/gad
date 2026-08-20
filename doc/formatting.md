# Formatting with `gad fmt`

[← Back to Get Started](getting-started.md)

`gad fmt` rewrites Gad source files with the canonical formatter. By default it
formats **in place**; with `--out` it writes elsewhere and leaves inputs
untouched; stdin is always written to stdout.

```sh
gad fmt main.gad          # format a single file in place
gad fmt src               # format the .gad/.gadt/.gadx files directly in ./src
gad fmt src/...           # recurse into ./src and its sub-directories
gad fmt -                 # read stdin, write formatted source to stdout
gad fmt --stdin-name a.gadx -   # read stdin as a Gadx buffer (see Dialects)
gad fmt --indent 2 main.gad     # indent with 2 spaces instead of a tab
```

A directory argument formats only the `.gad`, `.gadt` and `.gadx` files directly
inside it; append `/...` to recurse. Hidden files are ignored and hidden
directories are skipped. Files already formatted are left untouched; each file
that changes is printed.

## Dialects

The dialect is chosen by file extension, so one command formats every kind of
Gad source:

| Extension | Dialect                     | Formatter                                   |
|-----------|-----------------------------|---------------------------------------------|
| `.gad`    | plain Gad                   | the canonical Gad formatter                 |
| `.gadt`   | mixed template              | Gad formatter, parsed in template mode      |
| `.gadx`   | indentation (pug-style) template | the Gadx source formatter              |

A `.gadx` file is re-emitted in Gadx syntax (tags, components, indentation).
**Embedded Gad code** inside it — attribute expressions, `@if`/`@for` conditions,
`~` code lines and declaration values — is formatted with the **same rules as
plain Gad** (see [Layout Control](#layout-control)).

When reading from **stdin** there is no file name, so the dialect defaults to
plain Gad. Pass `--stdin-name FILE` to name the buffer and select the dialect —
this is what editors use to format an unsaved `.gadx`/`.gadt` buffer:

```sh
gad fmt --stdin-name page.gadx - < page.gadx
```

A failing file (e.g. a syntax error) does not stop the others — every target is
attempted, errors are reported to stderr, and the command exits with status `2`
(gofmt-style) when anything failed, otherwise `0`.

## Comments

`gad fmt` preserves `//` line comments and `/* … */` block comments. A comment
on the same line as a statement stays attached to it as a trailing comment;
comments on their own line are kept on their own line before the following
statement. (A comment sitting between the last statement of a block and its
closing `}` is currently re-emitted just after the `}`.)

## Selecting Files

| Flag            | Effect                                                            |
|-----------------|------------------------------------------------------------------|
| `--exclude GLOB`| Skip files matching GLOB (repeatable; comma-separated allowed).   |
| `--include GLOB`| Format matching files even if excluded (repeatable/comma).        |
| `--exclude-re RE` | Skip files matching the regex RE (repeatable; not comma-split).|
| `--include-re RE` | Re-include matching files even if excluded (repeatable).       |

Globs and regexes are tested against **both the full path and the base name**,
so either form works. An `include` match always wins over an `exclude`.

```sh
gad fmt --exclude '*_gen.gad' src/...
gad fmt --exclude-re '_(gen|test)\.gad$' src/...
```

## Output and Backups

| Flag                      | Effect                                                            |
|---------------------------|------------------------------------------------------------------|
| `--out PATH`              | Single input → output file `PATH`; otherwise `PATH` is an output directory mirroring the input tree. Inputs are not modified. |
| `--backup`                | Write a backup of each file before rewriting it in place.        |
| `--backup-format PATTERN` | Backup name pattern; `BASE_NAME` → file name without extension (default `BASE_NAME.backup.gad`). |

```sh
gad fmt --out dist src/...        # formatted copies under ./dist, src/ untouched
gad fmt --backup main.gad         # writes main.backup.gad, then formats main.gad
```

## Parallelism

| Flag           | Effect                                                              |
|----------------|--------------------------------------------------------------------|
| `--jobs N`     | Max concurrent jobs (default: number of CPUs).                      |

Each explicit file (and stdin) is one job; each directory is one job that
formats all of its files. Jobs run in parallel up to `--jobs`.

## Layout Control

By default the formatter is **column-aware**: a list construct stays on one line
and wraps to one item per line only when the inline form would overflow the
column budget (`--max-columns`, default 80). Each `--*-in-new-line` flag opts in
to **forcing** one construct onto separate lines regardless of width; `--format`
forces the full multi-line layout (every construct expanded).

| Flag                                 | Effect                                              |
|--------------------------------------|-----------------------------------------------------|
| `--max-columns N`                    | wrap budget before a construct breaks (0 uses 80)   |
| `--format`                           | force everything onto multiple lines                |
| `--array-item-in-new-line`           | force each array item onto its own line             |
| `--dict-item-in-new-line`            | force each dict item onto its own line              |
| `--key-value-array-item-in-new-line` | force each keyValueArray item onto its own line     |
| `--call-params-in-new-line`          | force each call argument onto its own line          |
| `--parem-values-in-new-line`         | force each parameter value onto its own line        |
| `--decl-item-in-new-line`            | force each declaration item onto its own line       |

## Indentation

`--indent` sets the indentation unit used per nesting level, for **every dialect**
(`.gad`, `.gadt` and the `.gadx` tag/component nesting). The default is a single
tab. The value is one of:

| `--indent` value | Unit                                   |
|------------------|----------------------------------------|
| `tab` / `tabs`   | one tab (the default)                  |
| `N` (an integer) | `N` spaces (e.g. `--indent 2`)         |
| `Nt` / `Ntab`    | `N` tabs (e.g. `--indent 2t`)          |
| any other string | that literal string (spaces, tabs or other characters); `\t` is expanded to a tab |

```sh
gad fmt --indent 4 main.gad     # four spaces per level
gad fmt --indent tab app.gadx   # one tab per level (default)
```

## Transpile

One `--transpile-NAME` flag is generated per field of the formatter's transpile
options (currently `--transpile-raw-str-func-start`,
`--transpile-raw-str-func-end`, `--transpile-write-func`). Setting any of them
emits transpiled output instead of plain formatting.

## Reports

`--report PATH` writes a per-file status report as **NDJSON** — one JSON object
per line. Use `--report -` to write the report to stdout. Each line carries
`file` (relative to its input directory when the file came from one),
`input_dir` (only for files discovered through a directory job), and `error`
(only on failure):

```sh
gad fmt --report report.ndjson src/...
```

```json
{"input_dir":"src","file":"a.gad"}
{"input_dir":"src","file":"b.gad","error":"Parse Error: ..."}
{"file":"oops.gad"}
```

- `--report-stream` writes each record as soon as its file is done, rather than
  buffering them all until the end. With no `--report`, the report streams to
  stdout.
- `--report-contents` adds a `result` key to each record holding the formatted
  source — useful with `--no-save` to format without touching files:

```sh
gad fmt --no-save --report-contents --report - src/...
```

```json
{"input_dir":"src","file":"a.gad","result":"x := 1\n"}
```

### Read-only formatting

`--no-save` formats every file but writes, creates and backs up nothing. Combine
it with `--report-contents` (and `--report -`) to obtain the formatted source as
data without modifying the working tree.

## Config File (`.gad/gad.yaml`)

Flag defaults can live in a YAML config under a `fmt:` key. The default file is
`.gad/gad.yaml` in the working directory; override with `--config PATH` or disable
with `--no-config`. Command-line flags override config values.

Keys use the flag names (without the leading `--`). A special `input_dirs` list
declares directories to format with their own include/exclude/backup/report
settings (these merge with the global include/exclude globs; `backup` defaults
to false per directory, and `backup_format` defaults to the global value). A
per-directory `report` writes that directory's NDJSON lines on its own.

```yaml
fmt:
  indent: "2"          # 2 spaces per level (default: tab); "tab", "2t", … also work
  exclude:
    - "*_gen.gad"
  backup-format: "BASE_NAME.bak.gad"
  report: report.ndjson
  input_dirs:
    - path: src
      backup: true
      excludes: ["*_test.gad"]
      report: src-report.ndjson
```

With such a file present, a bare `gad fmt` (no path arguments) formats the
configured `input_dirs`.
