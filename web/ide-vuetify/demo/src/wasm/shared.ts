// One shared WASM worker client for the whole demo (the VM runs off the UI
// thread; see client.ts + worker.ts).
import { WasmClient } from "./client";

const ASSET_BASE = (import.meta as { env?: { BASE_URL?: string } }).env?.BASE_URL || "/";

let client: WasmClient | null = null;

/** sharedClient returns the process-wide WASM worker client, creating it lazily. */
export function sharedClient(): WasmClient {
  if (!client) client = new WasmClient(ASSET_BASE);
  return client;
}
