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
  // A minimal stub for the embedded Markdown grammar, so the doc-comment
  // include resolves in-test (real editors ship text.html.markdown).
  const markdownStub = { scopeName: "text.html.markdown", patterns: [] };
  const registry = new Registry({
    onigLib,
    loadGrammar: async (scope) =>
      scope === "source.gad" ? (gad as any)
      : scope === "source.gadx" ? (gadx as any)
      : scope === "text.html.markdown" ? (markdownStub as any)
      : null,
  });
  const g = await registry.loadGrammar("source.gadx");
  if (!g) throw new Error("failed to load source.gadx");
  grammar = g;
});

function tokenize(line: string): { text: string; scopes: string[] }[] {
  const r = grammar.tokenizeLine(line, null);
  return r.tokens.map((t) => ({ text: line.slice(t.startIndex, t.endIndex), scopes: t.scopes }));
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

test("doc comment body is embedded markdown", () => {
  // opening line of a /** … **/ block
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
  // the prose line stays entirely inside the doc-comment markdown body
  expect(scoped[1].every((t) => t.scopes.some((s) => s.includes("comment.documentation.block.markdown.gadx")))).toBe(true);
  // the tag after the closing fence is real markup, not leaked comment/markdown
  const tag = scoped[3];
  expect(tag.some((t) => t.scopes.some((s) => s.includes("entity.name.tag.gadx")))).toBe(true);
  expect(tag.some((t) => t.scopes.some((s) => s.includes("comment") || s.includes("markdown")))).toBe(false);
});
