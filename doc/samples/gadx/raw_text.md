# raw_text

Script and stylesheet content.

A `<script>` or a `<style>` holds text, not markup, and Gadx reads it as such:

- **Braces are literal.** A CSS rule or a JS block is not an interpolation, so
  `.a { color: red }` and `if (x) { y() }` are written as they are.
- **Nothing is escaped.** A quote stays a quote; escaping it would break the
  string it belongs to.
- **Whitespace is kept.** Both languages can depend on it — a template literal
  or an indentation-sensitive stylesheet would not survive being reflowed.
- **`<` is content.** A `<` inside a script does not open a tag, so the element
  runs to its own close tag and nothing in between is read as markup.

Interpolation there is `#{= expr }#` — the closing `}#` is what makes it
unambiguous, since a lone `}` belongs to the CSS or the JS. `#{ expr }#` is the
control form: it runs and writes nothing, the same pair as `{= … }` and `{ … }`
in ordinary text. A written value goes out verbatim like the rest, so whatever
it carries must already be valid in the language it lands in.

`@raw_text` gives the same reading to a block of its own, for text that is not
inside a script or a stylesheet — an inline JSON payload, a shell snippet, a
license header. Its body is written out as given, minus the indentation the
block itself introduces: the indentation *inside* the block is content and is
kept, so a nested rule stays nested.

## Components

### main

## Example — `raw_text.gadx`

```gadx
~ accent := "#c94"
~ maxWidth := 960

@main
    <style>
        /* Braces, quotes and indentation all survive. */
        :root { --accent: #{= accent }#; }

        .banner {
            color: var(--accent);
            max-width: #{= maxWidth }#px;
            content: "a { not a block }";
        }
    </style>

    <script>
        // `<` and `{` are content here, not markup.
        const width = #{= maxWidth }#;
        if (width < 1000) {
            document.body.dataset.narrow = "yes";
        }
    </script>

    // The same rules, in a block of their own. The two leading levels of
    // indentation belong to the block and are dropped; the one inside the
    // object is content and survives.
    @raw_text
        {
            "accent": "#{= accent }#",
            "maxWidth": #{= maxWidth }#
        }
```
