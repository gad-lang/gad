import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import { fileURLToPath } from "node:url";

// Server-less demo for @gad-lang/ide-vuetify. The sibling workspace packages are
// resolved from their TypeScript source (no prior build needed for dev), and the
// single-instance-sensitive libraries are deduped to this app's copy.
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      "@gad-lang/ide-vuetify": fileURLToPath(new URL("../src/index.ts", import.meta.url)),
      "@gad-lang/codemirror-gad": fileURLToPath(new URL("../../codemirror-gad/src/index.ts", import.meta.url)),
      "@gad-lang/prism-gad": fileURLToPath(new URL("../../prism-gad/src/index.ts", import.meta.url)),
    },
    dedupe: [
      "vue",
      "vuetify",
      // CodeMirror requires a single instance of state/view or the editor's
      // StateFields (breakpoint gutter, debug line) silently do not apply.
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
});
