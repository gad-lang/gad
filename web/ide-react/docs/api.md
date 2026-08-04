# @gad-lang/ide-react — API reference

## `<Ide>` component

```tsx
import { Ide } from "@gad-lang/ide-react";
```

| Prop | Type | Default | Description |
| --- | --- | --- | --- |
| `api` | `IdeApi` | `httpIdeApi` | Backend implementation (the extension point). |
| `workspace` | `Workspace` | — | Initial `{ root, name, openFile }`. |
| `layoutConfig` | `unknown` | — | Serialized dockview layout to restore. |
| `onLayoutConfigChange` | `(layout: unknown) => void` | — | Fires when the layout changes. |
| `runProfiles` | `RunProfile[]` | `[]` | Named run/debug configurations. |
| `onRunProfilesChange` | `(profiles: RunProfile[]) => void` | — | Fires when profiles change. |
| `runMode` | `RunMode` | `"debug"` | Gates the Run / Debug / profile actions. |

### Types

```ts
interface Workspace { root: string; name: string; openFile: string }
interface RunProfile { name: string; path: string; args: string[] }
type RunMode = "none" | "run" | "debug" | "";
```

`runMode` semantics: `"none"`/`""` disables Run, Debug and the profile selector;
`"run"` enables Run (and the profile selector); `"debug"` also enables Debug.

## The `IdeApi` contract

`<Ide>` drives everything through the injected `api`. Implement this interface to
back the IDE with anything — the built-in HTTP client (`httpIdeApi`, a `gad ide`
server) or a fully in-browser WASM + LocalStorage backend.

### Workspace & files

| Method | Signature | Description |
| --- | --- | --- |
| `workspace` | `() => Promise<Workspace>` | The current workspace descriptor. |
| `tree` | `(hidden?: boolean) => Promise<TreeNode>` | The file tree (root node). |
| `read` | `(path) => Promise<{ path; content }>` | Read a file. |
| `write` | `(path, content) => Promise<{ path }>` | Write a file. |
| `mkfile` | `(path) => Promise<{ path }>` | Create an empty file. |
| `mkdir` | `(path) => Promise<{ path }>` | Create a directory. |
| `del` | `(path) => Promise<{ path }>` | Delete a file/dir. |
| `rename` | `(path, to) => Promise<{ path }>` | Rename/move. |
| `fetchUrl` | `(url, path) => Promise<{ path; size }>` | Download a URL into a file. |

```ts
interface TreeNode { name: string; path: string; dir: boolean; children?: TreeNode[] }
```

### Config & modules

| Method | Signature | Description |
| --- | --- | --- |
| `config` | `() => Promise<Record<string, unknown>>` | The workspace config document. |
| `saveConfig` | `(doc) => Promise<Record<string, unknown>>` | Persist the config. |
| `modules` | `() => Promise<ModuleInfo[]>` | Importable modules (`{ name, unsafe }`). |

### Language

| Method | Signature | Description |
| --- | --- | --- |
| `format` | `(source) => Promise<FormatResult>` | Format Gad source. |
| `transpile` | `(source, path?) => Promise<FormatResult>` | Lower a template to plain Gad. |
| `diagnose` | `(source) => Promise<GadDiagnostic[]>` | Positioned errors/warnings. |
| `doc` | `(source) => Promise<DocComment[]>` | Extract doc comments. |
| `eval` | `(req) => Promise<EvalResult>` | Evaluate an expression (fresh or in a paused frame). |
| `inspect` | `(req) => Promise<{ ok; inspect?: InspectResult; error? }>` | Tree-navigator description of a value. |

```ts
interface FormatResult { ok: boolean; source: string; diagnostics: GadDiagnostic[] }
interface RunResult { ok: boolean; stdout: string; stderr: string; result: string; diagnostics: GadDiagnostic[] }
interface DocComment { line: number; kind: string; title: string; content: string }
interface EvalResult { ok: boolean; value: string; error: string; stdout: string }
interface InspectResult { type: string; value: string; expandable: boolean; entries: InspectEntry[] }
interface InspectEntry { key: string; accessor: string; type: string; value: string; expandable: boolean }
```

### Run

| Method | Signature | Description |
| --- | --- | --- |
| `run` | `(req: { path?; source?; args?; … }) => Promise<RunResult>` | Compile & run; `args` are parsed into the script's `param (…)` (typed). |

### Debug

| Method | Signature | Description |
| --- | --- | --- |
| `dbgStart` | `(req: { source; breakpoints; stopOnEntry; path?; args?; … }) => Promise<DebugResponse>` | Start a session; run to the first stop. |
| `dbgCmd` | `(session, command) => Promise<DebugResponse>` | `continue` / `next` / `stepIn` / `stepOut` / `pause`. |
| `dbgEval` | `(session, expr, repr) => Promise<{ ok; value?; error? }>` | Evaluate in the paused frame. |

```ts
interface DebugResponse {
  session?: string;
  state: "stopped" | "terminated" | "error";
  reason?: string; file?: string; line?: number; column?: number;
  frames?: DebugFrame[]; locals?: DebugVariable[];
  output?: string; stdout?: string; stderr?: string; result?: string; error?: string;
  diagnostics?: GadDiagnostic[];
}
interface DebugFrame { name: string; file: string; line: number; column: number; locals: DebugVariable[] }
interface DebugVariable { name: string; type: string; value: string }
```

## Built-in HTTP client

```ts
import { httpIdeApi, probeIde } from "@gad-lang/ide-react";

const ws = await probeIde();      // Workspace | null (null when no gad ide server)
```

`httpIdeApi` implements `IdeApi` over a `gad ide` server's `/api/ide/*` endpoints.

## Building blocks

`Editor` (`EditorHandle`, `EditorLanguage`, `TemplateDelimiters`), `ReadonlyCode`,
`GadInput`, `useTheme`, `renderDocMarkdown`, `InspectDialog` (`InspectFn`).
