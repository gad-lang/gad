// Demo shell (TSX): the reusable <GadIde> driven by the in-browser localIdeApi.
// No Go server — files live in a LocalStorage overlay over the bundled samples,
// and the Gad VM runs in a Web Worker. Both the dockview layout (layoutConfig)
// and the settings (config) are v-models persisted to LocalStorage and reloaded.
import { computed, defineComponent, onMounted, ref } from "vue";
import { useTheme } from "vuetify";
import { VApp, VAppBar, VAppBarTitle, VMain } from "vuetify/components";
import yaml from "js-yaml";
import { GadIde } from "@gad-lang/ide-vuetify";
import type { RunMode, RunProfile, SerializedDockview, Workspace } from "@gad-lang/ide-vuetify";
import { localIdeApi, resetWorkspace } from "./backends/localIde";

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
    const workspace = ref<Workspace | null>(null);
    const layoutConfig = ref<SerializedDockview | null>(loadJSON<SerializedDockview | null>(LAYOUT_KEY, null));
    const config = ref<Record<string, unknown>>(loadJSON<Record<string, unknown>>(CONFIG_KEY, {}));
    const runProfiles = ref<RunProfile[]>([]);
    const runMode = ref<RunMode>((localStorage.getItem("gad-vuetify-runmode") as RunMode) || "debug");

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
          <VAppBarTitle>Gad IDE</VAppBarTitle>
        </VAppBar>
        <VMain class="gad-main">
          {workspace.value && (
            <GadIde
              api={localIdeApi}
              workspace={workspace.value}
              dark={dark.value}
              onReset={resetWorkspace}
              onToggleTheme={toggleTheme}
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
