# text

## Components

### main

## Parameters

### @param

```gadx
@param(; name="world")
```

@text — a literal-text block.

Every line indented under `@text` is emitted verbatim, with no tag or
directive parsing: bare words stay words (they are NOT turned into tags) and
the original line breaks are preserved. Blank lines are kept too, so the
block reproduces the source text exactly.

`{= expr }` interpolation still works inside the text — only tag/directive
parsing is disabled — and each interpolation keeps its source position, so
diagnostics point at the right column.

Use `@text` for preformatted content (license headers, ASCII art, embedded
config, `<pre>` bodies) where you want the text as-is.

## Example — `text.gadx`

```gadx
@param (; name="world")

@main
	@text
		Dear {= name },

		Thank you for trying Gadx.
		This whole block is literal text — <b>tags</b> are not parsed,
		and the blank line above is preserved.

		    indented lines keep their extra spaces too.
```
