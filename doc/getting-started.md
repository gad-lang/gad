# Getting Started

[← Back to index](README.md)

## Installing the CLI

Gad ships with a command-line tool that runs scripts and provides a REPL.

```sh
go install github.com/gad-lang/gad/cmd/gad@latest
```

Run `gad` with no arguments to start the interactive REPL, or pass a script
file to execute it:

```sh
gad                 # start the REPL
gad script.gad      # run a script
gad - < script.gad  # read the script from stdin
```

The CLI also exposes subcommands (`run`, `fmt`); see
[Subcommands](#subcommands) below. A bare `gad FILE` is shorthand for
`gad run FILE`.

## Your First Script

Create `hello.gad`:

```go
println("Hello, Gad!")
```

Run it:

```sh
$ gad hello.gad
Hello, Gad!
```

## Main Function, Parameters and Return

A Gad script is itself a function. It can declare parameters with
[`param`](variables-and-scopes.md#param) and produce a result with `return`. If
no `return` is reached, the script returns `nil`.

```go
param (name, *rest)

if !name {
    return "no name given"
}
return "hello " + name
```

Positional arguments come after the file name; named arguments use `--NAME` or
`--NAME=VALUE`:

```sh
$ gad greet.gad Gad world
```

## The REPL

The REPL evaluates expressions as you type and prints their values, which makes
it ideal for exploring the language:

```
» 1 + 2
3
» x := [1, 2, 3]
[1, 2, 3]
» [n * n for n in x]
[1, 4, 9]
```

## Passing Arguments

The script below joins its positional arguments and accepts named arguments
`--sep` and `--ln`:

```go
param (*args, sep=",", ln=no)
if !args { return }
for _, arg in args[:-1] { print(arg, sep) }
print(args[-1])
if ln { println() }
```

```sh
$ gad join.gad a b c            # a,b,c
$ gad join.gad a b c --sep +    # a+b+c
$ gad join.gad a b c --ln       # a,b,c\n
```

## Subcommands

The CLI is organised as subcommands. Run `gad help` for the list, or
`gad <cmd> --help` for a command's flags.

| Command       | Purpose                                                  |
|---------------|----------------------------------------------------------|
| `gad run`     | Run a script file/stdin, or start the REPL (the default).|
| `gad fmt`     | Format Gad source files in place.                        |
| `gad debug`   | Debug a script (interactive REPL or `--dap` for editors).|
| `gad ide`     | Start a local web IDE for a workspace directory or file. |
| `gad help`    | Show help and list subcommands.                          |

`gad` with no subcommand behaves as `gad run`, so `gad script.gad`,
`gad - < script.gad` and a bare `gad` (REPL) all keep working.

### Run flags (`gad run` / bare `gad`)

| Flag                       | Purpose                                            |
|----------------------------|----------------------------------------------------|
| `-no-optimizer`            | Disable the constant-folding optimizer.            |
| `-safe`                    | Disable external-access modules (`http`, `os`, …). |
| `-disabled-modules a,b`    | Disable specific modules.                          |
| `-timeout 5s`              | Abort the script after a duration.                 |
| `-trace parser,compiler`   | Trace the parse/optimize/compile steps.            |

### Formatting with `gad fmt`

`gad fmt PATH...` rewrites Gad source files with the canonical formatter, in
place by default:

```sh
gad fmt main.gad     # format a single file
gad fmt src/...      # recurse into ./src
gad fmt -            # format stdin to stdout
```

It supports include/exclude globs and regexes, `--out`, `--backup`, parallel
`--jobs`, layout `--no-*` flags, transpile flags, machine-readable `--report`
output and a `.gad.yaml` config file. See **[Formatting](formatting.md)** for
the full reference.

### The web IDE (`gad ide`)

`gad ide [PATH]` starts a local web IDE in your browser for a workspace
directory (or a single file to edit; PATH defaults to the current directory):

```sh
gad ide                # edit the current directory
gad ide src/           # open a project directory
gad ide main.gad       # open just one file
```

The IDE offers a file tree, multi-file tabs, formatting, running and debugging
(breakpoints, stepping, call stack and locals). Per-file **Run**/**Debug**
dialogs let you set arguments, enable or disable builtin modules, toggle safe
mode and save the combined STDOUT/STDERR to a file. Formatter settings and the
panel layout are stored in the workspace `.gad.yaml` (`fmt` and `ide` keys).

Useful flags:

| Flag             | Purpose                                                     |
|------------------|-------------------------------------------------------------|
| `--addr host:port` | Listen address (defaults to `0.0.0.0:17000`; the next free port is used if it is busy). |
| `--no-open`      | Do not open the browser automatically.                      |
| `--static DIR`   | Serve a pre-built web app from DIR instead of the bundled UI. |

In the React UI, set breakpoints by clicking the editor's left gutter (next to
the line numbers) and remove them with a double-click; the **Breakpoints** panel
lists them grouped per file (with *Current file* and *All* tabs) and they are
saved to `.gad.yaml` under `ide.breakpoints`.

The command ships a build-free bundled UI. For the richer **React + CodeMirror**
interface, build `web/app` and point `--static` at it (or use `make ide-react`):

```sh
make ide-react                                   # builds web/app and serves it
gad ide --static web/app/dist samples            # equivalent, after a build
```

To embed Gad in a Go program instead of running it from the CLI, see
[Embedding in Go](embedding.md).
