import { StreamLanguage, StringStream, LanguageSupport } from "@codemirror/language";
import { tags as t } from "@lezer/highlight";
import {
  GadState,
  gadToken,
  gadTokenTable,
  newGadState,
} from "./language";

// giomTokenTable extends the Gad token table (reused for embedded Gad inside
// `{= … }` interpolations and `~~ … ~~` code blocks) with the giom-specific
// markup token kinds. Every giom token maps to a lezer highlight tag so themes
// colour tags, classes, attributes and control keywords distinctly.
export const giomTokenTable = {
  ...gadTokenTable,
  giomTag: t.tagName,
  giomClass: t.className,
  giomId: t.attributeName,
  giomKeyword: t.keyword,
  giomComponent: t.function(t.variableName),
  giomComment: t.lineComment,
  giomDoctype: t.meta,
  giomText: t.content,
  giomFence: t.meta,
  // `{= … }` / `[ … ]` / `|` markers use tagName — the conventional lezer tag
  // for markup/template delimiters (as the Gad-template mode does).
  giomDelimiter: t.tagName,
};

// LineMode is the parser state within a single logical line (or a multi-line
// `[ … ]` attribute group). It is reset to "start" at every start-of-line unless
// the tokenizer is mid attribute group, `~~` code block or `{ … }` interpolation.
type LineMode = "start" | "tagHead" | "attr" | "text" | "html" | "gad";

/** GiomState is the StreamLanguage state for the Giom template language. */
export interface GiomState {
  // Embedded Gad tokenizer state, used inside interpolations and `~~` blocks.
  gad: GadState;
  // true while inside a `~~ … ~~` Gad code block.
  code: boolean;
  // Brace depth inside a `{ … }` / `{= … }` interpolation (0 = not interpolating).
  interp: number;
  // Bracket depth inside a `[ … ]` attribute group (may span multiple lines).
  attrDepth: number;
  // Current within-line parsing mode.
  line: LineMode;
  // Line mode to return to when the current interpolation closes.
  interpReturn: LineMode;
}

function newGiomState(): GiomState {
  return {
    gad: newGadState(),
    code: false,
    interp: 0,
    attrDepth: 0,
    line: "start",
    interpReturn: "text",
  };
}

const isIdentStart = (ch: string) => /[A-Za-z_]/.test(ch);
const isTagChar = (ch: string) => /[A-Za-z0-9_-]/.test(ch);

// tokenInterp tokenizes one token inside a `{ … }` / `{= … }` interpolation. The
// `{`/`}` braces are intercepted here to track depth; everything else is
// delegated to the Gad tokenizer so interpolations get full Gad highlighting.
function tokenInterp(stream: StringStream, state: GiomState): string | null {
  const ch = stream.peek() as string;
  if (ch === "{") {
    stream.next();
    state.interp++;
    return "operator";
  }
  if (ch === "}") {
    stream.next();
    state.interp--;
    if (state.interp === 0) {
      state.line = state.interpReturn;
      return "giomDelimiter";
    }
    return "operator";
  }
  return gadToken(stream, state.gad);
}

// enterInterp consumes an opening `{` (and an optional `=` buffered marker) and
// switches to interpolation mode, remembering the mode to resume afterwards.
function enterInterp(stream: StringStream, state: GiomState, ret: LineMode): string {
  stream.next(); // '{'
  if (stream.peek() === "=") stream.next(); // buffered `{= … }` marker
  state.interp = 1;
  state.interpReturn = ret;
  return "giomDelimiter";
}

// tokenText tokenizes plain text content, breaking out to interpolation at `{`.
function tokenText(stream: StringStream, state: GiomState): string | null {
  if (stream.peek() === "{") return enterInterp(stream, state, "text");
  while (!stream.eol() && stream.peek() !== "{") stream.next();
  return "giomText";
}

// tokenHTML tokenizes a raw-HTML line (`<tag …>`): angle brackets are markers,
// `{ … }` interpolates, and the rest is content.
function tokenHTML(stream: StringStream, state: GiomState): string | null {
  const ch = stream.peek() as string;
  if (ch === "{") return enterInterp(stream, state, "html");
  if (ch === "<" || ch === ">" || ch === "/") {
    stream.next();
    return "giomDelimiter";
  }
  if (ch === '"' || ch === "'") {
    const q = stream.next() as string;
    while (!stream.eol()) {
      if (stream.next() === q) break;
    }
    return "string";
  }
  while (!stream.eol()) {
    const c = stream.peek() as string;
    if (c === "{" || c === "<" || c === ">" || c === "/" || c === '"' || c === "'") break;
    stream.next();
  }
  return "giomText";
}

// tokenAttr tokenizes the inside of a `[ … ]` attribute group: brackets track
// depth (the group may span lines), everything else is Gad (attribute names read
// as variables, `=`, strings, expressions).
function tokenAttr(stream: StringStream, state: GiomState): string | null {
  const ch = stream.peek() as string;
  if (ch === "]") {
    stream.next();
    state.attrDepth--;
    if (state.attrDepth === 0) state.line = "tagHead";
    return "giomDelimiter";
  }
  if (ch === "[") {
    stream.next();
    state.attrDepth++;
    return "giomDelimiter";
  }
  return gadToken(stream, state.gad);
}

// tokenTagHead tokenizes the tag "head": the element name, `.class` and `#id`
// segments and `[ … ]` attribute groups. A space ends the head and the rest of
// the line becomes text.
function tokenTagHead(stream: StringStream, state: GiomState): string | null {
  if (stream.eatSpace()) {
    state.line = "text";
    return null;
  }
  const ch = stream.peek() as string;
  if (ch === ".") {
    stream.next();
    while (!stream.eol() && isTagChar(stream.peek() as string)) stream.next();
    return "giomClass";
  }
  if (ch === "#") {
    stream.next();
    while (!stream.eol() && isTagChar(stream.peek() as string)) stream.next();
    return "giomId";
  }
  if (ch === "[") {
    stream.next();
    state.attrDepth = 1;
    state.line = "attr";
    return "giomDelimiter";
  }
  if (ch === "{") return enterInterp(stream, state, "text");
  if (isTagChar(ch)) {
    while (!stream.eol() && isTagChar(stream.peek() as string)) stream.next();
    return "giomTag";
  }
  // Unknown head character: fall back to text.
  state.line = "text";
  return null;
}

// tokenStart dispatches at the beginning of a logical line (after indentation),
// classifying it as a comment, doctype, code fence, pipe-text, raw HTML, control
// keyword (`@…`), component call (`+…`) or a tag line.
function tokenStart(stream: StringStream, state: GiomState): string | null {
  if (stream.eatSpace()) return null;
  if (stream.eol()) return null;

  // Comments: `//` and silent `//-`.
  if (stream.match("//")) {
    stream.skipToEnd();
    return "giomComment";
  }
  // Doctype: `!!! 5`.
  if (stream.match(/^!!!/)) {
    stream.skipToEnd();
    return "giomDoctype";
  }

  const ch = stream.peek() as string;

  // Pipe text block: `| plain text`.
  if (ch === "|") {
    stream.next();
    state.line = "text";
    return "giomDelimiter";
  }
  // Raw HTML line.
  if (ch === "<") {
    state.line = "html";
    return tokenHTML(stream, state);
  }
  // Control keyword: `@main`, `@if`, `@for`, `@var`, `@enum`, `@import`, …
  if (ch === "@") {
    stream.next();
    while (!stream.eol() && /[A-Za-z_]/.test(stream.peek() as string)) stream.next();
    state.line = "gad"; // the remainder is a Gad expression / declaration
    return "giomKeyword";
  }
  // Component call: `+comp(...)` / `+mod.comp(...)`.
  if (ch === "+") {
    stream.next();
    while (!stream.eol() && /[A-Za-z0-9_.]/.test(stream.peek() as string)) stream.next();
    state.line = "gad";
    return "giomComponent";
  }
  // Interpolation at line start (bare `{= … }`).
  if (ch === "{") return enterInterp(stream, state, "text");
  // Otherwise a tag line (`div`, `.class`, `#id`, `section.hero`, …).
  if (isIdentStart(ch) || ch === "." || ch === "#") {
    state.line = "tagHead";
    return tokenTagHead(stream, state);
  }
  // Fallback.
  stream.next();
  return null;
}

/** giomToken tokenizes one Giom token. */
export function giomToken(stream: StringStream, state: GiomState): string | null {
  // Interpolation has top priority (it may span lines).
  if (state.interp > 0) return tokenInterp(stream, state);

  // `~~ … ~~` Gad code block body.
  if (state.code) {
    if (stream.sol() && stream.match(/^\s*~~\s*$/)) {
      state.code = false;
      return "giomFence";
    }
    return gadToken(stream, state.gad);
  }

  // Reset per-line state at the start of each line, except while a `[ … ]`
  // attribute group is still open (it may span multiple lines).
  if (stream.sol() && state.line !== "attr") state.line = "start";

  switch (state.line) {
    case "start": {
      // `~~` on its own line opens a Gad code block.
      if (stream.match(/^\s*~~\s*$/)) {
        state.code = true;
        return "giomFence";
      }
      return tokenStart(stream, state);
    }
    case "tagHead":
      return tokenTagHead(stream, state);
    case "attr":
      return tokenAttr(stream, state);
    case "text":
      return tokenText(stream, state);
    case "html":
      return tokenHTML(stream, state);
    case "gad":
      return gadToken(stream, state.gad);
  }
}

const giomStreamLanguage = StreamLanguage.define<GiomState>({
  name: "giom",
  startState: newGiomState,
  token: giomToken,
  tokenTable: giomTokenTable,
  languageData: { commentTokens: { line: "//" } },
});

/** The Giom template language (highlighting + comment metadata). */
export const giomLanguage = giomStreamLanguage;

/** LanguageSupport bundle for the Giom language, for plugging into an EditorState. */
export function giomLanguageSupport(): LanguageSupport {
  return new LanguageSupport(giomStreamLanguage);
}
