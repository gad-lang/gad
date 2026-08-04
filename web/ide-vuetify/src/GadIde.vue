<!-- GadIde — a reusable Vuetify 3 IDE for the Gad language. It renders a file
     explorer, a CodeMirror editor and Run / Doc / Debug panels, and is
     backend-agnostic: pass any IdeApi implementation via the `api` prop (the
     HTTP `gad ide` client, or a fully in-browser WASM + LocalStorage backend).
     An optional `onReset` restores a backend's pristine state (e.g. the demo). -->
<script setup lang="ts">
import { computed, onMounted, reactive, ref, shallowRef, watch } from "vue";
import type { GadDiagnostic } from "@gad-lang/codemirror-gad";
import GadEditor from "./GadEditor.vue";
import { langOf } from "./codemirror";
import { renderDocMarkdown } from "./docMarkdown";
import type { DebugResponse, DocComment, IdeApi, TreeNode, Workspace } from "./api";
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
const dialect = computed(() => {
  const l = langOf(openPath.value);
  return l === "giom" ? "giom" : l === "gadt" ? "gadTemplate" : "gad";
});

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
  // Auto-expand ancestors of the open file so it is visible.
  const parts = openPath.value.split("/");
  for (let i = 1; i < parts.length; i++) expanded.add(parts.slice(0, i).join("/"));
}

async function openFile(path: string) {
  openPath.value = path;
  const { content } = await props.api.read(path);
  source.value = content;
  panelTab.value = "run";
}

function toggleDir(path: string) {
  if (expanded.has(path)) expanded.delete(path);
  else expanded.add(path);
}

// Persist edits back to the backend (debounced-ish: on every change).
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

// --- panels ---------------------------------------------------------------
const panelTab = ref<"run" | "doc" | "debug">("run");
const busy = ref(false);

// Run
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

// Doc
const docHtml = ref("");
async function doDoc() {
  busy.value = true;
  try {
    const docs: DocComment[] = await props.api.doc(source.value);
    docHtml.value = docs.length
      ? docs.map((d) => `<h4>${escapeHtml(d.title || d.kind)}</h4>` + renderDocMarkdown(d.content)).join("\n")
      : `<p class="text-medium-emphasis">No documentation comments in this file.</p>`;
  } finally {
    busy.value = false;
  }
}

// Debug
const bpText = ref("");
const session = ref<string | null>(null);
const snap = ref<DebugResponse | null>(null);
const dbgOutput = ref("");
const breakpoints = ref<number[]>([]);
const debugLine = computed(() => (snap.value?.state === "stopped" ? snap.value.line ?? 0 : 0));
const debugColumn = computed(() => snap.value?.column ?? 1);
const stopped = computed(() => snap.value?.state === "stopped");

function bpLines(): number[] {
  const fromText = bpText.value
    .split(",")
    .map((p) => parseInt(p.trim(), 10))
    .filter((n) => !Number.isNaN(n));
  return [...new Set([...breakpoints.value, ...fromText])].sort((a, b) => a - b);
}

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
  try {
    applySnap(
      await props.api.dbgStart({
        source: source.value,
        path: openPath.value,
        breakpoints: bpLines(),
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

// Evaluate (debug)
const evalExpr = ref("");
const evalOut = ref("");
async function doEval() {
  if (!session.value || !evalExpr.value.trim()) return;
  const r = await props.api.dbgEval(session.value, evalExpr.value, true);
  evalOut.value = r.ok ? r.value ?? "" : "error: " + (r.error ?? "");
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
  <div class="gad-ide" :class="{ 'gad-ide--dark': dark }">
    <!-- Explorer -->
    <aside class="gad-ide__explorer">
      <div class="gad-ide__explorer-head">
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

    <!-- Editor -->
    <main class="gad-ide__editor-wrap">
      <div class="gad-ide__editor-head">
        <span class="text-caption">{{ openPath || "(no file)" }}</span>
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
        />
        <div v-else class="pa-4 text-medium-emphasis">Select or create a file to begin.</div>
      </div>
    </main>

    <!-- Panels -->
    <section class="gad-ide__panel">
      <v-tabs v-model="panelTab" density="compact">
        <v-tab value="run">Run</v-tab>
        <v-tab value="doc">Doc</v-tab>
        <v-tab value="debug">Debug</v-tab>
      </v-tabs>
      <div class="gad-ide__panel-body">
        <!-- Run -->
        <div v-show="panelTab === 'run'">
          <div class="mb-2">
            <v-btn size="small" color="primary" :loading="busy" prepend-icon="mdi-play" @click="doRun">Run</v-btn>
            <v-btn size="small" variant="tonal" class="ml-2" :loading="busy" @click="doFormat">Format</v-btn>
          </div>
          <template v-if="runRes">
            <pre v-if="runRes.stdout" class="gad-ide__out">{{ runRes.stdout }}</pre>
            <pre v-if="runRes.stderr" class="gad-ide__out gad-ide__out--err">{{ runRes.stderr }}</pre>
            <div v-if="runRes.ok && runRes.result" class="gad-ide__return">⇦ {{ runRes.result }}</div>
            <div v-for="(d, i) in runRes.diagnostics" :key="i" class="gad-ide__diag">
              {{ d.line }}:{{ d.column }} {{ d.message }}
            </div>
          </template>
        </div>

        <!-- Doc -->
        <div v-show="panelTab === 'doc'">
          <v-btn size="small" color="primary" class="mb-2" :loading="busy" @click="doDoc">Generate docs</v-btn>
          <div class="gad-ide__doc" v-html="docHtml" />
        </div>

        <!-- Debug -->
        <div v-show="panelTab === 'debug'">
          <v-text-field
            v-model="bpText"
            label="Breakpoints (lines, e.g. 2, 5)"
            density="compact"
            variant="outlined"
            hide-details
            :disabled="!!session"
            class="mb-2"
          />
          <div class="mb-2">
            <v-btn size="small" color="primary" :loading="busy" @click="dbgStart">{{ session ? "Restart" : "Start" }}</v-btn>
            <v-btn size="small" variant="tonal" class="ml-1" :disabled="!stopped || busy" @click="dbgCmd('continue')">Continue</v-btn>
            <v-btn size="small" variant="tonal" class="ml-1" :disabled="!stopped || busy" @click="dbgCmd('next')">Step Over</v-btn>
            <v-btn size="small" variant="tonal" class="ml-1" :disabled="!stopped || busy" @click="dbgCmd('stepIn')">Step In</v-btn>
            <v-btn size="small" variant="tonal" class="ml-1" :disabled="!stopped || busy" @click="dbgCmd('stepOut')">Step Out</v-btn>
          </div>

          <template v-if="snap">
            <div v-if="snap.state === 'stopped'" class="text-caption mb-1">
              stopped ({{ snap.reason }}) at {{ snap.line }}:{{ snap.column }}
            </div>
            <div v-else-if="snap.state === 'terminated'" class="gad-ide__return mb-1">
              terminated — returned {{ snap.result || "nil" }}
            </div>
            <div v-else class="gad-ide__out--err mb-1">compile error</div>

            <div v-for="(d, i) in snap.diagnostics" :key="i" class="gad-ide__diag">
              {{ d.line }}:{{ d.column }} {{ d.message }}
            </div>

            <template v-if="stopped">
              <div class="text-caption font-weight-medium mt-2">Call stack</div>
              <ul class="gad-ide__list">
                <li v-for="(f, i) in snap.frames" :key="i">{{ f.name }} <span class="text-medium-emphasis">@ {{ f.line }}:{{ f.column }}</span></li>
              </ul>
              <div class="text-caption font-weight-medium mt-2">Locals</div>
              <ul class="gad-ide__list">
                <li v-for="(v, i) in snap.locals" :key="i">{{ v.name }} = {{ v.value }} <span class="text-medium-emphasis">({{ v.type }})</span></li>
                <li v-if="!snap.locals?.length" class="text-medium-emphasis">(none)</li>
              </ul>
              <div class="d-flex mt-2" style="gap: 4px">
                <v-text-field v-model="evalExpr" label="Evaluate" density="compact" variant="outlined" hide-details
                              @keyup.enter="doEval" />
                <v-btn size="small" variant="tonal" @click="doEval">Eval</v-btn>
              </div>
              <pre v-if="evalOut" class="gad-ide__out">{{ evalOut }}</pre>
            </template>
          </template>

          <pre v-if="dbgOutput" class="gad-ide__out mt-2">{{ dbgOutput }}</pre>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.gad-ide {
  display: grid;
  grid-template-columns: 220px 1fr 340px;
  height: 100%;
  min-height: 0;
  font-size: 13px;
}
.gad-ide__explorer,
.gad-ide__editor-wrap,
.gad-ide__panel {
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  border-right: 1px solid rgba(var(--v-border-color), 0.3);
}
.gad-ide__panel {
  border-right: none;
  border-left: 1px solid rgba(var(--v-border-color), 0.3);
}
.gad-ide__explorer-head,
.gad-ide__editor-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 8px;
  border-bottom: 1px solid rgba(var(--v-border-color), 0.3);
}
.gad-ide__tree {
  flex: 1;
  overflow: auto;
  padding: 4px 0;
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
.gad-ide__panel-body {
  flex: 1;
  overflow: auto;
  padding: 8px;
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
.gad-ide__doc :deep(pre) {
  overflow-x: auto;
}
</style>
