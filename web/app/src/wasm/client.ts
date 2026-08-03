// Promise-based client for the Gad WASM Web Worker (worker.ts). Each call posts a
// request with a unique id and resolves when the matching reply arrives. The
// worker can be terminated (abort) — e.g. to stop a runaway debug session — and
// is recreated lazily on the next call.
import type { DebugResponse } from "../backends/debug";

type Pending = { resolve: (v: string) => void; reject: (e: Error) => void };

export class WasmClient {
  private worker: Worker | null = null;
  private seq = 0;
  private pending = new Map<number, Pending>();
  private base: string;

  constructor(assetBase = "/") {
    this.base = assetBase;
  }

  private ensureWorker(): Worker {
    if (this.worker) return this.worker;
    const w = new Worker(new URL("./worker.ts", import.meta.url), { type: "module" });
    w.onmessage = (e: MessageEvent<{ id: number; result?: string; error?: string }>) => {
      const p = this.pending.get(e.data.id);
      if (!p) return;
      this.pending.delete(e.data.id);
      if (e.data.error) p.reject(new Error(e.data.error));
      else p.resolve(e.data.result ?? "");
    };
    w.onerror = (e) => {
      // Fail every in-flight request; the worker is recreated on the next call.
      for (const [, p] of this.pending) p.reject(new Error(e.message || "worker error"));
      this.pending.clear();
      this.worker = null;
    };
    this.worker = w;
    // Initialize the module (loads wasm_exec.js + gad.wasm from base).
    this.raw("init", [], this.base).catch(() => {});
    return w;
  }

  private raw(fn: string, args: unknown[] = [], base?: string): Promise<string> {
    const w = this.ensureWorker();
    const id = ++this.seq;
    return new Promise<string>((resolve, reject) => {
      this.pending.set(id, { resolve, reject });
      w.postMessage({ id, fn, args, base });
    });
  }

  private async json<T>(fn: string, args: unknown[] = []): Promise<T> {
    return JSON.parse(await this.raw(fn, args)) as T;
  }

  /** Terminate the worker (aborting any blocked run) and drop pending calls. */
  terminate(): void {
    this.worker?.terminate();
    this.worker = null;
    for (const [, p] of this.pending) p.reject(new Error("aborted"));
    this.pending.clear();
  }

  run(source: string) {
    return this.json<{ ok: boolean; stdout?: string; stderr?: string; result?: string }>("gadRun", [source]);
  }
  format(source: string) {
    return this.json<{ ok: boolean; source?: string; error?: string }>("gadFormat", [source]);
  }
  diagnose(source: string) {
    return this.json<{ diagnostics: unknown[] }>("gadDiagnose", [source]);
  }
  /** doc extracts Markdown documentation from content and its sourceType. */
  doc(source: string, sourceType: "gad" | "gadTemplate" | "giom") {
    return this.json<{ markdown?: string; error?: string }>("gadDoc", [source, sourceType]);
  }

  // --- Debugger (mirrors backends/debug.ts) ---
  debugStart(source: string, path: string, breakpoints: number[], stopOnEntry: boolean, args: string[] = []) {
    return this.json<DebugResponse>("gadDebugStart", [
      source, path, JSON.stringify(breakpoints), stopOnEntry, JSON.stringify(args),
    ]);
  }
  debugCommand(session: string, command: string) {
    return this.json<DebugResponse>("gadDebugCommand", [session, command]);
  }
  debugEval(session: string, expr: string, repr = false) {
    return this.json<{ ok: boolean; value?: string; error?: string }>("gadDebugEval", [session, expr, repr]);
  }
  debugStop(session: string) {
    return this.json<{ ok: boolean }>("gadDebugStop", [session]);
  }
}
