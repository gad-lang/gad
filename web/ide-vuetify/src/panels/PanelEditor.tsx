// Editor dockview panel: the CodeMirror editor bound to the open file, with the
// editor/run control toolbar (Save, Format, Reload, Undo, Redo, Run, Debug, the
// run-profile selector and Doc, plus the debugger step controls while paused).
import { defineComponent, inject } from "vue";
import GadEditor from "../GadEditor";
import { VBtn, VDivider, VList, VListItem, VListSubheader, VMenu } from "../vuetify";
import { IdeControllerKey } from "../controller";
import type { GadEditorView } from "../codemirror";
import type { RunProfile } from "../api";

const baseName = (path: string) => path.slice(path.lastIndexOf("/") + 1);
const truncate = (s: string, n: number) => (s.length > n ? s.slice(0, n - 1) + "…" : s);

export default defineComponent({
  name: "PanelEditor",
  setup() {
    const ctx = inject(IdeControllerKey)!;
    const has = () => !!ctx.openPath.value;

    const iconBtn = (icon: string, title: string, onClick: () => void, opts: { disabled?: boolean; color?: string } = {}) => (
      <VBtn size="small" variant="text" icon={icon} title={title} disabled={opts.disabled} color={opts.color} onClick={onClick} />
    );

    return () => (
      <div class="pnl">
        {/* Open-file tabs — basename truncated to 15 chars; full name on hover. */}
        {ctx.tabs.value.length > 0 && (
          <div class="editor-tabs">
            {ctx.tabs.value.map((t, i) => {
              const dirty = ctx.isDirty(t.path);
              return (
                <div
                  key={t.path}
                  class={["editor-tab", { "editor-tab--active": i === ctx.active.value, "editor-tab--dirty": dirty }]}
                  title={baseName(t.path) + (dirty ? " • modified" : "")}
                  onClick={() => ctx.activateTab(i)}
                  onMousedown={(e: MouseEvent) => { if (e.button === 1) { e.preventDefault(); ctx.closeTab(i); } }}
                >
                  <span class="editor-tab-name">{truncate(baseName(t.path), ctx.tabNameMax.value)}</span>
                  <span
                    class={["editor-tab-close", { "editor-tab-close--dirty": dirty }]}
                    title="Close"
                    onClick={(e: MouseEvent) => { e.stopPropagation(); ctx.closeTab(i); }}
                  />
                </div>
              );
            })}
          </div>
        )}
        <div class="pnl-toolbar">
          {ctx.canEdit.value && (
            <>
              {iconBtn("mdi-content-save-outline", "Save", () => ctx.save(), { disabled: !has() })}
              {iconBtn("mdi-auto-fix", "Format", () => ctx.format(), { disabled: !has() })}
            </>
          )}
          {iconBtn("mdi-refresh", "Reload from disk", () => ctx.reload(), { disabled: !has() })}
          {ctx.canEdit.value && (
            <>
              {iconBtn("mdi-undo", "Undo", () => ctx.undo(), { disabled: !has() })}
              {iconBtn("mdi-redo", "Redo", () => ctx.redo(), { disabled: !has() })}
            </>
          )}
          <VDivider vertical class="mx-1" />
          {iconBtn("mdi-play", "Run", () => ctx.runActive(), { disabled: !has() || !ctx.canRun.value, color: "success" })}
          {iconBtn(ctx.session.value ? "mdi-restart" : "mdi-bug", ctx.session.value ? "Restart" : "Debug",
            () => ctx.debugActive(), { disabled: !has() || !ctx.canDebug.value, color: "warning" })}
          {/* Run/debug profile selector ("…" menu). */}
          <VMenu location="bottom start" disabled={!ctx.canRun.value}>
            {{
              activator: ({ props: menuProps }: { props: Record<string, unknown> }) => (
                <VBtn size="small" variant="text" class="text-none" appendIcon="mdi-chevron-down" disabled={!ctx.canRun.value} {...menuProps}>
                  {ctx.runLabel.value}
                </VBtn>
              ),
              default: () => (
                <VList density="compact" minWidth="220">
                  <VListItem title="Current file" active={ctx.activeProfile.value === null} onClick={() => (ctx.activeProfile.value = null)} />
                  {ctx.runProfiles.value.length > 0 && <VListSubheader>Profiles</VListSubheader>}
                  {ctx.runProfiles.value.map((p: RunProfile) => (
                    <VListItem
                      key={p.name}
                      title={p.name}
                      subtitle={p.path + (p.args.length ? " " + p.args.join(" ") : "")}
                      active={ctx.activeProfile.value === p.name}
                      onClick={() => (ctx.activeProfile.value = p.name)}
                    >
                      {{
                        append: () => (
                          <VBtn size="x-small" variant="text" icon="mdi-delete-outline" title="Delete profile"
                            onClick={(e: Event) => { e.stopPropagation(); ctx.deleteProfile(p.name); }} />
                        ),
                      }}
                    </VListItem>
                  ))}
                  <VListItem title="New profile…" prependIcon="mdi-plus" onClick={() => (ctx.profileDialog.value = true)} />
                </VList>
              ),
            }}
          </VMenu>
          {/* Debugger step controls (shown while paused). */}
          {ctx.stopped.value && (
            <>
              <VDivider vertical class="mx-1" />
              {iconBtn("mdi-play-outline", "Continue", () => ctx.debugCmd("continue"), { disabled: ctx.busy.value })}
              {iconBtn("mdi-debug-step-over", "Step Over", () => ctx.debugCmd("next"), { disabled: ctx.busy.value })}
              {iconBtn("mdi-debug-step-into", "Step In", () => ctx.debugCmd("stepIn"), { disabled: ctx.busy.value })}
              {iconBtn("mdi-debug-step-out", "Step Out", () => ctx.debugCmd("stepOut"), { disabled: ctx.busy.value })}
            </>
          )}

          {ctx.snap.value?.state === "stopped" && (
            <span class="text-caption ml-2">stopped ({ctx.snap.value.reason}) @ {ctx.snap.value.line}:{ctx.snap.value.column}</span>
          )}

          {/* Right-aligned: Doc then Settings. */}
          <span class="pnl-toolbar-spacer" />
          {iconBtn("mdi-file-document-outline", "Doc", () => ctx.requestDocs(), { disabled: !has() })}
          {iconBtn("mdi-cog-outline", "Settings", () => (ctx.settingsOpen.value = true))}
        </div>
        <div class="pnl-editor">
          {has() ? (
            <GadEditor
              key={ctx.openPath.value}
              modelValue={ctx.source.value}
              {...{ "onUpdate:modelValue": (v: string) => (ctx.source.value = v) }}
              breakpoints={ctx.breakpoints.value}
              {...{ "onUpdate:breakpoints": (b: number[]) => (ctx.breakpoints.value = b) }}
              path={ctx.openPath.value}
              dark={ctx.dark.value}
              readonly={ctx.readonly.value}
              customExtension={ctx.fileTypes.extensionFor(ctx.openPath.value)}
              diagnose={ctx.diagnose}
              debugLine={ctx.debugLine.value}
              debugColumn={ctx.debugColumn.value}
              getLocals={ctx.getLocals}
              gotoLine={ctx.gotoTarget.value.line}
              gotoSeq={ctx.gotoTarget.value.seq}
              onBreakpointContext={(line: number) => ctx.openBpCondition(ctx.openPath.value, line)}
              onReady={(v: GadEditorView) => ctx.registerEditor(v)}
            />
          ) : (
            <div class="pa-4 text-medium-emphasis">Select or create a file to begin.</div>
          )}
        </div>
        {/* Thin status bar: the full path of the open file. */}
        <div class="editor-statusbar" title={ctx.openPath.value}>{ctx.openPath.value || "(no file)"}</div>
      </div>
    );
  },
});
