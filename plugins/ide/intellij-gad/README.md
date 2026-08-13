# Gad Language — IntelliJ Platform plugin

Language support for **Gad** (`.gad` / `.gadt` / `.gadx`) in JetBrains IDEs
(IntelliJ IDEA, GoLand, WebStorm, PyCharm, …). Marketplace name **Gad Language**;
plugin ID `dev.gad-lang.gad`. See <https://gad-lang.github.io>.

Design and roadmap: [`PLAN.md`](PLAN.md).

## Features

- **Syntax highlighting** for the three dialects, via a TextMate bundle assembled
  at build time from the sibling [`../vscode-gad`](../vscode-gad) grammars (single
  source of truth — no hand-written lexer).
- **Run configurations** with execution profiles: script, arguments, working
  directory, environment variables and `GADPATH`. Runs `gad <script> …`.
- **Debugger** over the Gad Debug Adapter (`gad debug --dap`):
  - line breakpoints and **conditional breakpoints**;
  - **call stack** with per-frame navigation, including **into imported files**;
  - **step** in / over / out, **resume** and **pause**;
  - **variables** inspection (Locals) and **evaluate** (Watches / Debug Console /
    on-hover);
  - console output, terminate/disconnect.
- **Settings** (Settings ▸ Tools ▸ Gad): the `gad` executable location and a
  default `GADPATH`.
- **Config schema** validation/completion for `.gad.yaml` / `.gadide.yaml`
  (schemas reused from `../vscode-gad/schemas`).

## Requirements

- A `gad` binary on `PATH` (or set its location in Settings ▸ Tools ▸ Gad). The
  debugger requires a `gad` built **with** the `debug` command (the default
  build; the `nodebug` build tag removes it).
- JDK 21 to build the plugin.

## Build

The IntelliJ Platform Gradle plugin downloads the target IDE SDK on first build.
A `Makefile` wraps the common Gradle tasks:

```sh
cd plugins/ide/intellij-gad
make help        # list targets
make compile     # compile the Kotlin (fast sanity check)
make build       # → build/distributions/intellij-gad-<version>.zip
make test        # unit tests
make verify      # JetBrains Plugin Verifier
make check       # compile + test + verify
make run         # launch a sandbox IDE with the plugin
make clean
```

Each target just calls `./gradlew <task>` (the committed wrapper); override the
JDK with `make JAVA_HOME=/path build`. Install the built `.zip` via *Settings ▸
Plugins ▸ ⚙ ▸ Install Plugin from Disk*.

## Architecture

The plugin is a thin front-end over the Gad CLI's protocols:

| Concern | Implementation |
| --- | --- |
| Highlighting | `highlight/GadBundleProvider` ships the VS Code grammars as a TextMate bundle |
| File identity | `lang/GadFile` (extension check) + `lang/GadFileIconProvider` (icon) |
| Run | `run/*` — `GadRunConfiguration` + profile form + `GadCommandLineState` (`gad <script>`) |
| Debug | `debug/*` — `GadDebugProcess` bridges `debug/dap/DapClient` (DAP over stdio) to the XDebugger |
| Config | `config/GadJsonSchemaProviderFactory` maps the reused JSON schemas |
| Settings | `settings/*` — application-level `gad` path + default `GADPATH` |

The debugger relies on the adapter reporting **per-frame source paths** and
honoring **launch profiles** (`args`/`cwd`/`env`/`GADPATH`) — shipped in the Gad
CLI (`cmd/gad/dap.go`).

## Status

All layers are implemented (highlighting, run, debug, settings, config). `make
build` compiles the Kotlin and packages a valid plugin `.zip` (the reused
grammars and schemas are bundled and `plugin.xml` is verified). Exercising the
debugger in a running IDE, the Plugin Verifier pass and the Marketplace listing
are the remaining ship steps (PLAN.md, phase 4). The IntelliJ SDK download is out
of scope for this repo's Go CI, so the plugin builds in its own Gradle job.
