// @gad-lang/ide-vuetify — a reusable Vuetify 3 IDE for the Gad language.
//
// <GadIde> renders the full workspace UI (file explorer, CodeMirror editor,
// run/format/doc panels and a stepping debugger) and is backend-agnostic: pass
// any `IdeApi` implementation via the `api` prop. The default `httpIdeApi` talks
// to a `gad ide` server; a fully in-browser backend (WASM + LocalStorage) can be
// supplied instead. Requires Vue 3 and Vuetify 3 as peer dependencies.
export { default as GadIde } from "./GadIde.vue";
export { default as GadEditor } from "./GadEditor.vue";
export { default as InspectorNode, type InspectFn } from "./InspectorNode.vue";

// Backend contract + the built-in HTTP implementation.
export { httpIdeApi, ideApi, probeIde } from "./api";
export type {
  IdeApi,
  Workspace,
  TreeNode,
  ModuleInfo,
  DocComment,
  BreakpointSpec,
  EvalResult,
  InspectEntry,
  InspectResult,
  DebugFrame,
  DebugVariable,
  DebugResponse,
} from "./api";
export type { FormatResult, RunResult } from "./types";

// CodeMirror building blocks (for composing a custom shell).
export {
  GadEditorView,
  langExtension,
  langOf,
  type EditorLanguage,
  type TemplateDelimiters,
  type LocalVar,
} from "./codemirror";
export { renderDocMarkdown } from "./docMarkdown";
