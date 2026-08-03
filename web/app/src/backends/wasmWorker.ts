// Web-Worker-backed backends: run the Gad WebAssembly module off the UI thread
// (see ../wasm/worker.ts + ../wasm/client.ts) so a blocking program or a paused
// debug session never freezes the page. A single WasmClient (one worker) is
// shared by the run/format/diagnose backend and the stepping debugger.
import type { GadDiagnostic } from "@gad-lang/codemirror-gad";
import { WasmClient } from "../wasm/client";
import type { DebugBackend, DebugCommand, DebugResponse } from "./debug";
import type { FormatResult, GadBackend, RunResult } from "./types";

// The asset base for wasm_exec.js / gad.wasm. Vite serves them from the app
// root, so "/" (or import.meta.env.BASE_URL) is correct in dev and in a build.
const ASSET_BASE = (import.meta as { env?: { BASE_URL?: string } }).env?.BASE_URL || "/";

let client: WasmClient | null = null;

/** sharedClient returns the process-wide WASM worker client, creating it lazily. */
export function sharedClient(): WasmClient {
  if (!client) client = new WasmClient(ASSET_BASE);
  return client;
}

/** wasmWorkerBackend runs format/run/diagnose in the Web Worker. */
export const wasmWorkerBackend: GadBackend = {
  name: "WebAssembly (worker)",
  format: (source) => sharedClient().format(source) as Promise<FormatResult>,
  run: (source) => sharedClient().run(source) as Promise<RunResult>,
  diagnose: async (source): Promise<GadDiagnostic[]> =>
    (await sharedClient().diagnose(source)).diagnostics,
};

/**
 * wasmDebugBackend drives the in-browser stepping debugger through the worker.
 * Because the worker owns the VM goroutine, aborting a runaway program is a hard
 * terminate of the worker (stop) — the next call recreates it.
 */
export const wasmDebugBackend: DebugBackend = {
  name: "WebAssembly (worker)",
  needsServer: false,
  start(source, breakpoints, stopOnEntry): Promise<DebugResponse> {
    // Empty path → the bridge compiles plain Gad (the playground dialect).
    return sharedClient().debugStart(source, "", breakpoints, stopOnEntry, []);
  },
  command(session, command: DebugCommand): Promise<DebugResponse> {
    return sharedClient().debugCommand(session, command);
  },
  evaluate(session, expr, repr = false) {
    return sharedClient().debugEval(session, expr, repr);
  },
  stop(session) {
    // Best-effort: tell the manager to drop the session, then hard-terminate the
    // worker so a blocked VM goroutine cannot keep running.
    void sharedClient().debugStop(session).catch(() => {});
    sharedClient().terminate();
  },
};
