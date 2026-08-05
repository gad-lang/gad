// File-type registry: maps a file extension to a tree icon and an editor
// language (a built-in EditorLanguage or a custom CodeMirror Extension factory).
// The host can register handlers for new extensions via GadIde's `fileTypes`
// prop, so both the Explorer icon and the editor highlighting are extensible.
import type { Extension } from "@codemirror/state";
import type { EditorLanguage } from "./codemirror";

export interface FileTypeHandler {
  /** Extensions this handler matches, without the dot (e.g. ["gad"], ["md","markdown"]). */
  extensions: string[];
  /** Material Design Icon name for the file tree (e.g. "mdi-language-python"). */
  icon?: string;
  /** A built-in EditorLanguage, or a factory returning a CodeMirror Extension for
   * a custom language / plugin. */
  language?: EditorLanguage | (() => Extension);
}

/** DEFAULT_FILE_TYPES cover the languages the editor already knows, with icons. */
export const DEFAULT_FILE_TYPES: FileTypeHandler[] = [
  { extensions: ["gad"], icon: "mdi-language-go", language: "gad" },
  { extensions: ["gadt"], icon: "mdi-file-code-outline", language: "gadt" },
  { extensions: ["gadx"], icon: "mdi-xml", language: "gadx" },
  { extensions: ["json"], icon: "mdi-code-json", language: "json" },
  { extensions: ["yaml", "yml"], icon: "mdi-file-cog-outline", language: "yaml" },
  { extensions: ["html", "htm"], icon: "mdi-language-html5", language: "html" },
  { extensions: ["css"], icon: "mdi-language-css3", language: "css" },
  { extensions: ["scss"], icon: "mdi-sass", language: "scss" },
  { extensions: ["js", "mjs", "cjs"], icon: "mdi-language-javascript", language: "javascript" },
  { extensions: ["ts", "mts", "cts"], icon: "mdi-language-typescript", language: "typescript" },
  { extensions: ["jsx"], icon: "mdi-language-javascript", language: "jsx" },
  { extensions: ["tsx"], icon: "mdi-language-typescript", language: "tsx" },
  { extensions: ["md", "mdx", "markdown"], icon: "mdi-language-markdown-outline", language: "markdown" },
];

const DEFAULT_ICON = "mdi-file-outline";

/** extOf returns a path's lowercase extension without the dot. */
export function extOf(path: string): string {
  const slash = path.lastIndexOf("/");
  const name = slash === -1 ? path : path.slice(slash + 1);
  const dot = name.lastIndexOf(".");
  return dot === -1 ? "" : name.slice(dot + 1).toLowerCase();
}

/** FileTypeRegistry resolves the icon and editor language for a path from the
 * built-in handlers plus any the host registered (host handlers win). */
export class FileTypeRegistry {
  private byExt = new Map<string, FileTypeHandler>();

  constructor(handlers: FileTypeHandler[] = []) {
    for (const h of [...DEFAULT_FILE_TYPES, ...handlers]) {
      for (const e of h.extensions) this.byExt.set(e.toLowerCase(), h);
    }
  }

  handlerFor(path: string): FileTypeHandler | undefined {
    return this.byExt.get(extOf(path));
  }
  iconFor(path: string): string {
    return this.handlerFor(path)?.icon ?? DEFAULT_ICON;
  }
  /** languageFor returns the built-in EditorLanguage for a path, or "text" when
   * the handler has none / a custom extension factory (see extensionFor). */
  languageFor(path: string): EditorLanguage {
    const lang = this.handlerFor(path)?.language;
    return typeof lang === "string" ? lang : "text";
  }
  /** extensionFor returns a custom CodeMirror Extension factory when the handler
   * provides one, else undefined (use languageFor instead). */
  extensionFor(path: string): (() => Extension) | undefined {
    const lang = this.handlerFor(path)?.language;
    return typeof lang === "function" ? lang : undefined;
  }
}
