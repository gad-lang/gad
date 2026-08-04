<!-- GadIde — a reusable Vuetify 3 IDE for the Gad language. Layout: a full-height
     file explorer on the left; the editor on the right with its Run / Debug (and
     Format / Doc) actions in the header, and the debugger step controls directly
     below them; and three bottom panels — CALL STACK, LOCALS and STDOUT/STDERR.
     Hovering an identifier while paused shows its live value (like `gad ide`).
     Backend-agnostic: pass any IdeApi via the `api` prop; an optional `onReset`
     restores a backend's pristine state. -->
<script setup lang="ts">
import { computed, onMounted, reactive, ref, shallowRef, watch } from "vue";
import type { GadDiagnostic } from "@gad-lang/codemirror-gad";
import GadEditor from "./GadEditor.vue";
import InspectorNode, { type InspectFn } from "./InspectorNode.vue";
import { langOf } from "./codemirror";
import type { LocalVar } from "./codemirror";
import { renderDocMarkdown } from "./docMarkdown";
import type { DebugResponse, DocComment, IdeApi, InspectResult, TreeNode, Workspace } from "./api";
import type { RunResult } from "./types";

const props = defineProps<{
  api: IdeApi;
  workspace: Workspace;
  dark?: boolean;
  /** When provided, a "Reset" button restores the backend's pristine state. */
  onReset?: () => Promise<void> | void;
}>();

// --- file tree ------------------------------------------------------------
const tree = shallowRef<TreeNode | null>(null);
const expanded = reactive(new Set<string>());
const openPath = ref<string>(props.workspace.openFile || "");
const source = ref<string>("");

interface Row {
  node: TreeNode;
  depth: number;
}

/** Flatten the tree into visible rows honoring the expanded set (dirs open). */
const rows = computed<Row[]>(() => {
  const out: Row[] = [];
  const walk = (node: TreeNode, depth: number) => {
    for (const c of node.children ?? []) {
      out.push({ node: c, depth });
      if (c.dir && expanded.has(c.path)) walk(c, depth + 1);
    }
  };
  if (tree.value) walk(tree.value, 0);
  return out;
});

async function loadTree() {
  tree.value = await props.api.tree();
  const parts = openPath.value.split("/");
  for (let i = 1; i < parts.length; i++) expanded.add(parts.slice(0, i).join("/"));
}

async function openFile(path: string) {
  openPath.value = path;
  const { content } = await props.api.read(path);
  source.value = content;
}

function toggleDir(path: string) {
  if (expanded.has(path)) expanded.delete(path);
  else expanded.add(path);
}

// Persist edits back to the backend on every change.
watch(source, (v) => {
  if (openPath.value) void props.api.write(openPath.value, v);
});

// --- tree mutations -------------------------------------------------------
function currentDir(): string {
  const slash = openPath.value.lastIndexOf("/");
  return slash === -1 ? "" : openPath.value.slice(0, slash + 1);
}

async function newFile() {
  const name = prompt("New file name (e.g. hello.gad):", "untitled.gad");
  if (!name) return;
  const path = currentDir() + name;
  await props.api.mkfile(path);
  await loadTree();
  await openFile(path);
}
async function newDir() {
  const name = prompt("New folder name:", "folder");
  if (!name) return;
  await props.api.mkdir(currentDir() + name);
  await loadTree();
}
async function removeOpen() {
  if (!openPath.value || !confirm("Delete " + openPath.value + "?")) return;
  await props.api.del(openPath.value);
  openPath.value = "";
  source.value = "";
  await loadTree();
}
async function doReset() {
  if (!props.onReset || !confirm("Discard all your changes and restore the samples?")) return;
  await props.onReset();
  await loadTree();
  const first = firstFile(tree.value);
  if (first) await openFile(first);
}
function firstFile(node: TreeNode | null): string | null {
  for (const c of node?.children ?? []) {
    if (!c.dir) return c.path;
    const f = firstFile(c);
    if (f) return f;
  }
  return null;
}

// --- diagnose (linter) ----------------------------------------------------
const diagnose = (src: string): Promise<GadDiagnostic[]> => props.api.diagnose(src);

// --- run / format / doc ---------------------------------------------------
const busy = ref(false);
const runRes = ref<RunResult | null>(null);

async function doRun() {
  busy.value = true;
  try {
    runRes.value = await props.api.run({ source: source.value, path: openPath.value });
  } finally {
    busy.value = false;
  }
}
async function doFormat() {
  busy.value = true;
  try {
    const r = await props.api.format(source.value);
    if (r.ok) source.value = r.source;
    else runRes.value = { ok: false, stdout: "", stderr: "", result: "", diagnostics: r.diagnostics };
  } finally {
    busy.value = false;
  }
}

const docDialog = ref(false);
const docHtml = ref("");
async function doDoc() {
  busy.value = true;
  try {
    const docs: DocComment[] = await props.api.doc(source.value);
    docHtml.value = docs.length
      ? docs.map((d) => `<h4>${escapeHtml(d.title || d.kind)}</h4>` + renderDocMarkdown(d.content)).join("\n")
      : `<p class="text-medium-emphasis">No documentation comments in this file.</p>`;
    docDialog.value = true;
  } finally {
    busy.value = false;
  }
}

// --- debugger -------------------------------------------------------------
const session = ref<string | null>(null);
const snap = ref<DebugResponse | null>(null);
const dbgOutput = ref("");
const breakpoints = ref<number[]>([]);
const debugLine = computed(() => (snap.value?.state === "stopped" ? snap.value.line ?? 0 : 0));
const debugColumn = computed(() => snap.value?.column ?? 1);
const stopped = computed(() => snap.value?.state === "stopped");

// Locals of the current paused frame, keyed by name — read live by the editor's
// hover tooltip so hovering an identifier shows its type and value.
const localsMap = computed<Map<string, LocalVar>>(() => {
  const m = new Map<string, LocalVar>();
  for (const v of snap.value?.locals ?? []) m.set(v.name, v);
  return m;
});
const getLocals = () => localsMap.value;

function applySnap(r: DebugResponse) {
  snap.value = r;
  if (r.output) dbgOutput.value += r.output;
  if (r.state === "terminated" || r.state === "error") session.value = null;
  else if (r.session) session.value = r.session;
}

async function dbgStart() {
  busy.value = true;
  dbgOutput.value = "";
  snap.value = null;
  runRes.value = null;
  try {
    applySnap(
      await props.api.dbgStart({
        source: source.value,
        path: openPath.value,
        breakpoints: [...breakpoints.value].sort((a, b) => a - b),
        stopOnEntry: false,
      }),
    );
  } finally {
    busy.value = false;
  }
}
async function dbgCmd(command: "continue" | "next" | "stepIn" | "stepOut") {
  if (!session.value) return;
  busy.value = true;
  try {
    applySnap(await props.api.dbgCmd(session.value, command));
  } finally {
    busy.value = false;
  }
}

// Evaluate in the paused frame.
const evalExpr = ref("");
const evalOut = ref("");
async function doEval() {
  if (!session.value || !evalExpr.value.trim()) return;
  const r = await props.api.dbgEval(session.value, evalExpr.value, true);
  evalOut.value = r.ok ? r.value ?? "" : "error: " + (r.error ?? "");
}

// --- value inspector (tree navigator) ------------------------------------
const inspectOpen = ref(false);
const inspectExpr = ref("");
const inspectLabel = ref("");

// inspectFn drives the InspectorNode: it evaluates expr in the paused frame when
// a session is active, otherwise fresh against the file's top-level definitions.
const inspectFn: InspectFn = async (expr: string): Promise<InspectResult | null> => {
  const r = await props.api.inspect({
    expr,
    session: session.value ?? undefined,
    source: source.value,
    path: openPath.value,
  });
  return r.ok ? r.inspect ?? null : null;
};

function openInspect(expr: string, label: string) {
  if (!expr.trim()) return;
  inspectExpr.value = expr;
  inspectLabel.value = label;
  inspectOpen.value = true;
}

function escapeHtml(s: string): string {
  return s.replace(/[&<>]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;" })[c] as string);
}

onMounted(async () => {
  await loadTree();
  if (openPath.value) await openFile(openPath.value);
  else {
    const first = firstFile(tree.value);
    if (first) await openFile(first);
  }
});
</script>

<template>
  <div class="gad-ide">
    <!-- Explorer (full height, left) -->
    <aside class="gad-ide__explorer">
      <div class="gad-ide__head">
        <span class="text-caption font-weight-medium">EXPLORER</span>
        <div>
          <v-btn size="x-small" variant="text" icon="mdi-file-plus-outline" title="New file" @click="newFile" />
          <v-btn size="x-small" variant="text" icon="mdi-folder-plus-outline" title="New folder" @click="newDir" />
          <v-btn size="x-small" variant="text" icon="mdi-delete-outline" title="Delete open file"
                 :disabled="!openPath" @click="removeOpen" />
          <v-btn v-if="onReset" size="x-small" variant="text" icon="mdi-backup-restore" title="Reset changes"
                 @click="doReset" />
        </div>
      </div>
      <div class="gad-ide__tree">
        <div
          v-for="row in rows"
          :key="row.node.path"
          class="gad-ide__row"
          :class="{ 'gad-ide__row--active': row.node.path === openPath }"
          :style="{ paddingLeft: 6 + row.depth * 14 + 'px' }"
          @click="row.node.dir ? toggleDir(row.node.path) : openFile(row.node.path)"
        >
          <v-icon size="16" class="mr-1">
            {{ row.node.dir ? (expanded.has(row.node.path) ? "mdi-chevron-down" : "mdi-chevron-right") : "mdi-file-outline" }}
          </v-icon>
          <span class="gad-ide__row-name">{{ row.node.name }}</span>
        </div>
      </div>
    </aside>

    <!-- Editor with actions in the header right, debug controls below them -->
    <main class="gad-ide__editor-wrap">
      <div class="gad-ide__editor-head">
        <span class="text-caption gad-ide__path">{{ openPath || "(no file)" }}</span>
        <div class="gad-ide__actions">
          <div class="gad-ide__actions-row">
            <v-btn size="small" variant="tonal" :loading="busy" @click="doFormat">Format</v-btn>
            <v-btn size="small" variant="tonal" :loading="busy" @click="doDoc">Doc</v-btn>
            <v-btn size="small" color="primary" prepend-icon="mdi-play" :loading="busy" @click="doRun">Run</v-btn>
            <v-btn size="small" color="secondary" prepend-icon="mdi-bug" :loading="busy" @click="dbgStart">
              {{ session ? "Restart" : "Debug" }}
            </v-btn>
          </div>
          <div class="gad-ide__actions-row">
            <v-btn size="x-small" variant="text" prepend-icon="mdi-play-outline"
                   :disabled="!stopped || busy" @click="dbgCmd('continue')">Continue</v-btn>
            <v-btn size="x-small" variant="text" prepend-icon="mdi-debug-step-over"
                   :disabled="!stopped || busy" @click="dbgCmd('next')">Step Over</v-btn>
            <v-btn size="x-small" variant="text" prepend-icon="mdi-debug-step-into"
                   :disabled="!stopped || busy" @click="dbgCmd('stepIn')">Step In</v-btn>
            <v-btn size="x-small" variant="text" prepend-icon="mdi-debug-step-out"
                   :disabled="!stopped || busy" @click="dbgCmd('stepOut')">Step Out</v-btn>
            <span v-if="snap?.state === 'stopped'" class="text-caption ml-2 align-self-center">
              stopped ({{ snap.reason }}) @ {{ snap.line }}:{{ snap.column }}
            </span>
            <span v-else-if="snap?.state === 'terminated'" class="text-caption ml-2 align-self-center gad-ide__return">
              terminated — {{ snap.result || "nil" }}
            </span>
          </div>
        </div>
      </div>
      <div class="gad-ide__editor">
        <GadEditor
          v-if="openPath"
          v-model="source"
          v-model:breakpoints="breakpoints"
          :path="openPath"
          :dark="dark"
          :diagnose="diagnose"
          :debug-line="debugLine"
          :debug-column="debugColumn"
          :get-locals="getLocals"
        />
        <div v-else class="pa-4 text-medium-emphasis">Select or create a file to begin.</div>
      </div>
    </main>

    <!-- Bottom panels: CALL STACK | LOCALS | STDOUT/STDERR -->
    <section class="gad-ide__panels">
      <div class="gad-ide__panel">
        <div class="gad-ide__head"><span class="text-caption font-weight-medium">CALL STACK</span></div>
        <div class="gad-ide__panel-body">
          <ul class="gad-ide__list">
            <li v-for="(f, i) in snap?.frames ?? []" :key="i">
              {{ f.name }} <span class="text-medium-emphasis">@ {{ f.line }}:{{ f.column }}</span>
            </li>
            <li v-if="!snap?.frames?.length" class="text-medium-emphasis">(not paused)</li>
          </ul>
        </div>
      </div>

      <div class="gad-ide__panel">
        <div class="gad-ide__head"><span class="text-caption font-weight-medium">EVALUATE &amp; LOCALS</span></div>
        <div class="gad-ide__panel-body">
          <!-- EVALUATE — always above LOCALS -->
          <div class="d-flex align-center" style="gap: 4px">
            <v-text-field v-model="evalExpr" label="Evaluate" density="compact" variant="outlined" hide-details
                          @keyup.enter="doEval" />
            <v-btn size="small" variant="tonal" :disabled="!stopped" @click="doEval">Eval</v-btn>
            <v-btn size="x-small" variant="text" icon="mdi-file-tree-outline" title="Inspect value"
                   :disabled="!evalExpr.trim()" @click="openInspect(evalExpr, evalExpr)" />
          </div>
          <pre v-if="evalOut" class="gad-ide__out">{{ evalOut }}</pre>

          <!-- LOCALS -->
          <div class="text-caption text-medium-emphasis mt-2 mb-1">LOCALS</div>
          <ul class="gad-ide__list">
            <li v-for="(v, i) in snap?.locals ?? []" :key="i" class="gad-ide__local">
              <span class="gad-ide__local-text">
                {{ v.name }} = {{ v.value }} <span class="text-medium-emphasis">({{ v.type }})</span>
              </span>
              <v-btn size="x-small" variant="text" icon="mdi-file-tree-outline" title="Inspect value"
                     @click="openInspect(v.name, v.name)" />
            </li>
            <li v-if="!snap?.locals?.length" class="text-medium-emphasis">(none)</li>
          </ul>
        </div>
      </div>

      <div class="gad-ide__panel">
        <div class="gad-ide__head"><span class="text-caption font-weight-medium">STDOUT / STDERR</span></div>
        <div class="gad-ide__panel-body">
          <template v-if="runRes">
            <pre v-if="runRes.stdout" class="gad-ide__out">{{ runRes.stdout }}</pre>
            <pre v-if="runRes.stderr" class="gad-ide__out gad-ide__out--err">{{ runRes.stderr }}</pre>
            <div v-if="runRes.ok && runRes.result" class="gad-ide__return">⇦ {{ runRes.result }}</div>
            <div v-for="(d, i) in runRes.diagnostics" :key="i" class="gad-ide__diag">{{ d.line }}:{{ d.column }} {{ d.message }}</div>
          </template>
          <pre v-if="dbgOutput" class="gad-ide__out">{{ dbgOutput }}</pre>
          <div v-for="(d, i) in snap?.diagnostics ?? []" :key="'s' + i" class="gad-ide__diag">{{ d.line }}:{{ d.column }} {{ d.message }}</div>
          <div v-if="!runRes && !dbgOutput" class="text-medium-emphasis">Run or debug to see output.</div>
        </div>
      </div>
    </section>

    <!-- Value inspector dialog (tree navigator) -->
    <v-dialog v-model="inspectOpen" max-width="720" scrollable>
      <v-card>
        <v-card-title class="text-subtitle-1">Inspect — {{ inspectLabel }}</v-card-title>
        <v-card-text class="gad-ide__inspect">
          <InspectorNode
            v-if="inspectOpen"
            :key="inspectExpr"
            :inspect="inspectFn"
            :label="inspectLabel"
            :expr="inspectExpr"
            root
          />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn @click="inspectOpen = false">Close</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Doc dialog -->
    <v-dialog v-model="docDialog" max-width="720" scrollable>
      <v-card>
        <v-card-title class="text-subtitle-1">Documentation — {{ openPath }}</v-card-title>
        <v-card-text>
          <div class="gad-ide__doc" v-html="docHtml" />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn @click="docDialog = false">Close</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<style scoped>
.gad-ide {
  display: grid;
  grid-template-columns: 220px 1fr;
  grid-template-rows: minmax(0, 1fr) 220px;
  grid-template-areas:
    "explorer editor"
    "explorer panels";
  height: 100%;
  min-height: 0;
  font-size: 13px;
}
.gad-ide__explorer {
  grid-area: explorer;
  border-right: 1px solid rgba(var(--v-border-color), 0.3);
}
.gad-ide__editor-wrap {
  grid-area: editor;
}
.gad-ide__panels {
  grid-area: panels;
  display: grid;
  grid-template-columns: 1fr 1fr 1.2fr;
  border-top: 1px solid rgba(var(--v-border-color), 0.3);
}
.gad-ide__explorer,
.gad-ide__editor-wrap {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.gad-ide__panel {
  min-width: 0;
  display: flex;
  flex-direction: column;
  border-right: 1px solid rgba(var(--v-border-color), 0.3);
}
.gad-ide__panel:last-child {
  border-right: none;
}
.gad-ide__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 8px;
  border-bottom: 1px solid rgba(var(--v-border-color), 0.3);
}
.gad-ide__editor-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
  padding: 4px 8px;
  border-bottom: 1px solid rgba(var(--v-border-color), 0.3);
}
.gad-ide__path {
  align-self: center;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gad-ide__actions {
  display: flex;
  flex-direction: column;
  gap: 4px;
  align-items: flex-end;
}
.gad-ide__actions-row {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  justify-content: flex-end;
}
.gad-ide__tree,
.gad-ide__panel-body {
  flex: 1;
  overflow: auto;
}
.gad-ide__tree {
  padding: 4px 0;
}
.gad-ide__panel-body {
  padding: 6px 8px;
}
.gad-ide__row {
  display: flex;
  align-items: center;
  padding: 2px 6px;
  cursor: pointer;
  white-space: nowrap;
}
.gad-ide__row:hover {
  background: rgba(var(--v-theme-primary), 0.08);
}
.gad-ide__row--active {
  background: rgba(var(--v-theme-primary), 0.16);
}
.gad-ide__row-name {
  overflow: hidden;
  text-overflow: ellipsis;
}
.gad-ide__editor {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}
.gad-ide__out {
  white-space: pre-wrap;
  font-family: ui-monospace, monospace;
  font-size: 12px;
  margin: 4px 0;
}
.gad-ide__out--err {
  color: rgb(var(--v-theme-error));
}
.gad-ide__return {
  color: rgb(var(--v-theme-success, 46, 125, 50));
  font-family: ui-monospace, monospace;
}
.gad-ide__diag {
  color: rgb(var(--v-theme-error));
  font-family: ui-monospace, monospace;
  font-size: 12px;
}
.gad-ide__list {
  list-style: none;
  padding-left: 0;
  margin: 2px 0;
  font-family: ui-monospace, monospace;
  font-size: 12px;
}
.gad-ide__local {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 4px;
}
.gad-ide__local-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gad-ide__inspect {
  max-height: 60vh;
  overflow: auto;
}
.gad-ide__doc :deep(pre) {
  overflow-x: auto;
}
</style>
