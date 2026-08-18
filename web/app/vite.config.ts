import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { fileURLToPath } from "node:url";

// During development the Go server (web/server) runs on :8080; the Vite dev
// server proxies /api to it so the editor can use the backend example without
// CORS friction.
export default defineConfig({
  plugins: [react()],
  build: {
    // Two entry pages: the playground (index.html) and the standalone,
    // server-less embeddable IDE (webide.html).
    rollupOptions: {
      input: {
        index: fileURLToPath(new URL("./index.html", import.meta.url)),
        webide: fileURLToPath(new URL("./webide.html", import.meta.url)),
      },
    },
  },
  resolve: {
    // Resolve the sibling workspace plugins from their TypeScript source, so the
    // app runs (and hot-reloads) against `src/` without a prior `plugins:build`.
    // Their published `exports` point at `dist/`, which the dev server does not
    // need.
    alias: {
      "@gad-lang/codemirror-gad": fileURLToPath(
        new URL("../plugins/js/codemirror-gad/src/index.ts", import.meta.url),
      ),
      "@gad-lang/prism-gad": fileURLToPath(
        new URL("../plugins/js/prism-gad/src/index.ts", import.meta.url),
      ),
      "@gad-lang/ide-react": fileURLToPath(
        new URL("../ide-react/src/index.ts", import.meta.url),
      ),
    },
    // The codemirror-gad source (resolved above) has its own copy of these
    // packages under codemirror-gad/node_modules. CodeMirror requires a SINGLE
    // instance of @codemirror/state and @codemirror/view — otherwise the app's
    // decorations/StateFields (e.g. the debug current-line highlight) silently do
    // not apply to the editor. Dedupe them (and their @lezer bases) to the app's
    // copy. The production build already collapses these; this fixes dev only.
    dedupe: [
      // React/MUI/dockview must be single instances too: @gad-lang/ide-react is
      // aliased to its src, which pulls these from its own node_modules unless
      // deduped — duplicate React breaks hooks/context across the boundary.
      "react",
      "react-dom",
      "@mui/material",
      "@mui/icons-material",
      "@emotion/react",
      "@emotion/styled",
      "dockview-react",
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
      // The /api backend (web/server or `gad ide`) is optional in dev: the
      // playground probes it and falls back to the in-browser WASM backend, and
      // the standalone webide.html never calls it at all. When :8080 is down,
      // answer with a quiet 503 instead of spamming ECONNREFUSED stack traces.
      "/api": {
        target: "http://localhost:8080",
        configure: (proxy) => {
          proxy.on("error", (_err, _req, res) => {
            const r = res as { writeHead?: (code: number) => void; end?: (body: string) => void };
            if (r.writeHead && r.end && !("headersSent" in res && (res as { headersSent: boolean }).headersSent)) {
              r.writeHead(503);
              r.end("api backend not running");
            }
          });
        },
      },
    },
  },
});
