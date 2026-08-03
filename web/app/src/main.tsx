import { StrictMode, useEffect, useState } from "react";
import { createRoot } from "react-dom/client";
import { App } from "./App";
import { Ide } from "@gad-lang/ide-react";
import { probeIde, type Workspace } from "@gad-lang/ide-react";
import "./styles.css";

/**
 * Root chooses the IDE when served by `gad ide` (the /api/ide backend is
 * reachable), otherwise the playground App. The same build serves both.
 */
function Root() {
  const [mode, setMode] = useState<"loading" | "ide" | "app">("loading");
  const [workspace, setWorkspace] = useState<Workspace | null>(null);

  useEffect(() => {
    probeIde().then((ws) => {
      if (ws) {
        setWorkspace(ws);
        setMode("ide");
      } else {
        setMode("app");
      }
    });
  }, []);

  if (mode === "loading") return null;
  if (mode === "ide" && workspace) return <Ide workspace={workspace} />;
  return <App />;
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <Root />
  </StrictMode>,
);
