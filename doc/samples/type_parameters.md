
# Type Parameters (generic functions)

A function signature may declare **type parameters** between the name (or `func`)
and the parameter list: `func mySet[T indexable, K int|uint, V number](…)`. Each
entry is `NAME CONSTRAINT`, where the constraint is written exactly like a
parameter type — a single type, an interface, or a `|` union —; a name alone
(`[T]`) is constrained by nothing (`any`).

Inside the function a type parameter is a **type**: it types the parameters and
the return (`xs []T`, `<T>`), converts (`T(v)`) and builds types
(`DictType[K, V]`). What it holds is decided at each call:

- **bound** — `f[int](…)`: the type arguments, in order, bind the type
  parameters; `_` leaves one to its constraint (`pair[_, int]`). A type argument
  must satisfy its constraint (`f[str]` of `[T number]` is an error). Indexing
  a generic function with types gives its instance, a value of its own
  (`g := first[int]`).
- **inferred** — `f(…)`: without type arguments, a type parameter is bound to
  the type of the first argument whose parameter is it (`v T`: its type; `xs
  []T`: the type of the items, when they all have one) — a call is checked as
  it would be bound (`same(1, "a")` of `same[T](a T, b T)` is an error).
- **its constraint** — when nothing binds it: the call is checked against the
  constraint, as a parameter typed by it would be.

A type argument may be a union (`f[int|float](…)`): `|` between two types is
their union.

## Where they work

Type parameters are accepted in every function-signature form:

- **named** — `func mySet[T indexable](target T) <T> { … }`, and the shorthand
  `name[T](v T) => …` (as `export name[T](…)`)
- **anonymous** — `const f = func[T number](v T) => v` (a space is optional:
  `func [T number]`), and arrows: `[T](x T) => x`
- **dict method shorthand** — `{ scale[V number](v V, n int) <V>: v * n }`
- **a function with methods** — `func f { [T](x T) => …; (x, y) => … }`, each
  method its own type parameters (`f[int]` binds a function of one generic
  method)
- **class methods and properties** — `methods { conv[R](f) => … }`,
  `props { as[R]() => … }`: `inst.conv[str](…)` binds the method,
  `inst.as[str]` gets the property bound
- **func-header value** — `<[T number](v T) <T>>`
- **method interface** — `meti { [T number](v T) <T> }`

A class, an interface and a mixin declare type parameters too — each with its
alias, the type it is when none is given: see [generics](generics.gad).

```gad
// T is indexable, K is int|uint, V is a number; the return type reuses T.
func mySet[T indexable, K int|uint, V number](target T, k K, v V) <T> {
    target[k] = v
    return target
}
mySet([10, 20, 30], 1, 99)
= [10, 99, 30]
```

## Reusing one constraint across parameters

Naming the constraint once keeps related parameters in sync — here both operands
and the result share `T`:

```gad
// both arguments and the result share the single constraint T (str or array).
concat := func[T str|array](a T, b T) <T> => a + b
[concat("ab", "cd"), concat([1, 2], [3])]
= ["abcd", [1, 2, 3]]
```

## Binding, `_` and inference

```gad
errorOf := func(f) { try { f(); return "" } catch e { return str(e) } }
func first[T](xs []T) <T> => xs[0]
first[int]([1, 2])                    // T bound: xs is checked as []int
= 1
first(["a"])                          // T inferred: str
= "a"
errorOf(() => first[int](["a"]))
= "TypeError: invalid type for argument '1st (xs)': expected []T, found array"
g := first[str]                       // an instance: a value
g(["x"])
= "x"

conv := func[T](v) => T(v)            // T converts
[conv[str](12), conv(12)]             // unbound: any(12)
= ["12", 12]

pair := func[K, V](k K, v V) => [k, v]
pair[_, int]("a", 1)                  // `_`: K left to its constraint
= ["a", 1]

same := func[T](a T, b T) => [a, b]
errorOf(() => same(1, "a"))   // T inferred int
= "TypeError: invalid type for argument '2st (b)': expected int, found str"

func twice[T number](v T) => v * 2
errorOf(() => twice[str])     // str does not satisfy number
= "TypeError: twice: type argument T: str does not satisfy number"
```

## Unions

```gad
errorOf := func(f) { try { f(); return "" } catch e { return str(e) } }
func num[T int|float](v T) => v
[num(2), num[int|float](2.5)]
= [2, 2.5]
errorOf(() => num("a"))
= "ErrCall: ‹compiledFunction: (main).num(v T)›; caused by: ‹TypeError: invalid type for argument '1st (v)': expected int|float, found str›"
id := func[T](v T) => v
id[int|str]("a")                      // a union as the type argument
= "a"
int|float
= int|float
```

## Anonymous functions, arrows and dict methods

```gad
double := func[T number](v T) <T> => v * 2       // anonymous, `func[…]`
d := { scale[V number](v V, n int) <V>: v * n }  // dict method shorthand
[double(21), d.scale(5, 3)]
= [42, 15]
```

## Functions with methods, class methods and properties

```gad
func fm { [T](x T) => T(x); (x, y) => x + y }
[fm(1, 2), fm(5)]
= [3, 5]

class Box {
    v = 0
    methods {
        conv[R](f) => R(f(this.v))
    }
    props {
        as[R]() => R(this.v)
    }
}
b := Box(; v=3)
[b.conv((x) => x * 2), b.conv[str]((x) => x * 2)]
= [6, "6"]
[b.as, b.as[str]]
= [3, "3"]
m := b.conv                           // a method, bound to its instance
m((x) => x + 1)
= 4
```

## Methods of the generic types

`met DictType[K, V](…)` is a method of every dict type — of `DictType[str,
int]` and of each typed dict type —, its type parameters bound to the dict
type's: a call of the dict type that fits it (its arguments, their types) is
the method's, what it gives the dict's entries. `met ~NAME[_, V](…)` is one of
the typed dict type NAME alone, tried before the generic's. The same holds for
`KeyValueArrayType`. Written alone as a parameter type, `DictType` takes any
dict (typed or not) — and any dict type —, `KeyValueArrayType` any key-value
array.

```gad
met DictType[K, V](arr []V) {         // every dict type: [10, 20] the values
    d := {}
    for i, v in arr { d[str(i * 10)] = v }
    return d
}
DictType[str, int]([1, 2])
= {"0": 1, "10": 2}
type scores dict[str, int]
met ~scores[_, V](arr []V) {          // scores alone
    d := {}
    for i, v in arr { d["s" + str(i)] = v }
    return d
}
scores([5]).s0
= 5
func size(d DictType) => len(d)       // any dict
[size({a: 1}), size(scores(; a=1, b=2))]
= [1, 2]
```

## Iterables and iterators

`iterable[K, V]` and `iterator[K, V]` are the iterables and the iterators of
keys K and values V — the types named, not checked: an iteration is lazy. The
builtins are documented with them: `map[K, V, R](it iterable[K, V], callback
<(v V, k K) <R>>; …) <iterator[K, R]>`.

```gad
keysOf := func[K, V](it iterable[K, V]) <iterator[K, K]> => keys(it)
collect(keysOf({a: 1}))
= ["a"]
iterable[str, int]
= iterable[str, int]
```

## Headers and method interfaces

A generic `meti` describes a generic method; a generic function satisfies it:

```gad
m := meti { [T number](v T) <T> }                // a generic method interface
f := func[T number](v T) <T> => v                // a generic function
implements(f, m)
= true
```

## Example — `type_parameters.gad`

```gad
example basics {
    // T is indexable, K is int|uint, V is a number; the return type reuses T.
    func mySet[T indexable, K int|uint, V number](target T, k K, v V) <T> {
        target[k] = v
        return target
    }
    mySet([10, 20, 30], 1, 99)
    = [10, 99, 30]
}

example reuse {
    // both arguments and the result share the single constraint T (str or array).
    concat := func[T str|array](a T, b T) <T> => a + b
    [concat("ab", "cd"), concat([1, 2], [3])]
    = ["abcd", [1, 2, 3]]
}

example anon {
    double := func[T number](v T) <T> => v * 2       // anonymous, `func[…]`
    d := { scale[V number](v V, n int) <V>: v * n }  // dict method shorthand
    [double(21), d.scale(5, 3)]
    = [42, 15]
}

example meti {
    m := meti { [T number](v T) <T> }                // a generic method interface
    f := func[T number](v T) <T> => v                // a generic function
    implements(f, m)
    = true
}

example binding {
    errorOf := func(f) { try { f(); return "" } catch e { return str(e) } }
    func first[T](xs []T) <T> => xs[0]
    first[int]([1, 2])                    // T bound: xs is checked as []int
    = 1
    first(["a"])                          // T inferred: str
    = "a"
    errorOf(() => first[int](["a"]))
    = "TypeError: invalid type for argument '1st (xs)': expected []T, found array"
    g := first[str]                       // an instance: a value
    g(["x"])
    = "x"

    conv := func[T](v) => T(v)            // T converts
    [conv[str](12), conv(12)]             // unbound: any(12)
    = ["12", 12]

    pair := func[K, V](k K, v V) => [k, v]
    pair[_, int]("a", 1)                  // `_`: K left to its constraint
    = ["a", 1]

    same := func[T](a T, b T) => [a, b]
    errorOf(() => same(1, "a"))   // T inferred int
    = "TypeError: invalid type for argument '2st (b)': expected int, found str"

    func twice[T number](v T) => v * 2
    errorOf(() => twice[str])     // str does not satisfy number
    = "TypeError: twice: type argument T: str does not satisfy number"
}

example unions {
    errorOf := func(f) { try { f(); return "" } catch e { return str(e) } }
    func num[T int|float](v T) => v
    [num(2), num[int|float](2.5)]
    = [2, 2.5]
    errorOf(() => num("a"))
    = "ErrCall: ‹compiledFunction: (main).num(v T)›; caused by: ‹TypeError: invalid type for argument '1st (v)': expected int|float, found str›"
    id := func[T](v T) => v
    id[int|str]("a")                      // a union as the type argument
    = "a"
    int|float
    = int|float
}

example methods {
    func fm { [T](x T) => T(x); (x, y) => x + y }
    [fm(1, 2), fm(5)]
    = [3, 5]

    class Box {
        v = 0
        methods {
            conv[R](f) => R(f(this.v))
        }
        props {
            as[R]() => R(this.v)
        }
    }
    b := Box(; v=3)
    [b.conv((x) => x * 2), b.conv[str]((x) => x * 2)]
    = [6, "6"]
    [b.as, b.as[str]]
    = [3, "3"]
    m := b.conv                           // a method, bound to its instance
    m((x) => x + 1)
    = 4
}

example generic_types {
    met DictType[K, V](arr []V) {         // every dict type: [10, 20] the values
        d := {}
        for i, v in arr { d[str(i * 10)] = v }
        return d
    }
    DictType[str, int]([1, 2])
    = {"0": 1, "10": 2}
    type scores dict[str, int]
    met ~scores[_, V](arr []V) {          // scores alone
        d := {}
        for i, v in arr { d["s" + str(i)] = v }
        return d
    }
    scores([5]).s0
    = 5
    func size(d DictType) => len(d)       // any dict
    [size({a: 1}), size(scores(; a=1, b=2))]
    = [1, 2]
}

example iterables {
    keysOf := func[K, V](it iterable[K, V]) <iterator[K, K]> => keys(it)
    collect(keysOf({a: 1}))
    = ["a"]
    iterable[str, int]
    = iterable[str, int]
}

return "type parameters"
```
