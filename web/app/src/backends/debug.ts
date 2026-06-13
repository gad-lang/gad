import type { GadDiagnostic } from "@gad-lang/codemirror-gad";

export interface DebugFrame {
  name: string;
  line: number;
  column: number;
}

export interface DebugVar {
  name: string;
  type: string;
  value: string;
}

export type DebugState = "stopped" | "terminated" | "error";

export interface DebugResponse {
  session?: string;
  state: DebugState;
  reason?: string;
  line?: number;
  column?: number;
  frames?: DebugFrame[];
  locals?: DebugVar[];
  output?: string;
  result?: string;
  error?: string;
  diagnostics?: GadDiagnostic[];
}

export type DebugCommand = "continue" | "next" | "stepIn" | "stepOut" | "pause";

async function post<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) throw new Error(`${path}: HTTP ${res.status}`);
  return (await res.json()) as T;
}

/**
 * The debugger is request/response: start launches a session and runs to the
 * first stop (or end); command resumes to the next stop (or end). It is served
 * by the Go server only (/api/debug/*), so this page requires `make web-server`.
 */
export const debugBackend = {
  start(source: string, breakpoints: number[], stopOnEntry: boolean): Promise<DebugResponse> {
    return post<DebugResponse>("/api/debug/start", { source, breakpoints, stopOnEntry });
  },
  command(session: string, command: DebugCommand): Promise<DebugResponse> {
    return post<DebugResponse>("/api/debug/command", { session, command });
  },
};
