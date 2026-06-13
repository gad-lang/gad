import type { GadDiagnostic } from "@gad-lang/codemirror-gad";
import type { FormatResult, GadBackend, RunResult } from "./types";

// The Go WASM module installs these globals (see web/wasm/main.go).
declare global {
  interface Window {
    Go: new () => { importObject: WebAssembly.Imports; run(i: WebAssembly.Instance): Promise<void> };
    gadReady?: boolean;
    onGadReady?: () => void;
    gadFormat?: (source: string) => string;
    gadRun?: (source: string) => string;
    gadDiagnose?: (source: string) => string;
  }
}

let readyPromise: Promise<void> | null = null;

/** Load wasm_exec.js (once) by injecting a script tag. */
function loadExecScript(): Promise<void> {
  return new Promise((resolve, reject) => {
    if (window.Go) return resolve();
    const s = document.createElement("script");
    s.src = "/wasm_exec.js";
    s.onload = () => resolve();
    s.onerror = () => reject(new Error("failed to load wasm_exec.js"));
    document.head.appendChild(s);
  });
}

/** Instantiate the Gad WASM module and wait for it to install its globals. */
async function ensureReady(): Promise<void> {
  if (readyPromise) return readyPromise;
  readyPromise = (async () => {
    await loadExecScript();
    const go = new window.Go();
    const result = await WebAssembly.instantiateStreaming(fetch("/gad.wasm"), go.importObject);
    // go.run never resolves (the module blocks on select{}); start it detached.
    void go.run(result.instance);
    await new Promise<void>((resolve) => {
      if (window.gadReady) return resolve();
      window.onGadReady = () => resolve();
    });
  })();
  return readyPromise;
}

function call<T>(fn: ((s: string) => string) | undefined, source: string): T {
  if (!fn) throw new Error("WASM module not initialized");
  return JSON.parse(fn(source)) as T;
}

/** wasmBackend runs everything in-browser via the Go WebAssembly module. */
export const wasmBackend: GadBackend = {
  name: "WebAssembly",
  async format(source) {
    await ensureReady();
    return call<FormatResult>(window.gadFormat, source);
  },
  async run(source) {
    await ensureReady();
    return call<RunResult>(window.gadRun, source);
  },
  async diagnose(source) {
    await ensureReady();
    return call<{ diagnostics: GadDiagnostic[] }>(window.gadDiagnose, source).diagnostics;
  },
};
