// Demo: a CodeMirror 6 editor with Gad language support, switchable between a
// plain `.gad` script, a `.gadt` template, and a `.gad` file that turns on
// template mode part-way with a `# gad: mixed` directive. Serve with
// `bun run demo`, then open example/index.html.
import {
  EditorView,
  lineNumbers,
  highlightActiveLine,
  drawSelection,
} from "@codemirror/view";
import { EditorState, type Extension } from "@codemirror/state";
import {
  syntaxHighlighting,
  defaultHighlightStyle,
  bracketMatching,
  indentOnInput,
} from "@codemirror/language";
import { closeBrackets, autocompletion } from "@codemirror/autocomplete";
import { gad, type GadOptions } from "../src/index";

// --- the three example sources ---------------------------------------------
const GAD = `# A plain .gad script.
const Pi = 3.14159            // constant (PascalCase)
name := "gad"                 // short var declaration

func area {
    (r float)          => Pi * r * r
    (w float, h float) => w * h
}
met area(side int) => side * side

Stringer := meti { () <str> }
met ~area($old, side int) => $old(side) + 1   // $old wraps the previous method
func apply(cb met<(int) <int>>, v int) => cb(v)
x := 5 :: int :: any                          // the :: assign-to-type operator

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
{% if len(items) == 0 %}<p>no items</p>{% end %}
`;

const MIXED = `# gad: mixed
# A .gad file: plain Gad here, then template output after the directive.
title := "Report"
items := ["cpu", "mem", "disk"]
<h1>{%= title %}</h1>
<ul>
{% for it in items %}  <li>{%= it %}</li>
{% end %}
</ul>
`;

const examples: Record<string, { doc: string; opts: GadOptions }> = {
  ".gad": { doc: GAD, opts: {} },
  ".gadt": { doc: GADT, opts: { template: true } },
  ".gad (mixed)": { doc: MIXED, opts: { template: true, preamble: true } },
};

// --- editor ----------------------------------------------------------------
const base: Extension = [
  lineNumbers(),
  highlightActiveLine(),
  drawSelection(),
  indentOnInput(),
  bracketMatching(),
  closeBrackets(),
  autocompletion(),
  syntaxHighlighting(defaultHighlightStyle),
];

const parent = document.getElementById("editor")!;
let view: EditorView | undefined;

function show(name: string): void {
  const ex = examples[name];
  view?.destroy();
  view = new EditorView({
    state: EditorState.create({ doc: ex.doc, extensions: [base, gad(ex.opts)] }),
    parent,
  });
}

// tab buttons
const tabs = document.getElementById("tabs")!;
for (const name of Object.keys(examples)) {
  const btn = document.createElement("button");
  btn.textContent = name;
  btn.onclick = () => {
    for (const b of tabs.children) b.classList.remove("active");
    btn.classList.add("active");
    show(name);
  };
  tabs.appendChild(btn);
}
(tabs.firstElementChild as HTMLButtonElement).classList.add("active");
show(".gad");
