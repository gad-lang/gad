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
  expect(scopesOf(line, "{")).toContain("punctuation.section.embedded.gad");
  expect(scopesOf(line, "}")).toContain("punctuation.section.embedded.gad");
  // The body is real Gad: operators and numbers colorize inside the island.
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
  expect(scopesOf(raw, "{")).toContain("punctuation.section.embedded.gad");
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
