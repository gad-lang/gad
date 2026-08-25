// TextMate tokenization tests for the Gad grammar, run against the real
// vscode-textmate + Oniguruma engine — the same stack VS Code and the IntelliJ
// TextMate bundle use — so what we assert here is what editors actually render.
//
// The grammar is generated from the Gad vocabulary by the sibling Go package
// (cmd/internal/pluginsync, via `go run ./cmd/update-vscode-plugin -print`); this
// test consumes that generator output directly, so it guards the highlighting
// contract at its source of truth rather than a committed copy that could drift.
//
// Run with `bun test` here, or `make grammar-test` from the repo root.
import { test, expect, beforeAll } from "bun:test";
import fs from "node:fs";
import path from "node:path";
import { Registry, type IGrammar } from "vscode-textmate";
import * as oniguruma from "vscode-oniguruma";

// repo root is four levels up: tmtest → pluginsync → internal → cmd → root.
const repoRoot = path.resolve(import.meta.dir, "../../../..");

/** The exact grammar the generator emits (not the committed extension copy). */
function generateGrammar(): unknown {
  const r = Bun.spawnSync(
    ["go", "run", "./cmd/update-vscode-plugin", "-print"],
    { cwd: repoRoot, stdout: "pipe", stderr: "pipe" },
  );
  if (r.exitCode !== 0) {
    throw new Error(`grammar generator failed:\n${r.stderr.toString()}`);
  }
  return JSON.parse(r.stdout.toString());
}

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
  const raw = generateGrammar();
  const registry = new Registry({
    onigLib,
    loadGrammar: async (scope) => (scope === "source.gad" ? (raw as any) : null),
  });
  const g = await registry.loadGrammar("source.gad");
  if (!g) throw new Error("failed to load source.gad grammar");
  grammar = g;
});

/** Tokenize one line into `{ text, scopes }` entries. */
function tokenize(line: string): { text: string; scopes: string[] }[] {
  const r = grammar.tokenizeLine(line, null);
  return r.tokens.map((t) => ({
    text: line.slice(t.startIndex, t.endIndex),
    scopes: t.scopes,
  }));
}

/** Scopes of the first token whose trimmed text equals `needle`. */
function scopesOf(line: string, needle: string): string[] {
  const tok = tokenize(line).find((t) => t.text.trim() === needle);
  if (!tok) throw new Error(`no token ${JSON.stringify(needle)} in ${JSON.stringify(line)}`);
  return tok.scopes;
}

test("cooked double interpolation highlights the embedded expression", () => {
  const line = 'x := #"a { x + 1 } b"';
  expect(scopesOf(line, "a")).toContain("string.quoted.double.interpolated.gad");
  expect(scopesOf(line, "{")).toContain("punctuation.section.embedded.begin.gad");
  expect(scopesOf(line, "}")).toContain("punctuation.section.embedded.end.gad");
  // The body is real Gad: it is marked meta.embedded (so hosts re-highlight it as
  // code, not string) and operators/numbers get their code scopes.
  expect(scopesOf(line, "+")).toContain("meta.embedded.gad");
  expect(scopesOf(line, "+")).toContain("keyword.operator.gad");
  expect(scopesOf(line, "1")).toContain("constant.numeric.gad");
  expect(scopesOf(line, "1")).toContain("meta.interpolation.gad");
});

test("triple and raw interpolated forms embed expressions too", () => {
  const triple = 'z := #"""t { x } e"""';
  expect(scopesOf(triple, "t")).toContain("string.quoted.triple.interpolated.gad");
  expect(scopesOf(triple, "{")).toContain("meta.interpolation.gad");

  const raw = "y := #`r { x } e`";
  expect(scopesOf(raw, "r")).toContain("string.quoted.raw.interpolated.gad");
  expect(scopesOf(raw, "{")).toContain("punctuation.section.embedded.begin.gad");
});

test("cooked forms keep \\{ and \\} as escapes, not island starts", () => {
  const line = 'w := #"esc \\{ keep \\} then { x }"';
  const esc = tokenize(line).find((t) => t.text === "\\{");
  expect(esc?.scopes).toContain("constant.character.escape.gad");
  expect(esc?.scopes).not.toContain("meta.interpolation.gad");
  // The genuine, unescaped island still opens.
  expect(scopesOf(line, "{")).toContain("meta.interpolation.gad");
});

test("plain (non-#) strings do not interpolate braces", () => {
  const line = 'p := "plain { not code } here"';
  for (const t of tokenize(line)) {
    expect(t.scopes).not.toContain("meta.interpolation.gad");
    expect(t.scopes).not.toContain("string.quoted.double.interpolated.gad");
  }
});

test("nested braces balance and do not leak past the island", () => {
  const line = 'm := #"n { {a: 1} } tail"';
  const tail = tokenize(line).find((t) => t.text.includes("tail"));
  expect(tail?.scopes).toContain("string.quoted.double.interpolated.gad");
  expect(tail?.scopes).not.toContain("meta.interpolation.gad");
});

test("a `///` doc comment does not bleed into the next line (interpolation)", () => {
  // Regression: `///` embedding the Markdown grammar continued a paragraph
  // across lines, so the interpolated string on the next line lost its scopes
  // until a blank line. Tokenize line-by-line threading the rule stack.
  const doc = "/// := declares; = reassigns an existing binding.";
  const code = 'greeting := #"{name} is a fast language."';

  let r = grammar.tokenizeLine(doc, null);
  // the comment line is a doc comment
  expect(r.tokens.some((t) => t.scopes.some((s) => s.includes("comment.documentation.line.gad")))).toBe(true);

  // the NEXT line, with the doc's end-of-line rule stack, is code — the string
  // interpolates and is not swallowed by the comment/markdown.
  const next = grammar.tokenizeLine(code, r.ruleStack);
  const toks = next.tokens.map((t) => ({ text: code.slice(t.startIndex, t.endIndex), scopes: t.scopes }));
  const str = toks.find((t) => t.text.includes("is a fast"));
  expect(str, `no string token in ${JSON.stringify(toks)}`).toBeDefined();
  expect(str!.scopes.some((s) => s.includes("string.quoted"))).toBe(true);
  // and it must NOT be inside a comment/markdown embed leaked from the line above
  expect(next.tokens.some((t) => t.scopes.some((s) => s.includes("comment") || s.includes("markdown")))).toBe(false);
});

test("a single-line `/** x **/` block doc closes on its own line", () => {
  const doc = "/** the name **/";
  const code = 'greeting := #"{name} v1"';
  let r = grammar.tokenizeLine(doc, null);
  // the line IS a doc comment (not the plain `/* */` fallback) …
  expect(r.tokens.some((t) => t.scopes.some((s) => s.includes("comment.documentation.block.gad")))).toBe(true);
  // … and it closes here — the block-doc end must match `**/` at end of line, not
  // only a fence alone on its own line, or a single-line doc would never close.
  const next = grammar.tokenizeLine(code, r.ruleStack);
  expect(next.tokens.some((t) => t.scopes.some((s) => s.includes("comment") || s.includes("markdown")))).toBe(false);
});

test("an embedded `***/` in block-doc prose does not close the doc early", () => {
  // The doc text mentions the `/*** … ***/` fence form; the `***/` is mid-line, so
  // the end (`**/` at end of line) must not fire until the real fence line.
  const lines = [
    "/**",
    "wrapping a `/*** … ***/` root comment, like this one.",
    "**/",
    "export PI = 3.14",
  ];
  let stack: any = null;
  const scoped = lines.map((ln) => {
    const r = grammar.tokenizeLine(ln, stack);
    stack = r.ruleStack;
    return r.tokens.map((t) => ({ text: ln.slice(t.startIndex, t.endIndex), scopes: t.scopes }));
  });
  // the prose line stays entirely inside the doc-comment body
  expect(scoped[1].every((t) => t.scopes.some((s) => s.includes("comment.documentation.block.gad")))).toBe(true);
  // and the code after the closing fence is real code, not leaked comment
  const code = scoped[3];
  expect(code.some((t) => t.scopes.some((s) => s.includes("keyword.control.gad")))).toBe(true);
  expect(code.some((t) => t.scopes.some((s) => s.includes("comment")))).toBe(false);
});

test("a `[link](x)` then another line inside a block doc keeps the doc scope", () => {
  // Regression for the IntelliJ breakage: with the Markdown grammar embedded, a
  // body line after a `[link](url)` (and the closing `**/`) lost the doc-comment
  // scope in IntelliJ's engine. With no embed the whole body is uniformly scoped.
  const lines = ["/**", "See [Templates](09_template.gad).", "abc", "**/", "export X = 1"];
  let stack: any = null;
  const scoped = lines.map((ln) => {
    const r = grammar.tokenizeLine(ln, stack);
    stack = r.ruleStack;
    return r.tokens.map((t) => ({ text: ln.slice(t.startIndex, t.endIndex), scopes: t.scopes }));
  });
  // every line from `/**` through the closing `**/` is doc-comment scoped
  for (let i = 0; i <= 3; i++) {
    expect(scoped[i].every((t) => t.scopes.some((s) => s.includes("comment.documentation.block.gad")))).toBe(true);
  }
  // and the line after `**/` is code
  expect(scoped[4].some((t) => t.scopes.some((s) => s.includes("keyword.control.gad")))).toBe(true);
  expect(scoped[4].some((t) => t.scopes.some((s) => s.includes("comment")))).toBe(false);
});
