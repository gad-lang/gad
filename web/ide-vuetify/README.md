# @gad-lang/ide-vuetify

A reusable **Vuetify 3** IDE component for the [Gad](https://github.com/gad-lang/gad)
language: file explorer, CodeMirror editor, and Run / Doc / Debug panels with a
stepping debugger. It is the Vuetify counterpart of
[`@gad-lang/ide-react`](../ide-react) and shares the same **backend-agnostic**
`IdeApi` contract, so the same in-browser backend (Gad WASM + a LocalStorage
filesystem) can drive either UI.

## Install

```sh
bun add @gad-lang/ide-vuetify vue vuetify @gad-lang/codemirror-gad @gad-lang/prism-gad
```

`vue`, `vuetify`, `@gad-lang/codemirror-gad` and `@gad-lang/prism-gad` are peer
dependencies.

## Usage

```vue
<script setup lang="ts">
import { GadIde, httpIdeApi, type Workspace } from "@gad-lang/ide-vuetify";
import "@gad-lang/ide-vuetify/style.css";

const workspace: Workspace = await httpIdeApi.workspace();
</script>

<template>
  <GadIde :api="httpIdeApi" :workspace="workspace" :dark="true" />
</template>
```

- **`api`** — any `IdeApi` implementation. `httpIdeApi` talks to a `gad ide`
  server; pass a fully in-browser one (WASM + LocalStorage) for a server-less IDE
  (see [`demo/`](./demo)).
- **`workspace`** — the initial `{ root, name, openFile }`.
- **`dark`** — light/dark editor theme.
- **`onReset`** — optional callback; when given, a **Reset** button appears
  (restore a backend's pristine state).

The scoped component styles are extracted to `dist/style.css`; import it once as
shown above.

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
