<script setup lang="ts">
import { computed, nextTick, ref, shallowRef, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useDisplay } from "vuetify";
import { loadContent, appBase, type SiteContent, type DocPage } from "../content";

const props = defineProps<{ slug?: string }>();
const route = useRoute();
const router = useRouter();
const { lgAndUp } = useDisplay();

const content = shallowRef<SiteContent | null>(null);
const body = ref<HTMLElement | null>(null);
const showSource = ref(false);
let prismLoaded: Promise<void> | null = null;

loadContent().then((c) => {
  content.value = c;
  ensureSlug();
}).catch(() => {});

function firstSlug(c: SiteContent): string | undefined {
  return c.groups[0]?.pages[0]?.slug;
}

// Redirect /docs (no slug) to the first documentation page.
function ensureSlug() {
  const c = content.value;
  if (!c) return;
  if (!props.slug) {
    const s = firstSlug(c);
    if (s) router.replace({ name: "docs", params: { slug: s } });
  }
}

const page = computed<DocPage | null>(() => {
  const c = content.value;
  if (!c || !props.slug) return null;
  return c.pages[props.slug] ?? null;
});

// Highlight code blocks with the PrismJS bundle emitted next to content.json.
function loadPrism(): Promise<void> {
  if (!prismLoaded) {
    prismLoaded = new Promise<void>((resolve) => {
      const w = window as unknown as { Prism?: unknown };
      if (w.Prism) return resolve();
      const s = document.createElement("script");
      s.src = appBase() + "prism.js";
      s.onload = () => resolve();
      s.onerror = () => resolve();
      document.head.appendChild(s);
    });
  }
  return prismLoaded;
}

watch([page, body, showSource], async () => {
  if (!page.value || !body.value) return;
  await nextTick();
  await loadPrism();
  const P = (window as unknown as { Prism?: { highlightAllUnder?: (e: Element) => void } }).Prism;
  if (P?.highlightAllUnder && body.value) P.highlightAllUnder(body.value);
});

watch(() => props.slug, () => {
  showSource.value = false;
  ensureSlug();
});

// Route internal doc links (`href="foo.html"`) through the SPA instead of a full
// page load; leave external links and in-page anchors to the browser.
function onClick(e: MouseEvent) {
  const a = (e.target as HTMLElement)?.closest?.("a") as HTMLAnchorElement | null;
  if (!a) return;
  const href = a.getAttribute("href") || "";
  if (!href || href.startsWith("#") || /^[a-z]+:/i.test(href) || a.target === "_blank") return;
  const slug = href.replace(/^\.?\//, "").replace(/#.*$/, "").replace(/\.html$/, "");
  if (content.value?.pages[slug]) {
    e.preventDefault();
    const hash = href.includes("#") ? href.slice(href.indexOf("#")) : "";
    router.push({ name: "docs", params: { slug }, hash });
  }
}
</script>

<template>
  <v-container fluid class="docs pa-0">
    <div class="docs-grid" :class="{ 'has-toc': lgAndUp && page && page.toc.length }">
      <article ref="body" class="content" @click="onClick">
        <div v-if="page && page.source" class="source-bar mb-4">
          <v-btn
            v-if="!showSource"
            size="small"
            variant="tonal"
            color="primary"
            prepend-icon="mdi-code-tags"
            @click="showSource = true"
          >View source</v-btn>
          <v-btn
            v-else
            size="small"
            variant="tonal"
            prepend-icon="mdi-arrow-left"
            @click="showSource = false"
          >Back to docs</v-btn>
        </div>
        <template v-if="page">
          <div v-if="showSource && page.source" class="source-view">
            <div class="text-caption text-medium-emphasis mb-2">
              Source · <code>{{ page.slug.replace(/^lang-/, "") }}.{{ page.sourceLang }}</code>
            </div>
            <pre><code :class="'language-' + (page.sourceLang || 'gad')">{{ page.source }}</code></pre>
          </div>
          <div v-else v-html="page.html" />
        </template>
        <div v-else class="pa-8 text-center text-medium-emphasis">Loading…</div>
      </article>
      <aside v-if="lgAndUp && page && page.toc.length" class="toc">
        <div class="text-caption text-uppercase font-weight-bold text-medium-emphasis mb-2">On this page</div>
        <a
          v-for="t in page.toc"
          :key="t.id"
          :href="'#' + t.id"
          class="toc-link d-block"
          :class="'toc-l' + t.level"
        >{{ t.text }}</a>
      </aside>
    </div>
  </v-container>
</template>

<style scoped>
.docs-grid {
  max-width: 1180px;
  margin: 0 auto;
  padding: 1.5rem 1.4rem 4rem;
}
.docs-grid.has-toc {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 200px;
  gap: 2rem;
}
.content {
  min-width: 0;
}
.toc {
  position: sticky;
  top: 84px;
  align-self: start;
  max-height: calc(100vh - 100px);
  overflow-y: auto;
  overscroll-behavior: contain;
  font-size: 0.85rem;
}
.toc-link {
  color: rgb(var(--v-theme-on-surface));
  opacity: 0.72;
  padding: 0.15rem 0;
  text-decoration: none;
  border-left: 2px solid transparent;
}
.toc-link:hover {
  opacity: 1;
  color: rgb(var(--v-theme-primary));
}
.toc-l3 {
  padding-left: 0.8rem;
  font-size: 0.82rem;
}
</style>
