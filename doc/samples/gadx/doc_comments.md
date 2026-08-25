# doc_comments

Single-line doc comments — `/// …`.

A `///` line is a documentation comment: it attaches to the declaration on
the next line (a `@comp`, `@func`, `@param`, `@var`, `@const` or `@enum`) and
shows up in the generated docs / Public API. Consecutive `///` lines join into
one multi-line description. It is the line-oriented analog of this `/** …

## Components

### card

```gadx
card(title, body)
```

card renders a titled content box: title in the header, body below.

### main

## Functions

### greet

```gadx
greet(name)
```

greet builds a friendly greeting for name.

## Example — `doc_comments.gadx`

```gadx
| `
block doc comment, so short notes read cleanly inline.
| (A `//-` line is a silent comment instead: neither rendered in the output nor
collected as documentation.)
| **/

/** card renders a titled content box: title in the header, body below. **/
@comp card(title, body)
	section
		h2 {= title }
		p {= body }

/** greet builds a friendly greeting for name. **/
@func greet(name)
	| Hello, {= name }!

@main
	+card("Welcome", greet("world"))
```
