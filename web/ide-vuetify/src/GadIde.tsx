// GadIde — a reusable Vuetify 3 IDE for the Gad language. Panels are hosted in a
// dockview-vue layout (resizable, movable, dockable, tabbable): Explorer, Editor,
// Docs, Call Stack, Locals, Breakpoints and Output. The editor/run controls live
// in the Editor panel's toolbar; the host app owns Settings + the theme toggle
// (open the Settings dialog via the exposed `openSettings()`). Three v-models:
// `layoutConfig` (dockview layout), `config` (Settings document) and
// `runProfiles`, plus `runMode`. Backend-agnostic via `api`.
import { computed, defineComponent, onBeforeUnmount, onMounted, provide, reactive, ref, watch, type PropType } from "vue";
import { DockviewVue, themeDark, themeLight } from "dockview-vue";
import type { DockviewApi, DockviewReadyEvent, SerializedDockview, VueComponent } from "dockview-vue";
import { createController, IdeControllerKey } from "./controller";
import type { IdeApi, RunMode, RunProfile, UploadedFile, Workspace } from "./api";
import PanelExplorer from "./panels/PanelExplorer";
import PanelEditor from "./panels/PanelEditor";
import PanelCallStack from "./panels/PanelCallStack";
import PanelLocals from "./panels/PanelLocals";
import PanelOutput from "./panels/PanelOutput";
import PanelDocs from "./panels/PanelDocs";
import PanelBreakpoints from "./panels/PanelBreakpoints";
import BreakpointConditionDialog from "./BreakpointConditionDialog";
import SettingsDialog, { type PanelToggle } from "./SettingsDialog";
import RunProfileDialog from "./RunProfileDialog";
import { ConfirmDialog, PromptDialog } from "./PromptDialog";
import UrlImportDialog from "./UrlImportDialog";

// The dockview theme CSS is the consumer's responsibility (like Vuetify's
// styles): import "dockview-core/dist/styles/dockview.css" once in the host app.
import "./styles.css";

// The registered panels: dockview component name, title and default placement.
interface PanelDef {
  id: string;
  label: string;
  add: (api: DockviewApi) => void;
}
const PANELS: PanelDef[] = [
  { id: "explorer", label: "Explorer", add: (a) => a.addPanel({ id: "explorer", component: "explorer", title: "Explorer" }) },
  { id: "editor", label: "Editor", add: (a) => a.addPanel({ id: "editor", component: "editor", title: "Editor", position: { referencePanel: "explorer", direction: "right" } }) },
  { id: "docs", label: "Docs", add: (a) => a.addPanel({ id: "docs", component: "docs", title: "Docs", position: { referencePanel: "editor", direction: "right" } }) },
  { id: "output", label: "Output", add: (a) => a.addPanel({ id: "output", component: "output", title: "Output", position: { referencePanel: "editor", direction: "below" } }) },
  { id: "callstack", label: "Call Stack", add: (a) => a.addPanel({ id: "callstack", component: "callstack", title: "Call Stack", position: { referencePanel: "output", direction: "within" } }) },
  { id: "locals", label: "Locals", add: (a) => a.addPanel({ id: "locals", component: "locals", title: "Locals", position: { referencePanel: "output", direction: "within" } }) },
  { id: "breakpoints", label: "Breakpoints", add: (a) => a.addPanel({ id: "breakpoints", component: "breakpoints", title: "Breakpoints", position: { referencePanel: "output", direction: "within" } }) },
];

const components = {
  explorer: PanelExplorer,
  editor: PanelEditor,
  docs: PanelDocs,
  callstack: PanelCallStack,
  locals: PanelLocals,
  breakpoints: PanelBreakpoints,
  output: PanelOutput,
} as unknown as Record<string, VueComponent>;

export default defineComponent({
  name: "GadIde",
  props: {
    api: { type: Object as PropType<IdeApi>, required: true },
    workspace: { type: Object as PropType<Workspace>, required: true },
    dark: { type: Boolean, default: false },
    onReset: { type: Function as PropType<() => Promise<void> | void>, default: undefined },
    /** Handle files uploaded (Explorer button or drag-drop). When absent, files
     * are written to the workspace via the api. */
    onUpload: { type: Function as PropType<(files: UploadedFile[]) => Promise<void> | void>, default: undefined },
    layoutConfig: { type: Object as PropType<SerializedDockview | null>, default: null },
    config: { type: Object as PropType<Record<string, unknown>>, default: () => ({}) },
    runProfiles: { type: Array as PropType<RunProfile[]>, default: () => [] },
    /** Gates the run/debug actions (v-model). Defaults to "debug" (all enabled). */
    runMode: { type: String as PropType<RunMode>, default: "debug" },
  },
  emits: {
    "update:layoutConfig": (_v: SerializedDockview) => true,
    "update:config": (_v: Record<string, unknown>) => true,
    "update:runProfiles": (_v: RunProfile[]) => true,
    "update:runMode": (_v: RunMode) => true,
  },
  setup(props, { emit }) {
    const dark = computed(() => props.dark);
    const ctx = createController(props.api, props.workspace, dark, {
      onReset: props.onReset,
      onUpload: props.onUpload,
      getRunProfiles: () => props.runProfiles,
      getRunMode: () => props.runMode,
      emitRunProfiles: (p) => emit("update:runProfiles", p),
    });
    provide(IdeControllerKey, ctx);

    let dv: DockviewApi | null = null;
    let disposer: { dispose(): void } | null = null;
    let applyingExternal = false;
    let lastJSON = "";
    const visible = reactive(new Set<string>());

    function buildDefault(api: DockviewApi) {
      for (const p of PANELS) p.add(api);
    }
    function syncVisible() {
      if (!dv) return;
      visible.clear();
      for (const p of dv.panels) visible.add(p.id);
    }

    function onReady(e: DockviewReadyEvent) {
      dv = e.api;
      const initial = props.layoutConfig;
      try {
        if (initial && (initial as { grid?: unknown }).grid) dv.fromJSON(initial);
        else buildDefault(dv);
      } catch {
        dv.clear();
        buildDefault(dv);
      }
      syncVisible();
      lastJSON = JSON.stringify(dv.toJSON());
      disposer = dv.onDidLayoutChange(() => {
        if (!dv) return;
        syncVisible();
        if (applyingExternal) return;
        const json = dv.toJSON();
        const s = JSON.stringify(json);
        if (s !== lastJSON) {
          lastJSON = s;
          emit("update:layoutConfig", json);
        }
      });
    }

    // External layoutConfig changes (e.g. a restore) re-apply to the dockview.
    watch(
      () => props.layoutConfig,
      (cfg) => {
        if (!dv || !cfg) return;
        const s = JSON.stringify(cfg);
        if (s === lastJSON) return;
        applyingExternal = true;
        try {
          dv.fromJSON(cfg);
          lastJSON = JSON.stringify(dv.toJSON());
        } catch {
          /* ignore malformed layout */
        } finally {
          applyingExternal = false;
          syncVisible();
        }
      },
    );

    function togglePanel(id: string, show: boolean) {
      if (!dv) return;
      if (show) {
        if (!dv.getPanel(id)) PANELS.find((p) => p.id === id)?.add(dv);
      } else {
        const p = dv.getPanel(id);
        if (p) dv.removePanel(p);
      }
    }

    const panelToggles = computed<PanelToggle[]>(() =>
      PANELS.map((p) => ({ id: p.id, label: p.label, visible: visible.has(p.id) })),
    );

    // The editor toolbar's Doc button bumps ctx.docRequest; reveal/focus Docs.
    watch(
      () => ctx.docRequest.value,
      () => {
        if (dv && !dv.getPanel("docs")) PANELS.find((p) => p.id === "docs")?.add(dv);
        dv?.getPanel("docs")?.api.setActive();
      },
    );

    function saveConfig(next: Record<string, unknown>) {
      emit("update:config", next);
    }

    onMounted(() => void ctx.init());
    onBeforeUnmount(() => disposer?.dispose());

    // dockview v7 themes via a `theme` option object (not a CSS ancestor class);
    // the default is themeAbyss (dark), so it must be set and kept in sync.
    const dvTheme = computed(() => (props.dark ? themeDark : themeLight));

    return () => (
      <div class="gad-ide">
        <div class="gad-ide__dock">
          <DockviewVue style={{ height: "100%" }} theme={dvTheme.value} components={components} onReady={onReady} />
        </div>

        <SettingsDialog
          modelValue={ctx.settingsOpen.value}
          {...{ "onUpdate:modelValue": (v: boolean) => (ctx.settingsOpen.value = v) }}
          config={props.config}
          panels={panelToggles.value}
          onTogglePanel={togglePanel}
          onSave={saveConfig}
        />

        <RunProfileDialog
          modelValue={ctx.profileDialog.value}
          {...{ "onUpdate:modelValue": (v: boolean) => (ctx.profileDialog.value = v) }}
          defaultPath={ctx.openPath.value}
          onCreate={ctx.addProfile}
        />

        <BreakpointConditionDialog
          modelValue={!!ctx.bpDialog.value}
          {...{ "onUpdate:modelValue": (v: boolean) => { if (!v) ctx.bpDialog.value = null; } }}
          line={ctx.bpDialog.value?.line ?? 0}
          initial={ctx.bpDialog.value ? ctx.bpMetaFor(ctx.bpDialog.value.path)[ctx.bpDialog.value.line] ?? {} : {}}
          onSave={(m: { disabled?: boolean; condition?: string }) => {
            if (ctx.bpDialog.value) ctx.setBpMeta(ctx.bpDialog.value.path, ctx.bpDialog.value.line, m);
          }}
        />

        <PromptDialog request={ctx.promptReq.value} onDone={() => (ctx.promptReq.value = null)} />
        <ConfirmDialog request={ctx.confirmReq.value} onDone={() => (ctx.confirmReq.value = null)} />
        <UrlImportDialog
          modelValue={ctx.urlDialog.value}
          {...{ "onUpdate:modelValue": (v: boolean) => (ctx.urlDialog.value = v) }}
          progress={ctx.uploadProgress.value}
          onImport={(url: string, extract: boolean) => ctx.uploadUrl(url, extract)}
        />
      </div>
    );
  },
});
