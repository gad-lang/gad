
# Templates — a `.gadt` example

A `.gadt` file runs as a template automatically — no `# gad: mixed` directive and
no `--template` flag needed; the extension is enough (`gad run
samples/template_example.gadt`). Text outside the tags is emitted literally; `{% … %}`
runs statements (emitting nothing), `{%= expr %}` writes a value, and the `-` /
`--` trim markers strip adjacent whitespace (`{%-`/`-%}` keep a single newline,
`{%--`/`--%}` remove it too).

In a `.gadt` the **module doc lives inside the leading code island** — a
`{%-- … --%}` block wrapping a `/*** … *

## Example — `template_example.gadt`

```gadt
{%--
/**
# Templates — a `.gadt` example

A `.gadt` file runs as a template automatically — no `# gad: mixed` directive and
no `--template` flag needed; the extension is enough (`gad run
samples/template_example.gadt`). Text outside the tags is emitted literally; `{% … %}`
runs statements (emitting nothing), `{%= expr %}` writes a value, and the `-` /
`--` trim markers strip adjacent whitespace (`{%-`/`-%}` keep a single newline,
`{%--`/`--%}` remove it too).

In a `.gadt` the **module doc lives inside the leading code island** — a
`{%-- … --%}` block wrapping a `/*** … ***/` root comment, like this one — so it
is captured as prose without being emitted as template text. Part of
[Templates](template.gad).
**/
--%}
{%
var (
  title = "Gad users"
  users = [
    {name: "joe",    admin: true},
    {name: "mary",   admin: false},
    {name: "george", admin: false},
  ]
  lastIndex = len(users) - 1
)
--%}
<!doctype html>
<html>
  <head><title>{%= title %}</title></head>
  <body>
    <h1>{%= title %} ({%= len(users) %})</h1>
    <ul>
{%-- for i, u in users begin %}
      <li>{%= u.name %}{% if u.admin begin %} <strong>(admin)</strong>{% end %}</li>
{%-- end %}
    </ul>
    <p>last index: {%= lastIndex %}, total: {%= len(users) %}</p>
  </body>
</html>
```
