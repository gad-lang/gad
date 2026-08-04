// GadIde — a reusable Vuetify 3 IDE for the Gad language. Panels are hosted in a
// dockview-vue layout (resizable, movable, dockable, tabbable): Explorer, Editor,
// Call Stack, Locals and Output. A top toolbar carries the Run/Debug/Format/Doc/
// Settings actions and the debugger step controls. Two independent v-models:
// `layoutConfig` (the dockview layout, toJSON/fromJSON) and `config` (the project
// settings document edited by the Settings dialog). Backend-agnostic via `api`.
import { computed, defineComponent, onBeforeUnmount, onMounted, provide, reactive, ref, watch, type PropType } from "vue";
import { DockviewVue } from "dockview-vue";
import type { DockviewApi, DockviewReadyEvent, SerializedDockview, VueComponent } from "dockview-vue";
import { VBtn, VCard, VCardActions, VCardText, VCardTitle, VDialog, VSpacer } from "./vuetify";
import { createController, IdeControllerKey } from "./controller";
import type { IdeApi, Workspace } from "./api";
import PanelExplorer from "./panels/PanelExplorer";
import PanelEditor from "./panels/PanelEditor";
import PanelCallStack from "./panels/PanelCallStack";
import PanelLocals from "./panels/PanelLocals";
import PanelOutput from "./panels/PanelOutput";
import SettingsDialog, { type PanelToggle } from "./SettingsDialog";
import { renderDocMarkdown } from "./docMarkdown";

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
  { id: "output", label: "Output", add: (a) => a.addPanel({ id: "output", component: "output", title: "Output", position: { referencePanel: "editor", direction: "below" } }) },
  { id: "callstack", label: "Call Stack", add: (a) => a.addPanel({ id: "callstack", component: "callstack", title: "Call Stack", position: { referencePanel: "output", direction: "within" } }) },
  { id: "locals", label: "Locals", add: (a) => a.addPanel({ id: "locals", component: "locals", title: "Locals", position: { referencePanel: "output", direction: "within" } }) },
];

const components = {
  explorer: PanelExplorer,
  editor: PanelEditor,
  callstack: PanelCallStack,
  locals: PanelLocals,
  output: PanelOutput,
} as unknown as Record<string, VueComponent>;

export default defineComponent({
  name: "GadIde",
  props: {
    api: { type: Object as PropType<IdeApi>, required: true },
    workspace: { type: Object as PropType<Workspace>, required: true },
    dark: { type: Boolean, default: false },
    onReset: { type: Function as PropType<() => Promise<void> | void>, default: undefined },
    layoutConfig: { type: Object as PropType<SerializedDockview | null>, default: null },
    config: { type: Object as PropType<Record<string, unknown>>, default: () => ({}) },
  },
  emits: {
    "update:layoutConfig": (_v: SerializedDockview) => true,
    "update:config": (_v: Record<string, unknown>) => true,
  },
  setup(props, { emit }) {
    const dark = computed(() => props.dark);
    const ctx = createController(props.api, props.workspace, dark, props.onReset);
    provide(IdeControllerKey, ctx);

    let dv: DockviewApi | null = null;
    let disposer: { dispose(): void } | null = null;
    let applyingExternal = false;
    let lastJSON = "";

    const visible = reactive(new Set<string>());
    const settingsOpen = ref(false);
    const docOpen = ref(false);
    const docHtml = ref("");

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
        if (initial && (initial as { grid?: unknown }).grid) {
          dv.fromJSON(initial);
        } else {
          buildDefault(dv);
        }
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

    async function doDoc() {
      const docs = await ctx.loadDoc();
      docHtml.value = docs.length
        ? docs.map((d) => `<h4>${escapeHtml(d.title || d.kind)}</h4>` + renderDocMarkdown(d.content)).join("\n")
        : `<p class="text-medium-emphasis">No documentation comments in this file.</p>`;
      docOpen.value = true;
    }

    function saveConfig(next: Record<string, unknown>) {
      emit("update:config", next);
    }

    onMounted(() => void ctx.init());
    onBeforeUnmount(() => disposer?.dispose());

    const themeClass = computed(() => (props.dark ? "dockview-theme-dark" : "dockview-theme-light"));

    return () => (
      <div class="gad-ide">
        <div class="gad-ide__toolbar">
          <span class="text-caption pnl-ellipsis" style={{ alignSelf: "center" }}>
            {ctx.openPath.value || props.workspace.name}
          </span>
          <div class="gad-ide__actions">
            <div class="gad-ide__actions-row">
              <VBtn size="small" variant="tonal" loading={ctx.busy.value} onClick={() => ctx.format()}>Format</VBtn>
              <VBtn size="small" variant="tonal" loading={ctx.busy.value} onClick={doDoc}>Doc</VBtn>
              <VBtn size="small" color="primary" prependIcon="mdi-play" loading={ctx.busy.value} onClick={() => ctx.run()}>Run</VBtn>
              <VBtn size="small" color="secondary" prependIcon="mdi-bug" loading={ctx.busy.value} onClick={() => ctx.debugStart()}>
                {ctx.session.value ? "Restart" : "Debug"}
              </VBtn>
              <VBtn size="small" variant="text" icon="mdi-cog-outline" title="Settings" onClick={() => (settingsOpen.value = true)} />
            </div>
            <div class="gad-ide__actions-row">
              <VBtn size="x-small" variant="text" prependIcon="mdi-play-outline" disabled={!ctx.stopped.value || ctx.busy.value} onClick={() => ctx.debugCmd("continue")}>Continue</VBtn>
              <VBtn size="x-small" variant="text" prependIcon="mdi-debug-step-over" disabled={!ctx.stopped.value || ctx.busy.value} onClick={() => ctx.debugCmd("next")}>Step Over</VBtn>
              <VBtn size="x-small" variant="text" prependIcon="mdi-debug-step-into" disabled={!ctx.stopped.value || ctx.busy.value} onClick={() => ctx.debugCmd("stepIn")}>Step In</VBtn>
              <VBtn size="x-small" variant="text" prependIcon="mdi-debug-step-out" disabled={!ctx.stopped.value || ctx.busy.value} onClick={() => ctx.debugCmd("stepOut")}>Step Out</VBtn>
              {ctx.snap.value?.state === "stopped" && (
                <span class="text-caption ml-2 align-self-center">stopped ({ctx.snap.value.reason}) @ {ctx.snap.value.line}:{ctx.snap.value.column}</span>
              )}
              {ctx.snap.value?.state === "terminated" && (
                <span class="text-caption ml-2 align-self-center pnl-return">terminated — {ctx.snap.value.result || "nil"}</span>
              )}
            </div>
          </div>
        </div>

        <div class="gad-ide__dock">
          <DockviewVue
            style={{ height: "100%" }}
            class={themeClass.value}
            components={components}
            onReady={onReady}
          />
        </div>

        <SettingsDialog
          modelValue={settingsOpen.value}
          {...{ "onUpdate:modelValue": (v: boolean) => (settingsOpen.value = v) }}
          config={props.config}
          panels={panelToggles.value}
          onTogglePanel={togglePanel}
          onSave={saveConfig}
        />

        <VDialog
          modelValue={docOpen.value}
          {...{ "onUpdate:modelValue": (v: boolean) => (docOpen.value = v) }}
          maxWidth="720"
          scrollable
        >
          <VCard>
            <VCardTitle class="text-subtitle-1">Documentation — {ctx.openPath.value}</VCardTitle>
            <VCardText>
              <div class="gad-ide__doc" innerHTML={docHtml.value} />
            </VCardText>
            <VCardActions>
              <VSpacer />
              <VBtn onClick={() => (docOpen.value = false)}>Close</VBtn>
            </VCardActions>
          </VCard>
        </VDialog>
      </div>
    );
  },
});

function escapeHtml(s: string): string {
  return s.replace(/[&<>]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;" })[c] as string);
}
