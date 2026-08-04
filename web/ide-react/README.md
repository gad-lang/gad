# @gad-lang/ide-react

A reusable **React** IDE component for the [Gad](https://github.com/gad-lang/gad)
language: a dockview workspace (file explorer, multi-language CodeMirror editor,
run / format / doc panels) and a full **stepping debugger** with call stack,
locals, watches, a value **inspector** (tree navigator) and JetBrains-style named
**run profiles**. It is **backend-agnostic**: drive it with any `IdeApi`
implementation — the built-in HTTP client that talks to a `gad ide` server, or a
fully in-browser one (Gad WASM + a LocalStorage filesystem) for a **server-less**
IDE. `@gad-lang/ide-vuetify` is the Vuetify counterpart and shares the same
contract.

> **Status:** not yet published to npm. The install commands below are the
> intended usage once the `@gad-lang` packages are released.

## Install

```sh
npm install @gad-lang/ide-react react react-dom @mui/material @mui/icons-material dockview-react \
  @gad-lang/codemirror-gad @gad-lang/prism-gad
# or: bun add @gad-lang/ide-react react react-dom @mui/material @mui/icons-material dockview-react \
#   @gad-lang/codemirror-gad @gad-lang/prism-gad
```

`react`, `react-dom`, `@mui/material`, `@mui/icons-material`, `dockview-react`,
`@gad-lang/codemirror-gad` and `@gad-lang/prism-gad` are **peer dependencies**.

## Usage

```tsx
import { useEffect, useState } from "react";
import { Ide, httpIdeApi, type Workspace } from "@gad-lang/ide-react";
import "dockview-react/dist/styles/dockview.css";

export function App() {
  const [workspace, setWorkspace] = useState<Workspace | null>(null);
  useEffect(() => { httpIdeApi.workspace().then(setWorkspace); }, []);
  return workspace ? <Ide api={httpIdeApi} workspace={workspace} /> : null;
}
```

That drives the IDE against a running `gad ide` server (which serves the
`/api/ide/*` endpoints `httpIdeApi` calls). For a **server-less** IDE, pass an
in-browser `IdeApi` instead — see [Server-less backend](#server-less-backend).

## `<Ide>` props

| Prop | Type | Description |
| --- | --- | --- |
| `api` | `IdeApi` | Backend implementation. Defaults to `httpIdeApi`. |
| `workspace` | `Workspace` | Initial `{ root, name, openFile }`. |
| `layoutConfig` | `unknown` | Initial dockview layout to restore (overrides the config's saved panels). |
| `onLayoutConfigChange` | `(layout) => void` | Called whenever the layout changes, so the host can persist it. |
| `runProfiles` | `RunProfile[]` | Named run/debug configurations (`{ name, path, args }`). |
| `onRunProfilesChange` | `(profiles) => void` | Called when a profile is added/removed. |
| `runMode` | `"none" \| "run" \| "debug"` | Gates the Run / Debug / profile actions. `"run"` enables Run (and the profile selector); `"debug"` also enables Debug; `"none"`/`""` disables all three. Defaults to `"debug"`. |

`RunProfile` and `RunMode` and the layout callback are the React analogue of the
Vuetify component's `v-model`s (`layoutConfig`, `runProfiles`, `runMode`).

## The `IdeApi` contract

`<Ide>` never talks to a server directly — it calls methods on the injected
`api`. That is the whole extension point: implement `IdeApi` and the same UI
edits, runs, documents and debugs any workspace. The contract (see
[`docs/api.md`](./docs/api.md) for the full signatures) covers:

- **Workspace & files** — `workspace`, `tree`, `read`, `write`, `mkfile`,
  `mkdir`, `del`, `rename`, `fetchUrl`.
- **Config** — `config`, `saveConfig`, `modules`.
- **Language** — `format`, `transpile`, `diagnose`, `doc`, `eval`, `inspect`.
- **Run** — `run` (with typed `param (…)` args).
- **Debug** — `dbgStart`, `dbgCmd`, `dbgEval`.

`httpIdeApi` is the built-in implementation over a `gad ide` server; `probeIde()`
resolves the workspace when such a server is reachable, else `null`.

```tsx
import { probeIde, Ide, type Workspace } from "@gad-lang/ide-react";

const ws: Workspace | null = await probeIde(); // null when no gad ide server
```

## Server-less backend

For an in-browser IDE with no Go server, supply an `IdeApi` backed by the Gad
**WebAssembly** module (running in a Web Worker) and a LocalStorage filesystem.
The gad repository ships a complete reference implementation (`localIdeApi`) used
by the standalone `webide.html`; it wires the same WASM bridge as `gad ide`
(`web/gadbridge`) into every `IdeApi` method — run/debug with typed args,
`inspect`, doc extraction, etc. Persist `runProfiles` and the layout via the
callbacks (the reference stores them in the workspace, e.g.
`.gad/run-profiles.yaml`).

## Building blocks

For composing a custom shell, the package also exports the editor and helpers:
`Editor` (+ `EditorHandle`, `EditorLanguage`), `ReadonlyCode`, `GadInput`,
`useTheme`, `renderDocMarkdown`, and `InspectDialog` (the value tree navigator).

## Documentation

- [API reference](./docs/api.md) — the full `IdeApi` contract and component props.

## Publishing

Published to npm under the public `@gad-lang` scope. The package ships `dist/`
(built from `src/` by `tsc`); `prepublishOnly` rebuilds it.

```sh
bun run build            # emit dist/ (tsc: .js + .d.ts)
npm version <patch|minor|major>
bun publish --dry-run
bun publish              # publishConfig sets the public registry + access
```
