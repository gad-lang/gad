import { createApp } from "vue";
import { vuetify } from "./vuetify";
import { router } from "./router";
import App from "./App.vue";
import "./styles.css";

const app = createApp(App).use(vuetify).use(router);

// A GitHub-Pages 404.html deep link stashes its route in __SPA_REDIRECT__ (see
// index.html). Navigate to it once the router is ready — after the module and its
// assets have loaded from the site root — then mount.
const redirect = (window as unknown as { __SPA_REDIRECT__?: string }).__SPA_REDIRECT__;
router
  .isReady()
  .then(() => (redirect ? router.replace(redirect) : undefined))
  .catch(() => {})
  .finally(() => app.mount("#app"));
