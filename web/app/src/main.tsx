import { StrictMode, useEffect, useState } from "react";
import { createRoot } from "react-dom/client";
import { App } from "./App";
import { Ide, httpIdeApi, probeIde, type IdeApi, type Workspace } from "@gad-lang/ide-react";
import { localIdeApi } from "./backends/localIde";
import "./styles.css";

// Compute methods that a server-less `gad ide` runs in the browser (WASM); file
// I/O keeps using the Go server so the explorer stays real files.
const COMPUTE_METHODS = [
  "format", "transpile", "doc", "eval", "inspect", "diagnose", "run", "dbgStart", "dbgCmd", "dbgEval",
] as const;

// hybridApi routes file ops to the Go server (httpIdeApi) and compute ops to the
// in-browser WASM backend (localIdeApi) — used by `gad ide --serverless`.
function hybridApi(): IdeApi {
  const api = { ...httpIdeApi } as Record<string, unknown>;
  for (const k of COMPUTE_METHODS) api[k] = (localIdeApi as Record<string, unknown>)[k];
  return api as IdeApi;
}

/**
 * Root chooses the IDE when served by `gad ide` (the /api/ide backend is
 * reachable), otherwise the playground App. The same build serves both. When the
 * server reports `compute: "wasm"` (gad ide --serverless), compute runs in the
 * browser while files stay on the server (hybrid api).
 */
function Root() {
  const [mode, setMode] = useState<"loading" | "ide" | "app">("loading");
  const [workspace, setWorkspace] = useState<Workspace | null>(null);
  const [api, setApi] = useState<IdeApi>(httpIdeApi);

  useEffect(() => {
    probeIde().then((ws) => {
      if (ws) {
        setWorkspace(ws);
        setApi(ws.compute === "wasm" ? hybridApi() : httpIdeApi);
        setMode("ide");
      } else {
        setMode("app");
      }
    });
  }, []);

  if (mode === "loading") return null;
  if (mode === "ide" && workspace) return <Ide workspace={workspace} api={api} />;
  return <App />;
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <Root />
  </StrictMode>,
);
