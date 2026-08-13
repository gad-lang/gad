# Gad plugin for the IntelliJ Platform — implementation plan

Status: **planning**. This document is the design/build plan for a full-featured
Gad plugin for JetBrains IDEs. No plugin code has been written yet; this file is
the agreed scope and roadmap.

## 1. Name and target

- **Directory**: `plugins/intellij-gad` (sibling of `plugins/vscode-gad`).
- **What it targets**: the **IntelliJ Platform**, so a single plugin runs in
  IntelliJ IDEA, GoLand, WebStorm, PyCharm, RustRover, CLion, Rider, etc. The name
  `intellij-gad` follows the community convention for IntelliJ Platform language
  plugins (`intellij-rust`, `intellij-elixir`, `intellij-lua`, …) and is symmetric
  with `vscode-gad`; `jetbrains-gad` is an acceptable alternative. **Plugin ID**:
  `dev.gad-lang.gad` (reverse-DNS, stable). **Marketplace display name**:
  `Gad Language`. **Website**: <https://gad-lang.github.io>.
- **Languages**: `.gad` (scripts/modules), `.gadt` (mixed templates), `.gadx`
  (indentation templates) — the same three the VS Code extension and the web
  plugins handle.

## 2. Goal — feature parity with the web IDE + VS Code extension

Everything the browser IDE (`web/ide-*`) and the VS Code extension expose, on the
IntelliJ Platform:

- Syntax highlighting for the three dialects.
- **Run configurations** with named **execution profiles** (args, env, working
  dir), backed by the project's `.gad` config.
- Full **debugger**:
  - line breakpoints and **conditional breakpoints**;
  - **call stack** with frame selection;
  - **step in / over / out / resume / pause**;
  - **inspect** — Variables/Scopes tree per frame;
  - **evaluate** — Watches and expression evaluation, incl. on-hover;
  - **navigation into imported files** while stepping (open the frame's real
    source file, not just the entry script);
  - terminate/disconnect.
- **Settings** (locate the `gad` binary, default profile, format-on-save…).
- Awareness of the project config files (`.gad.yaml`, `.gadide.yaml`) with
  JSON-schema validation and completion.

## 3. Architecture — reuse, don't reinvent

The Gad CLI already ships the two protocols an editor needs; the plugin is a thin
IntelliJ front-end over them.

| Concern | Reused Gad asset | IntelliJ side |
| --- | --- | --- |
| Debugging | `gad debug --dap` — a **Debug Adapter Protocol** server over stdio (`cmd/gad/dap.go`) | a DAP client bridged to the **XDebugger** API |
| Highlighting | TextMate grammars in `plugins/vscode-gad/syntaxes/{gad,gadx}.tmLanguage.json` | IntelliJ **TextMate bundle** support (`org.jetbrains.plugins.textmate`) — single source of truth, no grammar rewrite |
| Config validation | JSON schemas in `plugins/vscode-gad/schemas/*.json` | **JSON Schema** mappings for `.gad.yaml` / `.gadide.yaml` |
| Formatting | `gad fmt -` (stdin/stdout) | an external formatter / format-on-save action |
| Running | `gad run <file>` / `gad <file>` | a `RunConfiguration` process |

**Build system**: Gradle with the **IntelliJ Platform Gradle Plugin 2.x**,
Kotlin, `sinceBuild` = 2024.2 (`242`). DAP JSON is spoken with a small
hand-rolled client (or `org.eclipse.lsp4j.debug` if we vendor it) — no dependency
on third-party marketplace plugins, so the plugin is self-contained.

### DAP capabilities already provided (`cmd/gad/dap.go`)

Requests handled: `initialize`, `launch`, `configurationDone`, `setBreakpoints`,
`threads`, `stackTrace`, `scopes`, `variables`, `evaluate`, `continue`, `next`,
`stepIn`, `stepOut`, `pause`, `terminate`, `disconnect`. Events: `initialized`,
`stopped`, `output`, `terminated`, `exited`. Capabilities advertised:
`supportsConditionalBreakpoints`, `supportsEvaluateForHovers`,
`supportsTerminateRequest`.

This already covers breakpoints (+conditional), stepping, call stack, inspect and
evaluate. Two gaps must be closed on the Go side for full parity (§5).

## 4. Feature → IntelliJ API mapping

| Feature | IntelliJ Platform extension point / API |
| --- | --- |
| File types `.gad/.gadt/.gadx` | `com.intellij.fileType` (+ `LanguageFileType`, icons) |
| Highlighting | TextMate bundle registered via `TextMateBundleProvider`; grammars copied at build time from `plugins/vscode-gad` |
| Config files (`.gad.yaml`, `.gadide.yaml`) | `JsonSchemaProviderFactory` mapping to the reused schemas |
| Run configuration + profiles | `ConfigurationType`, `ConfigurationFactory`, `RunConfigurationBase`, `SettingsEditor` (form: file, profile, args, env, cwd) + `CommandLineState` |
| Debugger entry | `XDebuggerRunner` / `ProgramRunner` producing an `XDebugProcess` |
| Debug session | `XDebugProcess` subclass wrapping the DAP client |
| Line/conditional breakpoints | `XLineBreakpointType` with a condition expression → DAP `setBreakpoints` (`condition`) |
| Call stack | `XExecutionStack` / `XStackFrame` from DAP `stackTrace` frames |
| Cross-file navigation | each `XStackFrame` exposes an `XSourcePosition` built from the frame's `Source.path` (needs §5.1) |
| Inspect | `XValueContainer`/`XValue` tree from DAP `scopes`+`variables` |
| Evaluate / Watches / hover | `XDebuggerEvaluator` → DAP `evaluate` (context `watch`/`hover`) |
| Stepping | `XDebugProcess.startStepOver/Into/Out/resume` → DAP `next`/`stepIn`/`stepOut`/`continue` |
| Console output | DAP `output` events → `ConsoleView` |
| Settings | `Configurable` (application + project): gad binary path, default profile, format-on-save |

## 5. Gad-side (Go) prerequisites

These small `cmd/gad/dap.go` (and debug-engine) changes are required for full
feature parity and are part of this effort:

### 5.1 Per-frame source paths (cross-file navigation) — **required**

Today `handleStackTrace` reports a single `Source{Path: s.program}` for **every**
frame, so stepping into an imported module still points the editor at the entry
script. The debug engine must expose each frame's originating **file path**, and
`handleStackTrace` must emit a per-frame `dap.Source{Path: frame.File}`.
`setBreakpoints` must likewise key breakpoints by the requested source path so a
breakpoint set inside an imported `.gad` file is honored.

### 5.2 Richer launch arguments (execution profiles) — **required**

`launchArgs` currently reads only `program` + `stopOnEntry`. Add `args []string`,
`cwd string`, `env map[string]string` (and optionally a named `profile` resolved
from `.gad.yaml`), so run profiles carry through to the launched program.

### 5.3 `setVariable` (edit values in the inspector) — optional/nice-to-have

Advertise `supportsSetVariable` and handle `SetVariableRequest` so the Variables
tree is editable. Deferrable to a later phase.

Each item ships as its own Go commit with a DAP test in `cmd/gad/dap_test.go`.

## 6. Project layout

```
plugins/intellij-gad/
├── PLAN.md                         # this file
├── README.md                       # build/usage docs
├── build.gradle.kts                # IntelliJ Platform Gradle plugin 2.x
├── settings.gradle.kts
├── gradle.properties               # platform version, plugin version/id
├── gradle/wrapper/…                # pinned Gradle wrapper
└── src/main/
    ├── kotlin/dev/gad/intellij/
    │   ├── lang/                    # file types, language, icons
    │   ├── highlight/               # TextMate bundle provider
    │   ├── config/                  # JSON-schema providers for .gad.yaml/.gadide.yaml
    │   ├── run/                     # ConfigurationType/Factory/State, profiles, settings UI
    │   ├── debug/                   # XDebugProcess, DAP client, breakpoints, frames, values, evaluator
    │   └── settings/               # Configurable(s)
    └── resources/
        ├── META-INF/plugin.xml      # all extension registrations
        ├── icons/                   # .gad/.gadt/.gadx file icons
        ├── textmate/                # grammars copied from plugins/vscode-gad (build step)
        └── schemas/                 # config schemas copied from plugins/vscode-gad (build step)
```

`textmate/` and `schemas/` are **copied at build time** from `plugins/vscode-gad`
by a Gradle `Copy` task, so there is a single source of truth for grammars and
schemas across the VS Code and JetBrains plugins.

## 7. Build, run, verify

- `./gradlew buildPlugin` → distributable `.zip` in `build/distributions/`.
- `./gradlew runIde` → launch a sandbox IDE with the plugin for manual testing.
- `./gradlew verifyPlugin` → JetBrains **Plugin Verifier** (API-compat across the
  declared IDE range).
- `./gradlew test` → unit tests (DAP-client message mapping, breakpoint/condition
  translation, frame/value adapters — testable without a running IDE).
- **Makefile**: add `build-intellij-plugin` (mirrors `build-vscode-plugin`) that
  runs `./gradlew buildPlugin` and copies the `.zip` into `dist/`. Wire into
  `dist` once the plugin builds green.
- **Note**: the IntelliJ Platform Gradle plugin downloads the IDE SDK (~1 GB) on
  first build; that download is not part of this repo's Go CI. The plugin is
  built/verified locally and in a dedicated JetBrains CI job.

## 8. Milestones

| Phase | Deliverable | Depends on |
| --- | --- | --- |
| **0 — scaffold** | Gradle project, `plugin.xml`, file types + icons, TextMate highlighting for the 3 dialects; opens/colors files. | — |
| **1 — run** | Run configuration + execution-profile form, settings (gad path), `.gad.yaml`/`.gadide.yaml` schema mapping. | §5.2 |
| **2 — debug MVP** | `XDebugProcess` over DAP: launch, line breakpoints, resume/step in-over-out, call stack, Variables tree, evaluate/watches, console. | §5.2 |
| **3 — full debug** | Conditional breakpoints, **cross-file navigation** into imports, on-hover evaluate, (optional) editable variables. | §5.1 (+5.3) |
| **4 — ship** | Plugin Verifier green, README, marketplace metadata, `build-intellij-plugin` in the Makefile, first Marketplace draft. | 0–3 |

## 9. Distribution

- Packaged as a Marketplace `.zip` (`buildPlugin`); `publishPlugin` with a
  `PUBLISH_TOKEN` for the JetBrains Marketplace.
- Compatible IDE range pinned in `gradle.properties` (`sinceBuild`/`untilBuild`);
  verified by the Plugin Verifier before each release.
- Versioning tracks the Gad release (same `v0.1.0-rc.N` cadence).

## 10. Risks / open questions

- **DAP source paths** must be absolute and match what IntelliJ resolves to a
  `VirtualFile`; relative/module paths need normalization (§5.1).
- **TextMate injection** gives coloring but not semantic features (rename, go-to);
  an LSP server would add those later — out of scope for v1 (a future `gad lsp`
  could reuse the same client bridge).
- **Platform version drift**: XDebugger APIs are stable but the IntelliJ Platform
  Gradle plugin 2.x and `sinceBuild` must be pinned and verified.
- **SDK download** makes CI heavier; runs in a separate JetBrains job, not Go CI.
