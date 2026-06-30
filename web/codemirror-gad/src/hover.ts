import { Extension } from "@codemirror/state";
import { hoverTooltip, EditorView } from "@codemirror/view";
import { builtins, atoms, constants } from "./keywords";

// Brief per-builtin descriptions shown in the hover tooltip.
const builtinDocs: Record<string, string> = {
  // Type constructors / conversions
  int:          "int(x) → int\nConvert to signed integer.",
  uint:         "uint(x) → uint\nConvert to unsigned integer.",
  float:        "float(x) → float\nConvert to float.",
  decimal:      "decimal(x) → decimal\nConvert to decimal.",
  bool:         "bool(x) → bool\nConvert to bool.",
  flag:         "flag(x) → flag\nConvert to flag (tri-state bool: true/false/nil).",
  char:         "char(x) → char\nConvert to Unicode code point.",
  string:       "string(x) → str\nConvert to string (alias: str).",
  str:          "str(x) → str\nConvert to string.",
  bytes:        "bytes(x) → bytes\nConvert to byte slice.",
  array:        "array(*args) → array\nCreate an array from arguments.",
  chars:        "chars(x) → array[char]\nSplit string into a char array.",
  error:        "error(msg) → error\nCreate an error value.",
  keyValue:     "keyValue(key, value) → keyValue\nCreate a key-value pair.",
  keyValueArray:"keyValueArray(*kvs) → keyValueArray\nCreate a key-value array.",
  // Type inspection
  typeName:     "typeName(x) → str\nReturn the type name of x.",
  typeof:       "typeof(x) → type\nReturn the type object of x.",
  is:           "is(x, type) → bool\nTest whether x is of the given type.",
  isArray:      "isArray(x) → bool",
  isBool:       "isBool(x) → bool",
  isBytes:      "isBytes(x) → bool",
  isCallable:   "isCallable(x) → bool",
  isChar:       "isChar(x) → bool",
  isDict:       "isDict(x) → bool",
  isError:      "isError(x) → bool",
  isFloat:      "isFloat(x) → bool",
  isFunction:   "isFunction(x) → bool",
  isInt:        "isInt(x) → bool",
  isIterable:   "isIterable(x) → bool",
  isIterator:   "isIterator(x) → bool",
  isNil:        "isNil(x) → bool",
  isRawStr:     "isRawStr(x) → bool",
  isStr:        "isStr(x) → bool",
  isUint:       "isUint(x) → bool",
  isSyncDict:   "isSyncDict(x) → bool",
  // Sequences / collections
  len:          "len(x) → int\nReturn the length of a string, array, bytes or dict.",
  append:       "append(collection, *items) → collection\nAppend items to an array, bytes or string.",
  delete:       "delete(dict, key)\nRemove key from dict (mutates).",
  copy:         "copy(x) → x\nShallow copy of an array, dict or bytes.",
  dcopy:        "dcopy(x) → x\nDeep copy.",
  repeat:       "repeat(x, n) → x\nRepeat array or string n times.",
  contains:     "contains(collection, item) → bool\nTest membership.",
  sort:         "sort(array) → array\nSort in ascending order (mutates).",
  sortReverse:  "sortReverse(array) → array\nSort in descending order (mutates).",
  keys:         "keys(dict) → iterator[str]\nLazy iterator over dict keys.",
  values:       "values(dict) → iterator\nLazy iterator over dict values.",
  items:        "items(dict) → iterator[[key,value]]\nLazy iterator over dict entries.",
  zip:          "zip(a, b, ...) → iterator[array]\nZip iterables into tuples.",
  enumerate:    "enumerate(iterable) → iterator[[index, value]]",
  // Iteration
  map:          "map(iterable, fn(value, key)) → iterator\nLazy map. Callback receives (value, key).",
  filter:       "filter(iterable, fn(value, key, it)) → iterator\nLazy filter.",
  reduce:       "reduce(iterable, fn(acc, value, key), init) → value\nEager fold.",
  each:         "each(iterable, fn(key, value))\nEager iteration. Callback receives (key, value).",
  iterate:      "iterate(x) → iterator\nWrap x as an iterator.",
  iterator:     "iterator(fn) → iterator\nCreate an iterator from a next() function.",
  collect:      "collect(iterator) → array\nMaterialise a lazy iterator into an array.",
  toArray:      "toArray(x) → array\nConvert iterable to array.",
  // IO / formatting
  print:        "print(*args)\nPrint to stdout (no newline).",
  println:      "println(*args)\nPrint to stdout followed by a newline.",
  printf:       "printf(fmt, *args)\nFormatted print to stdout.",
  sprintf:      "sprintf(fmt, *args) → str\nFormatted string.",
  repr:         "repr(x) → str\nReturn a diagnostic representation of x.",
  read:         "read() → str\nRead a line from stdin.",
  write:        "write(*args)\nWrite args to stdout (used in template mode).",
  flush:        "flush()\nFlush stdout.",
  stdio:        "stdio() → {in, out, err}\nReturn the stdio streams.",
  // Misc
  globals:      "globals() → syncDict\nReturn the global variables dict.",
  cast:         "cast(x, type) → type\nUnsafe type assertion.",
  wrap:         "wrap(goValue) → value\nWrap a Go value as a Gad object.",
  addMethod:    "addMethod(obj, fn)\nRegister a method on an object type.",
  Class:        "Class(name; define=(Type, define) => define(...)) → type\nCreate a new class type.",
  userData:     "userData(x) → any\nReturn the Go user data attached to x.",
};

// All identifiers that should have a hover tooltip.
const tooltipWords = new Set<string>([
  ...builtins,
  ...atoms,
  ...constants,
]);

const isWordChar = (c: string) => /[A-Za-z0-9_]/.test(c);

function wordAt(text: string, offset: number): { start: number; end: number; word: string } | null {
  let i = Math.min(Math.max(offset, 0), text.length);
  if (i >= text.length || !isWordChar(text[i])) {
    if (i > 0 && isWordChar(text[i - 1])) i -= 1;
    else return null;
  }
  let s = i;
  let e = i;
  while (s > 0 && isWordChar(text[s - 1])) s--;
  while (e < text.length && isWordChar(text[e])) e++;
  if (e <= s) return null;
  return { start: s, end: e, word: text.slice(s, e) };
}

/**
 * gadHoverTooltip shows a brief description popup when hovering over a known
 * builtin function, atom or constant in the gad editor.
 */
export function gadHoverTooltip(): Extension {
  return [
    hoverTooltip((view, pos) => {
      const line = view.state.doc.lineAt(pos);
      const w = wordAt(line.text, pos - line.from);
      if (!w || !tooltipWords.has(w.word)) return null;
      const doc = builtinDocs[w.word] ?? w.word;
      return {
        pos: line.from + w.start,
        end: line.from + w.end,
        above: true,
        create() {
          const dom = document.createElement("div");
          dom.className = "cm-builtin-tooltip";
          // First line is the signature, rest is description.
          const [sig, ...rest] = doc.split("\n");
          const sigEl = document.createElement("span");
          sigEl.className = "cm-builtin-tooltip-sig";
          sigEl.textContent = sig;
          dom.appendChild(sigEl);
          if (rest.length) {
            const desc = document.createElement("span");
            desc.className = "cm-builtin-tooltip-desc";
            desc.textContent = rest.join(" ");
            dom.appendChild(desc);
          }
          return { dom };
        },
      };
    }),
    EditorView.baseTheme({
      ".cm-builtin-tooltip": {
        display: "flex",
        flexDirection: "column",
        padding: "4px 8px",
        fontFamily: "ui-monospace, monospace",
        fontSize: "0.82em",
        gap: "2px",
      },
      ".cm-builtin-tooltip-sig": {
        fontWeight: "600",
      },
      ".cm-builtin-tooltip-desc": {
        opacity: "0.75",
        fontSize: "0.95em",
      },
    }),
  ];
}
