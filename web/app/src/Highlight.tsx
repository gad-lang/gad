import { useMemo } from "react";
import Prism from "prismjs";
import { registerGad } from "@gad-lang/prism-gad";

// Register the Gad grammar once at module load.
registerGad(Prism);

const SAMPLES: { title: string; code: string }[] = [
  {
    title: "Functions & closures",
    code: `adder := func(base) {
  return (x) => base + x
}
add5 := adder(5)
println(add5(4))   // 9`,
  },
  {
    title: "Comprehensions & spread",
    code: `nums := [1, 2, 3, 4]
doubled := [n * 2 for n in nums if n > 1]
all := [0, *doubled, 99]
m := {[k]: k * k for k in nums}`,
  },
  {
    title: "Match, defer, errors",
    code: `safe := func() {
  defer_err { $ret = "recovered"; $err = nil }
  throw "boom"
}
label := match (n) { 1: "one", else: "other" }
z := mayThrow() or 2`,
  },
  {
    title: "Strings, bytes, regex",
    code: `s := """
  heredoc
  """
data := h"ffccf1c2"   // bytes
hi := b"Hello"
re := /(\\d+)-(\\d+)/
re.replace("12-34", "$2/$1")`,
  },
];

function highlight(code: string): string {
  return Prism.highlight(code, Prism.languages.gad, "gad");
}

/**
 * Highlight is the PrismJS demo page: read-only Gad snippets rendered with the
 * @gad-lang/prism-gad grammar (static highlighting, no editor).
 */
export function Highlight() {
  const blocks = useMemo(
    () => SAMPLES.map((s) => ({ ...s, html: highlight(s.code) })),
    [],
  );
  return (
    <div className="highlight-page">
      <p className="hint">
        Static highlighting with <code>@gad-lang/prism-gad</code> (PrismJS). Useful
        for docs, blogs and read-only code blocks.
      </p>
      {blocks.map((b) => (
        <figure className="sample" key={b.title}>
          <figcaption>{b.title}</figcaption>
          <pre className="language-gad">
            <code
              className="language-gad"
              dangerouslySetInnerHTML={{ __html: b.html }}
            />
          </pre>
        </figure>
      ))}
    </div>
  );
}
