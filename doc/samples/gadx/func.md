# func

`@func` sample — a template function: what it returns, how to call it, and
where its slots come from.

`@func` and `@comp` lower to the same thing: a Gad function that takes a
`slots` named parameter and whose body builds into a fresh `gadx.Elements()`
fragment, which it returns. So a `@func` is not a statement that prints — it
is a value you can hold, append, append twice, or pass on.

At runtime the two are the same: a `@func` takes slots and can be filled by a
caller's `@slot #name` block exactly as a `@comp` can. What only `@comp` does
is record its slots on the declaration, which is what the generated docs and
the tooling list; nothing in the lowering reads that.

Where slots surprise people is that they come from the *call*, not from the
file: a helper invoked with `+helper` is called with none, so its defaults
render even when the caller filled a slot of that name on the surrounding
component — see `with_slot` below.

## Components

### panel

```gadx
panel(title)
```

### main

## Functions

### badge

```gadx
badge(text)
```

### stars

```gadx
stars(items)
```

### with_slot

A slot inside a `@func` reads `slots["main"]`, exactly as it would in a
`@comp`: filled when the caller passes it, the default body otherwise.

## Example — `func.gadx`

```gadx
@func badge(text)
	span[class="badge"] {= text }

@func stars(items)
	@for _ in items
		i[class="star"] ★

/**
A slot inside a `@func` reads `slots["main"]`, exactly as it would in a
`@comp`: filled when the caller passes it, the default body otherwise.
**/
@func with_slot()
	div[class="slotted"]
		@slot main
			p default

@comp panel(title)
	div[class="panel"]
		h3 {= title }
		// `+name` appends what the function returned.
		+badge("new")

		// The caller's slots are not forwarded on their own: `+with_slot`
		// calls it with none, so the default renders.
		+with_slot

		// Forwarding them explicitly is what reaches the caller's block.
		~ $el += with_slot(; slots=slots)

@main
	// 1) called for its markup
	+badge("direct")

	// 2) held as a value, appended where it is wanted
	~ b := badge("held")
	div[class="once"]
		~ $el += b

	// 3) the same fragment used twice — it is a value, not a statement
	div[class="twice"]
		~ $el += b
		~ $el += b

	// 4) a loop inside the function
	div[class="stars"]
		+stars(["a", "b", "c"])

	// 5) inside a component, with a slot the component forwards
	+panel("Title")
		@slot #main
			p from the caller
```
