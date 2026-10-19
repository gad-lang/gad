package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// `type<class>`, `type<enum>`, `type<interface>`, `type<typedArray>`: a type
// value of the category — any class, any enum… —, never an instance.
func TestMetaTypeKinds(t *testing.T) {
	const decls = `class A {}; enum E { x }; type ints []int; interface I { a int }
	ok := func(f) { try { f(); return true } catch { return false } }
	`
	testExpectRun(t, decls+`
	gc := (v type<class>) => 1; ge := (v type<enum>) => 1; gi := (v type<interface>) => 1; gt := (v type<typedArray>) => 1
	return [ok(() => gc(A)), ok(() => gc(E)), ok(() => gc(A())), ok(() => gc(int)),
		ok(() => ge(E)), ok(() => ge(A)), ok(() => gi(I)), ok(() => gi(A)), ok(() => gt(ints)), ok(() => gt(A))]`,
		nil, Array{True, False, False, False, True, False, True, False, True, False})
	// a union of categories, of a category and a type
	testExpectRun(t, decls+`g := (v type<class|enum>) => 1; return [ok(() => g(A)), ok(() => g(E)), ok(() => g(I))]`,
		nil, Array{True, True, False})
	// overloads told apart by the category (checked by value, as structural types)
	testExpectRun(t, decls+`d := func { (v type<class>) => "class"; (v type<enum>) => "enum"; (v int) => "int" }; return [d(A), d(E), d(1)]`,
		nil, Array{Str("class"), Str("enum"), Str("int")})
	// a variable of that name says otherwise: the name is the variable's
	testExpectRun(t, `class enumX {}; typedArray := enumX; f := (v type<typedArray>) => 1; return f(enumX)`, nil, Int(1))
	// reflection: the name
	testExpectRun(t, `f := (v type<class>) => 1; return str(f)`, nil, Str("‹compiledFunction: (main).#1(v type<class>)›"))
}

// `type<X>` in a closure reads X where the closure was made.
func TestMetaTypeInClosure(t *testing.T) {
	testExpectRun(t, `class A {}; enum E { x }
	ok := func(f) { try { f(); return true } catch { return false } }
	k := (T) => { return (v type<T>) => 1 }
	return [ok(() => k(A)(A)), ok(() => k(A)(E)), ok(() => k(E)(E))]`,
		nil, Array{True, False, True})
}
