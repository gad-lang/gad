// Entry for the standalone, server-less IDE (webide.html). It renders the same
// reusable <Ide> component as `gad ide`, but driven by localIdeApi — a fully
// in-browser backend (WebFS sample tree + LocalStorage overlay, and the Gad WASM
// module in a Web Worker). No Go server is involved.
import { StrictMode, useEffect, useState } from "react";
import { createRoot } from "react-dom/client";
import { Ide, type Workspace } from "@gad-lang/ide-react";
import { localIdeApi } from "./backends/localIde";
import "./styles.css";

function Root() {
  const [workspace, setWorkspace] = useState<Workspace | null>(null);
  useEffect(() => {
    localIdeApi.workspace().then(setWorkspace);
  }, []);
  if (!workspace) return null;
  return <Ide workspace={workspace} api={localIdeApi} />;
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <Root />
  </StrictMode>,
);
