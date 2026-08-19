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

// Editor plugins and shared grammar, each in its own repository with a live
// GitHub Pages site (the TextMate bundle has no demo site).
const ecosystem = [
  { icon: "mdi-microsoft-visual-studio-code", title: "VS Code", text: "Highlighting for .gad / .gadt / .gadx plus debugging over the Gad DAP.", repo: "https://github.com/gad-lang/vscode-gad", site: "https://gad-lang.github.io/vscode-gad/" },
  { icon: "mdi-application-brackets", title: "IntelliJ / GoLand", text: "JetBrains plugin: highlighting, run configurations and the full debugger.", repo: "https://github.com/gad-lang/intellij-gad", site: "https://gad-lang.github.io/intellij-gad/" },
  { icon: "mdi-code-tags", title: "CodeMirror 6", text: "@gad-lang/codemirror-gad — language support, completion, hover and linting.", repo: "https://github.com/gad-lang/codemirror-gad", site: "https://gad-lang.github.io/codemirror-gad/" },
  { icon: "mdi-palette-outline", title: "Prism", text: "@gad-lang/prism-gad — a PrismJS grammar for static syntax highlighting.", repo: "https://github.com/gad-lang/prism-gad", site: "https://gad-lang.github.io/prism-gad/" },
  { icon: "mdi-file-code-outline", title: "TextMate bundle", text: "gad-textmate — the shared grammars & schemas, generated from the Gad vocabulary.", repo: "https://github.com/gad-lang/gad-textmate", site: null },
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

    <v-container class="ecosystem">
      <h2 class="text-h5 font-weight-bold text-center mb-2">Editor support &amp; ecosystem</h2>
      <p class="text-center text-medium-emphasis mb-6">
        Syntax highlighting and editing for Gad across editors — each in its own repository, with a live demo site.
      </p>
      <v-row>
        <v-col v-for="e in ecosystem" :key="e.title" cols="12" sm="6" md="4">
          <v-card variant="outlined" height="100%" class="d-flex flex-column">
            <v-card-item>
              <template #prepend><v-icon :icon="e.icon" color="primary" size="28" /></template>
              <v-card-title class="text-subtitle-1 font-weight-bold">{{ e.title }}</v-card-title>
            </v-card-item>
            <v-card-text class="text-medium-emphasis flex-grow-1">{{ e.text }}</v-card-text>
            <v-card-actions>
              <v-btn v-if="e.site" :href="e.site" target="_blank" rel="noopener" variant="tonal" size="small" prepend-icon="mdi-open-in-new">Site</v-btn>
              <v-btn :href="e.repo" target="_blank" rel="noopener" variant="text" size="small" prepend-icon="mdi-github">Repo</v-btn>
            </v-card-actions>
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
  padding-bottom: 2rem;
}
.ecosystem {
  max-width: 1100px;
  padding-bottom: 4rem;
}
</style>
