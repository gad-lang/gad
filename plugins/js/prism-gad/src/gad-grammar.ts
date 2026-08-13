// gad-grammar.ts — the core Gad PrismJS grammar and its word lists.
//
// This is a leaf module (no imports from ./index, ./template or ./gadx) so the
// template and gadx grammars — which embed the Gad grammar — can import it
// without a circular dependency through the package entry point.
import type { Grammar } from "prismjs";

const keywords = [
  "if", "else", "for", "in", "func", "method", "return", "break", "continue",
  "try", "catch", "finally", "throw", "match",
  "defer_ok", "defer_err", "defer", "deferb_ok", "deferb_err", "deferb",
  "param", "global", "var", "const", "export",
  "import", "embed", "raw", "template",
  // `code … end` code-string fences; the body between them is itself Gad source.
  "begin", "end", "code", "or", "is",
  // added by update plugin
  "ain", "met", "meti", "prop", "with",
  // added by update plugin
  "class", "enum", "interface",
  // `env` environment-table keyword; `delete` statement.
  "env", "delete",
];

const atoms = ["true", "false", "yes", "no", "nil"];

const builtins = [
  "int", "uint", "float", "decimal", "bool", "flag", "char", "string", "str",
  "bytes", "array", "chars", "error", "keyValue", "keyValueArray",
  "typeName", "typeof", "isArray", "isBool", "isBytes", "isCallable", "isChar",
  "isDict", "isError", "isFloat", "isFunction", "isInt", "isIterable",
  "isIterator", "isNil", "isRawStr", "isStr", "isUint", "isSyncDict",
  "len", "copy", "dcopy", "repeat", "contains", "sort",
  "sortReverse", "keys", "values", "items", "zip", "enumerate",
  "map", "filter", "reduce", "each", "iterate", "iterator", "collect", "toArray",
  "print", "println", "printf", "sprintf", "repr", "read", "write", "flush",
  "globals", "cast", "wrap", "addMethod", "Class", "userData",
];

const word = (words: string[]) => new RegExp(`\\b(?:${words.join("|")})\\b`);

/**
 * The Gad PrismJS grammar. Greedy patterns are used for comments and the
 * several string forms so they win over later token rules.
 */
export const gadGrammar: Grammar = {
  // Doc comments: `///` single-line, and `/** … **/` block (the three-star
  // `/*** … ***/` form is deprecated but still recognized). The unified block
  // documents the statement it sits directly above, or the module/section when a
  // blank line follows it. Its OPENING fence must be alone on its line, so an
  // inline `/** text **/` (and the `/**= … **/` / `/**< … **/` doc-generation
  // markers) is an ordinary comment, not a doc block. A block ends only at a line
  // that is exactly the closing fence, so inline `**bold**`/`***hr***` Markdown
  // does not close it early. Doc comments come before ordinary comments so their
  // markers are not read as `//`/`/*`. Inside blocks, ``` fence markers and
  // `>>> ` result assertion lines are highlighted with distinct classes.
  "doc-comment": {
    pattern:
      /\/\*\*\*[ \t]*$[\s\S]*?(?:^[ \t]*\*\*\*\/[ \t]*$|$(?![\s\S]))|\/\*\*[ \t]*$[\s\S]*?(?:^[ \t]*\*\*\/[ \t]*$|$(?![\s\S]))|\/\/\/(?!\/).*/m,
    greedy: true,
    alias: "comment",
    inside: {
      "doc-code-fence": {
        pattern: /^[ \t]*```[^\n]*/m,
        alias: "punctuation",
      },
      "doc-result": {
        pattern: /^>>> .*/m,
        alias: "output",
      },
    },
  },
  comment: {
    pattern: /\/\/.*|\/\*[\s\S]*?(?:\*\/|$)/,
    greedy: true,
  },
  // Template strings: #"…{expr}…" and #`…{expr}…` (heredoc forms too).
  // Highlighted as strings with `{…}` interpolation delimiters distinguished.
  "template-string": {
    pattern: /#(?:"""[\s\S]*?"""|```[\s\S]*?```|"(?:\\.|[^"\\])*"|`[^`]*`)/,
    greedy: true,
    alias: "string",
    inside: {
      interpolation: {
        pattern: /\{[^}]*\}/,
        inside: {
          "interpolation-punctuation": {
            pattern: /^\{|\}$/,
            alias: "punctuation",
          },
        },
      },
    },
  },
  // Heredocs first (longer fences), then regular/raw strings and chars.
  string: {
    pattern:
      /"""[\s\S]*?"""|```[\s\S]*?```|[bh]?"(?:\\.|[^"\\])*"|[bh]?`[^`]*`|'(?:\\.|[^'\\])*'/,
    greedy: true,
  },
  regex: {
    // /pattern/ only after an operator/keyword/opening bracket or line start.
    pattern: /(^|[(,=:?[{}|&!]|\b(?:return|in|or)\s)\s*\/(?:\\.|[^/\\\r\n])+\/[a-z]*/,
    lookbehind: true,
    greedy: true,
    alias: "string",
  },
  keyword: word(keywords),
  boolean: word(atoms),
  builtin: word(builtins),
  "class-name": {
    // @-prefixed specials: @args, @module, @main, ...
    pattern: /@[A-Za-z_$][\w$]*/,
  },
  function: {
    pattern: /[A-Za-z_$][\w$]*(?=\s*\()/,
  },
  number: /\b0[xX][0-9a-fA-F]+\b|\b\d+(?:\.\d+)?(?:[eE][-+]?\d+)?[uUdD]?\b|\B\.\d+\b/,
  operator: /::|\?\?=?|\.\.|=>|:=|\|\||&&|\*\*=?|<<=?|>>=?|&\^=?|[-+*/%&|^!<>=]=?|[~?:]/,
  punctuation: /[{}[\];(),.]/,
};
