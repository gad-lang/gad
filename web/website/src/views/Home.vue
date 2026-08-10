<script setup lang="ts">
import { computed, shallowRef } from "vue";
import { loadContent, appBase, type SiteContent } from "../content";

const content = shallowRef<SiteContent | null>(null);
loadContent().then((c) => (content.value = c)).catch(() => {});
const site = computed(() => content.value?.site);
const logo = appBase() + "gad.svg";

const features = [
  { icon: "mdi-rocket-launch", title: "Fast VM", text: "Compiled to bytecode and run on a stack-based VM written in native Go." },
  { icon: "mdi-language-go", title: "Embeddable", text: "Drop Gad into any Go program; expose your own types, builtins and modules." },
  { icon: "mdi-file-code", title: "Templates & Gadx", text: "Mixed .gadt templates and the indentation-based Gadx HTML engine, lowered to Gad." },
  { icon: "mdi-shield-check", title: "Typed & safe", text: "Typed method overloads, interfaces and strict compiler types — no loose any." },
  { icon: "mdi-language-markdown", title: "Docs built in", text: "Doc comments generate Markdown/HTML docs; @md blocks render Markdown via goldmark." },
  { icon: "mdi-web", title: "Runs in the browser", text: "A WebAssembly build powers the in-browser Playground and Notebook." },
];
</script>

<template>
  <div class="home">
    <section class="hero text-center">
      <img :src="logo" alt="Gad logo" width="112" height="112" class="mb-4" />
      <h1 class="text-h3 text-md-h2 font-weight-bold mb-3">Gad</h1>
      <p class="text-h6 font-weight-regular text-medium-emphasis mx-auto hero-tagline">
        {{ site?.tagline || "A fast, dynamic scripting language embedded in Go." }}
      </p>
      <div class="d-flex flex-wrap justify-center ga-3 mt-6">
        <v-btn to="/docs" color="primary" size="large" prepend-icon="mdi-book-open-variant">Get started</v-btn>
        <v-btn v-if="site?.playHref" :href="appBase() + site.playHref" size="large" variant="tonal" prepend-icon="mdi-play">
          Playground
        </v-btn>
        <v-btn v-if="site" :href="site.repoURL" target="_blank" size="large" variant="text" prepend-icon="mdi-github">
          GitHub
        </v-btn>
      </div>
    </section>

    <v-container class="features">
      <v-row>
        <v-col v-for="f in features" :key="f.title" cols="12" sm="6" md="4">
          <v-card variant="tonal" height="100%" class="pa-2">
            <v-card-item>
              <template #prepend><v-icon :icon="f.icon" color="primary" size="28" /></template>
              <v-card-title class="text-subtitle-1 font-weight-bold">{{ f.title }}</v-card-title>
            </v-card-item>
            <v-card-text class="text-medium-emphasis">{{ f.text }}</v-card-text>
          </v-card>
        </v-col>
      </v-row>
    </v-container>
  </div>
</template>

<style scoped>
.hero {
  padding: clamp(3rem, 8vw, 6rem) 1rem 2rem;
}
.hero-tagline {
  max-width: 46rem;
}
.features {
  max-width: 1100px;
  padding-bottom: 4rem;
}
</style>
