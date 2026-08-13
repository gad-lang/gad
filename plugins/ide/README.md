# Gad IDE extensions

Editor/IDE extensions that bring Gad support (`.gad` / `.gadt` / `.gadx`) to
desktop editors. Each reuses the Gad CLI's own protocols rather than
reimplementing them: the `gad debug --dap` Debug Adapter for debugging and the
TextMate grammars in [`vscode-gad/syntaxes`](vscode-gad/syntaxes) for
highlighting.

| Extension | Target | Status |
| --- | --- | --- |
| [`vscode-gad`](vscode-gad) | VS Code (and forks) | Shipping — TextMate grammars + language client + DAP debugging. |
| [`intellij-gad`](intellij-gad) | IntelliJ Platform (IntelliJ IDEA, GoLand, WebStorm, PyCharm, …) | Planned — see [`intellij-gad/PLAN.md`](intellij-gad/PLAN.md). |

## VS Code — `vscode-gad`

The debug adapter is the `gad` CLI itself (`gad debug --dap`, DAP over stdio);
the extension wires it into VS Code alongside the TextMate grammars, formatting
(`gad fmt -`) and `.gad.yaml` / `.gadide.yaml` schema validation.

```sh
go run ./cmd/update-vscode-plugin -w   # regenerate the Gad TextMate grammar
cd plugins/ide/vscode-gad
bun install
bun run package                        # compile + produce vscode-gad.vsix
```

`make build-vscode-plugin` runs these steps and moves the `.vsix` into `dist/`.

## IntelliJ Platform — `intellij-gad`

A planned Gradle/Kotlin plugin for JetBrains IDEs providing highlighting, run
configurations with execution profiles, and a full DAP-backed debugger
(breakpoints incl. conditional, call stack, inspect, evaluate, cross-file
navigation). Marketplace display name **Gad Language**. Scope, architecture and
roadmap: [`intellij-gad/PLAN.md`](intellij-gad/PLAN.md).

---

See [`../README.md`](../README.md) for the full plugin tree (these IDE extensions
plus the JS libraries under [`../js`](../js)).
