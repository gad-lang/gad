package gad_test

import (
	"testing"

	. "github.com/gad-lang/gad"
)

// A field typed by an anonymous class (`a class { … }`) holds a record of its
// own: a dict given for it is constructed into the class, recursively; its
// fields keep their defaults and types; one not given is nil.
func TestVMFieldClass(t *testing.T) {
	base := "class O { page class { header class { title str = \"Home\"; sticky bool = false } }; theme str = \"light\" }\n"

	testExpectRun(t, base+`o := O(; page={header: {sticky: true}}); return [o.page.header.title, o.page.header.sticky, o.theme]`,
		nil, Array{Str("Home"), True, Str("light")})
	// named by its path from the outermost class
	testExpectRun(t, base+`o := O(; page={header: {}}); return [typeName(o.page), typeName(o.page.header)]`,
		nil, Array{Str("O.page"), Str("O.page.header")})
	// not given: nil; given empty: its own defaults
	testExpectRun(t, base+`return [O().page, O(; page={}).page.header, O(; page={header: {}}).page.header.title]`,
		nil, Array{Nil, Nil, Str("Home")})
	// a typed field of it rejects a value of another type, by name
	expectErrHas(t, base+`return O(; page={header: {title: 1}})`, nil, `field "title" expects str, got int`)
	// nullable: `a? class { … }` accepts nil
	testExpectRun(t, `class N { a? class { b int = 1 } }; return N(; a=nil).a`, nil, Nil)
	// an instance given is kept
	testExpectRun(t, base+`o := O(; page={header: {}}); return O(; page=o.page).page.header.title`, nil, Str("Home"))
	// an anonymous outermost class has a generated name, `#N`, before the path
	testExpectRun(t, `c := class { db class { port int = 1 } }; n := typeName(c(; db={}).db); return [n[:1], strings.hasSuffix(n, ".db")]`,
		nil, Array{Str("#"), True})
	// the class of a field in a class of a field: named all the way down
	testExpectRun(t, `class A { b class { c class { d class { e int = 5 } } } }; x := A(; b={c: {d: {}}}); return [typeName(x.b.c.d), x.b.c.d.e]`,
		nil, Array{Str("A.b.c.d"), Int(5)})
}

// `**Expr` adds the members EXPR gives at run time, after the declared ones:
// fields — a name to a default, or to a spec (types, nullable, meta, default)
// —, methods and props; several apply in order; nil adds nothing.
func TestVMClassSpread(t *testing.T) {
	// fields: a dict, by the order of its names, after the declared ones
	testExpectRun(t, `class G { name str = "g"; **{fields: {rows: 2, cols: 3}} }
		g := G(; rows=4); return [g.name, g.cols, g.rows]`,
		nil, Array{Str("g"), Int(3), Int(4)})
	// a spec: types enforced, nullable, default
	testExpectRun(t, `class B { **{fields: (; title=(; types=[str], nullable=true), size=(; types=[int], default=10))} }
		return [B().title, B().size, B(; size=2).size]`,
		nil, Array{Nil, Int(10), Int(2)})
	expectErrHas(t, `class B { **{fields: (; size=(; types=[int]))} }; return B(; size="x")`,
		nil, `field "size" expects int, got str`)
	// methods and props
	testExpectRun(t, `class C { n int = 0; **{methods: [twice(this) => this.n * 2]}; **{props: {label: func { (this) => "n" + str(this.n) }}} }
		c := C(; n=21); return [c.twice(), c.label]`,
		nil, Array{Int(42), Str("n21")})
	// built at run time, from what a loop found
	testExpectRun(t, `mods := {pt: 1, en: 2}; f := {}; for k, _ in mods { f[k] = (; types=[bool], default=false) }
		class L { **{fields: f} }; l := L(; pt=true); return [l.en, l.pt]`,
		nil, Array{False, True})
	// nil adds nothing
	testExpectRun(t, `class P { a = 1; **nil }; return P().a`, nil, Int(1))
	// anything else is an error
	expectErrHas(t, `class P { ** 1 }; return P`, nil, `**Expr expects a dict of fields, methods and props, got int`)
	expectErrHas(t, `class P { **{fields: (; x=(; types=[1]))} }; return P`, nil, `field "x": int is not a type`)
	// in an anonymous class too
	testExpectRun(t, `c := class { **{fields: {v: 7}} }; return c().v`, nil, Int(7))
}

// The fields a class gives at run time are its fields as the declared ones
// are: in RawFields, after them, with their types, nullability and metadata.
func TestVMClassSpreadRawFields(t *testing.T) {
	src := `class O {
    name str = "x"
    **{fields: (; columns=3, title=(; types=[str], nullable=true, meta=(; label="Título")))}
    **{fields: {z: 1, a: 2}}
}
return O`
	st := NewSymbolTable(NewBuiltins().NameSet)
	res, err := Compile(st, []byte(src), CompileOptions{})
	if err != nil {
		t.Fatal(err)
	}
	ret, err := NewVM(NewBuiltins().Build(), res.Bytecode).Run()
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, f := range ret.(*Class).RawFields() {
		names = append(names, f.Name)
	}
	want := []string{"name", "columns", "title", "a", "z"}
	if len(names) != len(want) {
		t.Fatalf("fields %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("fields %v, want %v", names, want)
		}
	}
	title := ret.(*Class).RawFields()[2]
	if !title.Nullable || len(title.Types) != 1 || title.Meta == nil {
		t.Errorf("title: %+v", title)
	}
	if v := ret.(*Class).RawFields()[1].Value; v != Int(3) {
		t.Errorf("columns: %v", v)
	}
}
