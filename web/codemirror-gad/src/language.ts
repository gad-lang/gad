import { StreamLanguage, StringStream, LanguageSupport } from "@codemirror/language";
import { tags as t } from "@lezer/highlight";
import { atoms, builtins, constants, keywords } from "./keywords";

const keywordSet = new Set(keywords);
const atomSet = new Set(atoms);
const constantSet = new Set(constants);
const builtinSet = new Set(builtins);

const isIdentStart = (ch: string) => /[A-Za-z_$]/.test(ch);
const isIdent = (ch: string) => /[A-Za-z0-9_$]/.test(ch);

interface GadState {
  // depth of the current /* */ block comment, 0 when not in one.
  blockComment: number;
  // closing fence of the current doc-comment block (`**/` or `***/`), or ""
  // when not inside a `/**`…`**/` / `/***`…`***/` doc block.
  docFence: string;
  // non-empty when inside a ```…``` code fence *within* a doc-comment block.
  // Set to "`\`\`" on open, cleared on close.
  docCodeFence: string;
}

// A pragmatic stream tokenizer for Gad. It is intentionally lightweight (it
// does not build a full syntax tree) but covers comments, the several string
// forms, char/number literals, keywords, builtins and operators well enough for
// editor highlighting.
//
// Doc-comment blocks additionally highlight embedded ``` code fences as normal
// Gad code and `>>> ` result assertion lines as a distinct token.
const gadStreamLanguage = StreamLanguage.define<GadState>({
  name: "gad",

  startState(): GadState {
    return { blockComment: 0, docFence: "", docCodeFence: "" };
  },

  token(stream: StringStream, state: GadState): string | null {
    // Inside a code fence within a doc-comment block: tokenize as Gad code.
    if (state.docFence && state.docCodeFence) {
      return tokenDocCodeBlock(stream, state);
    }
    // Inside a doc-comment block (not a code fence): consume as doc text.
    if (state.docFence) {
      return tokenDocBlock(stream, state);
    }
    // Inside a block comment.
    if (state.blockComment > 0) {
      return tokenBlockComment(stream, state);
    }

    if (stream.eatSpace()) return null;

    const ch = stream.peek() as string;

    // Doc comments (`///` single, `/**`…`**/` and `/***`…`***/` blocks) come
    // before the ordinary // and /* checks so their markers are not read as
    // plain `//`/`/*` comments.
    if (stream.match("/***")) {
      state.docFence = "***/";
      return tokenDocBlock(stream, state);
    }
    if (stream.match("/**")) {
      state.docFence = "**/";
      return tokenDocBlock(stream, state);
    }
    if (stream.match(/^\/\/\/(?!\/)/)) {
      stream.skipToEnd();
      return "docComment";
    }

    // Delegate all non-comment, non-doc-comment tokens to the shared helper so
    // the same logic is reused when highlighting code inside doc-code fences.
    return tokenCodeLine(stream, state, ch);
  },

  tokenTable: {
    lineComment: t.lineComment,
    blockComment: t.blockComment,
    docComment: t.docComment,
    docCodeFence: t.meta,
    docResult: t.special(t.comment),
    string: t.string,
    character: t.character,
    number: t.number,
    keyword: t.keyword,
    atom: t.atom,
    standard: t.standard(t.variableName),
    builtin: t.function(t.variableName),
    variable: t.variableName,
    operator: t.operator,
  },
});

// tokenDocBlock handles one token while inside a `/**`…`**/` (or `/***`…`***/`)
// doc-comment block but outside any embedded code fence. On each line it checks:
//   - whether the line is the outer closing fence (resets docFence)
//   - whether the line starts a ``` code fence (sets docCodeFence)
// Everything else is consumed to end-of-line as "docComment".
function tokenDocBlock(stream: StringStream, state: GadState): string {
  if (stream.sol()) {
    const rest = stream.string.slice(stream.pos);
    if (rest.trim() === state.docFence) {
      state.docFence = "";
      stream.skipToEnd();
      return "docComment";
    }
    if (rest.match(/^[ \t]*```/)) {
      state.docCodeFence = "```";
      stream.skipToEnd();
      return "docCodeFence";
    }
  }
  stream.skipToEnd();
  return "docComment";
}

// tokenDocCodeBlock handles one token while inside a ``` fence within a doc-
// comment block. Closing ``` lines reset docCodeFence; `>>> ` lines are tagged
// as docResult; everything else is delegated to tokenCodeLine (gad code, without
// doc-comment marker detection so `///` / `/**` are not treated as doc openers).
function tokenDocCodeBlock(stream: StringStream, state: GadState): string | null {
  if (stream.sol()) {
    const rest = stream.string.slice(stream.pos);
    if (rest.match(/^[ \t]*```/)) {
      state.docCodeFence = "";
      stream.skipToEnd();
      return "docCodeFence";
    }
    if (rest.match(/^>>> /)) {
      stream.skipToEnd();
      return "docResult";
    }
  }
  // Block comment open/close inside the code fence.
  if (state.blockComment > 0) {
    return tokenBlockComment(stream, state);
  }
  // Normal gad code (no doc-comment detection).
  return tokenCodeLine(stream, state, stream.peek() as string);
}

// tokenCodeLine tokenizes one normal Gad code token. It is shared between the
// main token() path and the doc-code-fence path so that code inside ``` blocks
// in doc comments gets full Gad highlighting. It does NOT check for doc-comment
// markers (`///`, `/**`, `/***`) — those are handled at a higher level.
function tokenCodeLine(stream: StringStream, state: GadState, ch: string): string | null {
  if (stream.eatSpace()) return null;

  // Comments (no doc-comment variants here — `///` → lineComment in code context)
  if (stream.match("//")) {
    stream.skipToEnd();
    return "lineComment";
  }
  if (stream.match("/*")) {
    state.blockComment = 1;
    return tokenBlockComment(stream, state);
  }

  // Heredocs and raw strings
  if (stream.match('"""') || stream.match("```")) {
    return tokenFenced(stream, ch === '"' ? '"""' : "```");
  }

  // Strings
  if (ch === '"') {
    stream.next();
    tokenString(stream, '"');
    return "string";
  }
  if (ch === "`") {
    stream.next();
    tokenRawString(stream);
    return "string";
  }
  if (ch === "'") {
    stream.next();
    tokenString(stream, "'");
    return "character";
  }

  // Numbers (incl. hex, suffixes u/d, decimals)
  if (/[0-9]/.test(ch) || (ch === "." && /[0-9]/.test(stream.string[stream.pos + 1] ?? ""))) {
    stream.match(/^0[xX][0-9a-fA-F]+/) ||
      stream.match(/^[0-9]+\.[0-9]*([eE][-+]?[0-9]+)?[dD]?/) ||
      stream.match(/^\.[0-9]+([eE][-+]?[0-9]+)?/) ||
      stream.match(/^[0-9]+([eE][-+]?[0-9]+)?[uUdD]?/);
    return "number";
  }

  // Bytes literals: b"..."/h"..." (and raw/back-tick forms)
  if ((ch === "b" || ch === "h") && /["`]/.test(stream.string[stream.pos + 1] ?? "")) {
    stream.next(); // prefix
    const q = stream.next() as string;
    if (q === "`") tokenRawString(stream);
    else tokenString(stream, q);
    return "string";
  }

  // Identifiers / keywords / builtins
  if (isIdentStart(ch)) {
    let word = "";
    while (!stream.eol() && isIdent(stream.peek() as string)) {
      word += stream.next();
    }
    if (keywordSet.has(word)) return "keyword";
    if (atomSet.has(word)) return "atom";
    if (constantSet.has(word)) return "standard";
    if (builtinSet.has(word)) return "builtin";
    return "variable";
  }

  // @-prefixed special idents (@args, @module, ...)
  if (ch === "@") {
    stream.next();
    while (!stream.eol() && isIdent(stream.peek() as string)) stream.next();
    return "standard";
  }

  // Operators / punctuation
  if (/[-+*/%<>=!&|^~?:.,;(){}\[\]]/.test(ch)) {
    stream.next();
    return "operator";
  }

  stream.next();
  return null;
}

function tokenBlockComment(stream: StringStream, state: GadState): string {
  while (!stream.eol()) {
    if (stream.match("*/")) {
      state.blockComment = 0;
      return "blockComment";
    }
    stream.next();
  }
  return "blockComment";
}

function tokenString(stream: StringStream, quote: string): void {
  let escaped = false;
  while (!stream.eol()) {
    const c = stream.next() as string;
    if (c === quote && !escaped) return;
    escaped = !escaped && c === "\\";
  }
}

function tokenRawString(stream: StringStream): void {
  while (!stream.eol()) {
    if (stream.next() === "`") return;
  }
}

function tokenFenced(stream: StringStream, fence: string): string {
  while (!stream.eol()) {
    if (stream.match(fence)) return "string";
    stream.next();
  }
  return "string";
}

/** The Gad language (highlighting + comment metadata). */
export const gadLanguage = gadStreamLanguage;

/** LanguageSupport bundle for plugging into an EditorState. */
export function gadLanguageSupport(): LanguageSupport {
  return new LanguageSupport(gadStreamLanguage);
}
