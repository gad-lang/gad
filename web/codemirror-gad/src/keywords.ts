// Gad keywords and builtins, used by both highlighting and autocompletion.

export const keywords: string[] = [
  "if", "else", "for", "in", "func", "method", "return", "break", "continue",
  "try", "catch", "finally", "throw", "match",
  "defer", "defer_ok", "defer_err", "deferb", "deferb_ok", "deferb_err",
  "param", "global", "var", "const", "export",
  "import", "embed", "raw", "template",
  // `code … end` code-string fences; the body between them is itself Gad source
  // and is highlighted by the same tokenizer.
  "begin", "end", "code",
  "or", "is",
];

export const atoms: string[] = ["true", "false", "yes", "no", "nil"];

export const constants: string[] = [
  "STDIN", "STDOUT", "STDERR",
];

// Builtin functions and type constructors available without an import.
export const builtins: string[] = [
  // type constructors / conversions
  "int", "uint", "float", "decimal", "bool", "flag", "char", "string", "str",
  "bytes", "array", "chars", "error", "keyValue", "keyValueArray",
  // type inspection
  "typeName", "typeof", "is", "isArray", "isBool", "isBytes", "isCallable",
  "isChar", "isDict", "isError", "isFloat", "isFunction", "isInt", "isIterable",
  "isIterator", "isNil", "isRawStr", "isStr", "isUint", "isSyncDict",
  // sequences / collections
  "len", "append", "delete", "copy", "dcopy", "repeat", "contains", "sort",
  "sortReverse", "keys", "values", "items", "zip", "enumerate",
  // iteration
  "map", "filter", "reduce", "each", "iterate", "iterator", "collect", "toArray",
  // io / formatting
  "print", "println", "printf", "sprintf", "repr", "read", "write", "flush",
  "stdio",
  // misc
  "globals", "cast", "wrap", "addMethod", "Class", "userData",
];
