
# Assignment and Append

Assignment lists, and the append operators for arrays and dicts.

## Example — `assign_and_append.gad`

```gad
// --- A comma-separated right-hand side is an array (single target) ---
x := 1, 2, 3
println("x := 1, 2, 3        ->", x)               // [1, 2, 3]

rest := [3, 4]
y := 1, 2, *rest
println("y := 1, 2, *rest    ->", y)               // [1, 2, 3, 4]

// --- Several targets still destructure ---
a, b := 10, 20
println("a, b := 10, 20      ->", [a, b])          // [10, 20]

head, *tail := 1, 2, 3, 4
println("head, *tail         ->", head, tail)      // 1 [2, 3, 4]

// --- Array append: expressions ---
println("[1] + 2             ->", [1] + 2)          // [1, 2]   (append one)
println("[1] + [2, 3]        ->", [1] + [2, 3])     // [1, 2, 3] (+ concatenates)
println("[1] ++ [2, 3]       ->", [1] ++ [2, 3])    // [1, 2, 3] (++ extends)

// --- Array append: in-place assignments ---
acc := []
acc += 1            // append one element
acc ++= [2, 3]      // extend with an iterable
acc ++= 4, 5        // ++= a, b  ==  ++= [a, b]
println("acc                 ->", acc)              // [1, 2, 3, 4, 5]

// --- Dict entries: the same two operators, the same rule ---
// `+` takes one entry, as it takes one element of an array; `++` spreads, so
// its right side has to be a sequence of entries.
d := {a: 1}
d += keyValue("b", 2)   // put one entry
d += {c: 3}             // a dict reads as a sequence of entries, and merges
d ++= {e: 5, f: 6}      // extend with a sequence
println("d                   ->", d)                // {a: 1, b: 2, c: 3, e: 5, f: 6}

// `d ++ keyValue(…)` is an error: `++` always spreads, and one entry is not a
// sequence of them.

return acc == [1, 2, 3, 4, 5] && d == {a: 1, b: 2, c: 3, e: 5, f: 6}
```
