
# Hello, Gad

The basics: printing, variables and string templates.

Run with:  gad run samples/hello.gad   (or open it in `gad ide`).
See doc/getting-started.md for detailed documentation.

`println` prints its arguments separated by spaces; `#"…"` is an interpolated
string (`{expr}` is replaced by the value). `:=` declares a variable and `=`
reassigns an existing binding.

```gad
name := "Gad"
version := 1

println("Hello,", name)
println(#"Welcome to {name} v{version}!")   // interpolated string

greeting := #"{name} is a fast, embeddable scripting language."
version = 2                                 // reassign
println(greeting, #"(v{version})")
```

Output:

```text
Hello, Gad
Welcome to Gad v1!
Gad is a fast, embeddable scripting language. (v2)
```

## Example — `hello.gad`

```gad
name := "Gad"
version := 1

println("Hello,", name)
println(#"Welcome to {name} v{version}!")   // interpolated string

greeting := #"{name} is a fast, embeddable scripting language."
version = 2                                 // reassign
println(greeting, #"(v{version})")

return greeting
```
