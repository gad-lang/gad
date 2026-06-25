// @gad-lang/prism-gad — PrismJS grammar for the Gad scripting language.
//
//   import Prism from "prismjs";
//   import { registerGad } from "@gad-lang/prism-gad";
//   registerGad(Prism);
//   const html = Prism.highlight(code, Prism.languages.gad, "gad");

import type { Grammar, Environment } from "prismjs";

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
];

const atoms = ["true", "false", "yes", "no", "nil"];

const builtins = [
  "int", "uint", "float", "decimal", "bool", "flag", "char", "string", "str",
  "bytes", "array", "chars", "error", "keyValue", "keyValueArray",
  "typeName", "typeof", "isArray", "isBool", "isBytes", "isCallable", "isChar",
  "isDict", "isError", "isFloat", "isFunction", "isInt", "isIterable",
  "isIterator", "isNil", "isRawStr", "isStr", "isUint", "isSyncDict",
  "len", "append", "delete", "copy", "dcopy", "repeat", "contains", "sort",
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
  // Doc comments (`/?` single, `/??`…`??` and `/???`…`???` blocks) come before
  // ordinary comments so the `/?` marker is not read as a `//`/`/*` comment. A
  // block ends only at a line that is exactly the fence, so an inline `??`/`???`
  // in the doc text (e.g. in a code span) does not close it early.
  "doc-comment": {
    pattern:
      /\/\?\?\?[\s\S]*?(?:^[ \t]*\?\?\?[ \t]*$|$(?![\s\S]))|\/\?\?[\s\S]*?(?:^[ \t]*\?\?[ \t]*$|$(?![\s\S]))|\/\?.*/m,
    greedy: true,
    alias: "comment",
  },
  comment: {
    pattern: /\/\/.*|\/\*[\s\S]*?(?:\*\/|$)/,
    greedy: true,
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
  operator: /\?\?=?|\.\.|=>|:=|\|\||&&|\*\*=?|<<=?|>>=?|&\^=?|[-+*/%&|^!<>=]=?|[~?:]/,
  punctuation: /[{}[\];(),.]/,
};

/** registerGad installs the grammar under Prism.languages.gad. */
export function registerGad(Prism: {
  languages: Record<string, Grammar>;
  hooks?: { add(name: string, cb: (env: Environment) => void): void };
}): void {
  Prism.languages.gad = gadGrammar;
}
