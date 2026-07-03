// Demo: static highlighting with PrismJS and @gad-lang/prism-gad, switchable
// between a plain `.gad` script, a `.gadt` template, and a `.gad` file that
// turns on template mode with a `# gad: mixed` directive (routed via
// detectGadTemplate). Bundle/serve with `bun run demo`.
import Prism from "prismjs";
import { registerGad, registerGadTemplate, detectGadTemplate } from "../src/index";

registerGad(Prism); // Prism.languages.gad
registerGadTemplate(Prism); // Prism.languages.gadt

// --- the three example sources ---------------------------------------------
const GAD = `# A plain .gad script.
const Pi = 3.14159
name := "gad"

func area {
    (r float)          => Pi * r * r
    (w float, h float) => w * h
}
met area(side int) => side * side

Stringer := meti { () <str> }
met ~area($old, side int) => $old(side) + 1
func apply(cb met<(int) <int>>, v int) => cb(v)
x := 5 :: int :: any

for i in 0..10 {
    if i % 2 == 0 { println("even", i) }
}
`;

const GADT = `<!-- A .gadt template: literal text with {% %} / {%= %} tags. -->
<ul>
{% for i, name in ["ann", "bob", "cy"] %}
  <li>{%= i + 1 %}. {%= name.upper() %}</li>
{% end %}
</ul>
`;

const MIXED = `# gad: mixed
title := "Report"
items := ["cpu", "mem", "disk"]
<h1>{%= title %}</h1>
<ul>
{% for it in items %}  <li>{%= it %}</li>
{% end %}
</ul>
`;

// A `.gadt` file is always a template; a `.gad` file uses detectGadTemplate to
// choose (a `# gad: mixed` directive routes it to the template grammar).
const examples: { name: string; source: string; lang: string }[] = [
  { name: ".gad", source: GAD, lang: langFor(GAD, false) },
  { name: ".gadt", source: GADT, lang: "gadt" },
  { name: ".gad (mixed)", source: MIXED, lang: langFor(MIXED, false) },
];

function langFor(source: string, isGadtFile: boolean): string {
  if (isGadtFile) return "gadt";
  return detectGadTemplate(source).mixed ? "gadt" : "gad";
}

// --- render tabs + a highlighted <pre> -------------------------------------
const tabs = document.getElementById("tabs")!;
const out = document.getElementById("out")!;

function render(ex: { source: string; lang: string }): void {
  const grammar = Prism.languages[ex.lang];
  const html = Prism.highlight(ex.source, grammar, ex.lang);
  out.innerHTML = `<pre class="language-${ex.lang}"><code>${html}</code></pre>`;
}

for (const ex of examples) {
  const btn = document.createElement("button");
  btn.textContent = ex.name;
  btn.onclick = () => {
    for (const b of tabs.children) b.classList.remove("active");
    btn.classList.add("active");
    render(ex);
  };
  tabs.appendChild(btn);
}
(tabs.firstElementChild as HTMLButtonElement).classList.add("active");
render(examples[0]);
