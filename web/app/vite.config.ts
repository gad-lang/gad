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
  },
  server: {
    proxy: {
      "/api": "http://localhost:8080",
    },
  },
});
