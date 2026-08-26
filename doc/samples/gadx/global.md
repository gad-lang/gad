# global

`@global` — ambient values the template reads.

`@global` is the Gadx form of Gad's `global` declaration and accepts exactly the
same forms (it delegates to the Gad parser). Globals are supplied by the host
(the `globals` dict passed to the render) rather than as call arguments.

Forms (all Gad-valid):

- `@global name`              — a single global.
- `@global a b c`             — the legacy space-separated form (normalized to
                                commas), same as `@global (a, b, c)`.
- `@global (a, b, c)`         — several globals.
- `@global (a, b; name = v)`  — positionals first, then a named section (after
                                `;`) whose entries may carry a default. A default
                                also works without the explicit `;` when it is
                                the last entry: `@global (a, b, name = v)`.

Default operators (as in Gad):

- `name = v`    — apply the default when the global is nil OR absent.
- `name !?= v`  — apply only when the global is absent (present-but-nil is kept).

Order rule (as in Gad): entries WITH a default belong to the named section, so
they come AFTER the plain positional names — `@global (url, active; title = v)`,
not `@global (title = v, url, active)`.

## Components

### main

## Example — `global.gadx`

```gadx
// a single global
@global (siteName)

// several globals, with a defaulted one in the named section
@global (url, active; title="Untitled")

// an absent-only default (`!?=`): kept only when `theme` is not provided at all
@global (lang; theme="light")

@main
	header[class=theme]
		h1 {= title }
		nav[class="menu", data-active=active]
			a[href=url] {= siteName }
		p {= lang }
```
