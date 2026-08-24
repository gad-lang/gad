// Tokenization tests for the Gad Template grammar (source.gadt): `{% … %}` code
// and `{%= … %}` output islands embed the Gad grammar, while everything outside
// an island stays literal template text.
import { test, expect, beforeAll } from "bun:test";
import fs from "node:fs";
import path from "node:path";
import { Registry, type IGrammar } from "vscode-textmate";
import * as oniguruma from "vscode-oniguruma";

const repoRoot = path.resolve(import.meta.dir, "../../../..");

function genGad(): unknown {
  const r = Bun.spawnSync(["go", "run", "./cmd/update-vscode-plugin", "-print"], {
    cwd: repoRoot,
    stdout: "pipe",
    stderr: "pipe",
  });
  if (r.exitCode !== 0) throw new Error(r.stderr.toString());
  return JSON.parse(r.stdout.toString());
}

let grammar: IGrammar;

beforeAll(async () => {
  const onigWasm = path.join(path.dirname(require.resolve("vscode-oniguruma")), "onig.wasm");
  await oniguruma.loadWASM(fs.readFileSync(onigWasm).buffer);
  const onigLib = Promise.resolve({
    createOnigScanner: (s: string[]) => new oniguruma.OnigScanner(s),
    createOnigString: (s: string) => new oniguruma.OnigString(s),
  });
  const gad = genGad();
  const gadt = JSON.parse(
    fs.readFileSync(path.join(repoRoot, "plugins/ide/vscode-gad/gad-textmate/syntaxes/gadt.tmLanguage.json"), "utf8"),
  );
  const registry = new Registry({
    onigLib,
    loadGrammar: async (s) => (s === "source.gad" ? (gad as any) : s === "source.gadt" ? (gadt as any) : null),
  });
  grammar = (await registry.loadGrammar("source.gadt"))!;
});

function toks(line: string) {
  return grammar.tokenizeLine(line, null).tokens.map((t) => ({ text: line.slice(t.startIndex, t.endIndex), scopes: t.scopes }));
}

test("a `{% … %}` code island embeds Gad; literal text stays plain", () => {
  const t = toks('Hello {% x := "hi" + 1 %} world');
  expect(t.find((x) => x.text.startsWith("{%"))?.scopes.some((s) => s.includes("keyword.control.template.begin.gadt"))).toBe(true);
  expect(t.find((x) => x.text.includes("hi"))?.scopes.some((s) => s.includes("string.quoted"))).toBe(true);
  expect(t.find((x) => x.text === "1")?.scopes.some((s) => s.includes("constant.numeric"))).toBe(true);
  // literal "world" is not embedded Gad code
  const world = t.find((x) => x.text.includes("world"));
  expect(world?.scopes.some((s) => s.includes("meta.embedded.block.gad"))).toBe(false);
});

test("a `{%= … %}` output island is recognized", () => {
  const t = toks("<b>{%= name %}</b>");
  expect(t.find((x) => x.text.startsWith("{%="))?.scopes.some((s) => s.includes("begin.gadt"))).toBe(true);
});
