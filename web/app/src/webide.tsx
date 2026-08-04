// Entry for the standalone, server-less IDE (webide.html). It renders the same
// reusable <Ide> component as `gad ide`, but driven by localIdeApi — a fully
// in-browser backend (WebFS sample tree + LocalStorage overlay, and the Gad WASM
// module in a Web Worker). No Go server is involved.
import { StrictMode, useEffect, useState } from "react";
import { createRoot } from "react-dom/client";
import yaml from "js-yaml";
import { Ide, type RunMode, type RunProfile, type Workspace } from "@gad-lang/ide-react";
import { localIdeApi } from "./backends/localIde";
import "./styles.css";

// Run profiles persist as YAML at the workspace config dir (in the WebFS overlay).
const RUN_PROFILES_PATH = ".gad/run-profiles.yaml";

function Root() {
  const [workspace, setWorkspace] = useState<Workspace | null>(null);
  const [runProfiles, setRunProfiles] = useState<RunProfile[]>([]);
  // Host-controlled run/debug gating (persisted; defaults to full access).
  const runMode = ((localStorage.getItem("gad-webide-runmode") as RunMode) || "debug");

  useEffect(() => {
    localIdeApi.workspace().then(setWorkspace);
    localIdeApi
      .read(RUN_PROFILES_PATH)
      .then(({ content }) => {
        const doc = content.trim() ? yaml.load(content) : null;
        if (Array.isArray(doc)) setRunProfiles(doc as RunProfile[]);
      })
      .catch(() => {});
  }, []);

  if (!workspace) return null;
  return (
    <Ide
      workspace={workspace}
      api={localIdeApi}
      runProfiles={runProfiles}
      onRunProfilesChange={(next) => {
        setRunProfiles(next);
        void localIdeApi.write(RUN_PROFILES_PATH, yaml.dump(next));
      }}
      runMode={runMode}
    />
  );
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <Root />
  </StrictMode>,
);
