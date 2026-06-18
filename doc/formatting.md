# Formatting with `gad fmt`

[← Back to index](README.md)

`gad fmt` rewrites Gad source files with the canonical formatter. By default it
formats **in place**; with `--out` it writes elsewhere and leaves inputs
untouched; stdin is always written to stdout.

```sh
gad fmt main.gad          # format a single file in place
gad fmt src               # format the .gad files directly in ./src
gad fmt src/...           # recurse into ./src and its sub-directories
gad fmt -                 # read stdin, write formatted source to stdout
```

A directory argument formats only the `.gad` files directly inside it; append
`/...` to recurse. Hidden files are ignored and hidden directories are skipped.
Files already formatted are left untouched; each file that changes is printed.

A failing file (e.g. a syntax error) does not stop the others — every target is
attempted, errors are reported to stderr, and the command exits with status `2`
(gofmt-style) when anything failed, otherwise `0`.

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

The multi-line layout is on by default. `--no-format` disables it entirely;
each `--no-*-in-new-line` flag keeps one construct on a single line:

| Flag                                    | Keeps on one line     |
|-----------------------------------------|-----------------------|
| `--no-format`                           | everything (no multi-line layout) |
| `--no-array-item-in-new-line`           | array items           |
| `--no-dict-item-in-new-line`            | dict items            |
| `--no-key-value-array-item-in-new-line` | keyValueArray items   |
| `--no-call-params-in-new-line`          | call arguments        |
| `--no-parem-values-in-new-line`         | parameter values      |
| `--no-decl-item-in-new-line`            | declaration items     |

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

## Config File (`.gad.yaml`)

Flag defaults can live in a YAML config under a `fmt:` key. The default file is
`.gad.yaml` in the working directory; override with `--config PATH` or disable
with `--no-config`. Command-line flags override config values.

Keys use the flag names (without the leading `--`). A special `input_dirs` list
declares directories to format with their own include/exclude/backup/report
settings (these merge with the global include/exclude globs; `backup` defaults
to false per directory, and `backup_format` defaults to the global value). A
per-directory `report` writes that directory's NDJSON lines on its own.

```yaml
fmt:
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
