
# array

An **array** is an ordered, growable list of values of any types. Its literal
is `[1, "a", [2]]`; a typed array (`type ints []int`) checks its items (see
[typed arrays](../typed_arrays.md)).

## Constructors

`array(*args)` collects its arguments; `array(*it)` an iterable's values.

```gad
[array(1, 2), array(*"ab"), array()]
= [[1, 2], ['a', 'b'], []]
```

## Operators

`+` appends an item (a new array: `a += v` grows `a`), `++` appends the
values of an iterable; `in` tests an item; `==` compares item by item. An
array indexes (`a[0]`, `a[-1]` from the end), slices (`a[1:]`) and is
iterated (`for i, v in a`).

```gad
a := [1, 2]
a += 3
a ++= [4, 5]
[a, 2 in a, a[-1], a[1:3], [1, 2] == [1, 2]]
= [[1, 2, 3, 4, 5], true, 5, [2, 3], true]
```

## See also

- [Collections](../collections.md), [comprehensions](../comprehensions.md),
  [sized arrays](../sized_arrays.md).

## Type

### array

```gad
array type
```

#### Constructors

```gad
array(…)
```

#### Operators

```gad
a + b
a ++ b
a ++= b
a += b
a < b
a <= b
a > b
a >= b
a in b
```

#### Conversions

```gad
bytes(v array)
```

## Example — `array.gad`

```gad
example constructors {
    [array(1, 2), array(*"ab"), array()]
    = [[1, 2], ['a', 'b'], []]
}

example operators {
    a := [1, 2]
    a += 3
    a ++= [4, 5]
    [a, 2 in a, a[-1], a[1:3], [1, 2] == [1, 2]]
    = [[1, 2, 3, 4, 5], true, 5, [2, 3], true]
}
```
