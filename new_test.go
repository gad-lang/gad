// Put relatively new features' tests in this test file.

package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

func TestVMDestructuring(t *testing.T) {
	expectErrHas(t, `x, y = nil; return x`,
		NewTestOpts().CompilerError(), `Compile Error: unresolved reference "x"`)
	expectErrHas(t, `var (x, y); x, y := nil; return x`,
		NewTestOpts().CompilerError(), `Compile Error: no new variable on left side`)
	expectErrHas(t, `x, y = 1, 2`, NewTestOpts().CompilerError(),
		`Compile Error: multiple expressions on the right side not supported`)

	TestExpectRun(t, `x, y := nil; return x`, nil, Nil)
	TestExpectRun(t, `x, y := nil; return y`, nil, Nil)
	TestExpectRun(t, `x, y := 1; return x`, nil, Int(1))
	TestExpectRun(t, `x, y := 1; return y`, nil, Nil)
	TestExpectRun(t, `x, y := []; return x`, nil, Nil)
	TestExpectRun(t, `x, y := []; return y`, nil, Nil)
	TestExpectRun(t, `x, y := [1]; return x`, nil, Int(1))
	TestExpectRun(t, `x, y := [1]; return y`, nil, Nil)
	TestExpectRun(t, `x, y := [1, 2]; return x`, nil, Int(1))
	TestExpectRun(t, `x, y := [1, 2]; return y`, nil, Int(2))
	TestExpectRun(t, `x, y := [1, 2, 3]; return x`, nil, Int(1))
	TestExpectRun(t, `x, y := [1, 2, 3]; return y`, nil, Int(2))
	TestExpectRun(t, `var x; x, y := [1]; return x`, nil, Int(1))
	TestExpectRun(t, `var x; x, y := [1]; return y`, nil, Nil)

	TestExpectRun(t, `x, y, z := nil; return x`, nil, Nil)
	TestExpectRun(t, `x, y, z := nil; return y`, nil, Nil)
	TestExpectRun(t, `x, y, z := nil; return z`, nil, Nil)
	TestExpectRun(t, `x, y, z := 1; return x`, nil, Int(1))
	TestExpectRun(t, `x, y, z := 1; return y`, nil, Nil)
	TestExpectRun(t, `x, y, z := 1; return z`, nil, Nil)
	TestExpectRun(t, `x, y, z := []; return x`, nil, Nil)
	TestExpectRun(t, `x, y, z := []; return y`, nil, Nil)
	TestExpectRun(t, `x, y, z := []; return z`, nil, Nil)
	TestExpectRun(t, `x, y, z := [1]; return x`, nil, Int(1))
	TestExpectRun(t, `x, y, z := [1]; return y`, nil, Nil)
	TestExpectRun(t, `x, y, z := [1]; return z`, nil, Nil)
	TestExpectRun(t, `x, y, z := [1, 2]; return x`, nil, Int(1))
	TestExpectRun(t, `x, y, z := [1, 2]; return y`, nil, Int(2))
	TestExpectRun(t, `x, y, z := [1, 2]; return z`, nil, Nil)
	TestExpectRun(t, `x, y, z := [1, 2, 3]; return x`, nil, Int(1))
	TestExpectRun(t, `x, y, z := [1, 2, 3]; return y`, nil, Int(2))
	TestExpectRun(t, `x, y, z := [1, 2, 3]; return z`, nil, Int(3))
	TestExpectRun(t, `x, y, z := [1, 2, 3, 4]; return z`, nil, Int(3))

	// test index assignments
	TestExpectRun(t, `
	var (x = {}, y, z)
	x.a, y, z = [1, 2, 3, 4]; return x`, nil, Dict{"a": Int(1)})
	TestExpectRun(t, `
	var (x = {}, y, z)
	x.a, y, z = [1, 2, 3, 4]; return y`, nil, Int(2))
	TestExpectRun(t, `
	var (x = {}, y, z)
	x.a, y, z = [1, 2, 3, 4]; return z`, nil, Int(3))

	TestExpectRun(t, `
	var (x = {}, y, z)
	y, x.a, z = [1, 2, 3, 4]; return x`, nil, Dict{"a": Int(2)})
	TestExpectRun(t, `
	var (x = {}, y, z)
	y, x.a, z = [1, 2, 3, 4]; return y`, nil, Int(1))
	TestExpectRun(t, `
	var (x = {}, y, z)
	y, x.a, z = [1, 2, 3, 4]; return z`, nil, Int(3))

	TestExpectRun(t, `
	var (x = [0], y, z)
	x[0], y, z = [1, 2, 3, 4]; return x`, nil, Array{Int(1)})
	TestExpectRun(t, `
	var (x = [0], y, z)
	x[0], y, z = [1, 2, 3, 4]; return y`, nil, Int(2))
	TestExpectRun(t, `
	var (x = [0], y, z)
	x[0], y, z = [1, 2, 3, 4]; return z`, nil, Int(3))

	TestExpectRun(t, `
	var (x = [0], y, z)
	y, x[0], z = [1, 2, 3, 4]; return x`, nil, Array{Int(2)})
	TestExpectRun(t, `
	var (x = [0], y, z)
	y, x[0], z = [1, 2, 3, 4]; return y`, nil, Int(1))
	TestExpectRun(t, `
	var (x = [0], y, z)
	y, x[0], z = [1, 2, 3, 4]; return z`, nil, Int(3))

	// test function calls
	TestExpectRun(t, `
	fn := func() { 
		return [1, error("abc")]
	}
	x, y := fn()
	return [x, str(y)]`, nil, Array{Int(1), Str("error: abc")})

	TestExpectRun(t, `
	fn := func() { 
		return [1]
	}
	x, y := fn()
	return [x, y]`, nil, Array{Int(1), Nil})
	TestExpectRun(t, `
	fn := func() { 
		return
	}
	x, y := fn()
	return [x, y]`, nil, Array{Nil, Nil})
	TestExpectRun(t, `
	fn := func() { 
		return [1, 2, 3]
	}
	x, y := fn()
	t := {a: x}
	return [x, y, t]`, nil, Array{Int(1), Int(2), Dict{"a": Int(1)}})
	TestExpectRun(t, `
	fn := func() { 
		return {}
	}
	x, y := fn()
	return [x, y]`, nil, Array{Dict{}, Nil})
	TestExpectRun(t, `
	fn := func(v) { 
		return [1, v, 3]
	}
	var x = 10
	x, y := fn(x)
	t := {a: x}
	return [x, y, t]`, nil, Array{Int(1), Int(10), Dict{"a": Int(1)}})

	// test any expression
	TestExpectRun(t, `x, y :=  {}; return [x, y]`, nil, Array{Dict{}, Nil})
	TestExpectRun(t, `
	var x = 2
	if x > 0 {
		fn := func(v) { 
			return [3*v, 4*v]
		}
		var y
		x, y = fn(x)
		if y != 8 {
			throw sprintf("y value expected: %s, got: %s", 8, y)
		}
	}
	return x
	`, nil, Int(6))
	TestExpectRun(t, `
	var x = 2
	if x > 0 {
		fn := func(v) { 
			return [3*v, 4*v]
		}
		// new x symbol is created within if block
		// x in upper block is not affected
		x, y := fn(x)
		if y != 8 {
			throw sprintf("y value expected: %s, got: %s", 8, y)
		}
	}
	return x
	`, nil, Int(2))

	TestExpectRun(t, `
	var x = 2
	if x > 0 {
		fn := func(v) {
			try {
				ret := v/2
			} catch err {
				return [0, err]
			} finally {
				if err == nil {
					return ret
				}
			}
		}
		a, err := fn("str")
		if !isError(err) {
			throw err
		}
		if a != 0 {
			throw "a is not 0"
		}
		a, err = fn(6)
		if err != nil {
			throw sprintf("unexpected error: %s", err)
		}
		if a != 3 {
			throw "a is not 3"
		}
		x = a
	}
	// return map to check stack pointer is correct
	return {x: x}
	`, nil, Dict{"x": Int(3)})
	TestExpectRun(t, `
	for x,y := [1, 2]; true; x++ {
		if x == 10 {
			return [x, y]
		}
	}
	`, nil, Array{Int(10), Int(2)})
	TestExpectRun(t, `
	if x,y := [1, 2]; true {
		return [x, y]
	}
	`, nil, Array{Int(1), Int(2)})
	TestExpectRun(t, `
	var x = 0
	for true {
		x, y := [x]
		x++
		break
	}
	return x`, nil, Int(0))
	TestExpectRun(t, `
	x, y := func(n) {
		return repeat([n], n)
	}(3)
	return [x, y]`, nil, Array{Int(3), Int(3)})
	// closures
	TestExpectRun(t, `
	var x = 10
	a, b := func(n) {
		x = n
	}(3)
	return [x, a, b]`, nil, Array{Int(3), Nil, Nil})
	TestExpectRun(t, `
	var x = 10
	a, b := func(*args) {
		x, y := args
		return [x, y]
	}(1, 2)
	return [x, a, b]`, nil, Array{Int(10), Int(1), Int(2)})
	TestExpectRun(t, `
	var x = 10
	a, b := func(*args) {
		var y
		x, y = args
		return [x, y]
	}(1, 2)
	return [x, a, b]`, nil, Array{Int(1), Int(1), Int(2)})

	// return implicit array if return statement's expressions are comma
	// separated which is a part of destructuring implementation to mimic multi
	// return values.
	parseErr := `Parse Error: expected operand, found 'EOF'`
	expectErrHas(t, `return 1,`,
		NewTestOpts().CompilerError(), parseErr)
	expectErrHas(t, `return 1, 2,`,
		NewTestOpts().CompilerError(), parseErr)
	expectErrHas(t, `var a; return a,`,
		NewTestOpts().CompilerError(), parseErr)
	expectErrHas(t, `var (a, b); return a, b,`,
		NewTestOpts().CompilerError(), parseErr)
	expectErrHas(t, `return 1,`,
		NewTestOpts().CompilerError(), parseErr)
	expectErrHas(t, `return 1, 2,`,
		NewTestOpts().CompilerError(), parseErr)
	expectErrHas(t, `var a; return a,`,
		NewTestOpts().CompilerError(), parseErr)
	expectErrHas(t, `var (a, b); return a, b,`,
		NewTestOpts().CompilerError(), parseErr)

	parseErr = `Parse Error: expected operand, found '}'`
	expectErrHas(t, `func(){ return 1, }`,
		NewTestOpts().CompilerError(), parseErr)
	expectErrHas(t, `func(){ return 1, 2,}`,
		NewTestOpts().CompilerError(), parseErr)

	expectErrHas(t, `func(){ var a; return a,}`,
		NewTestOpts().CompilerError(), parseErr)
	expectErrHas(t, `func(){ var (a, b); return a, b,}`,
		NewTestOpts().CompilerError(), parseErr)
	expectErrHas(t, `func(){ return 1,}`,
		NewTestOpts().CompilerError(), parseErr)
	expectErrHas(t, `func(){ return 1, 2,}`,
		NewTestOpts().CompilerError(), parseErr)
	expectErrHas(t, `func(){ var a; return a,}`,
		NewTestOpts().CompilerError(), parseErr)
	expectErrHas(t, `func(){ var (a, b); return a, b,}`,
		NewTestOpts().CompilerError(), parseErr)

	TestExpectRun(t, `return 1, 2`, nil, Array{Int(1), Int(2)})
	TestExpectRun(t, `a := 1; return a, a`, nil, Array{Int(1), Int(1)})
	TestExpectRun(t, `a := 1; return a, 2`, nil, Array{Int(1), Int(2)})
	TestExpectRun(t, `a := 1; return 2, a`, nil, Array{Int(2), Int(1)})
	TestExpectRun(t, `a := 1; return 2, a, [3]`, nil,
		Array{Int(2), Int(1), Array{Int(3)}})
	TestExpectRun(t, `a := 1; return [2, a], [3]`, nil,
		Array{Array{Int(2), Int(1)}, Array{Int(3)}})
	TestExpectRun(t, `return {}, []`, nil, Array{Dict{}, Array{}})
	TestExpectRun(t, `return func(){ return 1}(), []`, nil, Array{Int(1), Array{}})
	TestExpectRun(t, `return func(){ return 1}(), [2]`, nil,
		Array{Int(1), Array{Int(2)}})
	TestExpectRun(t, `
	f := func() {
		return 1, 2
	}
	a, b := f()
	return a, b`, nil, Array{Int(1), Int(2)})
	TestExpectRun(t, `
	a, b := func() {
		return 1, error("x")
	}()
	return a, "" + b`, nil, Array{Int(1), Str("error: x")})
	TestExpectRun(t, `
	a, b := func(a, b) {
		return a + 1, b + 1
	}(1, 2)
	return a, b, a*2, 3/b`, nil, Array{Int(2), Int(3), Int(4), Int(1)})
	TestExpectRun(t, `
	return func(a, b) {
		return a + 1, b + 1
	}(1, 2), 4`, nil, Array{Array{Int(2), Int(3)}, Int(4)})

	TestExpectRun(t, `
	param *args

	mapEach := func(seq, fn) {
	
		if !isArray(seq) {
			return error("want array, got " + typeName(seq))
		}
	
		var out = []
	
		if sz := len(seq); sz > 0 {
			out = repeat([0], sz)
		} else {
			return out
		}
	
		try {
			for i, v in seq {
				out[i] = fn(v)
			}
		} catch err {
			println(err)
		} finally {
			return out, err
		}
	}
	
	global multiplier
	
	v, err := mapEach(args, func(x) { return x*multiplier })
	if err != nil {
		return err
	}
	return v
	`, NewTestOpts().
		Globals(Dict{"multiplier": Int(2)}).
		Args(Int(1), Int(2), Int(3), Int(4)),
		Array{Int(2), Int(4), Int(6), Int(8)})

	TestExpectRun(t, `
	global goFunc
	// ...
	v, err := goFunc(2)
	if err != nil {
		return str(err)
	}
	`, NewTestOpts().
		Globals(Dict{"goFunc": &Function{
			Value: func(Call) (Object, error) {
				// ...
				return Array{
					Nil,
					ErrIndexOutOfBounds.NewError("message"),
				}, nil
			},
		}}),
		Str("IndexOutOfBoundsError: message"))
}

func TestVMConst(t *testing.T) {
	expectErrHas(t, `const x = 1; x = 2`, NewTestOpts().CompilerError(),
		`Compile Error: assignment to constant variable "x"`)
	expectErrHas(t, `const x = 1; x := 2`, NewTestOpts().CompilerError(),
		`Compile Error: "x" redeclared in this block`)
	expectErrHas(t, `const (x = 1, x = 2)`, NewTestOpts().CompilerError(),
		`Compile Error: "x" redeclared in this block`)
	expectErrHas(t, `const x`, NewTestOpts().CompilerError(),
		`Parse Error: missing initializer in const declaration`)
	expectErrHas(t, `const (x, y = 2)`, NewTestOpts().CompilerError(),
		`Parse Error: missing initializer in const declaration`)

	// After iota support, `const (x=1,y)` does not throw error, like ToInterface. It
	// uses last expression as initializer.
	TestExpectRun(t, `const (x = 1, y)`, nil, Nil)

	expectErrHas(t, `const (x, y)`, NewTestOpts().CompilerError(),
		`Parse Error: missing initializer in const declaration`)
	expectErrHas(t, `
	const x = 1
	func() {
		x = 2
	}`, NewTestOpts().CompilerError(),
		`Compile Error: assignment to constant variable "x"`)
	expectErrHas(t, `
	const x = 1
	if x > 0 {
		x = 2
	}`, NewTestOpts().CompilerError(),
		`Compile Error: assignment to constant variable "x"`)
	expectErrHas(t, `
	const x = 1
	if x > 0 {
		return func() {
			x = 2
		}
	}`, NewTestOpts().CompilerError(),
		`Compile Error: assignment to constant variable "x"`)
	expectErrHas(t, `
	const x = 1
	if x = 2; x > 0 {
		return
	}`, NewTestOpts().CompilerError(),
		`Compile Error: assignment to constant variable "x"`)
	expectErrHas(t, `
	const x = 1
	for x = 1; x < 10; x++ {
		return
	}`, NewTestOpts().CompilerError(),
		`Compile Error: assignment to constant variable "x"`)
	expectErrHas(t, `
	const x = 1
	func() {
		var y
		x, y = [1, 2]
	}`, NewTestOpts().CompilerError(),
		`Compile Error: assignment to constant variable "x"`)
	expectErrHas(t, `
	x := 1
	func() {
		const y = 2
		x, y = [1, 2]
	}`, NewTestOpts().CompilerError(),
		`Compile Error: assignment to constant variable "y"`)
	expectErrHas(t, `const x = 1;global x`, NewTestOpts().CompilerError(),
		`Compile Error: "x" redeclared in this block`)
	expectErrHas(t, `const x = 1;param x`, NewTestOpts().CompilerError(),
		`Compile Error: "x" redeclared in this block`)
	expectErrHas(t, `global x; const x = 1`, NewTestOpts().CompilerError(),
		`Compile Error: "x" redeclared in this block`)
	expectErrHas(t, `param x; const x = 1`, NewTestOpts().CompilerError(),
		`Compile Error: "x" redeclared in this block`)
	expectErrHas(t, `
	const x = 1
	if [2] { // not optimized
		x = 2
	}`, NewTestOpts().CompilerError(),
		`Compile Error: assignment to constant variable "x"`)
	expectErrHas(t, `
	const x = 1
	if [2] { // not optimized
		func() {
			x = 2
		}
	}`, NewTestOpts().CompilerError(),
		`Compile Error: assignment to constant variable "x"`)
	expectErrHas(t, `
	const x = 1
	if x {
		func() {
			func() {
				for {
					x = 2
				}
			}
		}
	}`, NewTestOpts().CompilerError(),
		`Compile Error: assignment to constant variable "x"`)

	// FIXME: Compiler does not compile if or else blocks if condition is
	// a *BoolLit (which may be reduced by optimizer). So compiler does not
	// check whether a constant is reassigned in block to throw an error.
	// A few examples for this issue.
	TestExpectRun(t, `
	const x = 1
	if true {
		
	} else {
		// block is not compiled
		x = 2
	}
	return x
	`, nil, Int(1))
	TestExpectRun(t, `
	const x = 1
	if false {
		// block is not compiled
		x = 2
	}
	return x
	`, nil, Int(1))

	TestExpectRun(t, `const x = 1; return x`, nil, Int(1))
	TestExpectRun(t, `const x = "1"; return x`, nil, Str("1"))
	TestExpectRun(t, `const x = []; return x`, nil, Array{})
	TestExpectRun(t, `const x = []; return x`, nil, Array{})
	TestExpectRun(t, `const x = nil; return x`, nil, Nil)
	TestExpectRun(t, `const (x = 1, y = "2"); return x, y`, nil,
		Array{Int(1), Str("2")})
	TestExpectRun(t, `
	const (
		x = 1
		y = "2"
	)
	return x, y`, nil, Array{Int(1), Str("2")})
	TestExpectRun(t, `
	const (
		x = 1
		y = x + 1
	)
	return x, y`, nil, Array{Int(1), Int(2)})
	TestExpectRun(t, `
	const x = 1
	return func() {
		const x = x + 1
		return x
	}()`, nil, Int(2))
	TestExpectRun(t, `
	const x = 1
	return func() {
		x := x + 1
		return x
	}()`, nil, Int(2))
	TestExpectRun(t, `
	const x = 1
	return func() {
		return func() {
			return x + 1
		}()
	}()`, nil, Int(2))
	TestExpectRun(t, `
	const x = 1
	for x := 10; x < 100; x++{
		return x
	}`, nil, Int(10))
	TestExpectRun(t, `
	const (i = 1, v = 2)
	for i,v in [10] {
		v = -1
		return i
	}`, nil, Int(0))
	TestExpectRun(t, `
	const x = 1
	return func() {
		const y = 2
		const x = y
		return x
	}() + x
	`, nil, Int(3))
	TestExpectRun(t, `
	const x = 1
	return func() {
		const y = 2
		var x = y
		return x
	}() + x
	`, nil, Int(3))
	TestExpectRun(t, `
	const x = 1
	func() {
		x, y := [2, 3]
	}()
	return x
	`, nil, Int(1))
	TestExpectRun(t, `
	const x = 1
	for i := 0; i < 1; i++ {
		x, y := [2, 3]
		break
	}
	return x
	`, nil, Int(1))
	TestExpectRun(t, `
	const x = 1
	if [1] {
		x, y := [2, 3]
	}
	return x
	`, nil, Int(1))

	TestExpectRun(t, `
	return func() {
		const x = 1
		func() {
			x, y := [2, 3]
		}()
		return x
	}()
	`, nil, Int(1))
	TestExpectRun(t, `
	return func() {
		const x = 1
		for i := 0; i < 1; i++ {
			x, y := [2, 3]
			break
		}
		return x
	}()
	`, nil, Int(1))
	TestExpectRun(t, `
	return func(){
		const x = 1
		if [1] {
			x, y := [2, 3]
		}
		return x
	}()
	`, nil, Int(1))
	TestExpectRun(t, `
	return func(){
		const x = 1
		if [1] {
			var y
			x, y := [2, 3]
		}
		return x
	}()
	`, nil, Int(1))
}

func TestConstIota(t *testing.T) {
	TestExpectRun(t, `const x = iota; return x`, nil, Int(0))
	TestExpectRun(t, `const x = iota; const y = iota; return x, y`, nil, Array{Int(0), Int(0)})
	TestExpectRun(t, `const(x = iota, y = iota); return x, y`, nil, Array{Int(0), Int(1)})
	TestExpectRun(t, `const(x = iota, y); return x, y`, nil, Array{Int(0), Int(1)})

	TestExpectRun(t, `const(x = 1+iota, y); return x, y`, nil, Array{Int(1), Int(2)})
	TestExpectRun(t, `const(x = 1+iota, y=iota); return x, y`, nil, Array{Int(1), Int(1)})
	TestExpectRun(t, `const(x = 1+iota, y, z); return x, y, z`, nil, Array{Int(1), Int(2), Int(3)})
	TestExpectRun(t, `const(x = iota+1, y, z); return x, y, z`, nil, Array{Int(1), Int(2), Int(3)})

	TestExpectRun(t, `const(_ = iota+1, y, z); return y, z`, nil, Array{Int(2), Int(3)})

	TestExpectRun(t, `
	const (
		x = [iota]
	)
	return x`, nil, Array{Int(0)})

	TestExpectRun(t, `
	const (
		x = []
	)
	return x`, nil, Array{})

	TestExpectRun(t, `
	const (
		x = [iota, iota]
	)
	return x`, nil, Array{Int(0), Int(0)})

	TestExpectRun(t, `
	const (
		x = [iota, iota]
		y
	)
	return x, y`, nil, Array{Array{Int(0), Int(0)}, Array{Int(1), Int(1)}})

	TestExpectRun(t, `
	const (
		x = [iota, iota]
		y
		z
	)
	return x, y, z`, nil,
		Array{Array{Int(0), Int(0)}, Array{Int(1), Int(1)}, Array{Int(2), Int(2)}})

	TestExpectRun(t, `
	const (
		x = [iota, iota]
		y
	)
	x[0] = 2
	return x, y`, nil, Array{Array{Int(2), Int(0)}, Array{Int(1), Int(1)}})

	TestExpectRun(t, `
	const (
		x = {}
	)
	return x`, nil, Dict{})

	TestExpectRun(t, `
	const (
		x = {iota: 1}
	)
	return x`, nil, Dict{"iota": Int(1)})

	TestExpectRun(t, `
	const (
		x = {k: iota}
	)
	return x`, nil, Dict{"k": Int(0)})

	TestExpectRun(t, `
	const (
		x = {k: iota}
		y
	)
	return x, y`, nil, Array{Dict{"k": Int(0)}, Dict{"k": Int(1)}})

	TestExpectRun(t, `
	const (
		x = {k: iota}
		y
	)
	x["k"] = 2
	return x, y`, nil, Array{Dict{"k": Int(2)}, Dict{"k": Int(1)}})

	TestExpectRun(t, `
	const (
		x = {k: iota}
		y
		z
	)
	return x, y, z`, nil,
		Array{Dict{"k": Int(0)}, Dict{"k": Int(1)}, Dict{"k": Int(2)}})

	TestExpectRun(t, `
	const (
		_ = 1 << iota
		x
		y
	)
	return x, y`, nil, Array{Int(2), Int(4)})

	TestExpectRun(t, `
	const (
		x = 1 << iota
		_
		y
	)
	return x, y`, nil, Array{Int(1), Int(4)})

	TestExpectRun(t, `
	const (
		x = 1 << iota
		a
		y = a
		z
	)
	return x, y, z`, nil, Array{Int(1), Int(2), Int(2)})

	TestExpectRun(t, `
	const (
		x = 1 << iota
		_
		_
		z
	)
	return x, z`, nil, Array{Int(1), Int(8)})

	TestExpectRun(t, `
	iota := 1
	const (
		x = 1 << iota
	)
	return x, iota`, nil, Array{Int(2), Int(1)})

	TestExpectRun(t, `
	iota := 1
	const (
		x = 1 << iota
		y
	)
	return x, y`, nil, Array{Int(2), Int(2)})

	expectErrHas(t, `const iota = 1`,
		NewTestOpts().CompilerError(), "Compile Error: assignment to iota")

	expectErrHas(t, `const iota = iota + 1`,
		NewTestOpts().CompilerError(), "Compile Error: assignment to iota")

	expectErrHas(t, `
	const (
		x = 1 << iota
		iota
		y
	)
	return x, iota, y`,
		NewTestOpts().CompilerError(), "Compile Error: assignment to iota")

	expectErrHas(t, `const x = iota; return iota`,
		NewTestOpts().CompilerError(), `Compile Error: unresolved reference "iota"`)

	TestExpectRun(t, `
	const (
		x = iota
		y
	)
	iota := 3
	return x, y, iota`, nil, Array{Int(0), Int(1), Int(3)})

	TestExpectRun(t, `
	const (
		x = iota
		y
	)
	iota := 3
	const (
		a = 10+iota
		b
	)
	return x, y, iota, a, b`, nil, Array{Int(0), Int(1), Int(3), Int(13), Int(13)})

	TestExpectRun(t, `
	const (
		x = iota
		y
	)
	const (
		a = 10+iota
		b
	)
	return x, y, a, b`, nil, Array{Int(0), Int(1), Int(10), Int(11)})

	TestExpectRun(t, `
	const (
		x = func() { return 1 }()
		y
		z
	)
	return x, y, z`, nil, Array{Int(1), Int(1), Int(1)})

	TestExpectRun(t, `
	const (
		x = func(x) { return x }(iota)
		y
		z
	)
	return x, y, z`, nil, Array{Int(0), Int(1), Int(2)})

	TestExpectRun(t, `
	a:=0
	const (
		x = func() { a++; return a }()
		y
		z
	)
	return x, y, z`, nil, Array{Int(1), Int(2), Int(3)})

	TestExpectRun(t, `
	const (
		x = 1+iota
		y = func() { return 1+x }()
		z
	)
	return x, y, z`, nil, Array{Int(1), Int(2), Int(2)})

	TestExpectRun(t, `
	const (
		x = func() { return 1 }
		y
		z
	)
	return x(), y(), z()`, nil, Array{Int(1), Int(1), Int(1)})

	TestExpectRun(t, `
	const (
		x = func() { return 1 }
		y
		z
	)
	return x == y && y == z`, nil, True)

	TestExpectRun(t, `
	var a
	const (
		x = func() { return a }
		y
		z
	)
	return x != y && y != z`, nil, True)

	TestExpectRun(t, `
	return func() {
		const (
			x = 1 << iota
			_
			y
		)
		return x, y
	}()`, nil, Array{Int(1), Int(4)})

	TestExpectRun(t, `
	iota := 2
	return func() {
		const (
			x = 1 << iota
			_
			y
		)
		return x, y
	}()`, nil, Array{Int(4), Int(4)})

	TestExpectRun(t, `
	const (
		x = 1 + iota + func() { 
			const (
				_ = iota
				r
			)
			return r
		}()
		y
		_
	)
	return x,y`, nil, Array{Int(2), Int(3)})

	TestExpectRun(t, `
	const (x = iota%2?"odd":"even", y, z)
	return x,y,z`, nil, Array{Str("even"), Str("odd"), Str("even")})
}

func TestVM_Invoke(t *testing.T) {
	applyPool := &Function{
		Name: "applyPool",
		Value: func(c Call) (Object, error) {
			inv := NewInvoker(c.VM, c.Args.Shift())
			inv.Acquire()
			defer inv.Release()
			return inv.Invoke(c.Args, &c.NamedArgs)
		},
	}
	applyNoPool := &Function{
		Name: "applyNoPool",
		Value: func(c Call) (Object, error) {
			args := make([]Object, 0, c.Args.Length()-1)
			for i := 1; i < c.Args.Length(); i++ {
				args = append(args, c.Args.Get(i))
			}
			inv := NewInvoker(c.VM, c.Args.Get(0))
			return inv.Invoke(Args{args}, &c.NamedArgs)
		},
	}
	for _, apply := range []*Function{applyPool, applyNoPool} {
		t.Run(apply.Name, func(t *testing.T) {
			t.Run("apply", func(t *testing.T) {
				scr := `
global apply
sum := func(*args) {
	println("called f", args)
	s := 0
	for v in args {
		println("v", v)
		s += v
	}
	return s
}
return apply(sum, 1, 2, 3)
`
				TestExpectRun(t, scr,
					NewTestOpts().Globals(Dict{"apply": apply}),
					Int(6),
				)
			})

			t.Run("apply indirect", func(t *testing.T) {
				scr := `
global apply
sum := func(*args) {
	println("sum args", args)
	s := 0
	for v in args {
		println("v", v)
		s += v
	}
	return s
}
f := func(fn, *args) {
	return fn(*args)
}
return apply(f, sum, 1, 2, 3)
`
				TestExpectRun(t, scr,
					NewTestOpts().Globals(Dict{"apply": apply}),
					Int(6),
				)
			})

			t.Run("apply indirect 2", func(t *testing.T) {
				scr := `
global apply
sum := func(*args) {
	println("sum args", args)
	s := 0
	for v in args {
		println("v", v)
		s += v
	}
	return s
}
f := func(fn, *args) {
	return apply(fn, *args)
}
return apply(f, sum, 1, 2, 3)
`
				TestExpectRun(t, scr,
					NewTestOpts().Globals(Dict{"apply": apply}),
					Int(6),
				)
			})

			t.Run("apply go func", func(t *testing.T) {
				sum := &Function{
					Value: func(c Call) (Object, error) {
						s := Int(0)
						for i := 0; i < c.Args.Length(); i++ {
							s += c.Args.Get(i).(Int)
						}
						return s, nil
					},
				}
				scr := `
global (apply, sum)
return apply(sum, 1, 2, 3)
`
				TestExpectRun(t, scr,
					NewTestOpts().Globals(Dict{"apply": apply, "sum": sum}),
					Int(6),
				)
			})

			t.Run("module state", func(t *testing.T) {
				scr := `
module := import("module")
module.counter = 1

global apply

inc := func(a) {
	module := import("module")
	module.counter += a
}
apply(inc, 3)
return module.counter
`
				t.Run("builtin", func(t *testing.T) {
					TestExpectRun(t, scr,
						NewTestOpts().
							Globals(Dict{"apply": apply}).
							Module("module", Dict{}),
						Int(4),
					)
				})
				t.Run("source", func(t *testing.T) {
					TestExpectRun(t, scr,
						NewTestOpts().
							Globals(Dict{"apply": apply}).
							Module("module", `return {}`),
						Int(4),
					)
				})
			})

			t.Run("closure", func(t *testing.T) {
				scr := `
global apply

counter := 1
f1 := func(a) {
	counter += a
}

f2 := func(a) {
	counter += a
}
apply(f1, 3)
apply(f2, 5)
return counter
`
				TestExpectRun(t, scr,
					NewTestOpts().Globals(Dict{"apply": apply}),
					Int(9),
				)
			})

			t.Run("global", func(t *testing.T) {
				scr := `
global apply
global counter

f1 := func(a) {
	counter += a
}

f2 := func(a) {
	counter += a
}
apply(f1, 3)
apply(f2, 5)
return counter
`
				expected := Int(9)
				globals := Dict{"apply": apply, "counter": Int(1)}
				TestExpectRun(t, scr,
					NewTestOpts().Globals(globals).Skip2Pass(),
					expected,
				)
				if expected != globals["counter"] {
					t.Fatalf("expected %s, got %v", expected, globals["counter"])
				}
			})
		})
	}
}

type nameCaller struct {
	Dict
	counts map[string]int
}

func (n *nameCaller) CallName(name string, c Call) (Object, error) {
	fn := n.Dict[name]
	args := make([]Object, 0, c.Args.Length())
	for i := 0; i < c.Args.Length(); i++ {
		args = append(args, c.Args.Get(i))
	}
	ret, err := NewInvoker(c.VM, fn).Invoke(Args{args}, &c.NamedArgs)
	n.counts[name]++
	return ret, err
}

var _ NameCallerObject = &nameCaller{}

func TestVMCallName(t *testing.T) {
	newobject := func() *nameCaller {
		var f = &Function{
			Value: func(c Call) (Object, error) {
				return c.Args.Get(0).(Int) + 1, nil
			},
		}

		return &nameCaller{Dict: Dict{"add1": f}, counts: map[string]int{}}
	}
	scr := `
global object

object.sub1 = func(a) {
	return a - 1
}

return [object.add1(10), object.sub1(10)]
`

	t.Run("basic", func(t *testing.T) {
		TestExpectRun(t, scr,
			NewTestOpts().Globals(Dict{"object": newobject()}),
			Array{Int(11), Int(9)},
		)
	})

	t.Run("counts single pass", func(t *testing.T) {
		object := newobject()
		TestExpectRun(t, scr,
			NewTestOpts().Globals(Dict{"object": object}).Skip2Pass(),
			Array{Int(11), Int(9)},
		)
		if object.counts["add1"] != 1 {
			t.Fatalf("expected 1, got %d", object.counts["add1"])
		}
		if object.counts["sub1"] != 1 {
			t.Fatalf("expected 1, got %d", object.counts["sub1"])
		}
	})

	t.Run("counts all pass", func(t *testing.T) {
		object := newobject()
		TestExpectRun(t, scr,
			NewTestOpts().Globals(Dict{"object": object}),
			Array{Int(11), Int(9)},
		)
		if object.counts["add1"] <= 0 {
			t.Fatalf("expected >0, got %d", object.counts["add1"])
		}
		if object.counts["sub1"] <= 0 {
			t.Fatalf("expected >0, got %d", object.counts["sub1"])
		}
	})
}
