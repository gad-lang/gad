# Destructuring

Currently, Gad supports only destructuring array assignments to handle multi
variable assignments.
Following examples will help you to understand how it works in Gad.

**Important Warnings**

- `:=` define operator can cause variable shadowing for local and global
variables.
- Destructuring has a cost, single assignments are faster.

```go
f := func() {
    // ...
    return 0, nil, error("message")
}

x, y, z := f() // x == 0    y == nil   z == error("message")
```

The example above is similar to the code below with boilerplate code and it
shows how it works under the hood. Note that a hidden builtin function call is
omitted for brevity.

```go
f := func() {
    // ...
    return [0, nil, error("message")]
}

temp := f()
x := temp[0]
y := temp[1]
z := temp[2]
temp = nil
```

Some examples:

```go
x, y := [1, 2]  // x == 1   y == 2

x, y, z := [1, 2]  // x == 1   y == 2   z == nil

x = [1, 2]  // x == [1, 2]  normal assignment :)
```

```go
x, y := func() { return 1, 2 }()    // x == 1   y == 2

// This throws compiler error because x and y are already defined above
x, y := []

// But if left hand side has a new variable, compiler does not throw an error
x, y, err := func() { return 1, 2, 3 }()
```

```go
x := {}
// This throws a compiler error because if a selector is used on the left hand side,
// := define operator cannot be used. Instead, use = assignment operator and
// declare variables before assignment.
x.y, z := [1, 2]
```

```go
x := {}
var z
x.y, z = [1, 2] // x == {"y": 1}    z == 2
```

```go
// If the array holds less elements than left hand side variables,
// nil value is set to those which don't have corresponding array element.
x, y := [1] // x == 1   y == nil
```

```go
// Right hand side of the assignment can be of any type but only arrays
// let assignment of array elements to corresponding variables.
var (x, y)
x, y = 1    //  x == 1  y == nil
```

To take the advantage of destructuring arrays, an array must be returned from
exported Go functions.

```go
script := `
global goFunc

// ...

v, err := goFunc(2)
if err != nil {
    // ...
}
// ...
`

bytecode, err := gad.Compile([]byte(script), gad.DefaultCompilerOptions)
if err != nil {
    log.Fatal(err)
}

g := gad.Map{
    "goFunc": &gad.Function{
        Value: func(args ...gad.Object) (gad.Object, error) {
            // ...
            return gad.Array{gad.Nil, gad.ErrIndexOutOfBounds}, nil
        },
    },
}

ret, err := gad.NewVM(bytecode).Run(g)
// ...
```
