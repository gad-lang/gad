// Gad template (mixed) language for CodeMirror 6.
//
// A `.gadt` file is plain text with embedded Gad code between delimiters:
//   {%  … %}   a code block
//   {%= … %}   a value expression
// Text outside the delimiters is emitted literally. The delimiters default to
// `{%` / `%}` but are configurable (matching the `template:` section of a
// project `.gad.yaml` and the `--template*` CLI flags).
import { StreamLanguage, StringStream } from "@codemirror/language";
import {
  GadState,
  gadInContinuation,
  gadToken,
  gadTokenTable,
  newGadState,
} from "./language";

/** Delimiters for the code/value tags. Defaults to `{%` / `%}`. */
export interface GadTemplateDelimiters {
  start?: string;
  end?: string;
}

const DEFAULT_START = "{%";
const DEFAULT_END = "%}";

// A `# gad:` config directive line (see parser ParseConfigStmt). In a `.gad`
// file, mixed mode only begins *after* such a directive; everything before it
// is ordinary Gad.
const CONFIG_DIRECTIVE = /^[ \t]*#[ \t]*gad:[^\n]*/;

interface GadtState {
  // Embedded Gad tokenizer state, used in the preamble and inside a tag.
  gad: GadState;
  // true while between the start and end delimiters.
  inTag: boolean;
  // true while in the leading Gad region of a `.gad` mixed file, before the
  // `# gad: mixed` directive switches on template text. Always false for `.gadt`
  // files (which are template from the first byte).
  preamble: boolean;
}

// startsWith reports whether the stream is positioned at `s` (without consuming).
function startsWith(stream: StringStream, s: string): boolean {
  return stream.match(s, false) === true;
}

/**
 * gadTemplateLanguage builds a StreamLanguage that highlights a Gad template:
 * literal text plus `{% … %}` / `{%= … %}` tags whose bodies are tokenized as
 * Gad. The tag delimiters are taken from `delims` (defaulting to `{%` / `%}`).
 *
 * When `preamble` is set, the document starts as ordinary Gad and switches to
 * template text only after a `# gad:` config directive line — matching a `.gad`
 * file that enables mixed mode with `# gad: mixed` (as opposed to a `.gadt`
 * file, which is template from the first byte).
 */
export function gadTemplateLanguage(delims: GadTemplateDelimiters = {}, preamble = false) {
  const start = delims.start || DEFAULT_START;
  const end = delims.end || DEFAULT_END;

  return StreamLanguage.define<GadtState>({
    name: "gadt",

    startState(): GadtState {
      return { gad: newGadState(), inTag: false, preamble };
    },

    token(stream: StringStream, state: GadtState): string | null {
      // Leading Gad region of a `.gad` mixed file: tokenize as Gad until the
      // `# gad:` directive, then fall through to template text.
      if (state.preamble) {
        if (stream.sol() && !gadInContinuation(state.gad) && stream.match(CONFIG_DIRECTIVE)) {
          state.preamble = false;
          return "templateDirective";
        }
        return gadToken(stream, state.gad);
      }

      if (state.inTag) {
        // Finish any multi-token Gad construct (block comment, string, …) before
        // looking for the closing delimiter, so a `%}` inside a string does not
        // end the tag early.
        if (!gadInContinuation(state.gad)) {
          // Optional whitespace-trim markers (`-` / `--`) precede the end delim.
          if (stream.match("--" + end) || stream.match("-" + end) || stream.match(end)) {
            state.inTag = false;
            return "tagDelimiter";
          }
        }
        return gadToken(stream, state.gad);
      }

      // Literal text: emit the start delimiter as a tag opener, or consume text
      // up to the next start delimiter.
      if (stream.match(start)) {
        // A value tag is `{%=`; trim markers are `{%-` / `{%--`.
        stream.eat("=") || stream.match("--") || stream.eat("-");
        state.inTag = true;
        return "tagDelimiter";
      }

      // Consume plain template text until the next start delimiter (or EOL).
      stream.next();
      while (!stream.eol() && !startsWith(stream, start)) stream.next();
      return "tagContent";
    },

    tokenTable: gadTokenTable,
  });
}
