// Tokenization tests for the hand-written Gadx grammar, run against the real
// vscode-textmate + Oniguruma engine. Loads the committed gadx grammar plus the
// generated gad grammar (embedded via `source.gad`), so `!`, `@test`, `~ …` and
// `+ …` coloring is asserted the way editors render it.
import { test, expect, beforeAll } from "bun:test";
import fs from "node:fs";
import path from "node:path";
import { Registry, type IGrammar } from "vscode-textmate";
import * as oniguruma from "vscode-oniguruma";

const repoRoot = path.resolve(import.meta.dir, "../../../..");

function generateGadGrammar(): unknown {
  const r = Bun.spawnSync(
    ["go", "run", "./cmd/update-vscode-plugin", "-print"],
    { cwd: repoRoot, stdout: "pipe", stderr: "pipe" },
  );
  if (r.exitCode !== 0) throw new Error(`gad grammar gen failed:\n${r.stderr.toString()}`);
  return JSON.parse(r.stdout.toString());
}

const gadxPath = path.join(
  repoRoot,
  "plugins/ide/vscode-gad/gad-textmate/syntaxes/gadx.tmLanguage.json",
);

let grammar: IGrammar;

beforeAll(async () => {
  const onigWasm = path.join(
    path.dirname(require.resolve("vscode-oniguruma")),
    "onig.wasm",
  );
  await oniguruma.loadWASM(fs.readFileSync(onigWasm).buffer);
  const onigLib = Promise.resolve({
    createOnigScanner: (s: string[]) => new oniguruma.OnigScanner(s),
    createOnigString: (s: string) => new oniguruma.OnigString(s),
  });
  const gad = generateGadGrammar();
  const gadx = JSON.parse(fs.readFileSync(gadxPath, "utf8"));
  const registry = new Registry({
    onigLib,
    loadGrammar: async (scope) =>
      scope === "source.gad" ? (gad as any)
      : scope === "source.gadx" ? (gadx as any)
      : scope === "source.js" ? (stub("source.js", "js") as any)
      : scope === "source.css" ? (stub("source.css", "css") as any)
      : null,
  });
  const g = await registry.loadGrammar("source.gadx");
  if (!g) throw new Error("failed to load source.gadx");
  grammar = g;
});

/**
 * A stand-in grammar for an embedded language. The real JavaScript and CSS
 * grammars ship with the editor, not with this bundle, so what is asserted here
 * is that the embedding *resolves* — every token the raw-text body hands over
 * comes back carrying this grammar's mark — rather than how JS or CSS is
 * coloured, which is the editor's business.
 */
function stub(scopeName: string, tag: string) {
  return {
    scopeName,
    name: tag,
    patterns: [{ name: `stub.${tag}`, match: "\\S+" }],
  };
}

function tokenize(line: string): { text: string; scopes: string[] }[] {
  const r = grammar.tokenizeLine(line, null);
  return r.tokens.map((t) => ({ text: line.slice(t.startIndex, t.endIndex), scopes: t.scopes }));
}

/** Tokenize a whole document, threading the rule stack line by line. */
function tokenizeLines(lines: string[]): { text: string; scopes: string[] }[][] {
  let stack: any = null;
  return lines.map((ln) => {
    const r = grammar.tokenizeLine(ln, stack);
    stack = r.ruleStack;
    return r.tokens.map((t) => ({ text: ln.slice(t.startIndex, t.endIndex), scopes: t.scopes }));
  });
}

/** Every token of a line whose text is not blank. */
function solid(toks: { text: string; scopes: string[] }[]) {
  return toks.filter((t) => t.text.trim() !== "");
}

/** The scopes of the token that exactly equals `text`. */
function scopesOf(line: string, text: string): string[] {
  const tok = tokenize(line).find((t) => t.text === text);
  if (!tok) throw new Error(`token ${JSON.stringify(text)} not found in ${JSON.stringify(line)}`);
  return tok.scopes;
}

test("@test is a control keyword and its name is an entity", () => {
  expect(scopesOf("@test greeting", "@test")).toContain("keyword.control.gadx");
  expect(scopesOf("@test greeting", "greeting")).toContain("entity.name.function.gadx");
});

test("`!` fluent call: marker highlighted, body is embedded gad", () => {
  const s = scopesOf("    ! t.equal a b", "!");
  expect(s).toContain("keyword.operator.call.gadx");
  // the callee resolves through the embedded gad grammar
  expect(tokenize("    ! t.equal a b").some((t) => t.scopes.some((x) => x.includes("source.gad")))).toBe(true);
});

test("`!!!` doctype is NOT a call", () => {
  // the doctype rule scopes the whole line; the `!` must not become a call marker
  const toks = tokenize("!!! 5");
  expect(toks.some((t) => t.scopes.some((x) => x.includes("keyword.other.doctype.gadx")))).toBe(true);
  expect(toks.some((t) => t.scopes.some((x) => x.includes("keyword.operator.call.gadx")))).toBe(false);
});

test("`~ EXPR` single code line is embedded gad", () => {
  const toks = tokenize("    ~ t.run(x)");
  expect(scopesOf("    ~ t.run(x)", "~")).toContain("punctuation.definition.code.gadx");
  expect(toks.some((t) => t.scopes.some((x) => x.includes("source.gad")))).toBe(true);
});

test("`~~` block still opens an embedded gad block (not the code line)", () => {
  expect(scopesOf("~~", "~~")).toContain("punctuation.definition.codeblock.gadx");
});

test("`+ EXPR` component call: marker + name, args embedded gad", () => {
  expect(scopesOf("    +box(; a=1)", "+")).toContain("keyword.operator.component.gadx");
  expect(scopesOf("    +box(; a=1)", "box")).toContain("entity.name.function.gadx");
});

test("a single-line `/** … **/` block doc is doc-comment scoped", () => {
  const toks = tokenize("/** # Title **/");
  expect(toks.some((t) => t.scopes.some((x) => x.includes("comment.documentation.block.gadx")))).toBe(true);
});

test("an embedded `***/` in block-doc prose does not close the doc early", () => {
  // The `***/` sits mid-line inside the prose; the block-doc end (`**/` at end of
  // line) must not fire there, or the rest of the doc leaks out as tag markup.
  const lines = [
    "/**",
    "wrapping a `/*** … ***/` root comment, like this one.",
    "**/",
    "div hello",
  ];
  let stack: any = null;
  const scoped = lines.map((ln) => {
    const r = grammar.tokenizeLine(ln, stack);
    stack = r.ruleStack;
    return r.tokens.map((t) => ({ text: ln.slice(t.startIndex, t.endIndex), scopes: t.scopes }));
  });
  // the prose line stays entirely inside the doc-comment body
  expect(scoped[1].every((t) => t.scopes.some((s) => s.includes("comment.documentation.block.gadx")))).toBe(true);
  // the tag after the closing fence is real markup, not leaked comment
  const tag = scoped[3];
  expect(tag.some((t) => t.scopes.some((s) => s.includes("entity.name.tag.gadx")))).toBe(true);
  expect(tag.some((t) => t.scopes.some((s) => s.includes("comment")))).toBe(false);
});

test("doc-comment body styles inline Markdown (#docmarkup)", () => {
  const lines = ["/**", "Some **bold** and `code` here.", "**/", "div x"];
  let stack: any = null;
  const scoped = lines.map((ln) => {
    const r = grammar.tokenizeLine(ln, stack);
    stack = r.ruleStack;
    return r.tokens.map((t) => ({ text: ln.slice(t.startIndex, t.endIndex), scopes: t.scopes }));
  });
  const bold = scoped[1].find((t) => t.text === "**bold**")!;
  const code = scoped[1].find((t) => t.text === "`code`")!;
  expect(bold.scopes.some((s) => s.includes("markup.bold.gadx"))).toBe(true);
  expect(bold.scopes.some((s) => s.includes("comment.documentation.block.gadx"))).toBe(true);
  expect(code.scopes.some((s) => s.includes("markup.inline.raw"))).toBe(true);
  // the fence still closes; the tag below is not comment
  expect(scoped[2][0].scopes[scoped[2][0].scopes.length - 1]).toContain("comment.documentation.block.gadx");
  expect(scoped[3].some((t) => t.scopes.some((s) => s.includes("entity.name.tag.gadx")))).toBe(true);
});

test("`script` in tag syntax: the body is JavaScript, the tag line is still a tag", () => {
  const doc = [
    "@main",
    "\tscript[src=\"a.js\"]",
    "\tscript",
    "\t\tconst a = { b: 1 }",
    "",
    "\t\tif (a) { run() }",
    "\tdiv after",
  ];
  const t = tokenizeLines(doc);

  // the tag's own line stays a tag, attributes included
  expect(t[1].some((x) => x.scopes.some((s) => s.includes("entity.name.tag.gadx")))).toBe(true);
  expect(t[1].some((x) => x.scopes.some((s) => s.includes("meta.attributes.gadx")))).toBe(true);

  // the indented body goes to the JavaScript grammar
  for (const line of [3, 5]) {
    expect(solid(t[line]).every((x) => x.scopes.includes("stub.js"))).toBe(true);
    expect(t[line].some((x) => x.scopes.some((s) => s.includes("meta.embedded.block.javascript")))).toBe(true);
  }
  // a blank line does not close the block
  expect(t[4].every((x) => x.scopes.some((s) => s.includes("meta.embedded.block.javascript")))).toBe(true);
  // and a line back at the tag's own indentation does
  expect(t[6].some((x) => x.scopes.some((s) => s.includes("entity.name.tag.gadx")))).toBe(true);
  expect(t[6].some((x) => x.scopes.some((s) => s.includes("javascript")))).toBe(false);
});

test("`style` in tag syntax: the body is CSS", () => {
  const t = tokenizeLines(["@main", "\tstyle", "\t\t.a { color: red }", "\tp x"]);
  expect(solid(t[2]).every((x) => x.scopes.includes("stub.css"))).toBe(true);
  expect(t[2].some((x) => x.scopes.some((s) => s.includes("meta.embedded.block.css")))).toBe(true);
  expect(t[3].some((x) => x.scopes.some((s) => s.includes("entity.name.tag.gadx")))).toBe(true);
});

test("`#{ … }#` in raw text is an interpolation, its body coloured as gad", () => {
  const t = tokenizeLines([
    "@main",
    "\tstyle",
    "\t\t.a { color: #{= accent }#; }",
    "\t\t#{ log(\"x\") }#",
  ]);

  const open = t[2].find((x) => x.text === "#{=")!;
  expect(open.scopes).toContain("keyword.control.interpolation.output.begin.gadx");
  const close = t[2].find((x) => x.text === "}#")!;
  expect(close.scopes).toContain("keyword.control.interpolation.output.end.gadx");

  // the code between them is gad, not CSS
  const code = t[2].find((x) => x.text.trim() === "accent")!;
  expect(code.scopes.some((s) => s.includes("meta.embedded.gad"))).toBe(true);
  expect(code.scopes).not.toContain("stub.css");

  // the control form is marked as such
  expect(t[3].find((x) => x.text === "#{")!.scopes)
    .toContain("keyword.control.interpolation.control.begin.gadx");
});

test("a lone `}` inside raw text does not close the interpolation", () => {
  // The closing `}#` is what makes it unambiguous: the `}` of the CSS rule
  // belongs to the CSS.
  const t = tokenizeLines(["@main", "\tstyle", "\t\t.a { color: #{= f({a: 1}) }#; }"]);
  const line = t[2];
  const closers = line.filter((x) => x.scopes.includes("keyword.control.interpolation.output.end.gadx"));
  expect(closers.length).toBe(1);
  // what follows the island is CSS again
  expect(line.filter((x) => x.text === ";").every((x) => x.scopes.includes("stub.css"))).toBe(true);
});

test("`@raw_text` reads its body verbatim, with `#{ … }#` as the interpolation", () => {
  const t = tokenizeLines([
    "@main",
    "\t@raw_text",
    "\t\t{ not an interpolation }",
    "\t\tvalue: #{= v }#",
    "\tp after",
  ]);

  expect(t[1].find((x) => x.text === "@raw_text")!.scopes).toContain("keyword.control.gadx");
  // a brace is literal text there
  expect(solid(t[2]).every((x) => x.scopes.some((s) => s.includes("string.unquoted.rawtext.gadx")))).toBe(true);
  expect(t[2].some((x) => x.scopes.some((s) => s.includes("keyword.control.interpolation")))).toBe(false);
  // `#{= … }#` is not
  expect(t[3].find((x) => x.text === "#{=")!.scopes)
    .toContain("keyword.control.interpolation.output.begin.gadx");
  expect(t[4].some((x) => x.scopes.some((s) => s.includes("entity.name.tag.gadx")))).toBe(true);
});

test("an inline `<script>` region is JavaScript across lines", () => {
  const t = tokenizeLines([
    "@main",
    "\t<div><script src=\"a.js\">",
    "\t\tif (x) { y() }",
    "\t</script></div>",
  ]);

  expect(t[1].some((x) => x.scopes.some((s) => s.includes("entity.name.tag.gadx")))).toBe(true);
  expect(solid(t[2]).every((x) => x.scopes.includes("stub.js"))).toBe(true);
  expect(t[3].some((x) => x.scopes.some((s) => s.includes("punctuation.definition.tag.gadx")))).toBe(true);
});

test("a tag merely named like a word starting with `script` is an ordinary tag", () => {
  const t = tokenizeLines(["@main", "\tscripts x", "\t\tp y"]);
  expect(t[2].some((x) => x.scopes.some((s) => s.includes("javascript")))).toBe(false);
});
