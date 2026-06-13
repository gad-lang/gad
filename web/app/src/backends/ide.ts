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
}

export interface DebugFrame {
  name: string;
  line: number;
  column: number;
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
  line?: number;
  column?: number;
  frames?: DebugFrame[];
  locals?: DebugVariable[];
  output?: string;
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
  tree: () => jsonFetch<TreeNode>("GET", "api/ide/tree"),
  read: (path: string) =>
    jsonFetch<{ path: string; content: string }>(
      "GET",
      "api/ide/file?path=" + encodeURIComponent(path),
    ),
  write: (path: string, content: string) =>
    jsonFetch<{ path: string }>("PUT", "api/ide/file", { path, content }),
  mkfile: (path: string) => jsonFetch<{ path: string }>("PUT", "api/ide/file", { path, content: "" }),
  del: (path: string) => jsonFetch<{ path: string }>("POST", "api/ide/delete", { path }),
  config: () => jsonFetch<Record<string, unknown>>("GET", "api/ide/config"),
  saveConfig: (doc: Record<string, unknown>) =>
    jsonFetch<Record<string, unknown>>("PUT", "api/ide/config", doc),
  modules: () => jsonFetch<ModuleInfo[]>("GET", "api/ide/modules"),
  format: (source: string) => jsonFetch<FormatResult>("POST", "api/ide/format", { source }),
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
  }) => jsonFetch<RunResult>("POST", "api/ide/run", req),
  dbgStart: (req: { source: string; breakpoints: number[]; stopOnEntry: boolean }) =>
    jsonFetch<DebugResponse>("POST", "api/ide/debug/start", req),
  dbgCmd: (session: string, command: string) =>
    jsonFetch<DebugResponse>("POST", "api/ide/debug/command", { session, command }),
};

/** probeIde resolves true when the IDE backend is reachable (served by gad ide). */
export async function probeIde(): Promise<Workspace | null> {
  try {
    return await ideApi.workspace();
  } catch {
    return null;
  }
}
