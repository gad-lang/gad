// Browser bundle entry for static syntax highlighting on the Gad docs website
// (cmd/build-website). It bundles PrismJS core + a few common languages and the
// Gad-family grammars (gad / gadt / gadx), then highlights every code block on
// load. Built with `bun build` into the site's prism.js. Not published — this is
// the docs site's own bundle source.
import Prism from "prismjs";
// Common languages used across the docs (Prism core already ships markup/css/
// clike/javascript). These augment the global Prism the core sets up.
import "prismjs/components/prism-go.js";
import "prismjs/components/prism-json.js";
import "prismjs/components/prism-bash.js"; // also provides the `sh`/`shell` aliases
import "prismjs/components/prism-yaml.js";
import "prismjs/components/prism-markdown.js";
import "prismjs/components/prism-ini.js";
// TypeScript / JSX / TSX for the @gad-lang JS module docs (jsx & typescript must
// load before tsx, which extends both).
import "prismjs/components/prism-typescript.js";
import "prismjs/components/prism-jsx.js";
import "prismjs/components/prism-tsx.js";
import { registerGad, registerGadx, registerGadTemplate } from "./src/index";

// The Gad family: gad, gadx and gadt (template). registerGad must run first —
// the template and gadx grammars embed the Gad grammar.
// The docs label fences gad / gadt / gadx, which these install as
// Prism.languages.gad / .gadt / .gadx.
registerGad(Prism);
registerGadx(Prism);
registerGadTemplate(Prism);

globalThis.Prism = Prism;

function highlight() {
  Prism.highlightAll();
}
if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", highlight);
} else {
  highlight();
}
