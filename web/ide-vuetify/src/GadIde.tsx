// GadIde — a reusable Vuetify 3 IDE for the Gad language. Panels are hosted in a
// dockview-vue layout (resizable, movable, dockable, tabbable): Explorer, Editor,
// Call Stack, Locals and Output. A top toolbar carries the Run/Debug/Format/Doc/
// Settings actions and the debugger step controls. Two independent v-models:
// `layoutConfig` (the dockview layout, toJSON/fromJSON) and `config` (the project
// settings document edited by the Settings dialog). Backend-agnostic via `api`.
import { computed, defineComponent, onBeforeUnmount, onMounted, provide, reactive, ref, watch, type PropType } from "vue";
import { DockviewVue, themeDark, themeLight } from "dockview-vue";
import type { DockviewApi, DockviewReadyEvent, SerializedDockview, VueComponent } from "dockview-vue";
import { VBtn, VList, VListItem, VListSubheader, VMenu } from "./vuetify";
import { createController, IdeControllerKey, type RunTarget } from "./controller";
import type { IdeApi, RunMode, RunProfile, Workspace } from "./api";
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
    /** When provided, a light/dark toggle appears next to Settings. */
    onToggleTheme: { type: Function as PropType<() => void>, default: undefined },
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
    const ctx = createController(props.api, props.workspace, dark, props.onReset);
    provide(IdeControllerKey, ctx);

    let dv: DockviewApi | null = null;
    let disposer: { dispose(): void } | null = null;
    let applyingExternal = false;
    let lastJSON = "";

    const visible = reactive(new Set<string>());
    const settingsOpen = ref(false);

    // --- run/debug profiles (JetBrains-style) -----------------------------
    const activeProfile = ref<string | null>(null);
    const profileDialog = ref(false);
    const activeProfileObj = computed(() => props.runProfiles.find((p) => p.name === activeProfile.value) ?? null);
    const runLabel = computed(() => activeProfileObj.value?.name ?? "Current file");
    // runMode gates the actions: "run"/"debug" enable Run (+ profile selector);
    // only "debug" enables Debug; "none"/"" disables all three.
    const canRun = computed(() => props.runMode === "run" || props.runMode === "debug");
    const canDebug = computed(() => props.runMode === "debug");

    // Resolve what Run/Debug execute: the active profile's file + args, else the
    // open file. The profile file's content is read fresh (unless it is open).
    async function effectiveTarget(): Promise<RunTarget> {
      const p = activeProfileObj.value;
      if (!p) return { source: ctx.source.value, path: ctx.openPath.value, args: [] };
      const source = p.path === ctx.openPath.value ? ctx.source.value : (await props.api.read(p.path)).content;
      return { source, path: p.path, args: p.args };
    }
    async function runActive() {
      await ctx.run(await effectiveTarget());
    }
    async function debugActive() {
      await ctx.debugStart(await effectiveTarget());
    }
    function addProfile(p: RunProfile) {
      emit("update:runProfiles", [...props.runProfiles.filter((x) => x.name !== p.name), p]);
      activeProfile.value = p.name;
    }
    function deleteProfile(name: string) {
      emit("update:runProfiles", props.runProfiles.filter((p) => p.name !== name));
      if (activeProfile.value === name) activeProfile.value = null;
    }

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

    // Reveal (re-adding if closed) and refresh the Docs panel.
    async function revealDocs() {
      if (dv && !dv.getPanel("docs")) PANELS.find((p) => p.id === "docs")?.add(dv);
      dv?.getPanel("docs")?.api.setActive();
      await ctx.refreshDoc();
    }

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
        <div class="gad-ide__toolbar">
          <span class="text-caption pnl-ellipsis" style={{ alignSelf: "center" }}>
            {ctx.openPath.value || props.workspace.name}
          </span>
          <div class="gad-ide__actions">
            <div class="gad-ide__actions-row">
              <VBtn size="small" variant="tonal" loading={ctx.busy.value} onClick={() => ctx.format()}>Format</VBtn>
              <VBtn size="small" variant="tonal" loading={ctx.busy.value} onClick={revealDocs}>Doc</VBtn>
              <VBtn size="small" color="primary" prependIcon="mdi-play" loading={ctx.busy.value} disabled={!canRun.value} onClick={runActive}>Run</VBtn>
              <VBtn size="small" color="secondary" prependIcon="mdi-bug" loading={ctx.busy.value} disabled={!canDebug.value} onClick={debugActive}>
                {ctx.session.value ? "Restart" : "Debug"}
              </VBtn>
              {/* Run/debug profile selector (JetBrains-style "…" menu). */}
              <VMenu location="bottom end" disabled={!canRun.value}>
                {{
                  activator: ({ props: menuProps }: { props: Record<string, unknown> }) => (
                    <VBtn size="small" variant="text" class="text-none" appendIcon="mdi-chevron-down" disabled={!canRun.value} {...menuProps}>
                      {runLabel.value}
                    </VBtn>
                  ),
                  default: () => (
                    <VList density="compact" minWidth="220">
                      <VListItem
                        title="Current file"
                        active={activeProfile.value === null}
                        onClick={() => (activeProfile.value = null)}
                      />
                      {props.runProfiles.length > 0 && <VListSubheader>Profiles</VListSubheader>}
                      {props.runProfiles.map((p) => (
                        <VListItem
                          key={p.name}
                          title={p.name}
                          subtitle={p.path + (p.args.length ? " " + p.args.join(" ") : "")}
                          active={activeProfile.value === p.name}
                          onClick={() => (activeProfile.value = p.name)}
                        >
                          {{
                            append: () => (
                              <VBtn
                                size="x-small"
                                variant="text"
                                icon="mdi-delete-outline"
                                title="Delete profile"
                                onClick={(e: Event) => {
                                  e.stopPropagation();
                                  deleteProfile(p.name);
                                }}
                              />
                            ),
                          }}
                        </VListItem>
                      ))}
                      <VListItem
                        title="New profile…"
                        prependIcon="mdi-plus"
                        onClick={() => (profileDialog.value = true)}
                      />
                    </VList>
                  ),
                }}
              </VMenu>
              <VBtn size="small" variant="text" icon="mdi-cog-outline" title="Settings" onClick={() => (settingsOpen.value = true)} />
              {props.onToggleTheme && (
                <VBtn
                  size="small"
                  variant="text"
                  icon={props.dark ? "mdi-weather-sunny" : "mdi-weather-night"}
                  title="Toggle light/dark"
                  onClick={() => props.onToggleTheme!()}
                />
              )}
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
          <DockviewVue style={{ height: "100%" }} theme={dvTheme.value} components={components} onReady={onReady} />
        </div>

        <SettingsDialog
          modelValue={settingsOpen.value}
          {...{ "onUpdate:modelValue": (v: boolean) => (settingsOpen.value = v) }}
          config={props.config}
          panels={panelToggles.value}
          onTogglePanel={togglePanel}
          onSave={saveConfig}
        />

        <RunProfileDialog
          modelValue={profileDialog.value}
          {...{ "onUpdate:modelValue": (v: boolean) => (profileDialog.value = v) }}
          defaultPath={ctx.openPath.value}
          onCreate={addProfile}
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
      </div>
    );
  },
});
