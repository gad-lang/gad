# assignment

Variable assignments in a template body.

`IDENT := EXPR` declares a template-local variable; `IDENT = EXPR` reassigns an
existing one; the compound forms `+=`, `-=`, `*=`, `/=`, `%=`, `??=` update it in
place. No `~` prefix is needed.

`IDENT` is a plain Gad identifier — letters, digits and `_`, with an optional
leading `$`, and **no `-`**. That is what tells an assignment apart from a tag: a
tag never has a bare `=` at that spot, and a name containing `-` stays a tag
(`my-el` is an element, not an assignment).

Like a `~` code line, the value may span several lines while its brackets
(`()` / `[]` / `{}`) are unbalanced — a multi-line array, dict, call or
comprehension.

## Components

### main

## Example — `assignment.gadx`

```gadx
@main
	// `:=` declares a local
	title := "Cart"

	// the value may span several lines (unbalanced brackets continue it)
	prices := [
		10,
		25,
		5]

	// a comprehension value, one clause per line
	labels := [
		"$" + str(p)
		for p in prices]

	// `+=` updates in place; `=` reassigns an existing local
	total := 0
	@for p in prices
		total += p
	discount := 0
	discount = total / 10

	// `$`-prefixed names are valid identifiers too
	$note := "prices in USD"

	h1 {= title }
	ul
		@for label in labels
			li {= label }
	p.total Total: {= str(total) }
	p.discount Discount: {= str(discount) }
	small {= $note }
```
