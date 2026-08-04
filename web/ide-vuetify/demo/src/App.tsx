// Demo shell (TSX): the reusable <GadIde> driven by the in-browser localIdeApi.
// No Go server — files live in a LocalStorage overlay over the bundled samples,
// and the Gad VM runs in a Web Worker. Both the dockview layout (layoutConfig)
// and the settings (config) are v-models persisted to LocalStorage and reloaded.
import { defineComponent, onMounted, ref } from "vue";
import { useTheme } from "vuetify";
import { VApp, VAppBar, VAppBarTitle, VMain } from "vuetify/components";
import { GadIde } from "@gad-lang/ide-vuetify";
import type { SerializedDockview, Workspace } from "@gad-lang/ide-vuetify";
import { localIdeApi, resetWorkspace } from "./backends/localIde";

const LAYOUT_KEY = "gad-vuetify-layout-v1";
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
    const dark = ref(theme.global.current.value.dark);
    const workspace = ref<Workspace | null>(null);
    const layoutConfig = ref<SerializedDockview | null>(loadJSON<SerializedDockview | null>(LAYOUT_KEY, null));
    const config = ref<Record<string, unknown>>(loadJSON<Record<string, unknown>>(CONFIG_KEY, {}));

    function toggleTheme() {
      const next = dark.value ? "light" : "dark";
      theme.change(next);
      dark.value = !dark.value;
      saveJSON("gad-vuetify-theme", next);
    }

    onMounted(async () => {
      workspace.value = await localIdeApi.workspace();
      // Seed the config from the backend when the user has none saved yet.
      if (Object.keys(config.value).length === 0) config.value = await localIdeApi.config();
    });

    return () => (
      <VApp>
        <VAppBar density="compact" flat>
          <VAppBarTitle>Gad IDE</VAppBarTitle>
          <button class="gad-theme-toggle" title="Toggle light/dark" onClick={toggleTheme}>
            {dark.value ? "☀" : "☾"}
          </button>
        </VAppBar>
        <VMain class="gad-main">
          {workspace.value && (
            <GadIde
              api={localIdeApi}
              workspace={workspace.value}
              dark={dark.value}
              onReset={resetWorkspace}
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
            />
          )}
        </VMain>
      </VApp>
    );
  },
});
