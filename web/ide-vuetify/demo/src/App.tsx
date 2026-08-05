// Demo shell (TSX): the reusable <GadIde> driven by the in-browser localIdeApi.
// No Go server — files live in a LocalStorage overlay over the bundled samples,
// and the Gad VM runs in a Web Worker. Both the dockview layout (layoutConfig)
// and the settings (config) are v-models persisted to LocalStorage and reloaded.
import { computed, defineComponent, onMounted, ref } from "vue";
import { useTheme } from "vuetify";
import { VApp, VAppBar, VAppBarTitle, VMain, VSpacer, VTab, VTabs } from "vuetify/components";
import yaml from "js-yaml";
import { GadIde, GadNotebook, GadPlayground } from "@gad-lang/ide-vuetify";
import type { GadRunner, RunMode, RunProfile, SerializedDockview, UploadedFile, Workspace } from "@gad-lang/ide-vuetify";
import { localIdeApi, resetWorkspace } from "./backends/localIde";
import { sharedClient } from "./wasm/shared";
import { base64ToBytes, extractArchive } from "./extract";

// The Playground/Notebook backend: format/run/diagnose through the WASM worker.
const runner: GadRunner = {
  name: "WebAssembly",
  format: (s, st) => sharedClient().format(s, st),
  run: (s, st, te) => sharedClient().run(s, st, [], te),
  diagnose: async (s, st) => (await sharedClient().diagnose(s, st)).diagnostics,
};

// onUpload persists uploaded files to the in-browser workspace. A plain file is
// written as-is; an archive (when the component asks to extract) is expanded
// under a folder named after it, next to where it was dropped.
async function onUpload(files: UploadedFile[]) {
  for (const f of files) {
    if (f.archive && f.bytes) {
      const dir = f.path.replace(/\.(zip|tar\.gz|tgz|tar)$/i, "") + "/";
      for (const e of extractArchive(f.archive, base64ToBytes(f.bytes))) {
        await localIdeApi.write(dir + e.path, e.content);
      }
    } else {
      await localIdeApi.write(f.path, f.content);
    }
  }
}

// Run profiles are persisted to the workspace config dir as YAML, the way a real
// gad workspace would keep them (here the file lives in the LocalStorage-backed
// WebFS at GAD_CONFIG_DIR/run-profiles.yaml).
const RUN_PROFILES_PATH = ".gad/run-profiles.yaml";

const LAYOUT_KEY = "gad-vuetify-layout-v2";
const CONFIG_KEY = "gad-vuetify-config-v1";

function loadJSON<T>(key: string, fallback: T): T {
  try {
    const raw = localStorage.getItem(key);
    return raw ? (JSON.parse(raw) as T) : fallback;
  } catch {
    return fallback;
  }
}
function saveJSON(key: string, value: unknown) {
  try {
    localStorage.setItem(key, JSON.stringify(value));
  } catch {
    /* storage full/unavailable */
  }
}

export default defineComponent({
  name: "App",
  setup() {
    const theme = useTheme();
    // Derive `dark` from Vuetify's active theme so the toolbar, the dockview
    // theme and the editor stay in sync (no manual ref that can drift).
    const dark = computed(() => theme.global.current.value.dark);
    const tab = ref<"ide" | "playground" | "notebook">("playground");
    const workspace = ref<Workspace | null>(null);
    const layoutConfig = ref<SerializedDockview | null>(loadJSON<SerializedDockview | null>(LAYOUT_KEY, null));
    const config = ref<Record<string, unknown>>(loadJSON<Record<string, unknown>>(CONFIG_KEY, {}));
    const runProfiles = ref<RunProfile[]>([]);
    const runMode = ref<RunMode>((localStorage.getItem("gad-vuetify-runmode") as RunMode) || "debug");
    const fontSize = ref<number>(Number(localStorage.getItem("gad-vuetify-fontsize")) || 13);

    async function loadRunProfiles() {
      try {
        const { content } = await localIdeApi.read(RUN_PROFILES_PATH);
        const doc = content.trim() ? yaml.load(content) : null;
        if (Array.isArray(doc)) runProfiles.value = doc as RunProfile[];
      } catch {
        /* no profiles yet */
      }
    }
    function saveRunProfiles(next: RunProfile[]) {
      runProfiles.value = next;
      void localIdeApi.write(RUN_PROFILES_PATH, yaml.dump(next));
    }

    function toggleTheme() {
      const next = dark.value ? "light" : "dark";
      theme.global.name.value = next;
      saveJSON("gad-vuetify-theme", next);
    }

    onMounted(async () => {
      workspace.value = await localIdeApi.workspace();
      // Seed the config from the backend when the user has none saved yet.
      if (Object.keys(config.value).length === 0) config.value = await localIdeApi.config();
      await loadRunProfiles();
    });

    return () => (
      <VApp>
        <VAppBar density="compact" flat>
          <VAppBarTitle style={{ flex: "none", marginRight: "16px" }}>
            <span style={{ display: "inline-flex", alignItems: "center", gap: "8px" }}>
              <img src="gad-24.svg" width="24" height="24" alt="Gad" />
              <span>GAD</span>
            </span>
          </VAppBarTitle>
          <VTabs modelValue={tab.value} {...{ "onUpdate:modelValue": (v: unknown) => (tab.value = v as typeof tab.value) }} density="compact">
            <VTab value="playground">Playground</VTab>
            <VTab value="notebook">Notebook</VTab>
            <VTab value="ide">IDE</VTab>
          </VTabs>
          <VSpacer />
          <button class="gad-theme-toggle" title="Toggle light/dark" onClick={toggleTheme}>
            {dark.value ? "☀" : "☾"}
          </button>
        </VAppBar>
        <VMain class="gad-main">
          {tab.value === "playground" && <GadPlayground runner={runner} dark={dark.value} />}
          {tab.value === "notebook" && <GadNotebook runner={runner} dark={dark.value} />}
          {tab.value === "ide" && workspace.value && (
            <GadIde
              api={localIdeApi}
              workspace={workspace.value}
              dark={dark.value}
              onReset={resetWorkspace}
              onUpload={onUpload}
              layoutConfig={layoutConfig.value}
              {...{
                "onUpdate:layoutConfig": (v: SerializedDockview) => {
                  layoutConfig.value = v;
                  saveJSON(LAYOUT_KEY, v);
                },
              }}
              config={config.value}
              {...{
                "onUpdate:config": (v: Record<string, unknown>) => {
                  config.value = v;
                  saveJSON(CONFIG_KEY, v);
                  void localIdeApi.saveConfig(v);
                },
              }}
              runProfiles={runProfiles.value}
              {...{ "onUpdate:runProfiles": (v: RunProfile[]) => saveRunProfiles(v) }}
              autosave={600000}
              fontSize={fontSize.value}
              {...{
                "onUpdate:fontSize": (v: number) => {
                  fontSize.value = v;
                  try {
                    localStorage.setItem("gad-vuetify-fontsize", String(v));
                  } catch {
                    /* ignore */
                  }
                },
              }}
              runMode={runMode.value}
              {...{
                "onUpdate:runMode": (v: RunMode) => {
                  runMode.value = v;
                  try {
                    localStorage.setItem("gad-vuetify-runmode", v);
                  } catch {
                    /* ignore */
                  }
                },
              }}
            />
          )}
        </VMain>
      </VApp>
    );
  },
});
