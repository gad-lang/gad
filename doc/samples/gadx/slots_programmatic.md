# slots_programmatic

Passing slots programmatically from gad source.

`@slot #name` / `+super` are sugar: a component compiles to a gad function
that takes a `slots` dict, and each entry is a slot function. You can build
that dict yourself in a `~~ … ~~` code block and call the component directly.

A slot function's first parameter is `super` (the component's default for
that slot); the slot's scope parameters follow. Like a component, a slot
function builds a fragment tag and returns it: create it with `gadx.Tag()`,
append content, and `return` it. The template sugar forwards super for you —
a raw call passes it by hand, so `super(…)` must supply super's own super (an
empty function) as its first argument. Append the component's own result with
`$el += …`.

## Components

### box

```gadx
box(; slots={})
```

### list

```gadx
list(items; slots={})
```

### main

## Example — `slots_programmatic.gadx`

```gadx
@export comp box(; slots={})
	div
		@slot main
			span default

@export comp list(items; slots={})
	ul
		@for i, it in items
			li
				@slot row(i, it)
					| {= i }: {= it }

@main
	// 1) equivalent to: +box { @slot #main { b hi; +super } }
	~ $el += box(; slots={ main(super) {$el := gadx.Tag(); gadx.Text($el, raw "<b>hi</b>"); $el += super(func(*_) {}); return $el} })
	// 2) a scoped slot, ignoring super — receives (super, i, it)
	~ $el += list(["a", "b"]; slots={ row(super, i, it) {$el := gadx.Tag(); gadx.Text($el, ((raw "<b>" + it) + "</b>")); return $el} })
	// 3) a scoped slot forwarding the scope to super: super(empty, i, it)
	~ $el += list(["a", "b"]; slots={ row(super, i, it) {$el := gadx.Tag(); gadx.Text($el, raw "* "); $el += super(func(*_) {}, i, it); return $el} })
```
