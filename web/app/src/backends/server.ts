import type { GadDiagnostic } from "@gad-lang/codemirror-gad";
import type { FormatResult, GadBackend, RunResult } from "./types";

async function post<T>(path: string, source: string): Promise<T> {
  const res = await fetch(path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ source }),
  });
  if (!res.ok) throw new Error(`${path}: HTTP ${res.status}`);
  return (await res.json()) as T;
}

/**
 * serverBackend talks to the Go HTTP server (web/server). In dev, Vite proxies
 * /api to it; in production the same server serves the built app.
 */
export const serverBackend: GadBackend = {
  name: "Go server",
  format: (source) => post<FormatResult>("/api/fmt", source),
  run: (source) => post<RunResult>("/api/run", source),
  diagnose: async (source) =>
    (await post<{ diagnostics: GadDiagnostic[] }>("/api/diagnose", source)).diagnostics,
};
