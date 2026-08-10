import "vuetify/styles";
import "@mdi/font/css/materialdesignicons.css";
import { createVuetify } from "vuetify";
import type { ThemeDefinition } from "vuetify";
import * as components from "vuetify/components";
import * as directives from "vuetify/directives";

// Palette drawn from the Gad logo: cyan #06B6D4 (primary), amber #D97706
// (secondary/gazelle), navy #0D1B2A / #1B263B (surfaces), slate #415A77.
const light: ThemeDefinition = {
  dark: false,
  colors: {
    background: "#f4f7fb",
    surface: "#ffffff",
    "surface-variant": "#eef3f9",
    primary: "#0e7490",
    secondary: "#b45309",
    accent: "#06b6d4",
    info: "#0891b2",
    success: "#2e8b57",
    warning: "#d97706",
    error: "#dc2626",
    "on-background": "#0d1b2a",
    "on-surface": "#0d1b2a",
  },
};

const dark: ThemeDefinition = {
  dark: true,
  colors: {
    background: "#0d1b2a",
    surface: "#14243a",
    "surface-variant": "#1b263b",
    primary: "#22d3ee",
    secondary: "#fbbf24",
    accent: "#06b6d4",
    info: "#38bdf8",
    success: "#34d399",
    warning: "#f59e0b",
    error: "#f87171",
    "on-background": "#e6edf5",
    "on-surface": "#e6edf5",
  },
};

export function initialTheme(): "light" | "dark" {
  const d = document.documentElement.dataset.theme;
  return d === "dark" ? "dark" : "light";
}

export const vuetify = createVuetify({
  components,
  directives,
  theme: {
    defaultTheme: initialTheme(),
    themes: { light, dark },
  },
  defaults: {
    VCard: { rounded: "lg" },
    VBtn: { rounded: "lg" },
  },
});
