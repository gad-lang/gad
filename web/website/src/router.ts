import { createRouter, createWebHistory } from "vue-router";
import { appBase } from "./content";
import Home from "./views/Home.vue";
import Docs from "./views/Docs.vue";

export const router = createRouter({
  history: createWebHistory(appBase()),
  routes: [
    { path: "/", name: "home", component: Home },
    { path: "/docs", name: "docs-index", component: Docs },
    { path: "/docs/:slug(.*)", name: "docs", component: Docs, props: true },
    { path: "/:pathMatch(.*)*", redirect: "/" },
  ],
  scrollBehavior(to) {
    if (to.hash) return { el: to.hash, top: 80 };
    return { top: 0 };
  },
});
