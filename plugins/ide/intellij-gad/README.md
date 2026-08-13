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

```sh
cd plugins/ide/intellij-gad
./gradlew buildPlugin      # → build/distributions/intellij-gad-<version>.zip
./gradlew runIde           # launch a sandbox IDE with the plugin
./gradlew verifyPlugin     # JetBrains Plugin Verifier
./gradlew test             # unit tests
```

Install the built `.zip` via *Settings ▸ Plugins ▸ ⚙ ▸ Install Plugin from Disk*.

> The Gradle wrapper JAR (`gradle/wrapper/gradle-wrapper.jar`) is not committed;
> generate it once with a local Gradle (`gradle wrapper --gradle-version 8.10.2`)
> or let the IDE's Gradle integration create it on import.

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

All layers are implemented (highlighting, run, debug, settings, config). The
project has not yet been compiled in this repository's CI (the IntelliJ SDK
download is out of scope for the Go CI); build it locally with `./gradlew
buildPlugin`. A dedicated JetBrains CI job and the Marketplace listing are the
remaining ship steps (PLAN.md, phase 4).
