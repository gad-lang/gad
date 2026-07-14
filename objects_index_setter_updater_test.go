package gad

import (
	"testing"
)

// TestIndexSetterUpdaterImplementers asserts, at compile time and at runtime,
// that every ToDictConverter also implements IndexSetterUpdater and that both
// methods agree: filling a fresh Dict through UpdateIndexSetter yields the same
// entries as ToDict.
func TestIndexSetterUpdaterImplementers(t *testing.T) {
	// Compile-time guarantee: each ToDictConverter is also an
	// IndexSetterUpdater. If any implementer regresses, this block fails to
	// build.
	var (
		_ interface {
			ToDictConverter
			IndexSetterUpdater
		} = (*Enum)(nil)
		_ interface {
			ToDictConverter
			IndexSetterUpdater
		} = (*EmbeddedNodeFS)(nil)
		_ interface {
			ToDictConverter
			IndexSetterUpdater
		} = (*ClassInstance)(nil)
		_ interface {
			ToDictConverter
			IndexSetterUpdater
		} = (*Module)(nil)
		_ interface {
			ToDictConverter
			IndexSetterUpdater
		} = Dict(nil)
		_ interface {
			ToDictConverter
			IndexSetterUpdater
		} = KeyValueArray(nil)
		_ interface {
			ToDictConverter
			IndexSetterUpdater
		} = (*MixedParams)(nil)
	)

	type converter interface {
		ToDictConverter
		IndexSetterUpdater
	}

	build := func(c converter) Dict {
		out := Dict{}
		c.UpdateIndexSetter(out)
		return out
	}

	t.Run("Enum", func(t *testing.T) {
		e := NewEnum("Perm", nil)
		e.AddValue("Read", Uint(1))
		e.AddValue("Write", Uint(2))
		e.AddValue("Exec", Int(10))
		assertSameDict(t, e.ToDict(), build(e))
	})

	t.Run("EmbeddedNodeFS", func(t *testing.T) {
		node := &Embedded{
			Name: "root",
			Entries: map[string]*Embedded{
				"a.txt": {Name: "a.txt"},
				"b.txt": {Name: "b.txt"},
			},
		}
		fs := &EmbeddedNodeFS{node: node}
		assertSameDict(t, fs.ToDict(), build(fs))
	})

	t.Run("ClassInstance", func(t *testing.T) {
		inst := &ClassInstance{
			class:  &Class{},
			fields: Dict{"x": Int(1), "y": Str("two")},
		}
		assertSameDict(t, inst.ToDict(), build(inst))
	})

	t.Run("Dict", func(t *testing.T) {
		d := Dict{"a": Int(1), "b": Int(2)}
		assertSameDict(t, d.ToDict(), build(d))
	})

	t.Run("KeyValueArray", func(t *testing.T) {
		kva := KeyValueArray{
			{K: Str("a"), V: Int(1)},
			{K: Str("b"), V: Int(2)},
		}
		assertSameDict(t, kva.ToDict(), build(kva))
	})
}

func assertSameDict(t *testing.T, want, got Dict) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("length mismatch: want %d (%v), got %d (%v)", len(want), want, len(got), got)
	}
	for k, wv := range want {
		gv, ok := got[k]
		if !ok {
			t.Fatalf("missing key %q in %v", k, got)
		}
		if !wv.Equal(gv) {
			t.Fatalf("value mismatch for key %q: want %v, got %v", k, wv, gv)
		}
	}
}
