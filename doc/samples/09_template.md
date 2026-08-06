# 09_template

this program is a gad template to generate JSON output of users list:

    {
      "users": [
        "joe",
        "mary",
        "george"
      ]
    }
See doc/templates.md for detailed documentation.

## Example — `09_template.gad`

```gad
# gad: mixed
{%
var (
  users = [    "joe", "mary", "george"   ]
  lastIndex = len(users)-1
)
--%}
{
  "users": [
{%--  for i,      name in    users begin %}
    "{%=name    %}"{%= i < lastIndex ? "," : "" %}
{%-- end %}
  ]
}
```
