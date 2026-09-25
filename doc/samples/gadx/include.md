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
- `@include ("parts/*.gad"; excludes=["*_draft.gad"])` — every file a glob
  matches, in path order, narrowed by the `includes` / `excludes` /
  `includes_re` / `excludes_re` filters (`*_test` files are skipped unless an
  include names `_test`); see [include](../include/main.gad).

Here a glob pulls in the plain-Gad data (`include_data.gad`: `pageTitle`,
`items`) and renders it.

## Components

### main

## Example — `include.gadx`

```gadx
@include ("include_*.gad")

@main
	article
		h1 {= pageTitle }
		ul
			@for it in items
				li {= it }
```
