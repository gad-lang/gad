# text_space

`*` — a single space between two elements.

A tag and its text each own a line, and a line's edges are trimmed: whatever
whitespace separated two elements in the source is gone by the time the line is
read. Most of the time that is what you want — the indentation of a template is
layout, not content. Between two inline elements it is not: the space in
`<a>one</a> <a>two</a>` is what keeps the two words apart, and without it the
page reads `onetwo`.

A line holding nothing but `*` is that space. It must be exactly `*` — `*x` is
a tag and `* x` is text — and to write a literal asterisk on its own, pipe it:
`| *`.

The long form is `{= " " }`: a line that is one interpolation needs no `| `
prefix, because a line opening with `{=` already reads as the value it carries.
That form is what to reach for when the run is more than one space, or when it
sits beside a tag on the same line (`span {= " ⋅ " }`), where `*` is not read.

`gad transpile page.html` writes both: `*` for the single spaces it finds
between inline elements, `{= "…" }` where a run's edges have to survive.

## Components

### main

## Example — `text_space.gadx`

```gadx
@main
	p
		a[href="#one"] one
		*
		a[href="#two"] two
	p
		| between these two
		{= "   " }
		| there are three spaces
	p
		span before
		span {= " ⋅ " }
		span after
	p
		| a literal asterisk:
		*
		| *
```
