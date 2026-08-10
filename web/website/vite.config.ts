import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";

// The docs SPA is deployed under a one-segment version directory (/latest/,
// /<sha>/, /<tag>/) at the gad-lang.github.io root, so all assets are referenced
// relatively (base "./"); the router base is derived at runtime. Vuetify and Vue
// are deduped to this app's single copy.
export default defineConfig({
  base: "./",
  plugins: [vue()],
  resolve: { dedupe: ["vue", "vuetify"] },
});
