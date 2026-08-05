// @gad-lang/ide-react — a reusable React IDE for the Gad language.
//
// The <Ide> component renders the full workspace UI (file tree, tabbed
// multi-language editor, run/format/doc panels and a stepping debugger) and is
// backend-agnostic: pass any `IdeApi` implementation via the `api` prop. The
// default is `httpIdeApi`, which talks to a `gad ide` server; a fully in-browser
// backend (WASM + LocalStorage) can be supplied instead. See the README.

export { Ide } from "./Ide";
export { GadPlayground, type GadPlaygroundProps } from "./GadPlayground";
export { GadNotebook, type GadNotebookProps } from "./GadNotebook";
export { FileTypeRegistry, DEFAULT_FILE_TYPES, type FileTypeHandler } from "./fileTypes";

// Backend contract + the built-in HTTP implementation.
export { httpIdeApi, ideApi, probeIde } from "./api";
export type {
  IdeApi,
  Workspace,
  TreeNode,
  ModuleInfo,
  RunConfig,
  DocComment,
  BreakpointSpec,
  RunProfile,
  RunMode,
  BreakpointMeta,
  EvalResult,
  InspectEntry,
  InspectResult,
  DebugFrame,
  DebugVariable,
  DebugResponse,
} from "./api";
export type { FormatResult, RunResult, GadRunner } from "./types";

// Editor and supporting building blocks (useful when composing a custom shell).
export { Editor } from "./Editor";
export type { EditorHandle, EditorLanguage, TemplateDelimiters } from "./Editor";
export type { LocalVar } from "./debugDecorations";
export { ReadonlyCode } from "./ReadonlyCode";
export { GadInput } from "./GadInput";
export { useTheme } from "./useTheme";
export { renderDocMarkdown } from "./docMarkdown";
export { InspectDialog } from "./TreeNavigator";
export type { InspectFn } from "./TreeNavigator";
