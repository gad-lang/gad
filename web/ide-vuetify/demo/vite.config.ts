import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import vueJsx from "@vitejs/plugin-vue-jsx";
import { fileURLToPath } from "node:url";
import { existsSync } from "node:fs";

// Server-less demo for @gad-lang/ide-vuetify. The sibling workspace packages are
// resolved from their TypeScript source (no prior build needed for dev) when this
// demo lives inside the gad-lang/gad monorepo; in the extracted standalone repo
// those source paths are absent, so the packages resolve from node_modules (the
// published npm versions) instead. The single-instance-sensitive libraries are
// deduped to this app's copy.
function srcAlias(rel: string): string | undefined {
  const p = fileURLToPath(new URL(rel, import.meta.url));
  return existsSync(p) ? p : undefined;
}
const alias: Record<string, string> = {
  // The package itself always lives beside the demo, in both layouts.
  "@gad-lang/ide-vuetify": fileURLToPath(new URL("../src/index.ts", import.meta.url)),
};
for (const [name, rel] of [
  ["@gad-lang/codemirror-gad", "../../plugins/js/codemirror-gad/src/index.ts"],
  ["@gad-lang/prism-gad", "../../plugins/js/prism-gad/src/index.ts"],
] as const) {
  const p = srcAlias(rel);
  if (p) alias[name] = p; // monorepo: use source; standalone: fall through to npm
}
export default defineConfig({
  plugins: [vue(), vueJsx()],
  resolve: {
    alias,
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
