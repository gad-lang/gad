// Web Worker hosting the Gad WebAssembly module. Running the VM here (rather than
// on the UI thread) means a blocking program — or a paused debug session waiting
// between steps — never freezes the page. The worker speaks a small request/reply
// protocol: the client posts { id, fn, args }, the worker replies { id, result }
// or { id, error }. The WASM exports (gadRun, gadDoc, gadDebug*, …) are plain
// JSON-returning functions installed on the worker's global object.
/// <reference lib="webworker" />

interface Go {
  importObject: WebAssembly.Imports;
  run(i: WebAssembly.Instance): Promise<void>;
}

interface GadGlobals {
  Go: new () => Go;
  gadReady?: boolean;
  onGadReady?: () => void;
  gadRun?: (src: string) => string;
  gadFormat?: (src: string) => string;
  gadDiagnose?: (src: string) => string;
  gadDoc?: (src: string, sourceType: string) => string;
  gadDocData?: (src: string, sourceType: string) => string;
  gadDocComments?: (src: string) => string;
  gadEval?: (...a: unknown[]) => string;
  gadTranspile?: (...a: unknown[]) => string;
  gadDebugStart?: (...a: unknown[]) => string;
  gadDebugCommand?: (...a: unknown[]) => string;
  gadDebugEval?: (...a: unknown[]) => string;
  gadDebugStop?: (session: string) => string;
}

const g = globalThis as unknown as GadGlobals;

// Base URL for the wasm assets; set by the client's first "init" message so the
// worker fetches them from the page's origin/path.
let assetBase = "/";

let readyPromise: Promise<void> | null = null;

// ensureReady loads wasm_exec.js (which defines globalThis.Go) and instantiates
// gad.wasm, resolving once the module has installed its exports.
function ensureReady(): Promise<void> {
  if (readyPromise) return readyPromise;
  readyPromise = (async () => {
    const execSrc = await (await fetch(assetBase + "wasm_exec.js")).text();
    // wasm_exec.js is a classic script; evaluate it so it defines globalThis.Go.
    new Function(execSrc)();
    const go = new g.Go();
    const res = await WebAssembly.instantiateStreaming(fetch(assetBase + "gad.wasm"), go.importObject);
    void go.run(res.instance); // never resolves (the module blocks on select{})
    await new Promise<void>((resolve) => {
      if (g.gadReady) return resolve();
      g.onGadReady = () => resolve();
    });
  })();
  return readyPromise;
}

type Req = { id: number; fn: string; args?: unknown[]; base?: string };

self.onmessage = async (e: MessageEvent<Req>) => {
  const { id, fn, args = [], base } = e.data;
  try {
    if (fn === "init") {
      if (base) assetBase = base;
      await ensureReady();
      (self as unknown as Worker).postMessage({ id, result: "ready" });
      return;
    }
    await ensureReady();
    const target = (g as unknown as Record<string, ((...a: unknown[]) => string) | undefined>)[fn];
    if (typeof target !== "function") throw new Error("unknown wasm function: " + fn);
    const json = target(...args);
    (self as unknown as Worker).postMessage({ id, result: json });
  } catch (err) {
    (self as unknown as Worker).postMessage({ id, error: err instanceof Error ? err.message : String(err) });
  }
};
