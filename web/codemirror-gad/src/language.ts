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
}

// A pragmatic stream tokenizer for Gad. It is intentionally lightweight (it
// does not build a full syntax tree) but covers comments, the several string
// forms, char/number literals, keywords, builtins and operators well enough for
// editor highlighting.
const gadStreamLanguage = StreamLanguage.define<GadState>({
  name: "gad",

  startState(): GadState {
    return { blockComment: 0 };
  },

  token(stream: StringStream, state: GadState): string | null {
    if (state.blockComment > 0) {
      return tokenBlockComment(stream, state);
    }

    if (stream.eatSpace()) return null;

    const ch = stream.peek() as string;

    // Comments
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
      // Consume to a matching fence on a (possibly later) line; approximate by
      // reading to the closing fence on the same stream window.
      return tokenFenced(stream, ch === '"' ? '"""' : "```", ch === "`");
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
  },

  tokenTable: {
    lineComment: t.lineComment,
    blockComment: t.blockComment,
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

function tokenFenced(stream: StringStream, fence: string, _raw: boolean): string {
  // Already consumed the opening fence. Read until the closing fence appears.
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
