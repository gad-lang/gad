# include_data

include_data.gad — plain Gad data for a template to `@include`.

It is ordinary Gad (not gadx): `@include` compiles it inline into the template,
so these bindings are visible to the markup below the include.

## Example — `include_data.gad`

```gad
pageTitle := "Release notes"
items := ["Faster VM", "New `include`", "Docs"]
```
