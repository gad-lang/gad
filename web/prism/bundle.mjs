// The PrismJS bundle of the prism package (prism.go): Prism's core, the Gad
// family (gad, gadt, gadx — @gad-lang/prism-gad) and the languages most
// documentation shows. Unlike the docs site's bundle (site-bundle.mjs), it
// also highlights the code a page puts in after it loaded — an application
// that draws its pages without a reload (an admin) —, watching the page for
// it. Built by `make doc-prism` into prism.js.
import Prism from "../plugins/js/prism-gad/node_modules/prismjs/prism.js";
// (core already has markup — html, xml, svg —, css, clike and javascript)
import "../plugins/js/prism-gad/node_modules/prismjs/components/prism-go.js";
import "../plugins/js/prism-gad/node_modules/prismjs/components/prism-json.js";
import "../plugins/js/prism-gad/node_modules/prismjs/components/prism-bash.js";
import "../plugins/js/prism-gad/node_modules/prismjs/components/prism-yaml.js";
import "../plugins/js/prism-gad/node_modules/prismjs/components/prism-markdown.js";
import "../plugins/js/prism-gad/node_modules/prismjs/components/prism-ini.js";
import "../plugins/js/prism-gad/node_modules/prismjs/components/prism-toml.js";
import "../plugins/js/prism-gad/node_modules/prismjs/components/prism-sql.js";
import "../plugins/js/prism-gad/node_modules/prismjs/components/prism-diff.js";
import "../plugins/js/prism-gad/node_modules/prismjs/components/prism-python.js";
import "../plugins/js/prism-gad/node_modules/prismjs/components/prism-docker.js";
import "../plugins/js/prism-gad/node_modules/prismjs/components/prism-makefile.js";
import "../plugins/js/prism-gad/node_modules/prismjs/components/prism-typescript.js";
import "../plugins/js/prism-gad/node_modules/prismjs/components/prism-jsx.js";
import "../plugins/js/prism-gad/node_modules/prismjs/components/prism-tsx.js";
import { registerGad, registerGadx, registerGadTemplate } from "../plugins/js/prism-gad/src/index";

// registerGad first: gadx and gadt embed it
registerGad(Prism);
registerGadx(Prism);
registerGadTemplate(Prism);

// the names a fence is written with that Prism knows by another
const L = Prism.languages;
L.golang = L.go;
L.gadtemplate = L.gadt;

Prism.manual = true; // highlighted here, not by Prism's own load
globalThis.Prism = Prism;

const SELECTOR = 'code[class*="language-"], code[class*="lang-"]';

// highlightUnder highlights the code blocks under root not highlighted yet.
function highlightUnder(root) {
  if (!root.querySelectorAll) return;
  const blocks = root.matches && root.matches(SELECTOR) ? [root] : root.querySelectorAll(SELECTOR);
  for (const code of blocks) {
    if (code.dataset.prism === "done") continue;
    code.dataset.prism = "done";
    // a language Prism does not know is left as it is
    const lang = (code.className.match(/\blang(?:uage)?-([\w-]+)/) || [])[1];
    if (lang && Prism.languages[lang.toLowerCase()]) {
      if (lang !== lang.toLowerCase()) code.classList.add("language-" + lang.toLowerCase());
      Prism.highlightElement(code);
    }
  }
}

globalThis.GadPrism = { highlightUnder };

function start() {
  highlightUnder(document);
  // what comes in later — a page drawn without a reload — is highlighted
  // as it comes
  new MutationObserver((records) => {
    for (const r of records) {
      for (const n of r.addedNodes) {
        if (n.nodeType === 1 && !(n.closest && n.closest('code[data-prism="done"]'))) highlightUnder(n);
      }
    }
  }).observe(document.documentElement, { childList: true, subtree: true });
}
if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", start);
} else {
  start();
}
