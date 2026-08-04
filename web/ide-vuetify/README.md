# @gad-lang/ide-vuetify

A reusable **Vuetify 3** IDE component for the [Gad](https://github.com/gad-lang/gad)
language: file explorer, CodeMirror editor, and Run / Doc / Debug panels with a
stepping debugger. It is the Vuetify counterpart of
[`@gad-lang/ide-react`](../ide-react) and shares the same **backend-agnostic**
`IdeApi` contract, so the same in-browser backend (Gad WASM + a LocalStorage
filesystem) can drive either UI.

> **Status:** not yet published to npm. The install command below is the intended
> usage once the `@gad-lang` packages are released.

## Install

```sh
npm install @gad-lang/ide-vuetify vue vuetify dockview-vue @gad-lang/codemirror-gad @gad-lang/prism-gad
# or: bun add @gad-lang/ide-vuetify vue vuetify dockview-vue @gad-lang/codemirror-gad @gad-lang/prism-gad
```

`vue`, `vuetify`, `dockview-vue`, `@gad-lang/codemirror-gad` and
`@gad-lang/prism-gad` are peer dependencies.

The views are authored in **TSX** (`defineComponent` + JSX), not `.vue` SFCs, so
the whole package — panels, controller and the `IdeApi` contract — is fully
typed. The panels are laid out with **dockview-vue** (resizable, movable,
dockable, tabbable).

## Usage

```tsx
import { defineComponent, ref, onMounted } from "vue";
import { GadIde, httpIdeApi, type SerializedDockview, type Workspace } from "@gad-lang/ide-vuetify";

// Import the stylesheets once in the host app:
import "@gad-lang/ide-vuetify/style.css";
import "dockview-core/dist/styles/dockview.css";
import "vuetify/styles";

export default defineComponent(() => {
  const workspace = ref<Workspace>();
  const layout = ref<SerializedDockview | null>(null);   // v-model:layoutConfig
  const config = ref<Record<string, unknown>>({});       // v-model:config
  onMounted(async () => (workspace.value = await httpIdeApi.workspace()));

  return () =>
    workspace.value && (
      <GadIde
        api={httpIdeApi}
        workspace={workspace.value}
        dark
        layoutConfig={layout.value}
        {...{ "onUpdate:layoutConfig": (v: SerializedDockview) => (layout.value = v) }}
        config={config.value}
        {...{ "onUpdate:config": (v: Record<string, unknown>) => (config.value = v) }}
      />
    );
});
```

- **`api`** — any `IdeApi` implementation. `httpIdeApi` talks to a `gad ide`
  server; pass a fully in-browser one (WASM + LocalStorage) for a server-less IDE
  (see [`demo/`](./demo)).
- **`workspace`** — the initial `{ root, name, openFile }`.
- **`dark`** — light/dark editor + dockview theme.
- **`onReset`** — optional callback; when given, a **Reset** button appears
  (restore a backend's pristine state).
- **`v-model:layoutConfig`** — the dockview layout (its `toJSON()` shape). Bind it
  to persist and restore the panel arrangement (the demo stores it in
  LocalStorage).
- **`v-model:config`** — the project settings document edited by the **Settings**
  dialog (Panels / Formatter / Transpile / Template).

Import the component styles (`@gad-lang/ide-vuetify/style.css`), the dockview
theme (`dockview-core/dist/styles/dockview.css`) and Vuetify's styles once in the
host app.

## Demo

[`demo/`](./demo) is a standalone, **server-less** app (like the React
`webide.html`): the sample tree is read-only, user edits/creates/deletes live in
LocalStorage, and the Gad VM runs in a Web Worker.

```sh
cd demo
bun install
bun run wasm    # build gad.wasm + copy wasm_exec.js into public/
bun run dev
```
