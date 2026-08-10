<script setup lang="ts">
import { computed, ref, shallowRef, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useTheme, useDisplay } from "vuetify";
import { loadContent, appBase, type SiteContent, type SearchDoc } from "./content";

const route = useRoute();
const router = useRouter();
const theme = useTheme();
const { mdAndUp } = useDisplay();

const content = shallowRef<SiteContent | null>(null);
loadContent().then((c) => (content.value = c)).catch(() => {});

const site = computed(() => content.value?.site);
const groups = computed(() => content.value?.groups ?? []);
const logo = appBase() + "gad.svg";

const onDocs = computed(() => String(route.name || "").startsWith("docs"));
const drawer = ref(true);
watch(mdAndUp, (v) => (drawer.value = v), { immediate: true });

// Collapsible nav sections: keep only the group holding the current page open.
const openGroups = ref<string[]>([]);
const activeSlug = computed(() => (onDocs.value ? String(route.params.slug || "") : ""));
watch(
  [groups, activeSlug],
  () => {
    const g = groups.value.find((gr) => gr.pages.some((p) => p.slug === activeSlug.value));
    openGroups.value = g ? [g.name] : groups.value[0] ? [groups.value[0].name] : [];
  },
  { immediate: true },
);

// Theme toggle, persisted and mirrored on <html data-theme> (used pre-paint).
function toggleTheme() {
  const next = theme.global.current.value.dark ? "light" : "dark";
  theme.global.name.value = next;
  localStorage.setItem("gad-theme", next);
  document.documentElement.dataset.theme = next;
}
const isDark = computed(() => theme.global.current.value.dark);

// Search over the emitted index.
const query = ref("");
const searchOpen = ref(false);
const results = computed<SearchDoc[]>(() => {
  const q = query.value.trim().toLowerCase();
  if (!q || !content.value) return [];
  const out: SearchDoc[] = [];
  for (const d of content.value.search) {
    if (d.title.toLowerCase().includes(q) || d.text.toLowerCase().includes(q)) {
      out.push(d);
      if (out.length >= 12) break;
    }
  }
  return out;
});
function go(slug: string) {
  query.value = "";
  searchOpen.value = false;
  router.push({ name: "docs", params: { slug } });
}
</script>

<template>
  <v-app>
    <v-app-bar flat border density="comfortable" class="px-2">
      <v-app-bar-nav-icon v-if="onDocs" class="d-md-none" @click="drawer = !drawer" />
      <router-link to="/" class="brand text-none">
        <img :src="logo" alt="" width="30" height="30" />
        <span class="brand-name">Gad</span>
      </router-link>

      <v-menu v-model="searchOpen" :close-on-content-click="false" location="bottom" offset="6" max-width="440">
        <template #activator="{ props }">
          <v-text-field
            v-bind="props"
            v-model="query"
            density="compact"
            variant="solo-filled"
            flat
            hide-details
            rounded
            prepend-inner-icon="mdi-magnify"
            placeholder="Search docs…"
            class="search mx-2"
            @focus="searchOpen = true"
          />
        </template>
        <v-list v-if="results.length" density="compact" class="py-0">
          <v-list-item v-for="r in results" :key="r.slug" :title="r.title" @click="go(r.slug)" />
        </v-list>
        <v-list v-else-if="query" density="compact"><v-list-item title="No matches" disabled /></v-list>
      </v-menu>

      <v-spacer />

      <div class="header-links d-none d-sm-flex">
        <v-btn v-if="site?.hasRelease" :href="appBase() + 'download'" variant="tonal" color="secondary" size="small" class="rel-chip">
          {{ site?.releaseName }}
        </v-btn>
        <v-btn v-if="site?.playHref" :href="appBase() + site.playHref" variant="text" size="small">Playground</v-btn>
        <v-btn :to="'/docs'" variant="text" size="small">Docs</v-btn>
        <v-btn v-if="site" :href="site.repoURL + '/issues'" target="_blank" variant="text" size="small">Issues</v-btn>
        <v-btn v-if="site" :href="site.repoURL" target="_blank" variant="text" size="small">Repo</v-btn>
      </div>
      <v-btn :icon="isDark ? 'mdi-weather-night' : 'mdi-weather-sunny'" variant="text" @click="toggleTheme" />
    </v-app-bar>

    <v-navigation-drawer
      v-if="onDocs"
      v-model="drawer"
      :permanent="mdAndUp"
      width="270"
    >
      <v-list v-model:opened="openGroups" density="compact" nav>
        <v-list-group v-for="g in groups" :key="g.name" :value="g.name">
          <template #activator="{ props }">
            <v-list-item
              v-bind="props"
              :title="g.name"
              class="group-title text-uppercase text-caption font-weight-bold"
            />
          </template>
          <v-list-item
            v-for="p in g.pages"
            :key="p.slug"
            :title="p.title"
            :to="p.href ? undefined : { name: 'docs', params: { slug: p.slug } }"
            :href="p.href ? appBase() + p.href : undefined"
            color="primary"
            density="compact"
          />
        </v-list-group>
      </v-list>
    </v-navigation-drawer>

    <v-main>
      <router-view />
    </v-main>
  </v-app>
</template>

<style scoped>
.brand {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  font-weight: 800;
  font-size: 1.25rem;
  letter-spacing: -0.01em;
  color: inherit;
  text-decoration: none;
}
.search {
  max-width: 320px;
  min-width: 140px;
}
.rel-chip {
  font-weight: 700;
}

/* Nav sections: tint the open group and indent its items to show the hierarchy. */
:deep(.v-navigation-drawer .v-list-group__items) {
  background: rgba(var(--v-theme-primary), 0.06);
  border-radius: 10px;
  margin: 2px 0 8px;
  padding: 2px 0;
}
:deep(.v-navigation-drawer .v-list-group__items .v-list-item) {
  padding-inline-start: 22px !important;
}
:deep(.v-navigation-drawer .group-title) {
  opacity: 0.85;
}
</style>
