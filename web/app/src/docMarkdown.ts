// A small, dependency-free Markdown renderer for doc-comment content. It is not
// a full CommonMark implementation — it covers what doc comments use: fenced
// code blocks (highlighted with the Gad Prism grammar), headings, bullet lists,
// blockquotes, paragraphs and inline code/bold/italic.
import Prism from "prismjs";
import { registerGad } from "@gad-lang/prism-gad";

registerGad(Prism);

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");
}

// Inline formatting on an already-block-level line: code spans, bold, italic.
function renderInline(s: string): string {
  let h = escapeHtml(s);
  h = h.replace(/`([^`]+)`/g, "<code>$1</code>");
  h = h.replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>");
  h = h.replace(/(^|[^*])\*([^*\n]+)\*/g, "$1<em>$2</em>");
  return h;
}

// Render the non-code text between fenced blocks: headings, bullet lists,
// blockquotes and paragraphs (blank-line separated).
function renderTextBlock(text: string): string {
  const lines = text.split("\n");
  let out = "";
  let para: string[] = [];
  let list: string[] = [];

  const flushPara = () => {
    if (para.length) {
      out += "<p>" + renderInline(para.join(" ")) + "</p>";
      para = [];
    }
  };
  const flushList = () => {
    if (list.length) {
      out += "<ul>" + list.map((li) => "<li>" + renderInline(li) + "</li>").join("") + "</ul>";
      list = [];
    }
  };

  for (const raw of lines) {
    const line = raw.trimEnd();
    const heading = /^(#{1,6})\s+(.*)$/.exec(line);
    const bullet = /^\s*[-*]\s+(.*)$/.exec(line);
    const quote = /^\s*>\s?(.*)$/.exec(line);
    if (line.trim() === "") {
      flushPara();
      flushList();
    } else if (heading) {
      flushPara();
      flushList();
      const level = heading[1].length;
      out += `<h${level}>${renderInline(heading[2])}</h${level}>`;
    } else if (bullet) {
      flushPara();
      list.push(bullet[1]);
    } else if (quote) {
      flushPara();
      flushList();
      out += "<blockquote>" + renderInline(quote[1]) + "</blockquote>";
    } else {
      flushList();
      para.push(line);
    }
  }
  flushPara();
  flushList();
  return out;
}

/** renderDocMarkdown converts doc-comment Markdown to HTML, highlighting fenced
 * code blocks with the Gad (or named) Prism grammar. Code defaults to `gad`. */
export function renderDocMarkdown(md: string): string {
  const parts = md.split("```");
  let out = "";
  for (let i = 0; i < parts.length; i++) {
    if (i % 2 === 1) {
      // Fenced code block. A leading `lang` token selects the grammar.
      let code = parts[i];
      let lang = "gad";
      const nl = code.indexOf("\n");
      if (nl >= 0) {
        const first = code.slice(0, nl).trim();
        if (/^[a-zA-Z][\w-]*$/.test(first)) {
          lang = first.toLowerCase();
          code = code.slice(nl + 1);
        }
      }
      code = code.replace(/\n$/, "");
      const grammar = Prism.languages[lang] || Prism.languages.gad;
      // Highlight the block line-by-line so that `>>> ` result assertions
      // are rendered with a distinct style rather than as code.
      const codeLines = code.split("\n");
      let codeHtml = "";
      let pending: string[] = [];
      const flushPending = () => {
        if (!pending.length) return;
        const src = pending.join("\n");
        codeHtml += grammar ? Prism.highlight(src, grammar, lang) : escapeHtml(src);
        if (codeHtml && !codeHtml.endsWith("\n")) codeHtml += "\n";
        pending = [];
      };
      for (const ln of codeLines) {
        if (ln.startsWith(">>> ")) {
          flushPending();
          codeHtml += `<span class="doc-result">${escapeHtml(ln)}</span>\n`;
        } else {
          pending.push(ln);
        }
      }
      flushPending();
      codeHtml = codeHtml.replace(/\n$/, "");
      out += `<pre class="doc-code language-${lang}"><code>${codeHtml}</code></pre>`;
    } else {
      out += renderTextBlock(parts[i]);
    }
  }
  return out;
}
