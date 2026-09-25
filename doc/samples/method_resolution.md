
# Method Resolution

How a call chooses a method.

A function may hold several methods (overloads). A call dispatches to the
method whose parameter types match the arguments, Julia-style: every argument
narrows the choice, and the most specific matching method wins. This file walks
through arity, type specificity, subtype hierarchies, catch-all fallbacks,
unions, variadics, `met` extension/override, and structural (`met<…>`) params
that dispatch by value. See doc/method-interfaces.md for detailed documentation.

## Arity and multiple dispatch

The same name resolves by how many arguments are passed, and every parameter's
type participates — not just the first.

```gad
func area {
    (r float)          => 3.14159 * r * r        // circle
    (w float, h float) => w * h                   // rectangle
}
func combine {
    (a int, b int) => "int+int"
    (a int, b str) => "int+str"
    (a str, b str) => "str+str"
}
[area(2.0), area(2.0, 3.0), combine(1, 2), combine(1, "x"), combine("a", "b")]
// => [12.56636, 6, "int+int", "int+str", "str+str"]
```

## Specificity and fallbacks

A subclass argument prefers the method typed for the subclass over the one typed
for its parent. An untyped parameter matches anything and is the fallback,
chosen only when no typed method matches.

```gad
class Animal {}
class Dog { *Animal }
func speak {
    (a Animal) => "some animal"
    (d Dog)    => "woof"               // Dog is more specific
}
func kind {
    (x int) => "integer"
    (x str) => "text"
    (x)     => "other"                 // the fallback
}
[speak(Animal()), speak(Dog()), kind(5), kind("hi"), kind(true)]
// => ["some animal", "woof", "integer", "text", "other"]
```

## Unions and variadics

`int|str` accepts either type in a single method. `*xs int` collects any number
of trailing int arguments; the no-arg method is a distinct, more specific
overload.

```gad
func label(x int|str) => "label:" + str(x)
func sum {
    ()        => 0
    (*xs int) {
        total := 0
        for _, x in xs { total += x }
        return total
    }
}
[label(7), label("q"), sum(), sum(1, 2, 3)]
// => ["label:7", "label:q", 0, 6]
```

## Extending and overriding (`met`, `met ~`, `$old`)

`met` adds a method to an existing callable after the fact; `met ~name`
overrides an existing signature instead of erroring (the last definition wins).
A `$old` first parameter on an override captures the method being replaced, so
the new method can wrap it (super / around advice). It is dropped from the real
signature; `$old` is resolved from the remaining parameter types and is nil when
there was no previous method. `gad.methodFromArgs(fn, …)` — what `$old` uses
under the hood — returns the method a call would dispatch to, chosen by example
value or by type name.

```gad
met area(side int) => side * side          // a new (int) method: a square

func greet(name str) => "hi " + name
pre := greet("a")
met ~greet(name str) => "HELLO " + name   // override

func step(n int) => n * 10
met ~step($old, n int) => $old(n) + 1     // wraps the previous `step`

[area(4), pre, greet("a"), step(3), gad.methodFromArgs(step, int)(4)]
// => [16, "hi a", "HELLO a", 31, 41]
```

## Structural parameter types (dispatch by value)

A `met<…>` (method-interface) parameter type is checked structurally, by value:
the argument must be a callable whose signature satisfies the header. A
function-with-methods is accepted when one of its methods fits; a non-callable —
or a callable with the wrong shape — is rejected at the call.

```gad
func apply(cb met<(int) <int>>, v int) => cb(v)
func poly() => "s"
met poly(n int) => n + 1                  // poly's (int) method fits the header
rejected := func() { try { apply(42, 1); return false } catch { return true } }
[apply(func(n int) => n * n, 6), apply(poly, 10), rejected()]
// => [36, 11, true]
```

## The assign-to-type operator `::`

`obj :: Type` checks assignability by the same rules the dispatcher uses and
returns obj unchanged (else it raises a catchable type error). It uses the type
kinds above — plain types, subclasses and structural `met<…>` — chains
left-to-right (`obj::T1::T2`) and binds tighter than `+`.

```gad
bad := func() { try { "hi" :: int; return false } catch { return true } }
[
    5 :: int,
    5 :: int :: any,                                   // chained
    2 + 3 :: int,                                      // binds tighter than +
    (func(n int) => n) :: met<(int) <int>> != nil,     // a structural fit
    bad(),                                             // a failing cast raises
]
// => [5, 5, 5, true, true]
```

## Example — `method_resolution.gad`

```gad
func area {
    (r float)          => 3.14159 * r * r        // circle
    (w float, h float) => w * h                   // rectangle
}
func combine {
    (a int, b int) => "int+int"
    (a int, b str) => "int+str"
    (a str, b str) => "str+str"
}
[area(2.0), area(2.0, 3.0), combine(1, 2), combine(1, "x"), combine("a", "b")]

class Animal {}
class Dog { *Animal }
func speak {
    (a Animal) => "some animal"
    (d Dog)    => "woof"               // Dog is more specific
}
func kind {
    (x int) => "integer"
    (x str) => "text"
    (x)     => "other"                 // the fallback
}
[speak(Animal()), speak(Dog()), kind(5), kind("hi"), kind(true)]

func label(x int|str) => "label:" + str(x)
func sum {
    ()        => 0
    (*xs int) {
        total := 0
        for _, x in xs { total += x }
        return total
    }
}
[label(7), label("q"), sum(), sum(1, 2, 3)]

met area(side int) => side * side          // a new (int) method: a square

func greet(name str) => "hi " + name
pre := greet("a")
met ~greet(name str) => "HELLO " + name   // override

func step(n int) => n * 10
met ~step($old, n int) => $old(n) + 1     // wraps the previous `step`

[area(4), pre, greet("a"), step(3), gad.methodFromArgs(step, int)(4)]

func apply(cb met<(int) <int>>, v int) => cb(v)
func poly() => "s"
met poly(n int) => n + 1                  // poly's (int) method fits the header
rejected := func() { try { apply(42, 1); return false } catch { return true } }
[apply(func(n int) => n * n, 6), apply(poly, 10), rejected()]

bad := func() { try { "hi" :: int; return false } catch { return true } }
[
    5 :: int,
    5 :: int :: any,                                   // chained
    2 + 3 :: int,                                      // binds tighter than +
    (func(n int) => n) :: met<(int) <int>> != nil,     // a structural fit
    bad(),                                             // a failing cast raises
]
```
