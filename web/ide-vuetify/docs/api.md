# @gad-lang/ide-vuetify — API reference

## `<GadIde>` component

```tsx
import { GadIde } from "@gad-lang/ide-vuetify";
```

| Prop / v-model | Type | Default | Description |
| --- | --- | --- | --- |
| `api` | `IdeApi` | — | Backend implementation (the extension point). |
| `workspace` | `Workspace` | — | Initial `{ root, name, openFile }`. |
| `dark` | `boolean` | `false` | Light/dark editor + dockview theme. |
| `onReset` | `() => void \| Promise<void>` | — | When set, a **Reset** button appears. |
| `onToggleTheme` | `() => void` | — | When set, a light/dark toggle appears next to Settings. |
| `v-model:layoutConfig` | `SerializedDockview \| null` | `null` | The dockview layout (`toJSON()`/`fromJSON()`). |
| `v-model:config` | `Record<string, unknown>` | `{}` | The Settings document (Panels / Formatter / Transpile / Template). |
| `v-model:runProfiles` | `RunProfile[]` | `[]` | Named run/debug configurations. |
| `v-model:runMode` | `RunMode` | `"debug"` | Gates the Run / Debug / profile actions. |

### Types

```ts
interface Workspace { root: string; name: string; openFile: string }
interface RunProfile { name: string; path: string; args: string[] }
type RunMode = "none" | "run" | "debug" | "";
```

`runMode` semantics: `"none"`/`""` disables Run, Debug and the profile selector;
`"run"` enables Run (and the profile selector); `"debug"` also enables Debug.

## Panels

The workspace is a **dockview-vue** layout — resizable, movable, dockable and
tabbable: **Explorer**, **Editor**, **Docs**, **Call Stack**, **Locals** (with an
Evaluate box and a value **inspector**) and **Output**. The Run / Debug / Format /
Doc / Settings actions and the debugger step controls live in the top toolbar,
with a JetBrains-style "…" run-profile selector.

## The `IdeApi` contract

`<GadIde>` never talks to a server directly — it calls methods on the injected
`api`. It is the **same contract** as
[`@gad-lang/ide-react`](../../ide-react/docs/api.md): workspace/files
(`workspace`, `tree`, `read`, `write`, `mkfile`, `mkdir`, `del`, `rename`,
`fetchUrl`), config (`config`, `saveConfig`, `modules`), language (`format`,
`transpile`, `diagnose`, `doc`, `eval`, `inspect`), run (`run`) and debug
(`dbgStart`, `dbgCmd`, `dbgEval`). See the ide-react
[API reference](../../ide-react/docs/api.md) for the full method signatures and
result types.

`httpIdeApi` implements it over a `gad ide` server; `probeIde()` resolves the
workspace when such a server is reachable, else `null`.

## Styles

Import once in the host app:

```ts
import "@gad-lang/ide-vuetify/style.css";
import "dockview-core/dist/styles/dockview.css";
import "vuetify/styles";
```

## Exports

- Components: `GadIde`, `GadEditor`, `InspectorNode` (`InspectFn`).
- Backend: `httpIdeApi`, `ideApi`, `probeIde`, and the `IdeApi` / `Workspace` /
  `RunProfile` / `RunMode` / `DebugResponse` / … types.
- CodeMirror building blocks: `GadEditorView`, `langExtension`, `langOf`
  (`EditorLanguage`, `TemplateDelimiters`, `LocalVar`), `renderDocMarkdown`.
- Controller: `createController`, `IdeControllerKey`, `IdeController` (for a
  custom shell that composes the panels itself).
- Layout: `SerializedDockview` (for typing `v-model:layoutConfig`).
