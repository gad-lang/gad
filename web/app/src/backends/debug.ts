// The debug wire types are the canonical ones from the reusable IDE package, so
// the playground Debug page and the WASM worker client share one definition
// (avoids two structurally-different DebugResponse types across the boundary).
import type { DebugResponse } from "@gad-lang/ide-react";
export type {
  DebugFrame,
  DebugVariable,
  DebugVariable as DebugVar,
  DebugResponse,
} from "@gad-lang/ide-react";

export type DebugState = "stopped" | "terminated" | "error";

export type DebugCommand = "continue" | "next" | "stepIn" | "stepOut" | "pause";

/**
 * DebugBackend abstracts the stepping debugger so the same UI works against the
 * Go HTTP server (/api/debug/*) or the in-browser WebAssembly module running in
 * a Web Worker. The protocol is request/response: start launches a session and
 * runs to the first stop (or end); command resumes to the next stop (or end).
 */
export interface DebugBackend {
  readonly name: string;
  /** Whether this backend needs a running Go server (used for hints/UI). */
  readonly needsServer: boolean;
  start(source: string, breakpoints: number[], stopOnEntry: boolean): Promise<DebugResponse>;
  command(session: string, command: DebugCommand): Promise<DebugResponse>;
  /** Evaluate an expression in the paused session's top frame, when supported. */
  evaluate?(session: string, expr: string, repr?: boolean): Promise<{ ok: boolean; value?: string; error?: string }>;
  /** Best-effort abort of a running/paused session. */
  stop?(session: string): void | Promise<void>;
}

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
 * serverDebugBackend is served by the Go server only (/api/debug/*), so it
 * requires `make web-server`.
 */
export const serverDebugBackend: DebugBackend = {
  name: "Go server",
  needsServer: true,
  start(source, breakpoints, stopOnEntry) {
    return post<DebugResponse>("/api/debug/start", { source, breakpoints, stopOnEntry });
  },
  command(session, command) {
    return post<DebugResponse>("/api/debug/command", { session, command });
  },
};

/** @deprecated use serverDebugBackend. */
export const debugBackend = serverDebugBackend;
