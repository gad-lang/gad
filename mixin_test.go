package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// TestMixinUseMergesMembers verifies that `use` pulls a mixin's field, property
// and method into the class as its own, and that a mixin field default can be
// overridden by a value passed at construction.
func TestMixinUseMergesMembers(t *testing.T) {
	testExpectRun(t, `
		mixin Counter {
			count = 0
			props { doubled => this.count * 2 }
			methods { inc() { this.count += 1 } }
		}
		class Widget { use Counter; name = "w" }
		w := Widget(; name="box", count=3)
		w.inc()
		return [w.count, w.name, w.doubled]`,
		// count 3 -> inc -> 4; doubled = 4*2 = 8.
		nil, Array{Int(4), Str("box"), Int(8)})
}

// TestMixinFieldsInitFirst verifies that mixin field defaults are initialised
// before the class's own, in declaration order — observed here through the order
// a shared log records each non-literal default's evaluation.
func TestMixinFieldsInitFirst(t *testing.T) {
	testExpectRun(t, `
		log := []
		mark := func(s) { log = log + [s]; return s }
		mixin M { a = mark("a"); b = mark("b") }
		class C { use M; z = mark("z") }
		C()
		return log`,
		nil, Array{Str("a"), Str("b"), Str("z")})
}

// TestMixinFieldsInitParentsFirst verifies that when a used mixin extends parent
// mixins, the parents' field defaults initialise before the child mixin's, and
// all mixin defaults before the class's own.
func TestMixinFieldsInitParentsFirst(t *testing.T) {
	testExpectRun(t, `
		log := []
		mark := func(s) { log = log + [s]; return s }
		mixin P { p = mark("p") }
		mixin Q { *P; q = mark("q") }
		class C { use Q; z = mark("z") }
		C()
		return log`,
		nil, Array{Str("p"), Str("q"), Str("z")})
}

// TestMixinFieldsMembership verifies a using class declares the mixin's fields as
// its own (order-independent membership, since @fields is a dict).
func TestMixinFieldsMembership(t *testing.T) {
	testExpectRun(t, `
		mixin M { a = 1; b = 2 }
		class C { use M; z = 3 }
		return sort(collect(keys(C.@fields)))`,
		nil, Array{Str("a"), Str("b"), Str("z")})
}

// TestMixinThisInterfaceTyping verifies that a mixin method's injected `this` is
// typed by the `this { … }` interface, so it can call a member the interface
// declares — resolved on the final class instance.
func TestMixinThisInterfaceTyping(t *testing.T) {
	testExpectRun(t, `
		mixin A {
			this { count() <int> }
			methods { plus2() => this.count() + 2 }
		}
		class C { use A; methods { count() => 5 } }
		return C().plus2()`,
		nil, Int(7))
}

// TestMixinParents verifies that a mixin can extend parent mixins (`*A`), and a
// class that uses it gains the parents' members too.
func TestMixinParents(t *testing.T) {
	testExpectRun(t, `
		mixin A { a = 1 }
		mixin B { b = 10 }
		mixin SubA { *A; *B }
		class C { use SubA }
		c := C()
		return [c.a, c.b, len(SubA.@parents)]`,
		nil, Array{Int(1), Int(10), Int(2)})
}

// TestMixinDedupFirstWins verifies that a mixin appearing more than once across
// the use list and hierarchy is merged only once (no duplicate-member error);
// the first occurrence wins.
func TestMixinDedupFirstWins(t *testing.T) {
	testExpectRun(t, `
		mixin A { a = 1 }
		mixin SubA { *A }
		class C { use SubA, A }
		return [C().a, len(C.@mixins)]`,
		nil, Array{Int(1), Int(2)})
}

// TestMixinAnonymous verifies the anonymous expression form `const M = mixin { … }`.
func TestMixinAnonymous(t *testing.T) {
	testExpectRun(t, `
		const M = mixin { x = 7 }
		class D { use M }
		return D().x`,
		nil, Int(7))
}

// TestMixinInitFieldsPerInstance verifies that a mixin field's non-literal default
// is evaluated per instance (a fresh composite), like a class field default.
func TestMixinInitFieldsPerInstance(t *testing.T) {
	testExpectRun(t, `
		mixin M { items = [1, 2] }
		class C { use M }
		a := C(); b := C()
		a.items[0] = 99
		return [a.items[0], b.items[0]]`,
		nil, Array{Int(99), Int(1)})
}

// TestMixinTypeCheck verifies that a mixin value is a `gad.Mixin`.
func TestMixinTypeCheck(t *testing.T) {
	testExpectRun(t, `
		mixin A {}
		return bool(A :: gad.Mixin)`,
		nil, True)
}

// TestMixinInterfaceAttr verifies that `@interface` returns a cached Interface
// instance named `Name$interface` in the mixin's module, mirroring the mixin's
// declared members (field with its type, getter/setter/prop, method).
func TestMixinInterfaceAttr(t *testing.T) {
	// It is an Interface value, cached (same instance across reads).
	testExpectRun(t, `
		mixin A { methods { run() => 1 } }
		return [bool(A.@interface :: gad.Interface), A.@interface == A.@interface]`,
		nil, Array{True, True})

	// Its rendering mirrors the declared members: a typed field, a getter-only
	// property (`get p`), and a method.
	testExpectRun(t, `
		mixin A { f int = 1; props { p => 2 }; methods { run() => 1 } }
		return str(A.@interface)`,
		nil, Str("interface (main).A$interface {*(main).A$class; *(main).A$members}"))
}

// TestMixinThisAttr verifies `@this` returns the declared `this { … }` interface,
// or nil when the mixin has no `this` block.
func TestMixinThisAttr(t *testing.T) {
	testExpectRun(t, `
		mixin A { this { size() <int> } }
		return [bool(A.@this :: gad.Interface), str(A.@this)]`,
		nil, Array{True, Str("interface (main).A$this {size()}")})

	testExpectRun(t, `
		mixin B { x = 1 }
		return B.@this == nil`,
		nil, True)
}

// TestMixinInterfaceExtendsThisAndParents verifies `@interface` extends the `this`
// interface and each parent mixin's `@interface`: a value satisfies it only when
// it satisfies its own members, its `this` requirement, and its parents'.
func TestMixinInterfaceExtendsThisAndParents(t *testing.T) {
	base := `
		mixin Named { name = "?" }
		mixin Sized {
			*Named
			this { size() <int> }
			methods { area() => this.size() * this.size() }
		}`
	// A class that uses Sized and provides size() satisfies Sized.@interface (own
	// area + parent Named's name + this-interface size()).
	testExpectRun(t, base+`
		class Box { use Sized; methods { size() => 3 } }
		return bool(Box(; name="b") :: Sized.@interface)`,
		nil, True)

	// Missing size() -> fails the extended `this` interface.
	testExpectRun(t, base+`
		class NoSize { use Named }
		return bool(NoSize(; name="x") :: Sized.@interface or false)`,
		nil, False)

	// The rendering shows the extends spreads.
	testExpectRun(t, base+`return str(Sized.@interface)`,
		nil, Str("interface (main).Sized$interface {*(main).Sized$class; *(main).Sized$members}"))
}

// TestMixinClassInterfaceAndMembers verifies the interface decomposition:
// `@classInterface` (the using-class contract: this-block + parents) and
// `@membersInterface` (the mixin's own members).
func TestMixinClassInterfaceAndMembers(t *testing.T) {
	testExpectRun(t, `
		mixin Named { name = "?" }
		mixin Sized { *Named; this { size() <int> }; methods { area() => 1 } }
		return [str(Sized.@classInterface), str(Sized.@membersInterface)]`,
		nil, Array{
			Str("interface (main).Sized$class {*(main).Sized$this; *(main).Named$interface}"),
			Str("interface (main).Sized$members {area()}"),
		})
}

// TestMixinContractValidation verifies that a class using a mixin is rejected at
// definition when it does not satisfy the mixin's `@classInterface` (its `this`
// block), and accepted when it provides the required members.
func TestMixinContractValidation(t *testing.T) {
	// A class providing the required `size()` is accepted.
	testExpectRun(t, `
		mixin Sized { this { size() <int> }; methods { area() => this.size() } }
		class Box { use Sized; methods { size() => 4 } }
		return Box().area()`,
		nil, Int(4))

	// A class missing `size()` is rejected at definition.
	expectErrHas(t, `
		mixin Sized { this { size() <int> }; methods { area() => this.size() } }
		class Bad { use Sized }`,
		nil, `does not satisfy mixin "Sized": missing "size"`)

	// The requirement may be satisfied by another used mixin's member, not only an
	// own method: Provider supplies `size`, Sized requires it via `this`.
	testExpectRun(t, `
		mixin Provider { methods { size() => 3 } }
		mixin Sized { this { size() <int> }; methods { area() => this.size() } }
		class Box { use Sized, Provider }
		return Box().area()`,
		nil, Int(3))
}

// TestMixinInterfaceDedupAndCollision verifies interface-graph flattening dedups
// by interface (a diamond `*A` + `*B(extends A)` reaches A once) but rejects one
// member name declared in two DIFFERENT interfaces (potentially clashing signatures).
func TestMixinInterfaceDedupAndCollision(t *testing.T) {
	// Diamond: C reaches A directly and through B — A's `foo` counted once, no error.
	testExpectRun(t, `
		mixin A { methods { foo() => 1 } }
		mixin B { *A }
		mixin C { *A; *B; methods { bar() => 2 } }
		class Ok { use C }
		return [Ok().foo(), Ok().bar()]`,
		nil, Array{Int(1), Int(2)})

	// Two distinct interfaces declaring `foo` -> rejected at the mixin's definition.
	expectErrHas(t, `
		mixin A { this { foo() <int> } }
		mixin B { this { foo() <str> } }
		mixin C { *A; *B }`,
		nil, `member "foo" is declared in two different interfaces`)
}

// TestInterfaceFlatten verifies `iface.@flat`: the extends graph flattened into a
// single interface with all members (dedup by interface), cached.
func TestInterfaceFlatten(t *testing.T) {
	testExpectRun(t, `
		mixin Named { name = "?" }
		mixin Sized { *Named; this { size() <int> }; methods { area() => 1 } }
		f := Sized.@interface.@flat
		return [str(f), f == Sized.@interface.@flat]`,
		nil, Array{Str("interface (main).Sized$interface {name; size(); area()}"), True})
}

// TestMixinReflectionAttrs verifies the mixin reflection attributes mirror class.
func TestMixinReflectionAttrs(t *testing.T) {
	testExpectRun(t, `
		mixin A {
			f = 1
			props { p => 1 }
			methods { m() => 1 }
		}
		return [A.@name, collect(keys(A.@fields)), collect(keys(A.@props)), collect(keys(A.@methods))]`,
		nil, Array{Str("A"), Array{Str("f")}, Array{Str("p")}, Array{Str("m")}})
}
