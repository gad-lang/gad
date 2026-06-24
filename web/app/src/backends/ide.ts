// Client for the `gad ide` backend (/api/ide/*). Used only when the app is
// served by `gad ide` (detected via probeIde).
import type { GadDiagnostic } from "@gad-lang/codemirror-gad";
import type { FormatResult, RunResult } from "./types";

export interface Workspace {
  root: string;
  name: string;
  openFile: string;
}

export interface TreeNode {
  name: string;
  path: string;
  dir: boolean;
  children?: TreeNode[];
}

export interface ModuleInfo {
  name: string;
  unsafe: boolean;
}

export interface RunConfig {
  args: string[];
  disabled: string[];
  safe: boolean;
  saveOut: string;
  /** Per-stream output capture; when combine is set both go to saveStdout. */
  saveStdout?: string;
  saveStderr?: string;
  combine?: boolean;
}

export interface DocComment {
  line: number;
  kind: string;
  title: string;
  content: string;
}

export interface BreakpointSpec {
  line: number;
  disabled?: boolean;
  condition?: string;
}

/** Per-line breakpoint metadata, keyed by line number. */
export type BreakpointMeta = Record<number, { disabled?: boolean; condition?: string }>;

export interface EvalResult {
  ok: boolean;
  value: string;
  error: string;
  stdout: string;
}

export interface InspectEntry {
  key: string;
  accessor: string;
  type: string;
  value: string;
  expandable: boolean;
}

export interface InspectResult {
  type: string;
  value: string;
  expandable: boolean;
  entries: InspectEntry[];
}

export interface DebugFrame {
  name: string;
  file: string;
  line: number;
  column: number;
  locals: DebugVariable[];
}

export interface DebugVariable {
  name: string;
  type: string;
  value: string;
}

export interface DebugResponse {
  session?: string;
  state: "stopped" | "terminated" | "error";
  reason?: string;
  file?: string; // workspace-relative file of the current stop
  line?: number;
  column?: number;
  frames?: DebugFrame[];
  locals?: DebugVariable[];
  output?: string;
  stdout?: string;
  stderr?: string;
  result?: string;
  error?: string;
  diagnostics?: GadDiagnostic[];
}

async function jsonFetch<T>(method: string, url: string, body?: unknown): Promise<T> {
  const r = await fetch(url, {
    method,
    headers: body ? { "Content-Type": "application/json" } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  });
  const data = await r.json().catch(() => ({}));
  if (!r.ok) throw new Error((data as { error?: string }).error || r.statusText);
  return data as T;
}

export const ideApi = {
  workspace: () => jsonFetch<Workspace>("GET", "api/ide/workspace"),
  tree: (hidden = false) =>
    jsonFetch<TreeNode>("GET", "api/ide/tree" + (hidden ? "?hidden=true" : "")),
  read: (path: string) =>
    jsonFetch<{ path: string; content: string }>(
      "GET",
      "api/ide/file?path=" + encodeURIComponent(path),
    ),
  write: (path: string, content: string) =>
    jsonFetch<{ path: string }>("PUT", "api/ide/file", { path, content }),
  mkfile: (path: string) => jsonFetch<{ path: string }>("PUT", "api/ide/file", { path, content: "" }),
  del: (path: string) => jsonFetch<{ path: string }>("POST", "api/ide/delete", { path }),
  rename: (path: string, to: string) =>
    jsonFetch<{ path: string }>("POST", "api/ide/rename", { path, to }),
  mkdir: (path: string) => jsonFetch<{ path: string }>("POST", "api/ide/mkdir", { path }),
  fetchUrl: (url: string, path: string) =>
    jsonFetch<{ path: string; size: number }>("POST", "api/ide/fetch", { url, path }),
  config: () => jsonFetch<Record<string, unknown>>("GET", "api/ide/config"),
  saveConfig: (doc: Record<string, unknown>) =>
    jsonFetch<Record<string, unknown>>("PUT", "api/ide/config", doc),
  modules: () => jsonFetch<ModuleInfo[]>("GET", "api/ide/modules"),
  format: (source: string) => jsonFetch<FormatResult>("POST", "api/ide/format", { source }),
  transpile: (source: string, path?: string) =>
    jsonFetch<FormatResult>("POST", "api/ide/transpile", { source, path }),
  doc: (source: string) =>
    jsonFetch<{ docs: DocComment[] }>("POST", "api/ide/doc", { source }).then((r) => r.docs || []),
  eval: (req: { expr: string; repr?: boolean; source?: string; path?: string }) =>
    jsonFetch<EvalResult>("POST", "api/ide/eval", req),
  inspect: (req: { expr: string; session?: string; source?: string; path?: string }) =>
    jsonFetch<{ ok: boolean; inspect?: InspectResult; error?: string }>("POST", "api/ide/inspect", req),
  diagnose: (source: string) =>
    jsonFetch<{ diagnostics: GadDiagnostic[] }>("POST", "api/ide/diagnose", { source }).then(
      (r) => r.diagnostics || [],
    ),
  run: (req: {
    path?: string;
    source?: string;
    args?: string[];
    disabled?: string[];
    safe?: boolean;
    saveOut?: string;
    saveStdout?: string;
    saveStderr?: string;
    combine?: boolean;
  }) => jsonFetch<RunResult>("POST", "api/ide/run", req),
  dbgStart: (req: {
    source: string;
    breakpoints: number[];
    breakpointSpecs?: BreakpointSpec[];
    stopOnEntry: boolean;
    path?: string;
    args?: string[];
    disabled?: string[];
    safe?: boolean;
  }) => jsonFetch<DebugResponse>("POST", "api/ide/debug/start", req),
  dbgCmd: (session: string, command: string) =>
    jsonFetch<DebugResponse>("POST", "api/ide/debug/command", { session, command }),
  dbgEval: (session: string, expr: string, repr: boolean) =>
    jsonFetch<{ ok: boolean; value?: string; error?: string }>("POST", "api/ide/debug/eval", {
      session,
      expr,
      repr,
    }),
};

/** probeIde resolves true when the IDE backend is reachable (served by gad ide). */
export async function probeIde(): Promise<Workspace | null> {
  try {
    return await ideApi.workspace();
  } catch {
    return null;
  }
}
