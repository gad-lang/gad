# text_blocks

## Components

### main

## Parameters

### @param

```gadx
@param(; name="Gad")
```

Text output forms — the ways to emit text inside a tag.

Gadx has several text forms; they differ in how line breaks are handled:

- `tag TEXT`      inline text right after a tag on the same line.
- `| TEXT`        one text line; consecutive `| ` lines are joined with **no**
                  separator (`| a` / `| b` -> `ab`).
- `|`  (block)    a YAML-style literal block (like YAML `|`): the indented lines
                  are text with **no** `| ` prefix, and line breaks are **kept**.
- `|>` (block)    a YAML-style folded block (like YAML `>`): line breaks become
                  **spaces**.
- `@text`         a literal-text block: verbatim lines, no tag/directive parsing,
                  line breaks and blank lines preserved.

`{= expr }` interpolation works in every form.

## Example — `text_blocks.gadx`

```gadx
@param (; name="Gad")

@main
	//- inline text after the tag
	h1 Hello {= name }
	//- per-line `| `: the two lines concatenate -> "firstsecond"
	p
		| first
		| second
	//- `|` literal block: line breaks kept -> "one\ntwo"
	pre
		|
			one
			two {= name }

	//- `|>` folded block: line breaks become spaces -> "one two three"
	p
		|>
			one
			two
			three

	//- `@text`: verbatim, no tag parsing (bare words stay words)
	@text
		License (c) {= name }
		  indented lines keep their spaces
```
