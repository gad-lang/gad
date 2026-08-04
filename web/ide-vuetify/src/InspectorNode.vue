<!-- InspectorNode — one lazily-expanding node of the value tree navigator. It
     shows a value's key, type and rendered value; expandable containers
     (array/dict/module/…) fetch their children on first expand via the injected
     `inspect` function and render child InspectorNodes recursively, reaching each
     child by appending its Gad accessor to this node's expression. -->
<script setup lang="ts">
import { onMounted, ref } from "vue";
import type { InspectEntry, InspectResult } from "./api";

export type InspectFn = (expr: string) => Promise<InspectResult | null>;

const props = defineProps<{
  inspect: InspectFn;
  /** Display label (dict key, array index, or the root expression). */
  label: string;
  /** Gad expression that reaches this value from the root. */
  expr: string;
  type?: string;
  value?: string;
  expandable?: boolean;
  depth?: number;
  /** The root node fetches its own type/value/entries on mount. */
  root?: boolean;
}>();

const depth = props.depth ?? 0;
const open = ref(false);
const loading = ref(false);
const entries = ref<InspectEntry[] | null>(null);
const selfType = ref(props.type ?? "");
const selfValue = ref(props.value ?? "");
const selfExpandable = ref(!!props.expandable);

async function load() {
  loading.value = true;
  const r = await props.inspect(props.expr);
  loading.value = false;
  if (!r) {
    entries.value = [];
    return;
  }
  selfType.value = r.type;
  selfValue.value = r.value;
  selfExpandable.value = r.expandable;
  entries.value = r.entries ?? [];
}

async function toggle() {
  if (!selfExpandable.value) return;
  if (entries.value === null) await load();
  open.value = !open.value;
}

onMounted(() => {
  if (props.root) void load();
});
</script>

<template>
  <div class="ins-node">
    <div class="ins-row" :style="{ paddingLeft: depth * 14 + 'px' }" @click="toggle">
      <span class="ins-twist">{{ selfExpandable ? (open ? "▾" : "▸") : "·" }}</span>
      <span class="ins-key">{{ label }}</span>
      <span class="ins-type">{{ selfType }}</span>
      <span class="ins-val">{{ selfValue }}</span>
      <span v-if="loading" class="ins-load">…</span>
    </div>
    <template v-if="open && entries">
      <InspectorNode
        v-for="e in entries"
        :key="e.accessor"
        :inspect="inspect"
        :label="e.key"
        :expr="expr + e.accessor"
        :type="e.type"
        :value="e.value"
        :expandable="e.expandable"
        :depth="depth + 1"
      />
    </template>
  </div>
</template>

<style scoped>
.ins-row {
  display: flex;
  align-items: baseline;
  gap: 8px;
  padding: 1px 4px;
  cursor: default;
  font-family: ui-monospace, monospace;
  font-size: 12px;
  white-space: nowrap;
}
.ins-row:hover {
  background: rgba(var(--v-theme-primary), 0.08);
}
.ins-twist {
  width: 10px;
  color: rgb(var(--v-theme-primary));
}
.ins-key {
  color: rgb(var(--v-theme-primary));
}
.ins-type {
  color: rgba(var(--v-theme-on-surface), 0.6);
}
.ins-val {
  overflow: hidden;
  text-overflow: ellipsis;
}
.ins-load {
  opacity: 0.6;
}
</style>
