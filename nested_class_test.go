package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// TestNestedClassInitialization covers class fields typed by another class: a
// plain dict passed for such a field is constructed into an instance of that
// class (github.com/gad-lang/gad issue #4), recursively and at any depth.
func TestNestedClassInitialization(t *testing.T) {
	base := "class Point { x int; y int }\nclass Rect { a Point; b Point }\n"

	// A dict passed for a class-typed field becomes an instance of that class.
	testExpectRun(t, base+`return typeName(Rect(;a={x:0,y:0}, b={x:1,y:1}).a)`,
		nil, Str("Point"))
	// The constructed instance carries the dict's values.
	testExpectRun(t, base+`r := Rect(;a={x:3,y:4}, b={x:1,y:1}); return r.a.x + r.a.y`,
		nil, Int(7))
	// Both fields are converted.
	testExpectRun(t, base+`r := Rect(;a={x:0,y:0}, b={x:1,y:1}); return typeName(r.b)`,
		nil, Str("Point"))

	// Depth 3: Scene -> Rect -> Point, all from nested dict literals.
	deep := base + "class Scene { name str; rect Rect }\n"
	testExpectRun(t, deep+`
		s := Scene(;name="s1", rect={a: {x:1, y:2}, b: {x:3, y:4}})
		return typeName(s.rect) + "/" + typeName(s.rect.a) + "/" + str(s.rect.a.y)`,
		nil, Str("Rect/Point/2"))

	// An already-constructed instance is stored as-is, not re-wrapped.
	testExpectRun(t, base+`
		p := Point(;x=9, y=9)
		return Rect(;a=p, b={x:1,y:1}).a.x`, nil, Int(9))

	// A field that is not provided defaults to nil (not coerced).
	testExpectRun(t, base+`return Rect(;a={x:1,y:2}).b == nil`, nil, True)
}

// TestClassFieldTypeChecking covers the rule that a typed field rejects a value
// of a different type, using the same value-based assignability as parameter
// types; an untyped field accepts anything.
func TestClassFieldTypeChecking(t *testing.T) {
	// A matching primitive is accepted.
	testExpectRun(t, `class P { x int; y int }; return P(;x=1, y=2).x`, nil, Int(1))

	// A mismatching primitive is rejected with a TypeError naming the field.
	expectErrHas(t, `class P { x int; y int }; return P(;x="hi", y=2)`,
		nil, `field "x" expects int, got str`)
	expectErrHas(t, `class P { name str }; return P(;name=3)`,
		nil, `field "name" expects str, got int`)

	// A union type accepts any of its members and rejects the rest.
	testExpectRun(t, `class B { v int|str }; return B(;v="a").v`, nil, Str("a"))
	testExpectRun(t, `class B { v int|str }; return B(;v=7).v`, nil, Int(7))
	expectErrHas(t, `class B { v int|str }; return B(;v=true)`,
		nil, `field "v" expects int|str, got bool`)

	// An untyped field accepts anything.
	testExpectRun(t, `class Bag { any }; return typeName(Bag(;any="x").any)`, nil, Str("str"))
	testExpectRun(t, `class Bag { any }; return typeName(Bag(;any=1).any)`, nil, Str("int"))

	// A class-typed field still rejects a non-dict of the wrong type.
	expectErrHas(t, `class Point { x int; y int }; class R { a Point }; return R(;a=5)`,
		nil, `field "a" expects Point, got int`)
}

// TestClassFieldInterfaceType covers a field typed by an interface: a value is
// accepted when it structurally satisfies the interface (an instance or a dict)
// and rejected otherwise — the same satisfaction check the `::` operator and
// interface-typed parameters use.
func TestClassFieldInterfaceType(t *testing.T) {
	base := `
	interface Greeter { name str; greet() <str> }
	class Person { name = ""; methods { greet() => "hi " + this.name } }
	class Registry { main Greeter }
	`
	// A class instance that satisfies the interface is accepted.
	testExpectRun(t, base+`return Registry(;main=Person(;name="Ada")).main.greet()`,
		nil, Str("hi Ada"))
	// A dict that structurally satisfies it is accepted too.
	testExpectRun(t, base+`return Registry(;main={name:"Bo", greet: func() => "hi Bo"}).main.greet()`,
		nil, Str("hi Bo"))
	// A value that does not satisfy the interface is rejected.
	expectErrHas(t, base+`return Registry(;main=42)`, nil, `field "main" expects`)
}
