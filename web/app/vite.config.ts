import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// During development the Go server (web/server) runs on :8080; the Vite dev
// server proxies /api to it so the editor can use the backend example without
// CORS friction.
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      "/api": "http://localhost:8080",
    },
  },
});
