package gad_test

import (
	"testing"

	"github.com/gad-lang/gad"
	"github.com/gad-lang/gad/parser/test"
)

// `** EXPR` in a class body adds the members EXPR gives at run time, after the
// declared ones: fields (a name to a default, or to a spec with types,
// nullable and meta — a dict in the order of its keys, a key-value array in
// its own), methods and props.
func TestClassSpreadMembers(t *testing.T) {
	src := `extra := {fields: (; columns=3, title=(; types=[str], nullable=true, meta=(; label="Título")))}
class Options {
    name str = "x"
    ** extra
    ** {methods: [func hello(this) { return "hi " + this.name }]}
}
return Options`
	st := gad.NewSymbolTable(gad.NewBuiltins().NameSet)
	res, err := gad.Compile(st, []byte(src), gad.CompileOptions{})
	if err != nil {
		t.Fatal(err)
	}
	ret, err := gad.NewVM(gad.NewBuiltins().Build(), res.Bytecode).Run()
	if err != nil {
		t.Fatal(err)
	}
	c := ret.(*gad.Class)
	var names []string
	for _, f := range c.RawFields() {
		names = append(names, f.Name)
	}
	if want := "name,columns,title"; join(names) != want {
		t.Errorf("fields %s, want %s", join(names), want)
	}
	title := c.RawFields()[2]
	if !title.Nullable || len(title.Types) != 1 || title.Meta == nil {
		t.Errorf("title: %+v", title)
	}
	if c.RawFields()[1].Value != gad.Int(3) {
		t.Errorf("columns: %v", c.RawFields()[1].Value)
	}
}

// A `** EXPR` item is written back as it was.
func TestFormatClassSpread(t *testing.T) {
	test.New(t, `class O { a str; ** extra }`).
		Code(`class O {a str; **extra}`).
		FormattedCode("class O {\n\ta str\n\t**extra\n}")
}

func join(s []string) (out string) {
	for i, v := range s {
		if i > 0 {
			out += ","
		}
		out += v
	}
	return
}
