import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import { fileURLToPath } from "node:url";

// Library build for @gad-lang/ide-vuetify. Vue, Vuetify, the CodeMirror packages
// and the sibling Gad plugins are externalized (peer/runtime deps of the host
// app), so the bundle ships only this package's own code.
const external = [
  "vue",
  "vuetify",
  "@gad-lang/codemirror-gad",
  "@gad-lang/prism-gad",
  /^@codemirror\//,
  /^@lezer\//,
  "codemirror",
  "marked",
  "prismjs",
];

export default defineConfig({
  plugins: [vue()],
  build: {
    lib: {
      entry: fileURLToPath(new URL("./src/index.ts", import.meta.url)),
      formats: ["es"],
      fileName: () => "ide-vuetify.js",
    },
    rollupOptions: { external },
  },
});
