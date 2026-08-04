import { createApp } from "vue";
import { createVuetify } from "vuetify";
import * as components from "vuetify/components";
import * as directives from "vuetify/directives";
import { aliases, mdi } from "vuetify/iconsets/mdi";
import App from "./App";

import "vuetify/styles";
import "@mdi/font/css/materialdesignicons.css";
import "dockview-core/dist/styles/dockview.css";
import "./styles.css";

const saved = (() => {
  try {
    return localStorage.getItem("gad-vuetify-theme");
  } catch {
    return null;
  }
})();
const prefersDark = window.matchMedia?.("(prefers-color-scheme: dark)").matches;
const defaultTheme = saved === "light" || saved === "dark" ? saved : prefersDark ? "dark" : "light";

const vuetify = createVuetify({
  components,
  directives,
  icons: { defaultSet: "mdi", aliases, sets: { mdi } },
  theme: { defaultTheme },
});

createApp(App).use(vuetify).mount("#app");
