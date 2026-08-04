<!-- Demo shell: the reusable <GadIde> driven by the in-browser localIdeApi. No
     Go server — files live in a LocalStorage overlay over the bundled samples,
     and the Gad VM runs in a Web Worker. -->
<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useTheme } from "vuetify";
import { GadIde, type Workspace } from "@gad-lang/ide-vuetify";
import { localIdeApi, resetWorkspace } from "./backends/localIde";

const workspace = ref<Workspace | null>(null);
const theme = useTheme();
const dark = ref(theme.global.current.value.dark);

function toggleTheme() {
  const next = dark.value ? "light" : "dark";
  theme.change(next);
  dark.value = !dark.value;
  try {
    localStorage.setItem("gad-vuetify-theme", next);
  } catch {
    /* ignore */
  }
}

onMounted(async () => {
  workspace.value = await localIdeApi.workspace();
});
</script>

<template>
  <v-app>
    <v-app-bar density="compact" flat>
      <v-app-bar-title>Gad IDE</v-app-bar-title>
      <template #append>
        <v-btn :icon="dark ? 'mdi-weather-sunny' : 'mdi-weather-night'" @click="toggleTheme" />
      </template>
    </v-app-bar>
    <v-main class="gad-main">
      <GadIde
        v-if="workspace"
        :api="localIdeApi"
        :workspace="workspace"
        :dark="dark"
        :on-reset="resetWorkspace"
      />
    </v-main>
  </v-app>
</template>

<style>
html,
body,
#app {
  height: 100%;
  margin: 0;
}
.gad-main {
  height: calc(100vh - 48px);
}
.gad-main > .v-main__wrap {
  height: 100%;
}
</style>
