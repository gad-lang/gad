package gad_test

import (
	"strings"
	"testing"

	"github.com/gad-lang/gad"
)

// ArrayOf/DictOf parameters get the gad collection itself (its typed checks
// applied); plain slice and map parameters get a copy synchronized back.
func TestGoCollections(t *testing.T) {
	g := gad.Dict{
		"double": gad.MustNewReflectValue(func(a *gad.ArrayOf[int]) error {
			for i := 0; i < a.Len(); i++ {
				v, err := a.Get(i)
				if err != nil {
					return err
				}
				if err = a.Set(i, v*2); err != nil {
					return err
				}
			}
			return nil
		}),
		"push": gad.MustNewReflectValue(func(a gad.ArrayOf[int], v int) error { return a.Append(v) }),
		"bump": gad.MustNewReflectValue(func(d *gad.DictOf[string, uint]) error {
			n, _, err := d.Get("n")
			if err != nil {
				return err
			}
			if err = d.Set("n", n+1); err != nil {
				return err
			}
			return d.Delete("old")
		}),
		"byID": gad.MustNewReflectValue(func(d gad.DictOf[int, string]) (string, error) {
			v, _, err := d.Get(2)
			if err != nil {
				return "", err
			}
			keys, err := d.Keys()
			return v + "/" + strings.Repeat("k", len(keys)), err
		}),
		"setFirst": gad.MustNewReflectValue(func(xs []int) { xs[0] = 99; _ = append(xs, 1) }),
		"addKey":   gad.MustNewReflectValue(func(m map[string]int) { m["new"] = 1; delete(m, "a") }),
		"types": gad.MustNewReflectValue(func(d gad.DictOf[string, int]) string {
			return d.Key.(gad.ObjectType).Name() + "," + d.Value.(gad.ObjectType).Name()
		}),
	}
	ret, err := runReflect(t, `global (double, push, bump, byID, setFirst, addKey, types)
a := [1, 2]; double(a)
b := [1]; push(&b, 5)
d := {n: 1u, old: true}; bump(d)
type ints []int
x := ints(3); push(x, 4); double(x)
type scores dict[str, int]
s := scores(; a=1)
c := [1, 2]; setFirst(c)
e := {a: 1}; addKey(e)
return [a, b, d, typeName(d.n), x, byID({"1": "a", "2": "b"}), c, e, types(s)]`, g)
	if err != nil {
		t.Fatal(err)
	}
	want := `[[2, 4], [1, 5], {n: 2u}, "uint", ints(6, 8), "b/kk", [99, 2], {new: 1}, "str,int"]`
	if got := ret.ToString(); got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}

	// an array given by value cannot grow; a typed array checks its items
	for src, msg := range map[string]string{
		`global push; a := [1]; push(a, 2)`:                      "cannot grow",
		`global double; type strs []str; double(strs("a"))`:      "should be int",
		`global bump; type sc dict[str, str]; bump(sc(; n="x"))`: "should be uint",
	} {
		_, err := runReflect(t, src, g)
		if err == nil || !strings.Contains(err.Error(), msg) {
			t.Errorf("%s: want %q, got %v", src, msg, err)
		}
	}
}
