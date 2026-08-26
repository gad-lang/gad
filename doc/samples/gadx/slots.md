# slots

Slots sample — default content, optional slots, scoped slots and `super`.
Render the `@main` block to see all four behaviors composed together.

## Components

### card

```gadx
card(title)
```

### list

```gadx
list(items)
```

### main

## Example — `slots.gadx`

```gadx
@export comp card(title)
	div[class="card"]
		// optional slot: renders only when the caller provides a badge
		@slot badge
		div[class="card-header"]
			// slot with default content
			@slot header
				h3 {= title }
		div[class="card-body"]
			@slot main
				p Nothing here yet.

@export comp list(items)
	ul[class="list"]
		@for it in items
			li
				// scoped slot: passes `it` to the caller's content
				@slot row(it)
					span {= it }

@main
	// 1) all defaults
	+card("Plain")
	// 2) override header, then render the default header via super
	// `super` is auto-injected as the override's first parameter.
	+card("Fancy")
		@slot #badge
			span[class="badge"] NEW
		@slot #header
			em ★
			+super
	// 3) scoped slot override receiving the row value
	+list(["a", "b", "c"])
		@slot #row(it)
			a[href=("/item/" + it)] {= it }
```
