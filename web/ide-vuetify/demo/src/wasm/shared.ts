// One shared WASM worker client for the whole demo (the VM runs off the UI
// thread; see client.ts + worker.ts).
import { WasmClient } from "./client";

// Resolve the asset base to an ABSOLUTE URL. The Web Worker fetches gad.wasm /
// wasm_exec.js relative to this base; a page-relative base (e.g. "./" from a
// relative Vite build embedded under /ide/) would otherwise resolve against the
// worker's own location, not the page. Anchoring it to location.href works at
// any deploy path (root, /gad/, /gad/ide/, …).
const ASSET_BASE = new URL(
  (import.meta as { env?: { BASE_URL?: string } }).env?.BASE_URL || "/",
  location.href,
).href;

let client: WasmClient | null = null;

/** sharedClient returns the process-wide WASM worker client, creating it lazily. */
export function sharedClient(): WasmClient {
  if (!client) client = new WasmClient(ASSET_BASE);
  return client;
}
