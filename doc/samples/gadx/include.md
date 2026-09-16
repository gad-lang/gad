# include

*
`@include` — compile source file(s) inline (the Gadx form of Gad's `include`).

`@include` reads another file and compiles its statements INLINE into the
template, so its declarations are visible to the markup below — unlike `@import`,
which loads an isolated module. It lowers to Gad's `include`, so the paths and
positions are preserved (a bad path reports the right .gadx line).

Forms:

- `@include "data.gad"`                — one file (a bare string).
- `@include ("a.gad", "b.gad")`        — several, in order.

Here it pulls in plain-Gad data (`pageTitle`, `items`) and renders it.

## Components

### main

## Example — `include.gadx`

```gadx
@include "include_data.gad"

@main
	article
		h1 {= pageTitle }
		ul
			@for it in items
				li {= it }
```
