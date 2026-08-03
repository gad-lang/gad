// Entry for the standalone, server-less embeddable IDE (webide.html). Unlike
// main.tsx it never probes a backend: everything runs in-browser via the Gad
// WebAssembly module in a Web Worker, with a LocalStorage-backed filesystem.
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { WebIde } from "./webide/WebIde";
import { useTheme } from "./useTheme";
import "./styles.css";

function Root() {
  const [theme, toggleTheme] = useTheme();
  const dark = theme === "dark";
  return (
    <div className="webide-shell">
      <header className="webide-topbar">
        <strong>Gad IDE</strong>
        <button className="theme-toggle" onClick={toggleTheme} title="Toggle light/dark">
          {dark ? "☀ Light" : "☾ Dark"}
        </button>
      </header>
      <WebIde dark={dark} />
    </div>
  );
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <Root />
  </StrictMode>,
);
