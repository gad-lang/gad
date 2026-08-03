import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { fileURLToPath } from "node:url";

// During development the Go server (web/server) runs on :8080; the Vite dev
// server proxies /api to it so the editor can use the backend example without
// CORS friction.
export default defineConfig({
  plugins: [react()],
  resolve: {
    // Resolve the sibling workspace plugins from their TypeScript source, so the
    // app runs (and hot-reloads) against `src/` without a prior `plugins:build`.
    // Their published `exports` point at `dist/`, which the dev server does not
    // need.
    alias: {
      "@gad-lang/codemirror-gad": fileURLToPath(
        new URL("../codemirror-gad/src/index.ts", import.meta.url),
      ),
      "@gad-lang/prism-gad": fileURLToPath(
        new URL("../prism-gad/src/index.ts", import.meta.url),
      ),
    },
    // The codemirror-gad source (resolved above) has its own copy of these
    // packages under codemirror-gad/node_modules. CodeMirror requires a SINGLE
    // instance of @codemirror/state and @codemirror/view — otherwise the app's
    // decorations/StateFields (e.g. the debug current-line highlight) silently do
    // not apply to the editor. Dedupe them (and their @lezer bases) to the app's
    // copy. The production build already collapses these; this fixes dev only.
    dedupe: [
      "@codemirror/state",
      "@codemirror/view",
      "@codemirror/language",
      "@codemirror/autocomplete",
      "@codemirror/lint",
      "@codemirror/commands",
      "@lezer/common",
      "@lezer/highlight",
      "@lezer/lr",
    ],
  },
  server: {
    proxy: {
      "/api": "http://localhost:8080",
    },
  },
});
